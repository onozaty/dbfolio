// Package diagram builds ER diagrams as Graphviz DOT and renders them to SVG.
package diagram

import (
	"fmt"
	"strings"

	"github.com/onozaty/dbfolio/internal/schema"
)

// Diagram is a set of tables and the relations among them.
type Diagram struct {
	Tables    []*schema.Table
	Relations []*schema.Relation
	// Focus is highlighted; nil for the whole-schema diagram.
	Focus *schema.Table
	// KeysOnly shows only key columns (primary keys and relation columns),
	// without types, and summarizes the rest as a count. It keeps large
	// diagrams compact.
	KeysOnly bool
	// SchemaRelations are all relations of the schema, used to mark FK columns
	// even when the parent table is outside the diagram.
	SchemaRelations []*schema.Relation
}

// Whole returns the diagram of all tables and relations.
func Whole(s *schema.Schema, keysOnly bool) Diagram {
	return Diagram{Tables: s.Tables, Relations: s.Relations, KeysOnly: keysOnly, SchemaRelations: s.Relations}
}

// ForTable returns the diagram centered on t: t, the tables within distance,
// and the relations among them.
func ForTable(s *schema.Schema, idx *schema.RelationIndex, t *schema.Table, distance int) Diagram {
	tables := idx.RelatedTables(t, distance)
	in := make(map[*schema.Table]bool, len(tables))
	for _, x := range tables {
		in[x] = true
	}
	d := Diagram{Tables: tables, Focus: t, SchemaRelations: s.Relations}
	for _, r := range s.Relations {
		if in[r.Table] && in[r.ParentTable] {
			d.Relations = append(d.Relations, r)
		}
	}
	return d
}

// Diagram colors. The page recolors them in dark mode by matching these exact
// values (and Graphviz's "white" / "black"), see internal/render/assets/app.css.
const (
	headerColor = "#e8eef7"
	focusColor  = "#ffe9b3"
	lineColor   = "#666666"
	typeColor   = "#666666"
)

// graphName is the name of every diagram graph; inlineSVG removes its title.
const graphName = "dbfolio"

// DOT returns the Graphviz source of d. href gives the link of each table node.
//
// Tables are drawn as HTML-like labels with one row per column. Children are
// ranked left of parents (rankdir=LR), so an edge leaves the child's east side
// and enters the parent's west side. With types shown, each column has separate
// west and east ports so that edges attach at the node border; a single port
// would make the edge cross the type cell.
func DOT(d Diagram, href func(*schema.Table) string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "digraph %s {\n", graphName)
	b.WriteString("  graph [rankdir=LR, pad=0.2, nodesep=0.3, ranksep=0.8, fontname=\"Helvetica\"];\n")
	b.WriteString("  node [shape=plaintext, fontname=\"Helvetica\", fontsize=11];\n")
	fmt.Fprintf(&b, "  edge [dir=both, color=%s];\n", quote(lineColor))

	var keys map[*schema.Table]map[string]bool
	if d.KeysOnly {
		keys = keyColumns(d)
	}
	// d.Relations too, so that child columns are always labeled FK: their rows
	// then have a key cell carrying the east port that edges leave from.
	fks := fkColumns(append(d.SchemaRelations[:len(d.SchemaRelations):len(d.SchemaRelations)], d.Relations...))
	ids := make(map[*schema.Table]string, len(d.Tables))
	for i, t := range d.Tables {
		// Index-based IDs keep DOT valid whatever the table name contains.
		ids[t] = fmt.Sprintf("t%d", i)
		writeNode(&b, ids[t], t, t == d.Focus, href(t), keys[t], fks[t], d.KeysOnly)
	}
	for _, r := range d.Relations {
		writeEdge(&b, r, ids)
	}
	b.WriteString("}\n")
	return b.String()
}

// fkColumns returns, per table, the columns that reference a parent table.
// Relations include virtual ones, which have no FK constraint.
func fkColumns(relations []*schema.Relation) map[*schema.Table]map[string]bool {
	fks := make(map[*schema.Table]map[string]bool)
	for _, r := range relations {
		if fks[r.Table] == nil {
			fks[r.Table] = make(map[string]bool)
		}
		for _, c := range r.Columns {
			fks[r.Table][c] = true
		}
	}
	return fks
}

