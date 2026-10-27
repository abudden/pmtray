package pwm

import (
	"math"
	"unicode/utf16"
)

// Strength is the result of PasswordStrength.
type Strength struct {
	Strength                             int // 0-100
	HasUpper, HasLower, HasDigit, HasSym bool
}

// PasswordStrength rates a password the same way as the extension's
// strength meter.
func PasswordStrength(pw string) Strength {
	units := utf16.Encode([]rune(pw))
	n := float64(len(units))

	uniques := map[uint16]bool{}
	var upper, lower, digits, syms float64
	for _, u := range units {
		uniques[u] = true
		switch {
		case u >= 'A' && u <= 'Z':
			upper++
		case u >= 'a' && u <= 'z':
			lower++
		case u >= '0' && u <= '9':
			digits++
		}
		if !(u >= 'A' && u <= 'Z' || u >= 'a' && u <= 'z' || u >= '0' && u <= '9' || u == '_') {
			syms++
		}
	}
	r0 := 0.0
	if len(uniques) != 1 {
		r0 = float64(len(uniques)) / n
	}

	var r1 float64
	switch {
	case n >= 15:
		r1 = 10
	case n < 5:
		r1 = -5
	default:
		r1 = n - 5
	}

	quarter := math.Floor(n/4 + 0.5)
	ratio := func(count float64) float64 {
		c := math.Abs(quarter - count)
		if count > quarter*2 {
			c = quarter
		}
		return 1 - c/quarter
	}
	s := ((r0+ratio(digits)+ratio(syms)+ratio(upper)+ratio(lower))/5)*100 + r1
	if s < 0 || math.IsNaN(s) {
		s = 0
	}
	if s > 100 {
		s = 100
	}
	return Strength{
		Strength: int(s),
		HasUpper: upper > 0,
		HasLower: lower > 0,
		HasDigit: digits > 0,
		HasSym:   syms > 0,
	}
}
