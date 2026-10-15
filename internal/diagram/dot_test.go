package diagram

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/onozaty/dbfolio/internal/input/tbls"
	"github.com/onozaty/dbfolio/internal/schema"
)

var update = flag.Bool("update", false, "update golden files")

func hrefByName(t *schema.Table) string { return t.Name + ".html" }

func loadSample(t *testing.T) *schema.Schema {
	t.Helper()
	s, err := tbls.Load(filepath.Join("..", "..", "testdata", "basic", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func findTable(t *testing.T, s *schema.Schema, name string) *schema.Table {
	t.Helper()
	for _, x := range s.Tables {
		if x.Name == name {
			return x
		}
	}
	t.Fatalf("table %s not found", name)
	return nil
}

// assertGolden compares got with testdata/basic/expected-dot/<name>.
// Run `go test ./internal/diagram -update` to regenerate.
func assertGolden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "basic", "expected-dot", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("%s differs from golden file (run with -update to regenerate)\n--- got ---\n%s", name, got)
	}
}

func TestDOT_WholeSchemaMatchesGolden(t *testing.T) {
	tests := map[string]bool{
		"schema.keys.dot": true,
		"schema.all.dot":  false,
	}
	for golden, keysOnly := range tests {
		t.Run(golden, func(t *testing.T) {
			// Arrange
			s := loadSample(t)

			// Act
			got := DOT(Whole(s, keysOnly), hrefByName)

			// Assert
			assertGolden(t, golden, got)
		})
	}
}

func TestDOT_TableDiagramMatchesGolden(t *testing.T) {
	// Arrange: shipments has a single-column FK and a composite FK sharing a column.
	s := loadSample(t)
	shipments := findTable(t, s, "public.shipments")

	// Act
	got := DOT(ForTable(s, schema.NewRelationIndex(s), shipments, 1), hrefByName)

	// Assert
	assertGolden(t, "public.shipments.dot", got)
}

func TestForTable_IncludesRelationsAmongNeighborsOnly(t *testing.T) {
	// Arrange: orders -> users -> regions; orders is 2 hops from regions.
	regions := &schema.Table{Name: "regions"}
	users := &schema.Table{Name: "users"}
	orders := &schema.Table{Name: "orders"}
	toRegion := &schema.Relation{Table: users, ParentTable: regions}
	toUser := &schema.Relation{Table: orders, ParentTable: users}
	s := &schema.Schema{Tables: []*schema.Table{regions, users, orders}, Relations: []*schema.Relation{toRegion, toUser}}

	// Act
	d := ForTable(s, schema.NewRelationIndex(s), orders, 1)

	// Assert
	if len(d.Tables) != 2 || d.Tables[0] != users || d.Tables[1] != orders {
		t.Errorf("Tables = %v, want [users orders]", d.Tables)
	}
	if len(d.Relations) != 1 || d.Relations[0] != toUser {
		t.Errorf("Relations = %v, want [toUser]", d.Relations)
	}
	if d.Focus != orders {
		t.Errorf("Focus = %v, want orders", d.Focus)
	}
}

func TestDOT_EscapesNamesAndComments(t *testing.T) {
	// Arrange
	table := &schema.Table{
		Name:    `a"b<c>&d`,
		Comment: `line1 \N "quoted"` + "\nline2",
		Columns: []*schema.Column{{Name: "<col>", Type: `varchar("x")`}},
	}
	d := Diagram{Tables: []*schema.Table{table}}

	// Act
	got := DOT(d, func(*schema.Table) string { return `x\y.html` })

	// Assert
	for _, want := range []string{
		`<b>a&quot;b&lt;c&gt;&amp;d</b>`,                     // HTML label
		`tooltip="a\"b<c>&d` + "\n" + `line1 \\N \"quoted\"`, // DOT string: \N must not expand to the node name
		`URL="x\\y.html"`,
		`&lt;col&gt;`,
		`varchar(&quot;x&quot;)`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DOT does not contain %q\n%s", want, got)
		}
	}
	if strings.Contains(got, "line1 \\N &quot;quoted&quot;") {
		t.Error("the comment should be in the tooltip only, not in the label")
	}
}

