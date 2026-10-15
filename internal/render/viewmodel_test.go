package render

import (
	"reflect"
	"slices"
	"testing"

	"github.com/onozaty/dbfolio/internal/schema"
)

// plan plans s after appending the columns its relations refer to but the
// fixture tables do not declare, so that tests only spell out what they check.
func plan(t *testing.T, s *schema.Schema) *Site {
	t.Helper()
	for _, r := range s.Relations {
		addMissingColumns(r.Table, r.Columns)
		addMissingColumns(r.ParentTable, r.ParentColumns)
	}
	site, err := Plan(s)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	return site
}

func addMissingColumns(t *schema.Table, names []string) {
	for _, n := range names {
		if !slices.ContainsFunc(t.Columns, func(c *schema.Column) bool { return c.Name == n }) {
			t.Columns = append(t.Columns, &schema.Column{Name: n})
		}
	}
}

func keyLabels(p tablePage) map[string][]string {
	keys := make(map[string][]string)
	for _, c := range p.Columns {
		var labels []string
		for _, k := range c.Keys {
			labels = append(labels, k.Label)
		}
		keys[c.Name] = labels
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
	site := plan(t, &schema.Schema{Tables: []*schema.Table{table}})

	// Act
	p := site.newTablePage(table)

	// Assert
	want := map[string][]string{"order_id": {"PK"}, "line_no": {"PK"}, "product": nil}
	if got := keyLabels(p); !reflect.DeepEqual(got, want) {
		t.Errorf("keys = %v, want %v", got, want)
	}
}

func TestNewTablePage_LinksOneFKBadgePerRelation(t *testing.T) {
	// Arrange: order_id is in a single-column relation and in a composite relation.
	orders := &schema.Table{Name: "orders"}
	items := &schema.Table{Name: "order_items"}
	shipments := &schema.Table{
		Name:    "shipments",
		Columns: []*schema.Column{{Name: "id"}, {Name: "order_id"}, {Name: "line_no"}},
	}
	s := &schema.Schema{
		Tables: []*schema.Table{orders, items, shipments},
		Relations: []*schema.Relation{
			{Table: shipments, Columns: []string{"order_id"}, ParentTable: orders, ParentColumns: []string{"id"}},
			{Table: shipments, Columns: []string{"order_id", "line_no"}, ParentTable: items, ParentColumns: []string{"order_id", "line_no"}},
		},
	}
	site := plan(t, s)

	// Act
	p := site.newTablePage(shipments)

	// Assert
	want := map[string][]keyBadge{
		"id": nil,
		"order_id": {
			{Label: "FK", Href: "../tables/orders.html", Title: "References orders id"},
			{Label: "FK", Href: "../tables/order_items.html", Title: "References order_items (order_id, line_no)"},
		},
		"line_no": {
			{Label: "FK", Href: "../tables/order_items.html", Title: "References order_items (order_id, line_no)"},
		},
	}
	for _, c := range p.Columns {
		if !reflect.DeepEqual(c.Keys, want[c.Name]) {
			t.Errorf("%s keys = %+v, want %+v", c.Name, c.Keys, want[c.Name])
		}
	}
}

func TestNewTablePage_MarksVirtualRelationFK(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users"}
	logs := &schema.Table{Name: "audit_logs", Columns: []*schema.Column{{Name: "user_id"}}}
	s := &schema.Schema{
		Tables:    []*schema.Table{users, logs},
		Relations: []*schema.Relation{{Table: logs, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}, Virtual: true}},
	}
	site := plan(t, s)

	// Act
	p := site.newTablePage(logs)

	// Assert
	want := []keyBadge{{Label: "FK", Href: "../tables/users.html", Title: "Virtual relation: References users id", Virtual: true}}
	if got := p.Columns[0].Keys; !reflect.DeepEqual(got, want) {
		t.Errorf("keys = %+v, want %+v", got, want)
	}
}