// keyColumns returns, per table, the primary key columns and the columns used
// by the diagram's relations (which edges attach to).
func keyColumns(d Diagram) map[*schema.Table]map[string]bool {
	keys := make(map[*schema.Table]map[string]bool, len(d.Tables))
	add := func(t *schema.Table, cols []string) {
		if keys[t] == nil {
			keys[t] = make(map[string]bool)
		}
		for _, c := range cols {
			keys[t][c] = true
		}
	}
	for _, t := range d.Tables {
		for _, c := range t.Constraints {
			if c.Kind == schema.ConstraintPrimaryKey {
				add(t, c.Columns)
			}
		}
	}
	for _, r := range d.Relations {
		add(r.Table, r.Columns)
		add(r.ParentTable, r.ParentColumns)
	}
	return keys
}

// writeNode writes a table as an HTML-like label.
//
// Each column row has the name cell, the type cell (unless KeysOnly) and a key
// cell labeled "PK" / "FK" (only when some column of the table is a key). The
// first cell carries the west port and the last cell the east port, so that
// edges attach at the node border without crossing other cells. With a single
// cell per row (KeysOnly, no key label: a parent-only table without PK), that
// cell only receives edges, on its west side. Child rows always have a label.
//
// Column widths are set explicitly and the same in every row. Otherwise, when
// the header is wider than the columns, Graphviz spreads the extra
// width over all columns; here it all goes to the name.
func writeNode(b *strings.Builder, id string, t *schema.Table, focus bool, href string, keys, fks map[string]bool, keysOnly bool) {
	bg := headerColor
	if focus {
		bg = focusColor
	}
	pks := primaryKeyColumns(t)

	type row struct {
		index int
		col   *schema.Column
	}
	var rows []row
	for i, c := range t.Columns {
		if !keysOnly || keys[c.Name] {
			rows = append(rows, row{i, c})
		}
	}
	hidden := len(t.Columns) - len(rows)
	more := ""
	if hidden > 0 {
		noun := "columns"
		if hidden == 1 {
			noun = "column"
		}
		more = fmt.Sprintf("… %d more %s", hidden, noun)
	}

	showKeys := false
	nameWidth, typeWidth, keyWidth := 0, 0, 0
	for _, r := range rows {
		c := r.col
		nameWidth = max(nameWidth, cellWidth(c.Name, 11))
		if !keysOnly {
			typeWidth = max(typeWidth, cellWidth(c.Type, 11))
		}
		if label := keyLabel(pks[c.Name], fks[c.Name]); label != "" {
			showKeys = true
			keyWidth = max(keyWidth, estimateWidth(label, keyFontSize)+2*keyCellPadding)
		}
	}
	// Bold text is wider than the estimate for regular text.
	headerWidth := cellWidth(t.Name, 11) * 11 / 10
	kind := t.KindLabel()
	if kind != "" {
		kind = "[" + kind + "]"
		headerWidth += estimateWidth(kind, 9) + kindGap
	}
	total := max(nameWidth+typeWidth+keyWidth, headerWidth, cellWidth(more, 9))
	nameWidth = total - typeWidth - keyWidth
	span := rowCells(keysOnly, showKeys)

	// The comment is shown only in the tooltip: a comment row would make tables
	// with and without comments look inconsistent.
	tooltip := t.Name
	if t.Comment != "" {
		tooltip += "\n" + t.Comment
	}
	fmt.Fprintf(b, "  %s [URL=%s, tooltip=%s, label=<\n", id, quote(href), quote(tooltip))
	fmt.Fprintf(b, "    <table border=\"0\" cellborder=\"1\" cellspacing=\"0\" cellpadding=\"%d\">\n", cellPadding)
	// Cells have a background color because unfilled cells are clickable only on their glyphs.
	fmt.Fprintf(b, "      <tr><td bgcolor=%q colspan=\"%d\" width=\"%d\" align=\"left\">%s</td></tr>\n", bg, span, total, headerContent(t.Name, kind, bg))
	for _, r := range rows {
		i, c := r.index, r.col
		cells := []string{fmt.Sprintf(`<td bgcolor="white" width="%d" align="left">%s</td>`, nameWidth, escapeHTML(c.Name))}
		if !keysOnly {
			// An empty <font></font> is invalid in Graphviz HTML labels and would drop the whole label (and its ports).
			typ := ""
			if c.Type != "" {
				typ = fmt.Sprintf("<font color=%q>%s</font>", typeColor, escapeHTML(c.Type))
			}
			cells = append(cells, fmt.Sprintf(`<td bgcolor="white" width="%d" align="left">%s</td>`, typeWidth, typ))
		}
		if showKeys {
			cells = append(cells, keyCell(keyWidth, pks[c.Name], fks[c.Name]))
		}
		cells[0] = strings.Replace(cells[0], "<td ", fmt.Sprintf(`<td port="w%d" `, i), 1)
		if last := len(cells) - 1; last > 0 {
			cells[last] = strings.Replace(cells[last], "<td ", fmt.Sprintf(`<td port="e%d" `, i), 1)
		}
		fmt.Fprintf(b, "      <tr>%s</tr>\n", strings.Join(cells, ""))
	}
	if more != "" {
		fmt.Fprintf(b, "      <tr><td bgcolor=\"white\" colspan=\"%d\" width=\"%d\" align=\"left\"><font point-size=\"9\" color=%q>%s</font></td></tr>\n",
			span, total, typeColor, more)
	}
	b.WriteString("    </table>>];\n")
}

