// Package pwm implements the PasswordMaker password generation algorithm,
// compatible with the PasswordMaker Pro Chrome extension and the original
// PasswordMaker Firefox extension.
package pwm

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
	"math/bits"
	"unicode/utf16"

	"golang.org/x/crypto/md4"       //nolint:staticcheck // required for compatibility
	"golang.org/x/crypto/ripemd160" //nolint:staticcheck // required for compatibility
)

// HashAlgorithms lists the supported algorithm identifiers (as used by the
// Chrome extension) together with a human readable name, in display order.
var HashAlgorithms = []struct{ ID, Name string }{
	{"md4", "MD4"},
	{"hmac-md4", "HMAC-MD4"},
	{"md5", "MD5"},
	{"md5_v6", "MD5 Version 0.6"},
	{"hmac-md5", "HMAC-MD5"},
	{"hmac-md5_v6", "HMAC-MD5 Version 0.6"},
	{"sha1", "SHA-1"},
	{"hmac-sha1", "HMAC-SHA-1"},
	{"sha256", "SHA-256"},
	{"hmac-sha256_fix", "HMAC-SHA-256"},
	{"hmac-sha256", "HMAC-SHA-256 Version 1.5.1"},
	{"rmd160", "RIPEMD-160"},
	{"hmac-rmd160", "HMAC-RIPEMD-160"},
}

// IsKnownAlgorithm reports whether id is a supported hash algorithm.
func IsKnownAlgorithm(id string) bool {
	for _, a := range HashAlgorithms {
		if a.ID == id {
			return true
		}
	}
	return false
}

// v6Bytes converts a string the way the JavaScript implementation sees it
// for the md5_v6 algorithms: one byte per UTF-16 code unit, high byte dropped.
func v6Bytes(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, len(units))
	for i, u := range units {
		out[i] = byte(u)
	}
	return out
}

func digest(newHash func() hash.Hash, data []byte) []byte {
	h := newHash()
	h.Write(data)
	return h.Sum(nil)
}

func hmacDigest(newHash func() hash.Hash, key, data []byte) []byte {
	m := hmac.New(newHash, key)
	m.Write(data)
	return m.Sum(nil)
}

// rawHash applies the named algorithm and returns its encoded output.
// key and data are the strings after any l33t conversion.
func rawHash(algorithm, key, data, charset string) (string, bool) {
	k, d := []byte(key), []byte(data)
	switch algorithm {
	case "md4":
		return toCharset(digest(md4.New, k), charset), true
	case "hmac-md4":
		return toCharset(hmacDigest(md4.New, k, d), charset), true
	case "md5":
		return toCharset(digest(md5.New, k), charset), true
	case "md5_v6":
		return hex.EncodeToString(digest(md5.New, v6Bytes(key))), true
	case "hmac-md5":
		return toCharset(hmacDigest(md5.New, k, d), charset), true
	case "hmac-md5_v6":
		return hex.EncodeToString(hmacDigest(md5.New, v6Bytes(key), v6Bytes(data))), true
	case "sha1":
		return toCharset(digest(sha1.New, k), charset), true
	case "hmac-sha1":
		return toCharset(hmacDigest(sha1.New, k, d), charset), true
	case "sha256":
		return toCharset(digest(sha256.New, k), charset), true
	case "hmac-sha256_fix":
		return toCharset(hmacDigest(sha256.New, k, d), charset), true
	case "hmac-sha256":
		return toCharset(buggyHMACSHA256(k, d), charset), true
	case "rmd160":
		return toCharset(digest(ripemd160.New, k), charset), true
	case "hmac-rmd160":
		return toCharset(hmacDigest(ripemd160.New, k, d), charset), true
	}
	return "", false
}

// toCharset converts a hash to a string in the given "base" (charset), the
// equivalent of rstr2any in the JavaScript implementation.
func toCharset(input []byte, charset string) string {
	chars := []rune(charset)
	divisor := big.NewInt(int64(len(chars)))
	n := new(big.Int).SetBytes(input)
	zero := big.NewInt(0)
	var digits []rune
	rem := new(big.Int)
	for {
		n.QuoRem(n, divisor, rem)
		digits = append(digits, chars[rem.Int64()])
		if n.Cmp(zero) == 0 {
			break
		}
	}
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}

// buggyHMACSHA256 reproduces the "HMAC-SHA-256 Version 1.5.1" algorithm,
// whose outer hash declares a message length of 512+160 bits instead of
// 512+256, causing the padding to be OR-ed into the inner digest.
func buggyHMACSHA256(key, data []byte) []byte {
	if len(key) > 64 {
		s := sha256.Sum256(key)
		key = s[:]
	}
	var ipad, opad [64]byte
	copy(ipad[:], key)
	copy(opad[:], key)
	for i := range ipad {
		ipad[i] ^= 0x36
		opad[i] ^= 0x5c
	}
	ih := sha256.New()
	ih.Write(ipad[:])
	ih.Write(data)
	inner := ih.Sum(nil)

	// Build the two 512-bit blocks exactly as the buggy JavaScript did.
	var w [32]uint32
	for i := 0; i < 16; i++ {
		w[i] = binary.BigEndian.Uint32(opad[i*4:])
	}
	for i := 0; i < 8; i++ {
		w[16+i] = binary.BigEndian.Uint32(inner[i*4:])
	}
	w[21] |= 0x80000000
	w[31] = 512 + 160

	h := [8]uint32{0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab, 0x5be0cd19}
	sha256Block(&h, w[0:16])
	sha256Block(&h, w[16:32])
	out := make([]byte, 32)
	for i, v := range h {
		binary.BigEndian.PutUint32(out[i*4:], v)
	}
	return out
}

var sha256K = [64]uint32{
	0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4, 0xab1c5ed5,
	0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe, 0x9bdc06a7, 0xc19bf174,
	0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f, 0x4a7484aa, 0x5cb0a9dc, 0x76f988da,
	0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7, 0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967,
	0x27b70a85, 0x2e1b2138, 0x4d2c6dfc, 0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85,
	0xa2bfe8a1, 0xa81a664b, 0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070,
	0x19a4c116, 0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
	0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7, 0xc67178f2,
}

func sha256Block(h *[8]uint32, block []uint32) {
	var w [64]uint32
	copy(w[:], block)
	for j := 16; j < 64; j++ {
		s0 := bits.RotateLeft32(w[j-15], -7) ^ bits.RotateLeft32(w[j-15], -18) ^ (w[j-15] >> 3)
		s1 := bits.RotateLeft32(w[j-2], -17) ^ bits.RotateLeft32(w[j-2], -19) ^ (w[j-2] >> 10)
		w[j] = w[j-16] + s0 + w[j-7] + s1
	}
	a, b, c, d, e, f, g, hh := h[0], h[1], h[2], h[3], h[4], h[5], h[6], h[7]
	for j := 0; j < 64; j++ {
		S1 := bits.RotateLeft32(e, -6) ^ bits.RotateLeft32(e, -11) ^ bits.RotateLeft32(e, -25)
		ch := (e & f) ^ (^e & g)
		t1 := hh + S1 + ch + sha256K[j] + w[j]
		S0 := bits.RotateLeft32(a, -2) ^ bits.RotateLeft32(a, -13) ^ bits.RotateLeft32(a, -22)
		maj := (a & b) ^ (a & c) ^ (b & c)
		t2 := S0 + maj
		hh, g, f, e, d, c, b, a = g, f, e, d+t1, c, b, a, t1+t2
	}
	h[0] += a
	h[1] += b
	h[2] += c
	h[3] += d
	h[4] += e
	h[5] += f
	h[6] += g
	h[7] += hh
}
