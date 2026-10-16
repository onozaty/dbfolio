package render

import (
	"testing"

	"github.com/onozaty/dbfolio/internal/schema"
)

func TestEnumResolver_Resolve(t *testing.T) {
	enums := []*schema.Enum{
		{Name: "public.status"},
		{Name: "other.status"},
		{Name: "public.OrderStatus"},
		{Name: "other.level"},
	}
	withPaths := &schema.Driver{SearchPaths: []string{"postgres", "public"}}
	tests := []struct {
		name   string
		driver *schema.Driver
		typ    string
		want   int
		found  bool
	}{
		{"qualified", nil, "other.status", 1, true},
		{"array of an enum", nil, "public.status[]", 0, true},
		{"multidimensional array", nil, "public.status[][]", 0, true},
		{"quoted name", nil, `public."OrderStatus"`, 2, true},
		{"not an enum", nil, "text", 0, false},
		{"qualified, unknown schema", nil, "missing.status", 0, false},
		// Without search paths, a name without schema matches only if unique.
		{"unique without paths", nil, "level", 3, true},
		{"ambiguous without paths", nil, "status", 0, false},
		// With search paths, the database's resolution order applies.
		{"first on the search path", withPaths, "status", 0, true},
		{"not on the search path", withPaths, "level", 0, false},
		{"array, on the search path", withPaths, "status[]", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			r := newEnumResolver(enums, tt.driver)

			// Act
			got, found := r.resolve(tt.typ)

			// Assert
			if found != tt.found || (found && got != tt.want) {
				t.Errorf("resolve(%q) = %d, %v; want %d, %v", tt.typ, got, found, tt.want, tt.found)
			}
		})
	}
}
