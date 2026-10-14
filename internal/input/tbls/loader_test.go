package tbls

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "schema.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad_ReadsSampleSchema(t *testing.T) {
	// Arrange
	path := filepath.Join("..", "..", "..", "testdata", "basic", "schema.json")

	// Act
	s, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(s.Tables) != 10 {
		t.Errorf("len(Tables) = %d, want 10", len(s.Tables))
	}
	if len(s.Relations) != 8 {
		t.Errorf("len(Relations) = %d, want 8", len(s.Relations))
	}
}

func TestLoad_ParsesRelationFields(t *testing.T) {
	// Arrange
	path := writeFile(t, `{
		"tables": [],
		"relations": [{
			"table": "public.shipments",
			"columns": ["order_id", "line_no"],
			"cardinality": "zero_or_more",
			"parent_table": "public.order_items",
			"parent_columns": ["order_id", "line_no"],
			"parent_cardinality": "exactly_one",
			"def": "FOREIGN KEY (order_id, line_no) REFERENCES order_items(order_id, line_no)",
			"virtual": true
		}]
	}`)

	// Act
	s, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	want := Relation{
		Table:             "public.shipments",
		Columns:           []string{"order_id", "line_no"},
		Cardinality:       "zero_or_more",
		ParentTable:       "public.order_items",
		ParentColumns:     []string{"order_id", "line_no"},
		ParentCardinality: "exactly_one",
		Def:               "FOREIGN KEY (order_id, line_no) REFERENCES order_items(order_id, line_no)",
		Virtual:           true,
	}
	if !reflect.DeepEqual(*s.Relations[0], want) {
		t.Errorf("Relation = %+v, want %+v", *s.Relations[0], want)
	}
}

func TestLoad_IgnoresUnknownFields(t *testing.T) {
	// Arrange
	path := writeFile(t, `{
		"name": "example",
		"future_field": {"nested": true},
		"tables": [{"name": "users", "type": "BASE TABLE", "columns": [], "unknown": 1}]
	}`)

	// Act
	s, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if s.Name != "example" || len(s.Tables) != 1 || s.Tables[0].Name != "users" {
		t.Errorf("unexpected schema: name=%q tables=%d", s.Name, len(s.Tables))
	}
}

func TestLoad_AcceptsEmptyTables(t *testing.T) {
	// Arrange
	path := writeFile(t, `{"tables": []}`)

	// Act
	s, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(s.Tables) != 0 {
		t.Errorf("len(Tables) = %d, want 0", len(s.Tables))
	}
}

func TestLoad_ReturnsErrorWhenFileDoesNotExist(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "missing.json")

	// Act
	_, err := Load(path)

	// Assert
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Load() error = %v, want os.ErrNotExist", err)
	}
}

func TestLoad_ReturnsErrorWhenJSONIsInvalid(t *testing.T) {
	// Arrange
	path := writeFile(t, `{"tables": [`)

	// Act
	_, err := Load(path)

	// Assert
	if err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}
}

func TestLoad_ReturnsErrorWhenTablesFieldIsMissing(t *testing.T) {
	// Arrange
	path := writeFile(t, `{"name": "not a tbls schema"}`)

	// Act
	_, err := Load(path)

	// Assert
	if !errors.Is(err, ErrNotTblsSchema) {
		t.Errorf("Load() error = %v, want ErrNotTblsSchema", err)
	}
}

func TestLoad_ReturnsErrorForNullEntries(t *testing.T) {
	tests := map[string]string{
		"null table":    `{"tables": [null]}`,
		"unnamed table": `{"tables": [{"columns": []}]}`,
		"null column":   `{"tables": [{"name": "t", "columns": [null]}]}`,
		"null index":    `{"tables": [{"name": "t", "columns": [], "indexes": [null]}]}`,
		"null relation": `{"tables": [], "relations": [null]}`,
	}
	for name, content := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			path := writeFile(t, content)

			// Act
			_, err := Load(path)

			// Assert
			if err == nil {
				t.Error("Load() error = nil, want validation error")
			}
		})
	}
}
