package render

import (
	"reflect"
	"testing"

	"github.com/onozaty/dbfolio/internal/schema"
)

func columnKeys(p tablePage) map[string][]string {
	keys := make(map[string][]string)
	for _, c := range p.Columns {
		keys[c.Name] = c.Keys
	}
	return keys
}

func TestNewTablePage_MarksCompositePrimaryKeyColumns(t *testing.T) {
	// Arrange
	table := &schema.Table{
		Name:    "order_items",
		Columns: []*schema.Column{{Name: "order_id"}, {Name: "line_no"}, {Name: "product"}},
		Constraints: []*schema.Constraint{
			{Name: "pk", Kind: schema.ConstraintPrimaryKey, Columns: []string{"order_id", "line_no"}},
		},
	}

	// Act
	p := newTablePage(&schema.Schema{Tables: []*schema.Table{table}}, table)

	// Assert
	want := map[string][]string{"order_id": {"PK"}, "line_no": {"PK"}, "product": nil}
	if got := columnKeys(p); !reflect.DeepEqual(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestNewTablePage_ShowsSingleFKForColumnInSeveralForeignKeys(t *testing.T) {
	// Arrange: order_id is in a single-column FK and in a composite FK.
	table := &schema.Table{
		Name:    "shipments",
		Columns: []*schema.Column{{Name: "id"}, {Name: "order_id"}, {Name: "line_no"}},
		Constraints: []*schema.Constraint{
			{Name: "pk", Kind: schema.ConstraintPrimaryKey, Columns: []string{"id"}},
			{Name: "fk1", Kind: schema.ConstraintForeignKey, Columns: []string{"order_id"}},
			{Name: "fk2", Kind: schema.ConstraintForeignKey, Columns: []string{"order_id", "line_no"}},
		},
	}

	// Act
	p := newTablePage(&schema.Schema{Tables: []*schema.Table{table}}, table)

	// Assert
	want := map[string][]string{"id": {"PK"}, "order_id": {"FK"}, "line_no": {"FK"}}
	if got := columnKeys(p); !reflect.DeepEqual(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestNewTablePage_MarksColumnThatIsBothPKAndFK(t *testing.T) {
	// Arrange
	table := &schema.Table{
		Name:    "user_roles",
		Columns: []*schema.Column{{Name: "user_id"}},
		Constraints: []*schema.Constraint{
			{Name: "fk", Kind: schema.ConstraintForeignKey, Columns: []string{"user_id"}},
			{Name: "pk", Kind: schema.ConstraintPrimaryKey, Columns: []string{"user_id"}},
		},
	}

	// Act
	p := newTablePage(&schema.Schema{Tables: []*schema.Table{table}}, table)

	// Assert
	if got := p.Columns[0].Keys; !reflect.DeepEqual(got, []string{"PK", "FK"}) {
		t.Errorf("keys = %v, want [PK FK]", got)
	}
}

func TestNewTablePage_ShowsExtraColumnOnlyWhenUsed(t *testing.T) {
	tests := map[string]struct {
		columns []*schema.Column
		want    bool
	}{
		"no extra":   {[]*schema.Column{{Name: "a"}}, false},
		"with extra": {[]*schema.Column{{Name: "a"}, {Name: "b", ExtraDef: "auto_increment"}}, true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			table := &schema.Table{Name: "t", Columns: tt.columns}

			// Act
			p := newTablePage(&schema.Schema{Tables: []*schema.Table{table}}, table)

			// Assert
			if p.ShowExtra != tt.want {
				t.Errorf("ShowExtra = %v, want %v", p.ShowExtra, tt.want)
			}
		})
	}
}

func TestNewIndexPage_CountsTablesAndViewsSeparately(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}, {Name: "name"}}, Indexes: []*schema.Index{{Name: "pk"}}}
	active := &schema.Table{Name: "active_users", View: true, Columns: []*schema.Column{{Name: "id"}}}
	s := &schema.Schema{Name: "db", Tables: []*schema.Table{users, active}, Relations: []*schema.Relation{{}}}
	files := map[*schema.Table]string{users: "users.html", active: "active_users.html"}

	// Act
	p := newIndexPage(s, files)

	// Assert
	want := stats{Tables: 1, Views: 1, Columns: 3, Relations: 1, Indexes: 1}
	if p.Stats != want {
		t.Errorf("Stats = %+v, want %+v", p.Stats, want)
	}
	if p.Tables[1].Href != "tables/active_users.html" {
		t.Errorf("Href = %q, want %q", p.Tables[1].Href, "tables/active_users.html")
	}
}

func TestNewIndexPage_FallsBackWhenDatabaseNameIsEmpty(t *testing.T) {
	// Act
	p := newIndexPage(&schema.Schema{}, nil)

	// Assert
	if p.DBName != "Database" {
		t.Errorf("DBName = %q, want %q", p.DBName, "Database")
	}
}
