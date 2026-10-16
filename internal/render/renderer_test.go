package render

import (
	"bytes"
	"context"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/onozaty/dbfolio/internal/input/tbls"
	"github.com/onozaty/dbfolio/internal/output"
	"github.com/onozaty/dbfolio/internal/schema"
)

var update = flag.Bool("update", false, "update golden files")

func TestPlan_RejectsTablesMappedToSameFileName(t *testing.T) {
	// Arrange: "Users" gets a hash suffix, and another table is literally named like the result.
	s := &schema.Schema{Tables: []*schema.Table{{Name: "Users"}, {Name: output.FileName("Users")}}}

	// Act
	_, err := Plan(s)

	// Assert
	if err == nil || !strings.Contains(err.Error(), "same file name") {
		t.Errorf("Plan() error = %v, want same file name error", err)
	}
}

func TestPlan_RejectsRelationsToTablesOutsideSchema(t *testing.T) {
	users := &schema.Table{Name: "users"}
	outside := &schema.Table{Name: "outside"}
	tests := map[string]*schema.Relation{
		"child outside":  {Table: outside, ParentTable: users},
		"parent outside": {Table: users, ParentTable: outside},
		"nil parent":     {Table: users},
	}
	for name, r := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			s := &schema.Schema{Tables: []*schema.Table{users}, Relations: []*schema.Relation{r}}

			// Act
			_, err := Plan(s)

			// Assert
			if err == nil || !strings.Contains(err.Error(), "not in the schema") {
				t.Errorf("Plan() error = %v, want not in the schema error", err)
			}
		})
	}
}

func TestPlan_RejectsRelationsWithInvalidColumns(t *testing.T) {
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}}}
	orders := &schema.Table{Name: "orders", Columns: []*schema.Column{{Name: "user_id"}}}
	tests := map[string]*schema.Relation{
		"no columns":     {Table: orders, ParentTable: users},
		"count mismatch": {Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id", "x"}},
		"unknown column": {Table: orders, Columns: []string{"x"}, ParentTable: users, ParentColumns: []string{"id"}},
		"unknown parent": {Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"x"}},
	}
	for name, r := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			s := &schema.Schema{Tables: []*schema.Table{users, orders}, Relations: []*schema.Relation{r}}

			// Act
			_, err := Plan(s)

			// Assert
			if err == nil {
				t.Error("Plan() error = nil, want column error")
			}
		})
	}
}

func TestPrefixIDs_PrefixesOnlyIDAttributes(t *testing.T) {
	// Arrange
	svg := []byte(`<g id="node1" class="node"><title>id="x"</title><text>grid="y"</text></g>`)

	// Act
	got := prefixIDs(svg, "keys-")

	// Assert
	want := `<g id="keys-node1" class="node"><title>id="x"</title><text>grid="y"</text></g>`
	if string(got) != want {
		t.Errorf("prefixIDs() = %s, want %s", got, want)
	}
}

func TestRenderDiagrams_PrefixesOnlyGraphvizIDs(t *testing.T) {
	// Arrange: user text that looks like an id attribute is escaped by Graphviz.
	s := &schema.Schema{Tables: []*schema.Table{{Name: `t id="evil"`, Comment: ` id="evil2"`, Columns: []*schema.Column{{Name: "c"}}}}}
	site, err := Plan(s)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	err = site.RenderDiagrams(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("RenderDiagrams() error = %v", err)
	}
	for _, svg := range [][]byte{site.wholeKeys, site.wholeAll} {
		if bytes.Contains(svg, []byte("-evil")) {
			t.Errorf("user text was prefixed as an id:\n%s", svg)
		}
		if !bytes.Contains(svg, []byte(`id=&quot;evil&quot;`)) {
			t.Errorf("table name is missing or not escaped:\n%s", svg)
		}
	}
}

