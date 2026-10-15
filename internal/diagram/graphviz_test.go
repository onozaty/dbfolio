package diagram

import (
	"bytes"
	"context"
	"testing"

	"github.com/onozaty/dbfolio/internal/schema"
)

func newRenderer(t *testing.T) *Renderer {
	t.Helper()
	r, err := NewRenderer(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Close)
	return r
}

func TestSVG_ReturnsInlineSVGWithLinks(t *testing.T) {
	// Arrange
	r := newRenderer(t)
	dot := `digraph { a [URL="a.html", label="ユーザー"]; }`

	// Act
	svg, err := r.SVG(context.Background(), dot)

	// Assert
	if err != nil {
		t.Fatalf("SVG() error = %v", err)
	}
	if !bytes.HasPrefix(svg, []byte("<svg")) {
		t.Errorf("SVG does not start with <svg (XML declaration or DOCTYPE left?): %.80s", svg)
	}
	for _, want := range []string{`xlink:href="a.html"`, "ユーザー"} {
		if !bytes.Contains(svg, []byte(want)) {
			t.Errorf("SVG does not contain %q", want)
		}
	}
}

func TestSVG_ReturnsErrorForInvalidDOT(t *testing.T) {
	// Arrange
	r := newRenderer(t)

	// Act
	_, err := r.SVG(context.Background(), "digraph {")

	// Assert
	if err == nil {
		t.Error("SVG() error = nil, want parse error")
	}
}

func TestInlineSVG_StripsPrologue(t *testing.T) {
	// Arrange
	in := []byte("<?xml version=\"1.0\"?>\n<!DOCTYPE svg>\n<!-- c -->\n<svg></svg>")

	// Act
	got := inlineSVG(in)

	// Assert
	if string(got) != "<svg></svg>" {
		t.Errorf("inlineSVG() = %q, want %q", got, "<svg></svg>")
	}
}

func TestSVG_KeepsLabelsContainingCharactersInvalidInXML(t *testing.T) {
	// Arrange: Graphviz drops an HTML-like label containing such characters
	// with only a warning, so check the rendered text, not just the DOT.
	r := newRenderer(t)
	table := &schema.Table{
		Name:    "a\x01b",
		Comment: "c\vd\x00e￿",
		Columns: []*schema.Column{{Name: "col\x02", Type: "int"}},
	}
	dot := DOT(Diagram{Tables: []*schema.Table{table}}, hrefByName)

	// Act
	svg, err := r.SVG(context.Background(), dot)

	// Assert
	if err != nil {
		t.Fatalf("SVG() error = %v", err)
	}
	for _, want := range []string{">a�b</text>", ">col�</text>", ">int</text>"} {
		if !bytes.Contains(svg, []byte(want)) {
			t.Errorf("SVG does not contain %q; the label was dropped", want)
		}
	}
}

func TestSVG_KeepsLabelsWithEmptyKeyCells(t *testing.T) {
	// Arrange: "note" has neither PK nor FK, so its key cell is empty.
	r := newRenderer(t)
	users := &schema.Table{
		Name:        "users",
		Columns:     []*schema.Column{{Name: "id", Type: "bigint"}, {Name: "note", Type: "text"}},
		Constraints: []*schema.Constraint{{Kind: schema.ConstraintPrimaryKey, Columns: []string{"id"}}},
	}
	dot := DOT(Diagram{Tables: []*schema.Table{users}}, hrefByName)

	// Act
	svg, err := r.SVG(context.Background(), dot)

	// Assert
	if err != nil {
		t.Fatalf("SVG() error = %v", err)
	}
	for _, want := range []string{">PK</text>", ">note</text>", ">text</text>"} {
		if !bytes.Contains(svg, []byte(want)) {
			t.Errorf("SVG does not contain %q; the label was dropped", want)
		}
	}
}

func TestSVG_HasNoGraphTitle(t *testing.T) {
	// Arrange: browsers show the graph title as a tooltip over empty space.
	r := newRenderer(t)

	// Act
	svg, err := r.SVG(context.Background(), DOT(Diagram{Tables: []*schema.Table{{Name: "t"}}}, hrefByName))

	// Assert
	if err != nil {
		t.Fatalf("SVG() error = %v", err)
	}
	if bytes.Contains(svg, []byte("<title>"+graphName+"</title>")) {
		t.Error("SVG has the graph title")
	}
}
