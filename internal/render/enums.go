package render

import (
	"strings"

	"github.com/onozaty/dbfolio/internal/schema"
)

// enumResolver finds the enum a column type refers to.
type enumResolver struct {
	byName map[string]int
	// searchPaths resolve type names without schema, as the database does;
	// nil when the input does not tell them.
	searchPaths []string
	// unique maps names without schema that only one enum has, for when the
	// search paths are unknown.
	unique map[string]int
}

func newEnumResolver(enums []*schema.Enum, d *schema.Driver) *enumResolver {
	r := &enumResolver{byName: make(map[string]int, len(enums)), unique: make(map[string]int)}
	if d != nil {
		r.searchPaths = d.SearchPaths
	}
	count := make(map[string]int)
	for i, e := range enums {
		r.byName[e.Name] = i
		if base, ok := baseName(e.Name); ok {
			count[base]++
			r.unique[base] = i
		}
	}
	for base, n := range count {
		if n > 1 {
			delete(r.unique, base)
		}
	}
	return r
}

// resolve returns the position in Schema.Enums of the enum a column type refers
// to. Arrays of an enum (order_status[]) refer to it too. PostgreSQL quotes
// names that need it (public."OrderStatus") while tbls joins enum names
// without quotes, and omits the schema when it is on the search path.
func (r *enumResolver) resolve(columnType string) (int, bool) {
	t := strings.TrimSpace(columnType)
	for strings.HasSuffix(t, "[]") {
		t = strings.TrimSuffix(t, "[]")
	}
	t = strings.ReplaceAll(t, `"`, "")
	if i, ok := r.byName[t]; ok {
		return i, true
	}
	if strings.Contains(t, ".") {
		return 0, false
	}
	if r.searchPaths != nil {
		for _, sp := range r.searchPaths {
			if i, ok := r.byName[sp+"."+t]; ok {
				return i, true
			}
		}
		return 0, false
	}
	i, ok := r.unique[t]
	return i, ok
}

// baseName returns a qualified name without its schema.
func baseName(name string) (string, bool) {
	i := strings.LastIndex(name, ".")
	if i < 0 {
		return "", false
	}
	return name[i+1:], true
}
