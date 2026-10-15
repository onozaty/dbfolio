package schema

import "testing"

func TestTable_KindLabel(t *testing.T) {
	tests := map[string]string{
		"":                  "",
		"BASE TABLE":        "",
		"table":             "",
		"TABLE":             "",
		"VIEW":              "VIEW",
		"view":              "view",
		"MATERIALIZED VIEW": "MATERIALIZED VIEW",
		"FOREIGN TABLE":     "FOREIGN TABLE",
	}
	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			// Act
			got := (&Table{Type: in}).KindLabel()

			// Assert
			if got != want {
				t.Errorf("KindLabel() for %q = %q, want %q", in, got, want)
			}
		})
	}
}
