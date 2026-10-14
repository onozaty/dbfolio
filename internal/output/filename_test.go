package output

import (
	"regexp"
	"strings"
	"testing"
)

func TestFileName_KeepsSafeNames(t *testing.T) {
	tests := []string{
		"users",
		"public.users",
		"user_roles",
		"order-items",
		"t1",
	}
	for _, name := range tests {
		t.Run(name, func(t *testing.T) {
			// Act
			got := FileName(name)

			// Assert
			if got != name {
				t.Errorf("FileName(%q) = %q, want unchanged", name, got)
			}
		})
	}
}

var hashedName = regexp.MustCompile(`^[a-z0-9._-]+-[0-9a-f]{8}$`)

func TestFileName_AppendsHashToTransformedNames(t *testing.T) {
	tests := []struct {
		name       string
		wantPrefix string
	}{
		{name: "Users", wantPrefix: "users-"},
		{name: "public.Users", wantPrefix: "public.users-"},
		{name: "order items", wantPrefix: "order_items-"},
		{name: "a/b\\c", wantPrefix: "a_b_c-"},
		{name: "ユーザー", wantPrefix: "____-"},
		{name: ".hidden", wantPrefix: "_hidden-"},
		{name: "con", wantPrefix: "con-"},
		{name: "NUL", wantPrefix: "nul-"},
		{name: "com0", wantPrefix: "com0-"},
		{name: "LPT0", wantPrefix: "lpt0-"},
		{name: "", wantPrefix: "_-"},
		{name: "com1.log", wantPrefix: "com1.log-"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := FileName(tt.name)

			// Assert
			if !strings.HasPrefix(got, tt.wantPrefix) || !hashedName.MatchString(got) {
				t.Errorf("FileName(%q) = %q, want %q followed by an 8-digit hash", tt.name, got, tt.wantPrefix)
			}
		})
	}
}

func TestFileName_DoesNotTreatReservedWordInSchemaPrefixAsReserved(t *testing.T) {
	// Act
	got := FileName("public.con")

	// Assert
	if got != "public.con" {
		t.Errorf(`FileName("public.con") = %q, want unchanged`, got)
	}
}

func TestFileName_DistinguishesNamesDifferingOnlyByCase(t *testing.T) {
	// Arrange
	names := []string{"users", "Users", "USERS"}

	// Act
	seen := map[string]string{}
	for _, n := range names {
		seen[strings.ToLower(FileName(n))] = n
	}

	// Assert
	if len(seen) != len(names) {
		t.Errorf("case-insensitive collision among %v: %v", names, seen)
	}
}

func TestFileName_DistinguishesNamesMappedToSameBase(t *testing.T) {
	// Arrange
	a, b := "order items", "order/items"

	// Act
	gotA, gotB := FileName(a), FileName(b)

	// Assert
	if gotA == gotB {
		t.Errorf("FileName(%q) and FileName(%q) both = %q", a, b, gotA)
	}
}

func TestFileName_IsDeterministic(t *testing.T) {
	// Act
	first, second := FileName("Users"), FileName("Users")

	// Assert
	if first != second {
		t.Errorf("FileName returned %q then %q", first, second)
	}
}

func TestFileName_TruncatesLongNames(t *testing.T) {
	// Arrange
	name := strings.Repeat("a", 300)

	// Act
	got := FileName(name)

	// Assert
	if len(got) != maxBaseLen+9 || !hashedName.MatchString(got) {
		t.Errorf("FileName(300 chars) = %q (len %d), want %d chars + hash", got, len(got), maxBaseLen)
	}
}