func TestNewTablePage_MarksColumnThatIsBothPKAndFK(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users"}
	userRoles := &schema.Table{
		Name:        "user_roles",
		Columns:     []*schema.Column{{Name: "user_id"}},
		Constraints: []*schema.Constraint{{Name: "pk", Kind: schema.ConstraintPrimaryKey, Columns: []string{"user_id"}}},
	}
	s := &schema.Schema{
		Tables:    []*schema.Table{users, userRoles},
		Relations: []*schema.Relation{{Table: userRoles, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}},
	}
	site := plan(t, s)

	// Act
	p := site.newTablePage(userRoles)

	// Assert
	if got := keyLabels(p)["user_id"]; !reflect.DeepEqual(got, []string{"PK", "FK"}) {
		t.Errorf("keys = %v, want [PK FK]", got)
	}
}

func TestNewTablePage_ListsParentsAndChildren(t *testing.T) {
	// Arrange: comments references users and itself.
	users := &schema.Table{Name: "users"}
	comments := &schema.Table{Name: "comments", Columns: []*schema.Column{{Name: "user_id"}, {Name: "parent_id"}}}
	toUser := &schema.Relation{Table: comments, Columns: []string{"user_id"}, Cardinality: "zero_or_more",
		ParentTable: users, ParentColumns: []string{"id"}, ParentCardinality: "exactly_one", Def: "FK user"}
	self := &schema.Relation{Table: comments, Columns: []string{"parent_id"}, Cardinality: "zero_or_more",
		ParentTable: comments, ParentColumns: []string{"id"}, ParentCardinality: "zero_or_one", Def: "FK self"}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{users, comments}, Relations: []*schema.Relation{toUser, self}})

	// Act
	p := site.newTablePage(comments)

	// Assert
	userRow := relationRow{Child: "comments", ChildHref: "../tables/comments.html", Parent: "users", ParentHref: "../tables/users.html",
		Columns: "user_id → id", Cardinality: "0..* : 1", Def: "FK user"}
	selfRow := relationRow{Child: "comments", ChildHref: "../tables/comments.html", Parent: "comments", ParentHref: "../tables/comments.html",
		Columns: "parent_id → id", Cardinality: "0..* : 0..1", Def: "FK self"}
	wantParents := relationTable{Rows: []relationRow{userRow, selfRow}, ShowParent: true}
	wantChildren := relationTable{Rows: []relationRow{selfRow}, ShowChild: true}
	if !reflect.DeepEqual(p.Parents, wantParents) {
		t.Errorf("Parents = %+v, want %+v", p.Parents, wantParents)
	}
	if !reflect.DeepEqual(p.Children, wantChildren) {
		t.Errorf("Children = %+v, want %+v", p.Children, wantChildren)
	}
}

func TestNewTablePage_OrdersPKBeforeFKsOfSeveralRelations(t *testing.T) {
	// Arrange: order_id is part of the PK and of two relations.
	orders := &schema.Table{Name: "orders"}
	other := &schema.Table{Name: "other"}
	items := &schema.Table{
		Name:        "order_items",
		Columns:     []*schema.Column{{Name: "order_id"}},
		Constraints: []*schema.Constraint{{Name: "pk", Kind: schema.ConstraintPrimaryKey, Columns: []string{"order_id"}}},
	}
	s := &schema.Schema{
		Tables: []*schema.Table{orders, other, items},
		Relations: []*schema.Relation{
			{Table: items, Columns: []string{"order_id"}, ParentTable: orders, ParentColumns: []string{"id"}},
			{Table: items, Columns: []string{"order_id"}, ParentTable: other, ParentColumns: []string{"id"}, Virtual: true},
		},
	}
	site := plan(t, s)

	// Act
	p := site.newTablePage(items)

	// Assert
	got := p.Columns[0].Keys
	if len(got) != 3 || got[0].Label != "PK" || got[1].Href != "../tables/orders.html" || got[2].Href != "../tables/other.html" || !got[2].Virtual {
		t.Errorf("keys = %+v, want PK, FK(orders), virtual FK(other)", got)
	}
}

