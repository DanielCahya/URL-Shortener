package url

import (
	"crypto/rand"
	"math/big"
	"regexp"
	"strings"
)

const (
	Base62Alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	DefaultCodeLen = 7
)

var (
	// customAliasRegex allows alphanumeric characters, underscores, and hyphens (3 to 64 chars)
	customAliasRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,64}$`)

	// reservedAliases holds system-level reserved path names
	reservedAliases = map[string]struct{}{
		"api":         {},
		"admin":       {},
		"health":      {},
		"metrics":     {},
		"login":       {},
		"register":    {},
		"favicon.ico": {},
		"robots.txt":  {},
		"swagger":     {},
		"docs":        {},
		"static":      {},
		"assets":      {},
	}
)

// GenerateRandomBase62 generates a cryptographically secure random Base62 string of the given length.
func GenerateRandomBase62(length int) (string, error) {
	if length <= 0 {
		length = DefaultCodeLen
	}

	alphabetLen := big.NewInt(int64(len(Base62Alphabet)))
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		result[i] = Base62Alphabet[n.Int64()]
	}

	return string(result), nil
}

// ValidateCustomAlias checks whether a custom alias complies with length, character, and reservation rules.
func ValidateCustomAlias(alias string) error {
	trimmed := strings.TrimSpace(alias)
	lower := strings.ToLower(trimmed)
	if _, reserved := reservedAliases[lower]; reserved {
		return ErrReservedAlias
	}

	if !customAliasRegex.MatchString(trimmed) {
		return ErrInvalidAlias
	}

	return nil
}