// kindGap is the minimum space in points between the table name and its kind.
const kindGap = 8

// headerContent returns the header cell content: the table name on the left
// and, if any, its kind on the right. The kind sits in a borderless inner
// table, which Graphviz stretches to the cell width; its two cells then hold
// the name at the left end and the kind at the right end.
func headerContent(name, kind, bg string) string {
	bold := fmt.Sprintf("<b>%s</b>", escapeHTML(name))
	if kind == "" {
		return bold
	}
	return fmt.Sprintf(`<table border="0" cellborder="0" cellspacing="0" cellpadding="0">`+
		`<tr><td bgcolor=%q align="left">%s</td><td bgcolor=%q align="right"><font point-size="9" color=%q>%s</font></td></tr></table>`,
		bg, bold, bg, typeColor, escapeHTML(kind))
}

// rowCells returns the number of cells in a column row.
func rowCells(keysOnly, showKeys bool) int {
	n := 1
	if !keysOnly {
		n++
	}
	if showKeys {
		n++
	}
	return n
}

func primaryKeyColumns(t *schema.Table) map[string]bool {
	pks := make(map[string]bool)
	for _, c := range t.Constraints {
		if c.Kind == schema.ConstraintPrimaryKey {
			for _, col := range c.Columns {
				pks[col] = true
			}
		}
	}
	return pks
}

// keyLabel returns the key label of a column. A column that is both PK and FK
// is labeled PK only, to keep the column narrow.
func keyLabel(pk, fk bool) string {
	switch {
	case pk:
		return "PK"
	case fk:
		return "FK"
	default:
		return ""
	}
}

const (
	// keyFontSize is smaller than the column text so that labels do not stand out.
	keyFontSize = 9
	// keyCellPadding with valign="bottom" puts the smaller label on the same
	// baseline as the name and type (Graphviz centers text vertically by its
	// size, so a centered smaller label would sit higher). Measured with the
	// 11pt column text and cellPadding 5.
	keyCellPadding = 6
)

// keyCell returns the cell labeling a column as PK or FK. Labels are in the
// type color: the text tells the kind, and colored labels would stand out from
// the otherwise monochrome rows.
func keyCell(width int, pk, fk bool) string {
	label := keyLabel(pk, fk)
	if label == "" {
		return fmt.Sprintf(`<td bgcolor="white" width="%d" cellpadding="%d"></td>`, width, keyCellPadding)
	}
	// Not bold: Graphviz estimates bold text taller, which also raises it.
	return fmt.Sprintf(`<td bgcolor="white" width="%d" cellpadding="%d" valign="bottom"><font point-size="%d" color=%q>%s</font></td>`,
		width, keyCellPadding, keyFontSize, typeColor, label)
}