func TestNewTablePage_ListsVirtualSelfReferenceAsChild(t *testing.T) {
	// Arrange
	nodes := &schema.Table{Name: "nodes", Columns: []*schema.Column{{Name: "parent_id"}}}
	self := &schema.Relation{Table: nodes, Columns: []string{"parent_id"}, ParentTable: nodes, ParentColumns: []string{"id"}, Virtual: true}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{nodes}, Relations: []*schema.Relation{self}})

	// Act
	p := site.newTablePage(nodes)

	// Assert
	if len(p.Children.Rows) != 1 || !p.Children.Rows[0].Virtual || p.Children.Rows[0].ChildHref != "../tables/nodes.html" {
		t.Errorf("Children = %+v, want one virtual self reference", p.Children)
	}
	if len(p.Parents.Rows) != 1 || !p.Parents.Rows[0].Virtual {
		t.Errorf("Parents = %+v, want one virtual self reference", p.Parents)
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
			site := plan(t, &schema.Schema{Tables: []*schema.Table{table}})

			// Act
			p := site.newTablePage(table)

			// Assert
			if p.ShowExtra != tt.want {
				t.Errorf("ShowExtra = %v, want %v", p.ShowExtra, tt.want)
			}
		})
	}
}

func TestNewRelationshipsPage_ListsAllRelationsWithBothEnds(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users"}
	items := &schema.Table{Name: "order_items"}
	shipments := &schema.Table{Name: "shipments"}
	s := &schema.Schema{
		Tables: []*schema.Table{users, items, shipments},
		Relations: []*schema.Relation{
			{Table: shipments, Columns: []string{"order_id", "line_no"}, ParentTable: items, ParentColumns: []string{"order_id", "line_no"}},
			{Table: shipments, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}, Virtual: true},
		},
	}
	site := plan(t, s)

	// Act
	p := site.newRelationshipsPage()

	// Assert
	want := relationTable{
		ShowChild:  true,
		ShowParent: true,
		Rows: []relationRow{
			{Child: "shipments", ChildHref: "tables/shipments.html", Parent: "order_items", ParentHref: "tables/order_items.html",
				Columns: "(order_id, line_no) → (order_id, line_no)"},
			{Child: "shipments", ChildHref: "tables/shipments.html", Parent: "users", ParentHref: "tables/users.html",
				Columns: "user_id → id", Virtual: true},
		},
	}
	if !reflect.DeepEqual(p.Relations, want) {
		t.Errorf("Relations = %+v, want %+v", p.Relations, want)
	}
}

func TestCardinalityPair(t *testing.T) {
	tests := []struct {
		child, parent schema.Cardinality
		want          string
	}{
		{"zero_or_more", "exactly_one", "0..* : 1"},
		{"one_or_more", "zero_or_one", "1..* : 0..1"},
		{"exactly_one", "exactly_one", "1 : 1"},
		{"", "", ""},
		{"zero_or_more", "", "0..* : ?"},
		{"many", "one", "many : one"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			// Act
			got := cardinalityPair(&schema.Relation{Cardinality: tt.child, ParentCardinality: tt.parent})

			// Assert
			if got != tt.want {
				t.Errorf("cardinalityPair(%q, %q) = %q, want %q", tt.child, tt.parent, got, tt.want)
			}
		})
	}
}

func TestNewTablesPage_CountsTablesAndViewsSeparately(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}, {Name: "name"}}, Indexes: []*schema.Index{{Name: "pk"}}}
	active := &schema.Table{Name: "active_users", View: true, Columns: []*schema.Column{{Name: "id"}}}
	s := &schema.Schema{Name: "db", Tables: []*schema.Table{users, active}, Relations: []*schema.Relation{{Table: active, Columns: []string{"id"}, ParentTable: users, ParentColumns: []string{"id"}}}}
	site := plan(t, s)

	// Act
	p := site.newTablesPage()

	// Assert
	if p.Count != "1 table, 1 view" {
		t.Errorf("Count = %q, want %q", p.Count, "1 table, 1 view")
	}
	if p.Tables[1].Href != "tables/active_users.html" {
		t.Errorf("Href = %q, want %q", p.Tables[1].Href, "tables/active_users.html")
	}
}

func TestNewTablesPage_FallsBackWhenDatabaseNameIsEmpty(t *testing.T) {
	// Arrange
	site := plan(t, &schema.Schema{})

	// Act
	p := site.newTablesPage()

	// Assert
	if p.DBName != "Database" {
		t.Errorf("DBName = %q, want %q", p.DBName, "Database")
	}
}

