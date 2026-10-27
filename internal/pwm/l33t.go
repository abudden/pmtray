package pwm

import "strings"

// L33tOptions lists the "where to use l33t" values with display names.
var L33tOptions = []struct{ ID, Name string }{
	{"off", "not at all"},
	{"before-hashing", "before generating password"},
	{"after-hashing", "after generating password"},
	{"both", "before and after generating password"},
}

var l33tLevels = [9][26]string{
	{"4", "b", "c", "d", "3", "f", "g", "h", "i", "j", "k", "1", "m", "n", "0", "p", "9", "r", "s", "7", "u", "v", "w", "x", "y", "z"},
	{"4", "b", "c", "d", "3", "f", "g", "h", "1", "j", "k", "1", "m", "n", "0", "p", "9", "r", "5", "7", "u", "v", "w", "x", "y", "2"},
	{"4", "8", "c", "d", "3", "f", "6", "h", "'", "j", "k", "1", "m", "n", "0", "p", "9", "r", "5", "7", "u", "v", "w", "x", "'/", "2"},
	{"@", "8", "c", "d", "3", "f", "6", "h", "'", "j", "k", "1", "m", "n", "0", "p", "9", "r", "5", "7", "u", "v", "w", "x", "'/", "2"},
	{"@", "|3", "c", "d", "3", "f", "6", "#", "!", "7", "|<", "1", "m", "n", "0", "|>", "9", "|2", "$", "7", "u", "\\/", "w", "x", "'/", "2"},
	{"@", "|3", "c", "|)", "&", "|=", "6", "#", "!", ",|", "|<", "1", "m", "n", "0", "|>", "9", "|2", "$", "7", "u", "\\/", "w", "x", "'/", "2"},
	{"@", "|3", "[", "|)", "&", "|=", "6", "#", "!", ",|", "|<", "1", "^^", "^/", "0", "|*", "9", "|2", "5", "7", "(_)", "\\/", "\\/\\/", "><", "'/", "2"},
	{"@", "8", "(", "|)", "&", "|=", "6", "|-|", "!", "_|", "|(", "1", "|\\/|", "|\\|", "()", "|>", "(,)", "|2", "$", "|", "|_|", "\\/", "\\^/", ")(", "'/", "\"/_"},
	{"@", "8", "(", "|)", "&", "|=", "6", "|-|", "!", "_|", "|{", "|_", "/\\/\\", "|\\|", "()", "|>", "(,)", "|2", "$", "|", "|_|", "\\/", "\\^/", ")(", "'/", "\"/_"},
}

// L33t converts message to l33t-speak at the given level (1-9). Levels
// outside that range return the message unchanged.
func L33t(level int, message string) string {
	if level < 1 || level > 9 {
		return message
	}
	table := &l33tLevels[level-1]
	var b strings.Builder
	for _, r := range strings.ToLower(message) {
		if r >= 'a' && r <= 'z' {
			b.WriteString(table[r-'a'])
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
