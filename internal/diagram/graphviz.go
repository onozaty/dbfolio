package diagram

import (
	"bytes"
	"context"
	"fmt"

	"github.com/goccy/go-graphviz"
)

// Renderer renders DOT to SVG with the Graphviz bundled in go-graphviz (WASM),
// so no system Graphviz is needed. Create one per build: initialization is the
// costly part.
type Renderer struct {
	gv *graphviz.Graphviz
}

// NewRenderer initializes Graphviz.
func NewRenderer(ctx context.Context) (*Renderer, error) {
	gv, err := graphviz.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("initialize graphviz: %w", err)
	}
	return &Renderer{gv: gv}, nil
}

// Close releases Graphviz.
//
// The error from go-graphviz's Close is ignored: Graphviz reports the number of
// errors and warnings seen during the whole context there, including harmless
// font-metric warnings for non-ASCII text, and nothing tells which graph caused
// them. Some problems, such as a syntax error in an HTML-like label, are only
// warnings and never reach SVG's error either, so DOT must avoid them up front
// (see sanitize and escapeHTML).
func (r *Renderer) Close() {
	_ = r.gv.Close()
}

// SVG lays out dot and returns an SVG fragment for embedding in HTML.
func (r *Renderer) SVG(ctx context.Context, dot string) ([]byte, error) {
	g, err := graphviz.ParseBytes([]byte(dot))
	if err != nil {
		return nil, fmt.Errorf("parse dot: %w", err)
	}
	defer func() { _ = g.Close() }() // the rendered output is already complete
	var buf bytes.Buffer
	if err := r.gv.Render(ctx, g, graphviz.SVG, &buf); err != nil {
		return nil, fmt.Errorf("render svg: %w", err)
	}
	return inlineSVG(buf.Bytes()), nil
}

// inlineSVG drops the XML declaration, DOCTYPE and comments that Graphviz
// emits before the <svg> element; they are not allowed inside HTML. It also
// drops the graph's <title> (the graph name), which browsers would show as a
// tooltip over the empty area of the diagram.
func inlineSVG(b []byte) []byte {
	if i := bytes.Index(b, []byte("<svg")); i > 0 {
		b = b[i:]
	}
	return bytes.Replace(b, []byte("<title>"+graphName+"</title>\n"), nil, 1)
}