func TestNewColumnsPage_ListsColumnsOfAllTablesWithKeys(t *testing.T) {
	// Arrange
	users := &schema.Table{
		Name:        "users",
		Columns:     []*schema.Column{{Name: "id"}},
		Constraints: []*schema.Constraint{{Kind: schema.ConstraintPrimaryKey, Columns: []string{"id"}}},
	}
	orders := &schema.Table{Name: "orders", Columns: []*schema.Column{{Name: "user_id", ExtraDef: "x"}}}
	r := &schema.Relation{Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{users, orders}, Relations: []*schema.Relation{r}})

	// Act
	p := site.newColumnsPage()

	// Assert
	if len(p.Columns) != 2 {
		t.Fatalf("len(Columns) = %d, want 2", len(p.Columns))
	}
	idRow, fkRow := p.Columns[0], p.Columns[1]
	if idRow.Table != "users" || idRow.TableHref != "tables/users.html" || idRow.Name != "id" || idRow.Keys[0].Label != "PK" {
		t.Errorf("first row = %+v", idRow)
	}
	// Links are relative to the root, not to a table page.
	if fkRow.Table != "orders" || len(fkRow.Keys) != 1 || fkRow.Keys[0].Href != "tables/users.html" {
		t.Errorf("second row = %+v", fkRow)
	}
	if !p.ShowExtra || p.Current != menuColumns {
		t.Errorf("ShowExtra = %v, Current = %q", p.ShowExtra, p.Current)
	}
}

func TestNewConstraintsPage_ListsDatabaseConstraintsOnly(t *testing.T) {
	// Arrange: the virtual relation is not a constraint.
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}},
		Constraints: []*schema.Constraint{{Name: "users_pkey", Kind: schema.ConstraintPrimaryKey, Columns: []string{"id"}}}}
	logs := &schema.Table{Name: "logs", Columns: []*schema.Column{{Name: "user_id"}},
		Constraints: []*schema.Constraint{{Name: "logs_check", Kind: schema.ConstraintCheck}}}
	virtual := &schema.Relation{Table: logs, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}, Virtual: true}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{users, logs}, Relations: []*schema.Relation{virtual}})

	// Act
	p := site.newConstraintsPage()

	// Assert
	var got []string
	for _, c := range p.Constraints {
		got = append(got, c.Table+"/"+c.Name+"@"+c.TableHref)
	}
	want := []string{"users/users_pkey@tables/users.html", "logs/logs_check@tables/logs.html"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("constraints = %v, want %v", got, want)
	}
}

func TestPages_MarkCurrentMenu(t *testing.T) {
	// Arrange
	table := &schema.Table{Name: "t"}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{table}})

	// Act
	got := map[string]string{
		"tables":        site.newTablesPage().Current,
		"columns":       site.newColumnsPage().Current,
		"constraints":   site.newConstraintsPage().Current,
		"relationships": site.newRelationshipsPage().Current,
		"table page":    site.newTablePage(table).Current,
	}

	// Assert
	want := map[string]string{
		"tables":        menuTables,
		"columns":       menuColumns,
		"constraints":   menuConstraints,
		"relationships": menuRelationships,
		"table page":    "",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Current = %v, want %v", got, want)
	}
}

func TestPages_ShowCounts(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}, {Name: "name"}},
		Constraints: []*schema.Constraint{{Name: "pk", Kind: schema.ConstraintPrimaryKey, Columns: []string{"id"}}}}
	orders := &schema.Table{Name: "orders", Columns: []*schema.Column{{Name: "user_id"}}}
	r := &schema.Relation{Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{users, orders}, Relations: []*schema.Relation{r}})

	// Act
	got := []string{
		site.newTablesPage().Count,
		site.newColumnsPage().Count,
		site.newConstraintsPage().Count,
		site.newRelationshipsPage().Count,
	}

	// Assert: no views, so the tables count does not mention them.
	want := []string{"2 tables", "3 columns", "1 constraint", "1 relationship"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("counts = %v, want %v", got, want)
	}
}