// writeEdge writes r as an edge from the child's east side to the parent's west side.
func writeEdge(b *strings.Builder, r *schema.Relation, ids map[*schema.Table]string) {
	style := "solid"
	if r.Virtual {
		style = "dashed"
	}
	// Composite relations attach to their first columns only; edges per column
	// would clutter the diagram. The tooltip shows the full column mapping.
	tooltip := r.Table.Name + " " + columnList(r.Columns) + " → " + r.ParentTable.Name + " " + columnList(r.ParentColumns)
	if r.Virtual {
		tooltip += " (virtual)"
	}
	if r.Def != "" {
		tooltip += "\n" + r.Def
	}
	// Only the child end gets a crow's foot mark. Many edges often meet at the
	// same parent column (e.g. users.id), where marks would overlap into a
	// blot; the parent cardinality is on the table and relationships pages.
	fmt.Fprintf(b, "  %s:e%d:e -> %s:w%d:w [arrowtail=%s, arrowhead=none, style=%s, tooltip=%s];\n",
		ids[r.Table], columnIndex(r.Table, r.Columns[0]),
		ids[r.ParentTable], columnIndex(r.ParentTable, r.ParentColumns[0]),
		arrow(r.Cardinality), style, quote(tooltip))
}

// arrow maps a cardinality to a crow's foot arrow. Arrow shapes are drawn from
// the tip outward, so the mark nearest the table comes first.
func arrow(c schema.Cardinality) string {
	switch c {
	case "exactly_one":
		return "teetee"
	case "zero_or_one":
		return "teeodot"
	case "zero_or_more":
		return "crowodot"
	case "one_or_more":
		return "crowtee"
	default:
		return "none"
	}
}

// columnIndex returns the position of the named column. Relation columns are
// guaranteed to exist by the input layer and re-checked by render.Plan.
func columnIndex(t *schema.Table, name string) int {
	for i, c := range t.Columns {
		if c.Name == name {
			return i
		}
	}
	panic(fmt.Sprintf("diagram: column %q not found in table %s", name, t.Name))
}

func columnList(cols []string) string {
	if len(cols) == 1 {
		return cols[0]
	}
	return "(" + strings.Join(cols, ", ") + ")"
}

// cellPadding is the space between a cell's text and its border, in points.
const cellPadding = 5

// cellWidth returns the width in points of a cell holding s, including the
// cell padding (Graphviz's width attribute includes it); 0 for empty text.
// The 1pt borders, also included by Graphviz, fit in the estimate's margin.
//
// The WASM build of Graphviz has no font files: it estimates text widths from
// built-in metrics, which fit ASCII but are far too narrow for CJK and other
// wide glyphs. Widths are therefore set from dbfolio's own estimate, which is
// also needed to keep column widths fixed (see writeNode).
func cellWidth(s string, pt int) int {
	if s == "" {
		return 0
	}
	return estimateWidth(s, pt) + 2*cellPadding
}

// estimateWidth returns the width in points of s rendered at the given font size.
func estimateWidth(s string, pt int) int {
	w := 0.0
	for _, r := range s {
		if isWide(r) {
			w += float64(pt) * 1.05
		} else {
			// Helvetica / Arial average about 0.47em; leave a little room.
			w += float64(pt) * 0.5
		}
	}
	return int(w + 0.5)
}

// isWide reports whether r is typically rendered full-width (CJK, kana, full-width forms).
func isWide(r rune) bool {
	return r >= 0x2E80 && r <= 0xA4CF || // CJK radicals .. Yi
		r >= 0xAC00 && r <= 0xD7A3 || // Hangul syllables
		r >= 0xF900 && r <= 0xFAFF || // CJK compatibility ideographs
		r >= 0xFE30 && r <= 0xFE4F || // CJK compatibility forms
		r >= 0xFF00 && r <= 0xFF60 || // full-width forms
		r >= 0xFFE0 && r <= 0xFFE6 ||
		r >= 0x20000 && r <= 0x3FFFD // CJK extensions
}

// quote returns a DOT double-quoted string. Backslashes are doubled because
// attributes such as tooltip and URL treat sequences like \N as escapes.
func quote(s string) string {
	s = sanitize(s)
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

// escapeHTML escapes text for Graphviz HTML-like labels.
func escapeHTML(s string) string {
	return htmlEscaper.Replace(sanitize(s))
}

var htmlEscaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")

// sanitize replaces characters that XML 1.0 does not allow. Graphviz parses
// HTML-like labels as XML: such a character makes it drop the whole label (and
// its ports) with only a warning, which go-graphviz does not report. NUL also
// breaks DOT quoted strings. Names and comments come from the database and can
// contain them.
func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r == '\t' || r == '\n' || r == '\r':
			return r
		case r < 0x20, r == 0xFFFE, r == 0xFFFF, r >= 0xD800 && r <= 0xDFFF:
			return '\uFFFD'
		}
		return r
	}, s)
}
