// Package rdf reads and writes PasswordMaker profiles in the RDF format
// used by the PasswordMaker Pro Chrome extension's import/export feature
// (and, for reading, the original Firefox extension).
package rdf

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/abudden/pmtray/internal/pwm"
)

const (
	aboutDefaults = "http://passwordmaker.mozdev.org/defaults"
	aboutGlobal   = "http://passwordmaker.mozdev.org/globalSettings"
	aboutRemotes  = "http://passwordmaker.mozdev.org/remotes"
)

// Firefox names for hash algorithms that differ from the Chrome names.
var ffToChromeHash = map[string]string{
	"hmac-sha256-fixed": "hmac-sha256_fix",
	"md5-v0.6":          "md5_v6",
	"hmac-md5-v0.6":     "hmac-md5_v6",
}

type description struct {
	attrs    map[string]string // keyed by local name
	children map[string]string // multi-line values stored as child elements
}

func (d description) get(name string) (string, bool) {
	if v, ok := d.children[name]; ok {
		return v, true
	}
	v, ok := d.attrs[name]
	return v, ok
}

func parseDescriptions(data []byte) ([]description, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = false
	var out []description
	var cur *description
	var child string
	var text strings.Builder
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if cur == nil && t.Name.Local == "Description" {
				d := description{attrs: map[string]string{}, children: map[string]string{}}
				for _, a := range t.Attr {
					d.attrs[a.Name.Local] = a.Value
				}
				out = append(out, d)
				cur = &out[len(out)-1]
				child = ""
			} else if cur != nil && child == "" {
				child = t.Name.Local
				text.Reset()
			}
		case xml.CharData:
			if child != "" {
				text.Write(t)
			}
		case xml.EndElement:
			if cur == nil {
				continue
			}
			if child != "" && t.Name.Local == child {
				cur.children[child] = text.String()
				child = ""
			} else if t.Name.Local == "Description" {
				cur = nil
			}
		}
	}
	return out, nil
}

func strToBool(v string) bool {
	return strings.Contains(strings.ToLower(v), "true")
}

func parseInt(v string) int {
	m := regexp.MustCompile(`^\s*[-+]?\d+`).FindString(v)
	n, _ := strconv.Atoi(strings.TrimSpace(m))
	return n
}

var patternAttr = regexp.MustCompile(`^pattern(|type|enabled)(\d+)$`)