func TestWrite_RendersEmptyStates(t *testing.T) {
	tests := map[string]struct {
		schema *schema.Schema
		want   map[string]string
	}{
		"no tables": {
			&schema.Schema{},
			map[string]string{
				"index.html":         "<p>No tables.</p>",
				"columns.html":       "<p>No columns.</p>",
				"constraints.html":   "<p>No constraints.</p>",
				"relationships.html": "<p>No tables.</p>",
			},
		},
		"no relations": {
			&schema.Schema{Tables: []*schema.Table{{Name: "t", Columns: []*schema.Column{{Name: "c"}}}}},
			map[string]string{
				"constraints.html":   "<p>No constraints.</p>",
				"relationships.html": "<p>No relationships.</p>",
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			site, err := Plan(tt.schema)
			if err != nil {
				t.Fatal(err)
			}
			if err := site.RenderDiagrams(context.Background()); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()

			// Act
			err = site.Write(dir)

			// Assert
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}
			for file, want := range tt.want {
				b, err := os.ReadFile(filepath.Join(dir, file))
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(b), want) {
					t.Errorf("%s does not contain %q", file, want)
				}
			}
		})
	}
}

func TestWrite_WritesFunctionsAndEnumsPagesOnlyWhenPresent(t *testing.T) {
	tests := map[string]struct {
		schema *schema.Schema
		want   bool
	}{
		"absent":  {&schema.Schema{}, false},
		"present": {&schema.Schema{Functions: []*schema.Function{{Name: "f"}}, Enums: []*schema.Enum{{Name: "e"}}}, true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Arrange
			site, err := Plan(tt.schema)
			if err != nil {
				t.Fatal(err)
			}
			if err := site.RenderDiagrams(context.Background()); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()

			// Act
			err = site.Write(dir)

			// Assert
			if err != nil {
				t.Fatalf("Write() error = %v", err)
			}
			for _, file := range []string{"functions.html", "enums.html"} {
				_, statErr := os.Stat(filepath.Join(dir, file))
				if exists := statErr == nil; exists != tt.want {
					t.Errorf("%s exists = %v, want %v", file, exists, tt.want)
				}
			}
		})
	}
}

func TestWrite_RequiresRenderDiagrams(t *testing.T) {
	// Arrange
	site, err := Plan(&schema.Schema{Tables: []*schema.Table{{Name: "t"}}})
	if err != nil {
		t.Fatal(err)
	}

	// Act
	err = site.Write(t.TempDir())

	// Assert
	if err == nil || !strings.Contains(err.Error(), "RenderDiagrams") {
		t.Errorf("Write() error = %v, want RenderDiagrams error", err)
	}
}

func TestPlan_AcceptsNamesDifferingOnlyByCase(t *testing.T) {
	// Arrange
	s := &schema.Schema{Tables: []*schema.Table{{Name: "users"}, {Name: "Users"}}}

	// Act
	site, err := Plan(s)

	// Assert
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if site.files[s.Tables[0]] == site.files[s.Tables[1]] {
		t.Errorf("both tables map to %s", site.files[s.Tables[0]])
	}
}

func TestRenderDiagrams_RendersWholeAndRelatedTablesOnly(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users", Columns: []*schema.Column{{Name: "id"}}}
	orders := &schema.Table{Name: "orders", Columns: []*schema.Column{{Name: "user_id"}}}
	isolated := &schema.Table{Name: "isolated"}
	s := &schema.Schema{
		Tables:    []*schema.Table{users, orders, isolated},
		Relations: []*schema.Relation{{Table: orders, Columns: []string{"user_id"}, ParentTable: users, ParentColumns: []string{"id"}}},
	}
	site, err := Plan(s)
	if err != nil {
		t.Fatal(err)
	}

	// Act
	err = site.RenderDiagrams(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("RenderDiagrams() error = %v", err)
	}
	for name, svg := range map[string][]byte{"keys": site.wholeKeys, "all": site.wholeAll} {
		if !bytes.Contains(svg, []byte(`xlink:href="tables/isolated.html"`)) {
			t.Errorf("whole diagram (%s) does not link isolated table with a root-relative path", name)
		}
		if !bytes.Contains(svg, []byte(` id="`+name+`-graph0"`)) {
			t.Errorf("whole diagram (%s) IDs are not prefixed", name)
		}
	}
	if !bytes.Contains(site.wholeAll, []byte(">user_id</text>")) || bytes.Contains(site.wholeKeys, []byte(">bigint<")) {
		t.Error("whole diagram variants do not differ in columns shown")
	}
	if !bytes.Contains(site.tableDiagrams[orders], []byte(`xlink:href="../tables/users.html"`)) {
		t.Error("table diagram does not link the parent with a page-relative path")
	}
	if _, ok := site.tableDiagrams[isolated]; ok {
		t.Error("table without relations has a diagram")
	}
}

