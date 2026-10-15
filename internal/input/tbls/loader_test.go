package tbls

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/onozaty/dbfolio/internal/schema"
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
	if len(s.Relations) != 10 {
		t.Errorf("len(Relations) = %d, want 10", len(s.Relations))
	}
}

func TestLoad_ResolvesRelationEndpoints(t *testing.T) {
	// Arrange
	path := writeFile(t, `{
		"tables": [
			{"name": "public.order_items", "columns": [{"name": "order_id"}, {"name": "line_no"}]},
			{"name": "public.shipments", "columns": [{"name": "order_id"}, {"name": "line_no"}]}
		],
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
	want := schema.Relation{
		Table:             s.Tables[1],
		Columns:           []string{"order_id", "line_no"},
		Cardinality:       "zero_or_more",
		ParentTable:       s.Tables[0],
		ParentColumns:     []string{"order_id", "line_no"},
		ParentCardinality: "exactly_one",
		Def:               "FOREIGN KEY (order_id, line_no) REFERENCES order_items(order_id, line_no)",
		Virtual:           true,
	}
	got := *s.Relations[0]
	// Compare pointers by identity: the endpoints must be the tables of the same schema.
	if got.Table != want.Table || got.ParentTable != want.ParentTable {
		t.Errorf("endpoints = %s -> %s, want tables of the schema", got.Table.Name, got.ParentTable.Name)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Relation = %+v, want %+v", got, want)
	}
}

func TestLoad_ResolvesSelfReference(t *testing.T) {
	// Arrange
	path := writeFile(t, `{
		"tables": [{"name": "comments", "columns": [{"name": "id"}, {"name": "parent_id"}]}],
		"relations": [{"table": "comments", "columns": ["parent_id"], "parent_table": "comments", "parent_columns": ["id"]}]
	}`)

	// Act
	s, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	r := s.Relations[0]
	if r.Table != s.Tables[0] || r.ParentTable != s.Tables[0] {
		t.Error("self reference is not resolved to the same table")
	}
}

func TestLoad_ReturnsErrorForInvalidRelations(t *testing.T) {
	const tables = `"tables": [
		{"name": "users", "columns": [{"name": "id"}]},
		{"name": "orders", "columns": [{"name": "user_id"}, {"name": "line_no"}]}
	]`
	tests := map[string]struct {
		relation string
		wantErr  string
	}{
		"unknown table": {
			`{"table": "order", "columns": ["user_id"], "parent_table": "users", "parent_columns": ["id"]}`,
			`table "order" not found`,
		},
		"unknown parent table": {
			`{"table": "orders", "columns": ["user_id"], "parent_table": "user", "parent_columns": ["id"]}`,
			`parent table "user" not found`,
		},
		"unknown column": {
			`{"table": "orders", "columns": ["userid"], "parent_table": "users", "parent_columns": ["id"]}`,
			`column "userid" not found in table orders`,
		},
		"unknown parent column": {
			`{"table": "orders", "columns": ["user_id"], "parent_table": "users", "parent_columns": ["uid"]}`,
			`column "uid" not found in table users`,
		},
		"column count mismatch": {
			`{"table": "orders", "columns": ["user_id", "line_no"], "parent_table": "users", "parent_columns": ["id"]}`,
			"2 columns but 1 parent columns",
		},
		"no columns": {
			`{"table": "orders", "columns": [], "parent_table": "users", "parent_columns": []}`,
			"no columns",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			path := writeFile(t, `{`+tables+`, "relations": [`+tt.relation+`]}`)

			// Act
			_, err := Load(path)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %v, want containing %q", err, tt.wantErr)
			}
		})
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
		"null table":      `{"tables": [null]}`,
		"unnamed table":   `{"tables": [{"columns": []}]}`,
		"null column":     `{"tables": [{"name": "t", "columns": [null]}]}`,
		"null index":      `{"tables": [{"name": "t", "columns": [], "indexes": [null]}]}`,
		"null relation":   `{"tables": [], "relations": [null]}`,
		"duplicate table": `{"tables": [{"name": "t"}, {"name": "t"}]}`,
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

func loadTable(t *testing.T, tableJSON string) *schema.Table {
	t.Helper()
	s, err := Load(writeFile(t, `{"tables": [`+tableJSON+`]}`))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	return s.Tables[0]
}

func TestLoad_ConvertsColumns(t *testing.T) {
	// Arrange
	table := `{"name": "t", "type": "BASE TABLE", "columns": [
		{"name": "id", "type": "int", "nullable": false, "default": "0", "extra_def": "auto_increment", "comment": "ID"},
		{"name": "note", "type": "text", "nullable": true}
	]}`

	// Act
	got := loadTable(t, table)

	// Assert
	zero := "0"
	want := []*schema.Column{
		{Name: "id", Type: "int", Nullable: false, Default: &zero, ExtraDef: "auto_increment", Comment: "ID"},
		{Name: "note", Type: "text", Nullable: true},
	}
	if !reflect.DeepEqual(got.Columns, want) {
		t.Errorf("Columns = %+v, want %+v", got.Columns, want)
	}
}

func TestLoad_ConvertsConstraintKinds(t *testing.T) {
	// Arrange
	table := `{"name": "t", "columns": [], "constraints": [
		{"name": "pk", "type": "PRIMARY KEY", "def": "PRIMARY KEY (a, b)", "columns": ["a", "b"], "referenced_table": ""},
		{"name": "fk", "type": "FOREIGN KEY", "def": "FOREIGN KEY (a, b) REFERENCES p(x, y)", "columns": ["a", "b"],
		 "referenced_table": "public.p", "referenced_columns": ["x", "y"]},
		{"name": "uq", "type": "unique", "columns": ["c"]},
		{"name": "ck", "type": "CHECK", "columns": ["c"]},
		{"name": "ex", "type": "EXCLUSION", "columns": ["c"]}
	]}`

	// Act
	got := loadTable(t, table)

	// Assert
	want := []*schema.Constraint{
		{Name: "pk", Kind: schema.ConstraintPrimaryKey, Type: "PRIMARY KEY", Def: "PRIMARY KEY (a, b)", Columns: []string{"a", "b"}},
		{Name: "fk", Kind: schema.ConstraintForeignKey, Type: "FOREIGN KEY", Def: "FOREIGN KEY (a, b) REFERENCES p(x, y)", Columns: []string{"a", "b"},
			ReferencedTable: "public.p", ReferencedColumns: []string{"x", "y"}},
		{Name: "uq", Kind: schema.ConstraintUnique, Type: "unique", Columns: []string{"c"}},
		{Name: "ck", Kind: schema.ConstraintCheck, Type: "CHECK", Columns: []string{"c"}},
		{Name: "ex", Kind: schema.ConstraintOther, Type: "EXCLUSION", Columns: []string{"c"}},
	}
	if !reflect.DeepEqual(got.Constraints, want) {
		t.Errorf("Constraints =\n%+v\nwant\n%+v", got.Constraints, want)
	}
}

func TestLoad_ExcludesNotNullConstraints(t *testing.T) {
	// Arrange: PostgreSQL 18 emits NOT NULL as constraints of type "n".
	table := `{"name": "t", "columns": [], "constraints": [
		{"name": "t_id_not_null", "type": "n", "def": "NOT NULL id", "columns": ["id"]},
		{"name": "t_pkey", "type": "PRIMARY KEY", "columns": ["id"]}
	]}`

	// Act
	got := loadTable(t, table)

	// Assert
	if len(got.Constraints) != 1 || got.Constraints[0].Name != "t_pkey" {
		t.Errorf("Constraints = %+v, want only t_pkey", got.Constraints)
	}
}

func TestLoad_DetectsViews(t *testing.T) {
	tests := map[string]bool{
		"BASE TABLE":        false,
		"FOREIGN TABLE":     false,
		"table":             false,
		"VIEW":              true,
		"MATERIALIZED VIEW": true,
		"view":              true,
	}
	for tableType, want := range tests {
		t.Run(tableType, func(t *testing.T) {
			// Act
			got := loadTable(t, `{"name": "t", "type": "`+tableType+`", "columns": []}`)

			// Assert
			if got.View != want || got.Type != tableType {
				t.Errorf("View = %v, Type = %q; want %v, %q", got.View, got.Type, want, tableType)
			}
		})
	}
}

func TestLoad_ConvertsIndexesTriggersAndDef(t *testing.T) {
	// Arrange
	table := `{"name": "v", "type": "VIEW", "columns": [], "def": "SELECT 1",
		"indexes": [{"name": "i", "def": "CREATE INDEX i ON v (a)", "table": "v", "columns": ["a"], "comment": "idx"}],
		"triggers": [{"name": "tr", "def": "CREATE TRIGGER tr ...", "comment": "trg"}]
	}`

	// Act
	got := loadTable(t, table)

	// Assert
	wantIndexes := []*schema.Index{{Name: "i", Def: "CREATE INDEX i ON v (a)", Columns: []string{"a"}, Comment: "idx"}}
	wantTriggers := []*schema.Trigger{{Name: "tr", Def: "CREATE TRIGGER tr ...", Comment: "trg"}}
	if !reflect.DeepEqual(got.Indexes, wantIndexes) {
		t.Errorf("Indexes = %+v, want %+v", got.Indexes, wantIndexes)
	}
	if !reflect.DeepEqual(got.Triggers, wantTriggers) {
		t.Errorf("Triggers = %+v, want %+v", got.Triggers, wantTriggers)
	}
	if got.Def != "SELECT 1" {
		t.Errorf("Def = %q, want %q", got.Def, "SELECT 1")
	}
}