// siteList builds the space separated site list from patternN attributes.
func siteList(d description) string {
	type pat struct{ value, typ, enabled string }
	pats := map[int]*pat{}
	maxIdx := -1
	for name, v := range d.attrs {
		m := patternAttr.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		idx, _ := strconv.Atoi(m[2])
		if pats[idx] == nil {
			pats[idx] = &pat{}
		}
		switch m[1] {
		case "":
			pats[idx].value = v
		case "type":
			pats[idx].typ = v
		case "enabled":
			pats[idx].enabled = v
		}
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	var parts []string
	for i := 0; i <= maxIdx; i++ {
		p := pats[i]
		if p == nil || p.enabled != "true" {
			continue
		}
		if p.typ == "regex" {
			parts = append(parts, "/"+p.value+"/")
		} else {
			parts = append(parts, p.value)
		}
	}
	return strings.Join(parts, " ")
}

// applyAttrs sets the profile fields present in d.
func applyAttrs(p *pwm.Profile, d description) {
	str := func(name string, dst *string) {
		if v, ok := d.get(name); ok {
			*dst = v
		}
	}
	num := func(name string, dst *int) {
		if v, ok := d.get(name); ok {
			*dst = parseInt(v)
		}
	}
	boolean := func(name string, dst *bool) {
		if v, ok := d.get(name); ok {
			*dst = strToBool(v)
		}
	}
	str("name", &p.Title)
	if v, ok := d.get("timestamp"); ok {
		p.Timestamp = int64(parseInt(v))
	}
	str("urlToUse", &p.StrUseText)
	str("whereLeetLB", &p.WhereToUseL33t)
	num("leetLevelLB", &p.L33tLevel)
	if v, ok := d.get("hashAlgorithmLB"); ok {
		if c, ok := ffToChromeHash[v]; ok {
			v = c
		}
		p.HashAlgorithm = v
	}
	num("passwordLength", &p.PasswordLength)
	str("usernameTB", &p.Username)
	str("counter", &p.Modifier)
	str("charset", &p.SelectedCharset)
	str("prefix", &p.PasswordPrefix)
	str("suffix", &p.PasswordSuffix)
	str("description", &p.Description)
	boolean("protocolCB", &p.URLProtocol)
	boolean("subdomainCB", &p.URLSubdomain)
	boolean("domainCB", &p.URLDomain)
	boolean("pathCB", &p.URLPath)
}

// Parse reads the profiles from an RDF document. Like the Chrome
// extension's importer, a Firefox export (recognised by its "remotes"
// section) has its "defaults" entry as the first profile, providing
// missing settings for the others; in a Chrome export the "defaults"
// entry is only a copy of the first profile and is skipped.
func Parse(data []byte) ([]pwm.Profile, error) {
	descs, err := parseDescriptions(data)
	if err != nil {
		return nil, fmt.Errorf("parsing RDF: %w", err)
	}
	var defaults *description
	var profiles []description
	for i := range descs {
		d := descs[i]
		about := d.attrs["about"]
		if about == aboutGlobal {
			continue
		}
		if _, ok := d.get("charset"); !ok {
			continue // groups, remotes, etc.
		}
		if about == aboutDefaults {
			defaults = &descs[i]
		} else {
			profiles = append(profiles, d)
		}
	}

	fromFirefox := bytes.Contains(data, []byte(aboutRemotes))
	var out []pwm.Profile
	if fromFirefox && defaults != nil {
		base := pwm.NewProfile()
		applyAttrs(&base, *defaults)
		base.SiteList = siteList(*defaults)
		out = append(out, base)
		for _, d := range profiles {
			p := base
			applyAttrs(&p, d)
			p.SiteList = siteList(d)
			out = append(out, p)
		}
	} else {
		if len(profiles) == 0 && defaults != nil {
			profiles = append(profiles, *defaults)
		}
		for _, d := range profiles {
			p := pwm.NewProfile()
			applyAttrs(&p, d)
			p.SiteList = siteList(d)
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no profiles found in RDF file")
	}
	return out, nil
}

type attr struct{ name, value string }

func profileAttrs(p pwm.Profile) []attr {
	b := strconv.FormatBool
	attrs := []attr{
		{"name", p.Title},
		{"timestamp", strconv.FormatInt(p.Timestamp, 10)},
		{"urlToUse", p.StrUseText},
		{"whereLeetLB", p.WhereToUseL33t},
		{"leetLevelLB", strconv.Itoa(p.L33tLevel)},
		{"hashAlgorithmLB", p.HashAlgorithm},
		{"passwordLength", strconv.Itoa(p.PasswordLength)},
		{"usernameTB", p.Username},
		{"counter", p.Modifier},
		{"charset", p.SelectedCharset},
		{"prefix", p.PasswordPrefix},
		{"suffix", p.PasswordSuffix},
		{"description", p.Description},
		{"protocolCB", b(p.URLProtocol)},
		{"subdomainCB", b(p.URLSubdomain)},
		{"domainCB", b(p.URLDomain)},
		{"pathCB", b(p.URLPath)},
	}
	for k, pat := range p.SitePatterns() {
		typ := "wildcard"
		if len(pat) >= 2 && strings.HasPrefix(pat, "/") && strings.HasSuffix(pat, "/") {
			typ = "regex"
			pat = pat[1 : len(pat)-1]
		}
		n := strconv.Itoa(k)
		attrs = append(attrs,
			attr{"pattern" + n, pat},
			attr{"patternenabled" + n, "true"},
			attr{"patterndesc" + n, ""},
			attr{"patterntype" + n, typ})
	}
	return attrs
}

func escape(s string) string {
	r := strings.NewReplacer("&", "&amp;", ">", "&gt;", "<", "&lt;", `"`, "&quot;", "'", "&#39;", "`", "&#96;",
		"\t", "&#9;", "\r", "&#13;")
	return r.Replace(s)
}

// Marshal writes profiles in the format produced by the Chrome extension's
// export feature, so the file can also be imported into the extension.
func Marshal(profiles []pwm.Profile) []byte {
	var buf bytes.Buffer
	buf.WriteString(`<?xml version="1.0"?>
<RDF:RDF xmlns:NS1="http://passwordmaker.mozdev.org/rdf#"
         xmlns:NC="http://home.netscape.com/NC-rdf#"
         xmlns:RDF="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
`)
	var abouts []string
	write := func(about string, p pwm.Profile) {
		abouts = append(abouts, about)
		fmt.Fprintf(&buf, "<RDF:Description RDF:about=\"%s\"\n", escape(about))
		var multi []attr
		for _, a := range profileAttrs(p) {
			if strings.Contains(a.value, "\n") {
				multi = append(multi, a)
				continue
			}
			fmt.Fprintf(&buf, " NS1:%s=\"%s\"\n", a.name, escape(a.value))
		}
		if len(multi) == 0 {
			buf.WriteString(" />\n")
			return
		}
		buf.WriteString(" >\n")
		for _, a := range multi {
			fmt.Fprintf(&buf, " <NS1:%s>%s</NS1:%s>\n", a.name, escape(a.value), a.name)
		}
		buf.WriteString(" </RDF:Description>\n")
	}
	if len(profiles) > 0 {
		// The first profile is duplicated as "defaults", for Firefox.
		d := profiles[0]
		d.Title = "Defaults"
		write(aboutDefaults, d)
	}
	for i, p := range profiles {
		write("rdf:#$CHROME"+strconv.Itoa(i+1), p)
	}
	buf.WriteString("<RDF:Seq RDF:about=\"http://passwordmaker.mozdev.org/accounts\">\n")
	for _, a := range abouts {
		fmt.Fprintf(&buf, "<RDF:li RDF:resource=\"%s\"/>\n", escape(a))
	}
	buf.WriteString("</RDF:Seq>\n</RDF:RDF>\n")
	return buf.Bytes()
}
