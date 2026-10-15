package render

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/onozaty/dbfolio/internal/schema"
)

func TestSearchEntries_IndexesTablesAndColumns(t *testing.T) {
	// Arrange
	users := &schema.Table{Name: "users", Type: "BASE TABLE", Comment: "ユーザー",
		Columns: []*schema.Column{{Name: "id", Type: "bigint", Comment: "ID"}, {Name: "name", Type: "text"}}}
	active := &schema.Table{Name: "active_users", Type: "VIEW", View: true}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{users, active}})

	// Act
	got := site.searchEntries()

	// Assert
	want := []searchEntry{
		{Kind: "t", Name: "users", Comment: "ユーザー", Href: "tables/users.html"},
		{Kind: "c", Name: "id", Table: "users", Type: "bigint", Comment: "ID", Href: "tables/users.html#column-0"},
		{Kind: "c", Name: "name", Table: "users", Type: "text", Href: "tables/users.html#column-1"},
		{Kind: "t", Name: "active_users", Type: "VIEW", Href: "tables/active_users.html"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("searchEntries() =\n%+v\nwant\n%+v", got, want)
	}
}

func TestWriteSearchIndex_IsSafeJavaScript(t *testing.T) {
	// Arrange: text that could end the script or break a string literal.
	comment := "</script><script>alert(1)</script>\n\u2028\"quoted\" & more"
	site := plan(t, &schema.Schema{Tables: []*schema.Table{{Name: "t", Comment: comment}}})
	var buf bytes.Buffer

	// Act
	err := site.writeSearchIndex(&buf)

	// Assert
	if err != nil {
		t.Fatalf("writeSearchIndex() error = %v", err)
	}
	out := buf.String()
	for _, bad := range []string{"</script", "<script", "\u2028"} {
		if strings.Contains(out, bad) {
			t.Errorf("index contains %q:\n%s", bad, out)
		}
	}
	const prefix, suffix = "window.DBFOLIO_SEARCH_INDEX = ", ";\n"
	if !strings.HasPrefix(out, prefix) || !strings.HasSuffix(out, suffix) || strings.Count(out, "\n") != 1 {
		t.Fatalf("unexpected format:\n%s", out)
	}
	var entries []searchEntry
	if err := json.Unmarshal([]byte(strings.TrimSuffix(strings.TrimPrefix(out, prefix), suffix)), &entries); err != nil {
		t.Fatalf("index is not valid JSON: %v", err)
	}
	if entries[0].Comment != comment {
		t.Errorf("comment = %q, want %q", entries[0].Comment, comment)
	}
}

func TestWriteSearchIndex_WritesEmptyArrayForEmptySchema(t *testing.T) {
	// Arrange
	site := plan(t, &schema.Schema{})
	var buf bytes.Buffer

	// Act
	err := site.writeSearchIndex(&buf)

	// Assert
	if err != nil || buf.String() != "window.DBFOLIO_SEARCH_INDEX = [];\n" {
		t.Errorf("writeSearchIndex() = %q, %v", buf.String(), err)
	}
}

func TestNewTablePage_AnchorsColumnRows(t *testing.T) {
	// Arrange
	table := &schema.Table{Name: "t", Columns: []*schema.Column{{Name: "a"}, {Name: "b"}}}
	site := plan(t, &schema.Schema{Tables: []*schema.Table{table}})
	var buf bytes.Buffer

	// Act
	err := tableTmpl.ExecuteTemplate(&buf, "layout", site.newTablePage(table))

	// Assert
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`<tr id="column-0"><td>a</td>`, `<tr id="column-1"><td>b</td>`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("table page does not contain %q", want)
		}
	}
}

// TestSearchJS runs the tests of the search logic in assets/search.js with node.
func TestSearchJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not found")
	}
	out, err := exec.Command(node, filepath.Join("testdata", "search_test.js")).CombinedOutput()
	if err != nil {
		t.Fatalf("search_test.js failed: %v\n%s", err, out)
	}
}