func TestRenderDiagrams_SkipsWholeDiagramForEmptySchema(t *testing.T) {
	// Arrange
	site, err := Plan(&schema.Schema{})
	if err != nil {
		t.Fatal(err)
	}

	// Act
	err = site.RenderDiagrams(context.Background())

	// Assert
	if err != nil || site.wholeKeys != nil || site.wholeAll != nil {
		t.Errorf("RenderDiagrams() error = %v, whole diagrams = %q / %q; want none", err, site.wholeKeys, site.wholeAll)
	}
}

// TestWrite_MatchesGolden compares the whole generated site with testdata/basic/expected.
// Run `go test ./internal/render -update` to regenerate the expected files.
func TestWrite_MatchesGolden(t *testing.T) {
	// Arrange
	base := filepath.Join("..", "..", "testdata", "basic")
	expected := filepath.Join(base, "expected")
	s, err := tbls.Load(filepath.Join(base, "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, err := Plan(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := site.RenderDiagrams(context.Background()); err != nil {
		t.Fatal(err)
	}
	outDir := t.TempDir()

	// Act
	err = site.Write(outDir)

	// Assert
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	got := readTree(t, outDir)
	if *update {
		if err := os.RemoveAll(expected); err != nil {
			t.Fatal(err)
		}
		for name, content := range got {
			path := filepath.Join(expected, name)
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		return
	}
	want := readTree(t, expected)
	for name, content := range want {
		if got[name] != content {
			t.Errorf("%s differs from golden file (run with -update to regenerate)", name)
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("%s is generated but has no golden file", name)
		}
	}
}

func readTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

// TestDiagramColorsAreThemed checks that every color Graphviz writes in the
// diagrams is recolored by app.css in dark mode, so that a color added to the
// DOT source is not left out of the theme.
func TestDiagramColorsAreThemed(t *testing.T) {
	// Arrange
	s, err := tbls.Load(filepath.Join("..", "..", "testdata", "basic", "schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	site, err := Plan(s)
	if err != nil {
		t.Fatal(err)
	}
	if err := site.RenderDiagrams(context.Background()); err != nil {
		t.Fatal(err)
	}
	css, err := content.ReadFile("assets/app.css")
	if err != nil {
		t.Fatal(err)
	}
	svgs := [][]byte{site.wholeKeys, site.wholeAll}
	for _, svg := range site.tableDiagrams {
		svgs = append(svgs, svg)
	}

	// Act
	colors := regexp.MustCompile(`(fill|stroke)="([^"]+)"`)
	seen := make(map[string]bool)
	for _, svg := range svgs {
		for _, m := range colors.FindAllSubmatch(svg, -1) {
			seen[string(m[1])+`="`+string(m[2])+`"`] = true
		}
	}

	// Assert
	for attr := range seen {
		if strings.HasSuffix(attr, `="none"`) {
			continue
		}
		if !bytes.Contains(css, []byte(".diagram svg ["+attr+"]")) {
			t.Errorf("app.css has no dark mode rule for %s", attr)
		}
	}
	// Text without a fill attribute is black by default.
	if !bytes.Contains(css, []byte(".diagram svg text:not([fill])")) {
		t.Error("app.css has no dark mode rule for text without fill")
	}
}
