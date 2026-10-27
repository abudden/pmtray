package pwm

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"

	"golang.org/x/crypto/md4"       //nolint:staticcheck
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck
)

// testdata/vectors.json was produced by a literal (word-array based) port of
// the extension's JavaScript, including its HMAC-SHA-256 1.5.1 bug.
type vectors struct {
	Passwords []struct {
		Alg, Master, Where       string
		Level                    int
		Charset                  string
		Length                   int
		Modifier, Prefix, Suffix string
		Used, User, Result       string
	}
	Verification []struct{ Master, Result string }
}

func loadVectors(t *testing.T) vectors {
	data, err := os.ReadFile("testdata/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var v vectors
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestGeneratePasswordVectors(t *testing.T) {
	v := loadVectors(t)
	if len(v.Passwords) == 0 {
		t.Fatal("no vectors")
	}
	for i, c := range v.Passwords {
		p := NewProfile()
		p.HashAlgorithm = c.Alg
		p.WhereToUseL33t = c.Where
		p.L33tLevel = c.Level
		p.SelectedCharset = c.Charset
		p.PasswordLength = c.Length
		p.Modifier = c.Modifier
		p.PasswordPrefix = c.Prefix
		p.PasswordSuffix = c.Suffix
		got := p.GeneratePassword(c.Used, c.Master, c.User)
		if got != c.Result {
			t.Errorf("vector %d (%s, master %q, l33t %s/%d, len %d): got %q want %q",
				i, c.Alg, c.Master, c.Where, c.Level, c.Length, got, c.Result)
		}
	}
}

func TestVerificationCode(t *testing.T) {
	for _, c := range loadVectors(t).Verification {
		if got := VerificationCode(c.Master); got != c.Result {
			t.Errorf("VerificationCode(%q) = %q, want %q", c.Master, got, c.Result)
		}
	}
}

func TestMD4AndRIPEMD160(t *testing.T) {
	// RFC 1320 and the RIPEMD-160 reference test vectors.
	h := md4.New()
	h.Write([]byte("abc"))
	if got := hex.EncodeToString(h.Sum(nil)); got != "a448017aaf21d8525fc10ae87aa6729d" {
		t.Errorf("md4: %s", got)
	}
	h = ripemd160.New()
	h.Write([]byte("abc"))
	if got := hex.EncodeToString(h.Sum(nil)); got != "8eb208f7e05d987a9b044a8e98c6b087f15a0bfc" {
		t.Errorf("rmd160: %s", got)
	}
	p := NewProfile()
	p.HashAlgorithm = "hmac-rmd160"
	if got := p.GeneratePassword("example.com", "pw", ""); len(got) != 8 {
		t.Errorf("hmac-rmd160 produced %q", got)
	}
}

func TestToCharset(t *testing.T) {
	if got := toCharset([]byte{0, 0}, "ab"); got != "a" {
		t.Errorf("zero: %q", got)
	}
	if got := toCharset([]byte{0, 5}, "01"); got != "101" {
		t.Errorf("five: %q", got)
	}
}

func TestURLText(t *testing.T) {
	const (
		proto = 1 << iota
		sub
		dom
		path
	)
	cases := []struct {
		url  string
		mask int
		want string
	}{
		{"https://www.google.com/search?q=1#frag", dom, "google.com"},
		{"https://www.google.com/search?q=1#frag", proto | sub | dom | path, "https://www.google.com/search?q=1"},
		{"https://www.google.com/search?q=1#frag", proto | dom, "https://google.com"},
		{"https://www.google.com/search?q=1#frag", sub | dom, "www.google.com"},
		{"https://www.google.com/search?q=1#frag", sub, "www"},
		{"https://www.google.com/search?q=1#frag", path, "/search?q=1"},
		{"http://mail.google.co.uk:8080/x/y", dom, "google.co.uk"},
		{"http://mail.google.co.uk:8080/x/y", sub | dom | path, "mail.google.co.uk:8080/x/y"},
		{"google.com", dom, "google.com"},
		{"google.com", proto | dom, "google.com"},
		{"localhost", dom, ".localhost"},
		{"sub.foo.bar.gov.uk/path", dom, "bar.gov.uk"},
		{"sub.foo.bar.gov.uk/path", sub, "sub.foo"},
		{"a.b.c.example.com", sub | dom, "a.b.c.example.com"},
		{"", dom, "."},
	}
	for _, c := range cases {
		p := NewProfile()
		p.URLProtocol = c.mask&proto != 0
		p.URLSubdomain = c.mask&sub != 0
		p.URLDomain = c.mask&dom != 0
		p.URLPath = c.mask&path != 0
		if got := p.URLText(c.url); got != c.want {
			t.Errorf("URLText(%q, %04b) = %q, want %q", c.url, c.mask, got, c.want)
		}
	}
}

func TestMatchProfile(t *testing.T) {
	profiles := DefaultProfiles()
	profiles = append(profiles, NewProfile(), NewProfile())
	profiles[2].SiteList = "*.example.com  foo.org"
	profiles[3].SiteList = "/^https?://bank\\./ invalid(regex"
	cases := []struct {
		texts []string
		want  int
	}{
		{[]string{"https://www.example.com/login"}, 2},
		{[]string{"", "WWW.EXAMPLE.COM"}, 2},
		{[]string{"foo.org"}, 2},
		{[]string{"fooxorg"}, -1},
		{[]string{"https://bank.co.uk/"}, 3},
		{[]string{"http://notbank.co.uk/"}, -1},
		{[]string{"invalidregex"}, 3},
	}
	for _, c := range cases {
		if got := MatchProfile(profiles, c.texts...); got != c.want {
			t.Errorf("MatchProfile(%q) = %d, want %d", c.texts, got, c.want)
		}
	}
}

func TestL33t(t *testing.T) {
	if got := L33t(1, "Hello"); got != "h3110" {
		t.Errorf("got %q", got)
	}
	if got := L33t(0, "Hello"); got != "Hello" {
		t.Errorf("got %q", got)
	}
}

func TestSuffixLongerThanPassword(t *testing.T) {
	p := NewProfile()
	p.PasswordLength = 4
	p.PasswordSuffix = "123456"
	// JavaScript: pw.slice(0, 4-6) + "123456" then .slice(0, 4), so the
	// suffix is pushed off the end when the hash output is long enough.
	plain := p
	plain.PasswordSuffix = ""
	if got, want := p.GeneratePassword("x", "y", ""), plain.GeneratePassword("x", "y", ""); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