func TestDOT_DrawsVirtualRelationDashed(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}}}
	logs := &schema.Table{Name: "logs", Columns: []*schema.Column{{Name: "x"}, {Name: "user_id"}}}
	r := &schema.Relation{Table: logs, Columns: []string{"user_id"}, Cardinality: "zero_or_more",
		ParentTable: users, ParentColumns: []string{"id"}, ParentCardinality: "zero_or_one", Virtual: true}
	d := Diagram{Tables: []*schema.Table{users, logs}, Relations: []*schema.Relation{r}}

	// Act
	got := DOT(d, hrefByName)

	// Assert: logs (t1) column 1 east -> users (t0) column 0 west.
	want := `t1:e1:e -> t0:w0:w [arrowtail=crowodot, arrowhead=none, style=dashed, tooltip="logs user_id → users id (virtual)"];`
	if !strings.Contains(got, want) {
		t.Errorf("DOT does not contain %q\n%s", want, got)
	}
}

func TestArrow(t *testing.T) {
	tests := map[schema.Cardinality]string{
		"exactly_one":  "teetee",
		"zero_or_one":  "teeodot",
		"zero_or_more": "crowodot",
		"one_or_more":  "crowtee",
		"":             "none",
		"unknown":      "none",
	}
	for c, want := range tests {
		t.Run(string(c), func(t *testing.T) {
			// Act
			got := arrow(c)

			// Assert
			if got != want {
				t.Errorf("arrow(%q) = %q, want %q", c, got, want)
			}
		})
	}
}

func TestEstimateWidth_CountsWideGlyphsWider(t *testing.T) {
	// Act
	ascii := estimateWidth("abcd", 10)
	japanese := estimateWidth("ユーザー", 10)
	empty := estimateWidth("", 10)

	// Assert
	if ascii != 20 { // 4 * 5
		t.Errorf("estimateWidth(ascii) = %d, want 20", ascii)
	}
	if japanese != 42 { // 4 * 10.5
		t.Errorf("estimateWidth(japanese) = %d, want 42", japanese)
	}
	if empty != 0 {
		t.Errorf("estimateWidth(empty) = %d, want 0", empty)
	}
}

func TestDOT_OmitsFontElementForEmptyType(t *testing.T) {
	// Arrange
	d := Diagram{Tables: []*schema.Table{{Name: "t", Columns: []*schema.Column{{Name: "c"}}}}}

	// Act
	got := DOT(d, hrefByName)

	// Assert
	if strings.Contains(got, "<font color=\"#666666\"></font>") {
		t.Errorf("DOT contains an empty font element\n%s", got)
	}
}

func TestSanitize_ReplacesCharactersInvalidInXML(t *testing.T) {
	tests := map[string]string{
		"keeps tab and newlines": "a\tb\nc\rd",
		"keeps unicode":          "ユーザー😀",
	}
	for name, in := range tests {
		t.Run(name, func(t *testing.T) {
			// Act
			got := sanitize(in)

			// Assert
			if got != in {
				t.Errorf("sanitize(%q) = %q, want unchanged", in, got)
			}
		})
	}
	t.Run("replaces control and non-characters", func(t *testing.T) {
		// Act
		got := sanitize("\x00\x01\v\x1f\uFFFE\uFFFF")

		// Assert
		if want := strings.Repeat("\uFFFD", 6); got != want {
			t.Errorf("sanitize() = %q, want %q", got, want)
		}
	})
}

func TestCellWidth_IncludesPadding(t *testing.T) {
	// Act
	ascii := cellWidth("user_id", 10)
	japanese := cellWidth("ユーザーID", 10)
	empty := cellWidth("", 10)

	// Assert
	if ascii != 45 { // 7 * 5 + 2 * 5
		t.Errorf("cellWidth(ascii) = %d, want 45", ascii)
	}
	if japanese != 62 { // 4 * 10.5 + 2 * 5 + 2 * 5
		t.Errorf("cellWidth(japanese) = %d, want 62", japanese)
	}
	if empty != 0 {
		t.Errorf("cellWidth(empty) = %d, want 0", empty)
	}
}

