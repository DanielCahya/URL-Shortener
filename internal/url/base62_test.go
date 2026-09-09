package url

import (
	"strings"
	"testing"
)

func TestGenerateRandomBase62(t *testing.T) {
	t.Run("default length", func(t *testing.T) {
		code, err := GenerateRandomBase62(DefaultCodeLen)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(code) != DefaultCodeLen {
			t.Fatalf("expected length %d, got %d", DefaultCodeLen, len(code))
		}
		for _, c := range code {
			if !strings.ContainsRune(Base62Alphabet, c) {
				t.Fatalf("character %c not in Base62 alphabet", c)
			}
		}
	})

	t.Run("custom length", func(t *testing.T) {
		code, err := GenerateRandomBase62(10)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(code) != 10 {
			t.Fatalf("expected length 10, got %d", len(code))
		}
	})

	t.Run("uniqueness across consecutive generations", func(t *testing.T) {
		seen := make(map[string]bool)
		for i := 0; i < 1000; i++ {
			code, err := GenerateRandomBase62(DefaultCodeLen)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if seen[code] {
				t.Fatalf("collision detected in 1000 generations: %s", code)
			}
			seen[code] = true
		}
	})
}

func TestValidateCustomAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr error
	}{
		{"valid lowercase", "myalias", nil},
		{"valid mixed case", "MyProject123", nil},
		{"valid with hyphen and underscore", "my-cool_link", nil},
		{"too short", "ab", ErrInvalidAlias},
		{"empty", "", ErrInvalidAlias},
		{"contains space", "my link", ErrInvalidAlias},
		{"contains special char", "link@home", ErrInvalidAlias},
		{"reserved api", "api", ErrReservedAlias},
		{"reserved health", "health", ErrReservedAlias},
		{"reserved admin uppercase", "Admin", ErrReservedAlias},
		{"reserved metrics", "metrics", ErrReservedAlias},
		{"reserved favicon", "favicon.ico", ErrReservedAlias},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomAlias(tt.alias)
			if err != tt.wantErr {
				t.Errorf("ValidateCustomAlias(%q) = %v, want %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}
