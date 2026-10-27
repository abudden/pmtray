package rdf

import (
	"reflect"
	"testing"

	"github.com/abudden/pmtray/internal/pwm"
)

const chromeExport = `<?xml version="1.0"?>
<RDF:RDF xmlns:NS1="http://passwordmaker.mozdev.org/rdf#"
         xmlns:NC="http://home.netscape.com/NC-rdf#"
         xmlns:RDF="http://www.w3.org/1999/02/22-rdf-syntax-ns#"><RDF:Description RDF:about="http://passwordmaker.mozdev.org/defaults"
 NS1:name="Defaults"
 NS1:timestamp="0"
 NS1:urlToUse=""
 NS1:whereLeetLB="off"
 NS1:leetLevelLB="0"
 NS1:hashAlgorithmLB="md5"
 NS1:passwordLength="8"
 NS1:usernameTB=""
 NS1:counter=""
 NS1:charset="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789&#96;~!@#$%^&amp;*()_-+={}|[]\:&quot;;&#39;&lt;&gt;?,./"
 NS1:prefix=""
 NS1:suffix=""
 NS1:description=""
 NS1:protocolCB="false"
 NS1:subdomainCB="false"
 NS1:domainCB="true"
 NS1:pathCB="false"
 />
<RDF:Description RDF:about="rdf:#$CHROME1"
 NS1:name="Default"
 NS1:timestamp="0"
 NS1:urlToUse=""
 NS1:whereLeetLB="off"
 NS1:leetLevelLB="0"
 NS1:hashAlgorithmLB="md5"
 NS1:passwordLength="8"
 NS1:usernameTB=""
 NS1:counter=""
 NS1:charset="ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789&#96;~!@#$%^&amp;*()_-+={}|[]\:&quot;;&#39;&lt;&gt;?,./"
 NS1:prefix=""
 NS1:suffix=""
 NS1:description=""
 NS1:protocolCB="false"
 NS1:subdomainCB="false"
 NS1:domainCB="true"
 NS1:pathCB="false"
 />
<RDF:Description RDF:about="rdf:#$CHROME2"
 NS1:name="Work"
 NS1:timestamp="1700000000000"
 NS1:urlToUse=""
 NS1:whereLeetLB="both"
 NS1:leetLevelLB="4"
 NS1:hashAlgorithmLB="hmac-sha256_fix"
 NS1:passwordLength="16"
 NS1:usernameTB="me"
 NS1:counter="2"
 NS1:charset="0123456789"
 NS1:prefix="P"
 NS1:suffix="S"
 NS1:protocolCB="true"
 NS1:subdomainCB="true"
 NS1:domainCB="true"
 NS1:pathCB="false"
 NS1:pattern0="*.example.com"
 NS1:patternenabled0="true"
 NS1:patterndesc0=""
 NS1:patterntype0="wildcard"
 NS1:pattern1="^https://work\."
 NS1:patternenabled1="true"
 NS1:patterndesc1=""
 NS1:patterntype1="regex"
 >
 <NS1:description>line one
line two</NS1:description>
 </RDF:Description>
<RDF:Seq RDF:about="http://passwordmaker.mozdev.org/accounts">
<RDF:li RDF:resource="http://passwordmaker.mozdev.org/defaults"/>
<RDF:li RDF:resource="rdf:#$CHROME1"/>
<RDF:li RDF:resource="rdf:#$CHROME2"/>
</RDF:Seq></RDF:RDF>`

func TestParseChrome(t *testing.T) {
	got, err := Parse([]byte(chromeExport))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d profiles", len(got))
	}
	if !reflect.DeepEqual(got[0], pwm.NewProfile()) {
		t.Errorf("default profile mismatch:\n%+v\n%+v", got[0], pwm.NewProfile())
	}
	want := pwm.Profile{
		Title: "Work", Timestamp: 1700000000000, SiteList: `*.example.com /^https://work\./`,
		Description: "line one\nline two", URLProtocol: true, URLSubdomain: true, URLDomain: true,
		HashAlgorithm: "hmac-sha256_fix", Username: "me", Modifier: "2", PasswordLength: 16,
		SelectedCharset: "0123456789", PasswordPrefix: "P", PasswordSuffix: "S",
		WhereToUseL33t: "both", L33tLevel: 4,
	}
	if !reflect.DeepEqual(got[1], want) {
		t.Errorf("got\n%+v\nwant\n%+v", got[1], want)
	}
}

const firefoxExport = `<?xml version="1.0"?>
<RDF:RDF xmlns:NS1="http://passwordmaker.mozdev.org/rdf#"
         xmlns:NC="http://home.netscape.com/NC-rdf#"
         xmlns:RDF="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
  <RDF:Seq RDF:about="http://passwordmaker.mozdev.org/accounts">
    <RDF:li RDF:resource="http://passwordmaker.mozdev.org/defaults"/>
    <RDF:li RDF:resource="rdf:#$abc"/>
  </RDF:Seq>
  <RDF:Description RDF:about="http://passwordmaker.mozdev.org/remotes" NS1:x="1"/>
  <RDF:Description RDF:about="http://passwordmaker.mozdev.org/globalSettings" NS1:maskMasterPassword="true"/>
  <RDF:Description RDF:about="http://passwordmaker.mozdev.org/defaults"
                   NS1:name="Defaults" NS1:charset="abc" NS1:hashAlgorithmLB="md5-v0.6"
                   NS1:passwordLength="10" NS1:protocolCB="false" NS1:subdomainCB="true"
                   NS1:domainCB="true" NS1:pathCB="false" NS1:whereLeetLB="off" NS1:leetLevelLB="1"/>
  <RDF:Description RDF:about="rdf:#$abc" NS1:name="Bank" NS1:charset="0123456789"
                   NS1:pattern0="*bank*" NS1:patterntype0="wildcard" NS1:patternenabled0="true"
                   NS1:pattern1="*old*" NS1:patterntype1="wildcard" NS1:patternenabled1="false"/>
  <RDF:Description RDF:about="rdf:#$group" NS1:name="A group"/>
</RDF:RDF>`

func TestParseFirefox(t *testing.T) {
	got, err := Parse([]byte(firefoxExport))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d profiles: %+v", len(got), got)
	}
	if got[0].Title != "Defaults" || got[0].HashAlgorithm != "md5_v6" || !got[0].URLSubdomain {
		t.Errorf("defaults: %+v", got[0])
	}
	b := got[1]
	if b.Title != "Bank" || b.HashAlgorithm != "md5_v6" || b.PasswordLength != 10 ||
		b.SelectedCharset != "0123456789" || !b.URLSubdomain || b.SiteList != "*bank*" {
		t.Errorf("bank: %+v", b)
	}
}

func TestRoundTrip(t *testing.T) {
	in, err := Parse([]byte(chromeExport))
	if err != nil {
		t.Fatal(err)
	}
	in[0].Description = "tab\there & <there> \"quoted\" 'single'"
	in[0].PasswordPrefix = "`"
	out, err := Parse(Marshal(in))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Errorf("round trip mismatch:\n%+v\n%+v", in, out)
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse([]byte("<RDF:RDF/>")); err == nil {
		t.Error("expected error")
	}
	if _, err := Parse([]byte("not xml <")); err == nil {
		t.Error("expected error")
	}
}