func TestDOT_KeysOnlyShowsKeyColumnsAndCountsTheRest(t *testing.T) {
	// Arrange
	users := &schema.Table{
		Name:        "users",
		Columns:     []*schema.Column{{Name: "id", Type: "bigint"}, {Name: "name", Type: "text"}, {Name: "email", Type: "text"}},
		Constraints: []*schema.Constraint{{Kind: schema.ConstraintPrimaryKey, Columns: []string{"id"}}},
	}
	orders := &schema.Table{
		Name:    "orders",
		Columns: []*schema.Column{{Name: "note", Type: "text"}, {Name: "user_id", Type: "bigint"}},
	}
	r := &schema.Relation{Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}
	d := Diagram{Tables: []*schema.Table{users, orders}, Relations: []*schema.Relation{r}, KeysOnly: true, SchemaRelations: []*schema.Relation{r}}

	// Act
	got := DOT(d, hrefByName)

	// Assert
	for _, want := range []string{
		// PK of users; the name gets the width of "… 2 more columns" beyond the key cell.
		`<tr><td port="w0" bgcolor="white" width="61" align="left">id</td><td port="e0" bgcolor="white" width="21" cellpadding="6" valign="bottom"><font point-size="9" color="#666666">PK</font></td></tr>`,
		// The relation column keeps its column index.
		`<tr><td port="w1" bgcolor="white" width="57" align="left">user_id</td><td port="e1" bgcolor="white" width="21" cellpadding="6" valign="bottom"><font point-size="9" color="#666666">FK</font></td></tr>`,
		"… 2 more columns",
		"… 1 more column<",
		`t1:e1:e -> t0:w0:w`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DOT does not contain %q\n%s", want, got)
		}
	}
	for _, unwanted := range []string{">name<", ">note<", "bigint"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("DOT contains %q, want key columns only without types", unwanted)
		}
	}
}

func TestWhole_SetsKeysOnly(t *testing.T) {
	for _, keysOnly := range []bool{true, false} {
		// Act
		d := Whole(&schema.Schema{}, keysOnly)

		// Assert
		if d.KeysOnly != keysOnly {
			t.Errorf("Whole(%v).KeysOnly = %v", keysOnly, d.KeysOnly)
		}
	}
}

func TestDOT_LabelsKeyColumns(t *testing.T) {
	// Arrange: user_id is PK and FK, role_id is a virtual FK to a table outside
	// the diagram, note is neither; users has no key at all.
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}}}
	roles := &schema.Table{Name: "roles", Columns: []*schema.Column{{Name: "id"}}}
	userRoles := &schema.Table{
		Name:        "user_roles",
		Columns:     []*schema.Column{{Name: "user_id"}, {Name: "role_id"}, {Name: "note"}},
		Constraints: []*schema.Constraint{{Kind: schema.ConstraintPrimaryKey, Columns: []string{"user_id"}}},
	}
	toUser := &schema.Relation{Table: userRoles, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}
	toRole := &schema.Relation{Table: userRoles, Columns: []string{"role_id"}, ParentTable: roles, ParentColumns: []string{"id"}, Virtual: true}
	d := Diagram{
		Tables:          []*schema.Table{users, userRoles},
		Relations:       []*schema.Relation{toUser},
		SchemaRelations: []*schema.Relation{toUser, toRole},
	}

	// Act
	got := DOT(d, hrefByName)

	// Assert
	row := `<tr><td port="w%d" bgcolor="white" width="50" align="left">%s</td><td bgcolor="white" width="0" align="left"></td><td port="e%d" bgcolor="white" width="21" cellpadding="6"%s</td></tr>`
	for _, want := range []string{
		fmt.Sprintf(row, 0, "user_id", 0, ` valign="bottom"><font point-size="9" color="#666666">PK</font>`), // PK and FK: PK only
		fmt.Sprintf(row, 1, "role_id", 1, ` valign="bottom"><font point-size="9" color="#666666">FK</font>`),
		fmt.Sprintf(row, 2, "note", 2, ">"),
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DOT does not contain %q\n%s", want, got)
		}
	}
	// A table without keys has no key column; the type cell carries the east port.
	if !strings.Contains(got, `<tr><td port="w0" bgcolor="white" width="41" align="left">id</td><td port="e0" bgcolor="white" width="0" align="left"></td></tr>`) {
		t.Errorf("table without keys should have no key column\n%s", got)
	}
}

