package pwm

import (
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"
)

// CharsetOptions are the predefined character sets offered by the Chrome
// extension. The first is the default.
var CharsetOptions = []string{
	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789`~!@#$%^&*()_-+={}|[]\\:\";'<>?,./",
	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
	"0123456789abcdef",
	"0123456789",
	"ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
	"`~!@#$%^&*()_-+={}|[]\\:\";'<>?,./",
}

// Profile holds the settings used to generate a password. Field names
// follow the Chrome extension.
type Profile struct {
	Title       string
	Timestamp   int64 // milliseconds since the epoch
	SiteList    string
	Description string

	URLProtocol  bool
	URLSubdomain bool
	URLDomain    bool
	URLPath      bool

	// StrUseText is used instead of the URL when not empty.
	StrUseText string

	HashAlgorithm   string
	Username        string
	Modifier        string
	PasswordLength  int
	SelectedCharset string
	PasswordPrefix  string
	PasswordSuffix  string
	WhereToUseL33t  string
	L33tLevel       int
}

// NewProfile returns a profile with the extension's default settings.
func NewProfile() Profile {
	return Profile{
		Title:           "Default",
		URLDomain:       true,
		HashAlgorithm:   "md5",
		PasswordLength:  8,
		SelectedCharset: CharsetOptions[0],
		WhereToUseL33t:  "off",
	}
}

// DefaultProfiles returns the profiles created when no configuration exists.
func DefaultProfiles() []Profile {
	normal := NewProfile()
	alpha := NewProfile()
	alpha.Title = "Alphanumeric"
	alpha.SelectedCharset = CharsetOptions[1]
	return []Profile{normal, alpha}
}

// jsLen returns the length of s as JavaScript would report it.
func jsLen(s string) int {
	n := 0
	for _, r := range s {
		n += utf16.RuneLen(r)
	}
	return n
}

// jsSlice returns s.slice(0, end) with JavaScript semantics (negative end
// counts from the end of the string).
func jsSlice(s string, end int) string {
	units := utf16.Encode([]rune(s))
	if end < 0 {
		end += len(units)
		if end < 0 {
			end = 0
		}
	}
	if end > len(units) {
		end = len(units)
	}
	return string(utf16.Decode(units[:end]))
}

// GeneratePassword generates the password for usedText using masterKey.
// username overrides the profile's username.
func (p Profile) GeneratePassword(usedText, masterKey, username string) string {
	if jsLen(p.SelectedCharset) < 2 {
		return "Not enough chars!"
	}
	if !IsKnownAlgorithm(p.HashAlgorithm) {
		return "Unknown hash algorithm!"
	}

	data := usedText + username + p.Modifier
	password := ""
	for count := 0; jsLen(password) < p.PasswordLength; count++ {
		key := masterKey
		if count > 0 {
			key += "\n" + strconv.Itoa(count)
		}
		password += p.generateCharacters(key, data)
	}

	if p.PasswordPrefix != "" {
		password = p.PasswordPrefix + password
	}
	if p.PasswordSuffix != "" {
		password = jsSlice(password, p.PasswordLength-jsLen(p.PasswordSuffix)) + p.PasswordSuffix
	}
	return jsSlice(password, p.PasswordLength)
}

func (p Profile) generateCharacters(key, data string) string {
	usingHMAC := strings.Contains(strings.ToLower(p.HashAlgorithm), "hmac")
	if !usingHMAC {
		key += data
	}
	if p.WhereToUseL33t == "both" || p.WhereToUseL33t == "before-hashing" {
		key = L33t(p.L33tLevel, key)
		if usingHMAC {
			data = L33t(p.L33tLevel, data)
		}
	}
	password, _ := rawHash(p.HashAlgorithm, key, data, p.SelectedCharset)
	if p.WhereToUseL33t == "both" || p.WhereToUseL33t == "after-hashing" {
		return L33t(p.L33tLevel, password)
	}
	return password
}

// UsedText returns the text used for generation: the profile's
// "Use this text" setting if present, otherwise the parts of url selected
// by the profile.
func (p Profile) UsedText(url string) string {
	if p.StrUseText != "" {
		return p.StrUseText
	}
	return p.URLText(url)
}

var urlRegexp = regexp.MustCompile(`^([^:/]*?://)?([^:/]*)([^#]*)`)

// URLText extracts the parts of url selected by the profile's URL
// component settings.
func (p Profile) URLText(url string) string {
	m := urlRegexp.FindStringSubmatchIndex(url)
	group := func(i int) string {
		if m[2*i] < 0 {
			return ""
		}
		return url[m[2*i]:m[2*i+1]]
	}
	segments := strings.Split(group(2), ".")
	for len(segments) < 3 {
		segments = append([]string{""}, segments...)
	}
	sub, dom := splitSubdomain(segments)

	result := ""
	if p.URLProtocol && m[2] >= 0 {
		result += group(1)
	}
	if p.URLSubdomain {
		result += sub
	}
	if p.URLDomain {
		if result != "" && !strings.HasSuffix(result, ".") && !strings.HasSuffix(result, "/") {
			result += "."
		}
		result += dom
	}
	if p.URLPath {
		result += group(3)
	}
	return result
}

// splitSubdomain splits domain segments like [www google co uk] into the
// subdomain ("www") and domain ("google.co.uk").
func splitSubdomain(segments []string) (string, string) {
	for i := range segments {
		if topLevelDomains[strings.Join(segments[i:], ".")] {
			pivot := i - 1
			if pivot < 0 {
				pivot = 0
			}
			return strings.Join(segments[:pivot], "."), strings.Join(segments[pivot:], ".")
		}
	}
	n := len(segments)
	return strings.Join(segments[:n-2], "."), strings.Join(segments[n-2:], ".")
}

// SitePatterns returns the individual patterns of the profile's site list.
func (p Profile) SitePatterns() []string {
	return strings.Fields(p.SiteList)
}

var wildcardStrip = regexp.MustCompile(`[$+()^\[\]\\|{},]`)

// patternRegexp converts a site pattern (a wildcard pattern, or a regular
// expression enclosed in slashes) to a case-insensitive regexp.
func patternRegexp(pattern string) (*regexp.Regexp, error) {
	if len(pattern) >= 2 && strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		return regexp.Compile("(?i)" + pattern[1:len(pattern)-1])
	}
	s := wildcardStrip.ReplaceAllString(pattern, "")
	s = strings.ReplaceAll(s, ".", `\.`)
	s = strings.ReplaceAll(s, "?", ".")
	s = strings.ReplaceAll(s, "*", ".*")
	return regexp.Compile("(?i)" + s)
}

// Matches reports whether any of the profile's site patterns match any of
// the given texts.
func (p Profile) Matches(texts ...string) bool {
	for _, pat := range p.SitePatterns() {
		re, err := patternRegexp(pat)
		if err != nil {
			continue
		}
		for _, t := range texts {
			if t != "" && re.MatchString(t) {
				return true
			}
		}
	}
	return false
}

// MatchProfile returns the index of the first profile whose site list
// matches any of texts, or -1 if none do.
func MatchProfile(profiles []Profile, texts ...string) int {
	for i := range profiles {
		if profiles[i].Matches(texts...) {
			return i
		}
	}
	return -1
}

// VerificationCode returns the three letter code used to check that the
// master password was typed correctly.
func VerificationCode(masterKey string) string {
	p := NewProfile()
	p.HashAlgorithm = "sha256"
	p.PasswordLength = 3
	p.SelectedCharset = CharsetOptions[4]
	return p.GeneratePassword("", masterKey, "")
}