func TestDOT_LabelsChildColumnsEvenWithoutSchemaRelations(t *testing.T) {
	// Arrange: KeysOnly, no SchemaRelations, users has no PK (a parent-only
	// table whose rows have a single cell).
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}}}
	orders := &schema.Table{Name: "orders", Columns: []*schema.Column{{Name: "user_id"}}}
	r := &schema.Relation{Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}
	d := Diagram{Tables: []*schema.Table{users, orders}, Relations: []*schema.Relation{r}, KeysOnly: true}

	// Act
	got := DOT(d, hrefByName)

	// Assert
	for _, want := range []string{
		`<tr><td port="w0" bgcolor="white" width="41" align="left">id</td></tr>`, // single cell: receives edges on its west side
		`>FK</font></td></tr>`,
		`t1:e0:e -> t0:w0:w`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DOT does not contain %q\n%s", want, got)
		}
	}
}

func TestDOT_HandlesTablesWithoutRows(t *testing.T) {
	tests := map[string]struct {
		d    Diagram
		want string
	}{
		"no columns": {
			Diagram{Tables: []*schema.Table{{Name: "empty"}}},
			`<tr><td bgcolor="#e8eef7" colspan="2" width="41" align="left"><b>empty</b></td></tr>
    </table>`,
		},
		"keys only without keys": {
			Diagram{Tables: []*schema.Table{{Name: "v", Columns: []*schema.Column{{Name: "a"}, {Name: "b"}}}}, KeysOnly: true},
			`<tr><td bgcolor="white" colspan="1" width="82" align="left"><font point-size="9" color="#666666">… 2 more columns</font></td></tr>`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Act
			got := DOT(tt.d, hrefByName)

			// Assert
			if !strings.Contains(got, tt.want) {
				t.Errorf("DOT does not contain %q\n%s", tt.want, got)
			}
		})
	}
}

func TestDOT_GivesExtraWidthToNameColumn(t *testing.T) {
	// Arrange: the table name is wider than the columns.
	table := &schema.Table{Name: strings.Repeat("n", 40), Columns: []*schema.Column{{Name: "id", Type: "int"}}}

	// Act
	got := DOT(Diagram{Tables: []*schema.Table{table}}, hrefByName)

	// Assert: total = (40 * 5.5 + 10) * 1.1 = 253; type 27 (3 * 5.5 + 10), name gets the rest.
	for _, want := range []string{
		`colspan="2" width="253" align="left"><b>`,
		`<td port="w0" bgcolor="white" width="226" align="left">id</td>`,
		`width="27" align="left"><font color="#666666">int</font>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("DOT does not contain %q\n%s", want, got)
		}
	}
}

func TestDOT_ShowsCommentInTooltipOnly(t *testing.T) {
	// Act
	got := DOT(Diagram{Tables: []*schema.Table{{Name: "t", Comment: "注文"}}}, hrefByName)

	// Assert
	if !strings.Contains(got, `tooltip="t`+"\n"+`注文"`) {
		t.Errorf("tooltip should have the comment\n%s", got)
	}
	if strings.Count(got, "注文") != 1 {
		t.Errorf("comment should not be in the label\n%s", got)
	}
}

func TestKindLabel(t *testing.T) {
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
			got := kindLabel(in)

			// Assert
			if got != want {
				t.Errorf("kindLabel(%q) = %q, want %q", in, got, want)
			}
		})
	}
}

func TestDOT_ShowsKindOfNonOrdinaryTables(t *testing.T) {
	// Arrange
	d := Diagram{Tables: []*schema.Table{
		{Name: "users", Type: "BASE TABLE", Columns: []*schema.Column{{Name: "id"}}},
		{Name: "active_users", Type: "VIEW", View: true, Columns: []*schema.Column{{Name: "id"}}},
	}}

	// Act
	got := DOT(d, hrefByName)

	// Assert
	if !strings.Contains(got, `<td bgcolor="#e8eef7" align="left"><b>active_users</b></td><td bgcolor="#e8eef7" align="right"><font point-size="9" color="#666666">[VIEW]</font></td>`) {
		t.Errorf("view header should show the name on the left and its kind on the right\n%s", got)
	}
	if !strings.Contains(got, `align="left"><b>users</b></td></tr>`) {
		t.Errorf("ordinary table header should show the name only, left-aligned\n%s", got)
	}
}
