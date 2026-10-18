// Package render generates the HTML documentation site.
package render

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"slices"

	"github.com/onozaty/dbfolio/internal/diagram"
	"github.com/onozaty/dbfolio/internal/output"
	"github.com/onozaty/dbfolio/internal/schema"
)

//go:embed templates assets
var content embed.FS

const (
	tablesDir = "tables"
	assetsDir = "assets"
)

var (
	tablesTmpl        = parsePage("templates/index.html")
	relationshipsTmpl = parsePage("templates/relationships.html")
	tableTmpl         = parsePage("templates/table.html")
	functionsTmpl     = parsePage("templates/functions.html")
	enumsTmpl         = parsePage("templates/enums.html")
)

// tableDiagramDistance is how far table-centered diagrams reach.
const tableDiagramDistance = 1

func parsePage(name string) *template.Template {
	return template.Must(template.New("").Funcs(template.FuncMap{
		"columnAnchor":     columnAnchor,
		"constraintAnchor": constraintAnchor,
		"indexAnchor":      indexAnchor,
	}).
		ParseFS(content, "templates/layout.html", "templates/partials.html", name))
}

// Site is a planned documentation site. Planning is separated from writing so
// that problems are detected before the output directory is touched.
type Site struct {
	schema *schema.Schema
	// files maps each table to its page file name under tablesDir.
	files     map[*schema.Table]string
	relations *schema.RelationIndex
	// tablesByName resolves referenced tables of views.
	tablesByName map[string]*schema.Table
	enums        *enumResolver
	// wholeKeys, wholeAll and tableDiagrams hold inline SVG, set by RenderDiagrams.
	// The whole-schema diagram comes in two variants (key columns only / all
	// columns) that the page switches between. Tables without relations have no diagram.
	wholeKeys     []byte
	wholeAll      []byte
	tableDiagrams map[*schema.Table][]byte
	// customCSS is the user's stylesheet, written as customCSSFile and loaded
	// after app.css; hasCustomCSS tells an empty one from none.
	customCSS    []byte
	hasCustomCSS bool
}

// customCSSFile is the user's stylesheet in the assets directory.
const customCSSFile = "custom.css"

// SetCustomCSS adds the user's stylesheet to every page, after the default
// one so that it overrides it.
func (site *Site) SetCustomCSS(css []byte) {
	site.customCSS = css
	site.hasCustomCSS = true
}

// Plan decides the page file names, failing if two tables map to the same file.
func Plan(s *schema.Schema) (*Site, error) {
	files := make(map[*schema.Table]string, len(s.Tables))
	owners := make(map[string]string, len(s.Tables))
	for _, t := range s.Tables {
		name := output.FileName(t.Name) + ".html"
		if other, ok := owners[name]; ok {
			return nil, fmt.Errorf("tables %s and %s map to the same file name %s", other, t.Name, name)
		}
		owners[name] = t.Name
		files[t] = name
	}
	// The input layer already guarantees this; checking again keeps Write from
	// failing (after the output directory is cleaned) for a Schema built elsewhere.
	for i, r := range s.Relations {
		if _, ok := files[r.Table]; !ok {
			return nil, fmt.Errorf("relations[%d]: child table is not in the schema", i)
		}
		if _, ok := files[r.ParentTable]; !ok {
			return nil, fmt.Errorf("relations[%d]: parent table is not in the schema", i)
		}
		if err := checkRelationColumns(r); err != nil {
			return nil, fmt.Errorf("relations[%d]: %w", i, err)
		}
	}
	tablesByName := make(map[string]*schema.Table, len(s.Tables))
	for _, t := range s.Tables {
		tablesByName[t.Name] = t
	}
	return &Site{
		schema:       s,
		files:        files,
		relations:    schema.NewRelationIndex(s),
		tablesByName: tablesByName,
		enums:        newEnumResolver(s.Enums, s.Driver),
	}, nil
}

// hasFunctions and hasEnums decide both whether the optional pages are
// written and whether the menu links to them.
func (site *Site) hasFunctions() bool { return len(site.schema.Functions) > 0 }

func (site *Site) hasEnums() bool { return len(site.schema.Enums) > 0 }

// checkRelationColumns ensures what the diagram relies on to place edges.
func checkRelationColumns(r *schema.Relation) error {
	if len(r.Columns) == 0 || len(r.Columns) != len(r.ParentColumns) {
		return errors.New("columns are empty or do not match parent columns")
	}
	for _, side := range []struct {
		table *schema.Table
		cols  []string
	}{{r.Table, r.Columns}, {r.ParentTable, r.ParentColumns}} {
		for _, name := range side.cols {
			if !slices.ContainsFunc(side.table.Columns, func(c *schema.Column) bool { return c.Name == name }) {
				return fmt.Errorf("column %q not found in table %s", name, side.table.Name)
			}
		}
	}
	return nil
}

// RenderDiagrams lays out the ER diagrams. Call it before preparing the output
// directory so that a Graphviz failure does not leave the output half-written.
func (site *Site) RenderDiagrams(ctx context.Context) error {
	r, err := diagram.NewRenderer(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	s := site.schema
	if len(s.Tables) > 0 {
		href := func(t *schema.Table) string { return site.tableHref("", t) }
		if site.wholeKeys, err = r.SVG(ctx, diagram.DOT(diagram.Whole(s, true), href)); err != nil {
			return fmt.Errorf("render ER diagram: %w", err)
		}
		if site.wholeAll, err = r.SVG(ctx, diagram.DOT(diagram.Whole(s, false), href)); err != nil {
			return fmt.Errorf("render ER diagram: %w", err)
		}
		// Both variants are on the same page; keep their element IDs unique.
		site.wholeKeys = prefixIDs(site.wholeKeys, "keys-")
		site.wholeAll = prefixIDs(site.wholeAll, "all-")
	}
	site.tableDiagrams = make(map[*schema.Table][]byte, len(s.Tables))
	for _, t := range s.Tables {
		if len(site.relations.Parents(t)) == 0 && len(site.relations.Children(t)) == 0 {
			continue
		}
		d := diagram.ForTable(s, site.relations, t, tableDiagramDistance)
		dot := diagram.DOT(d, func(t *schema.Table) string { return site.tableHref("../", t) })
		svg, err := r.SVG(ctx, dot)
		if err != nil {
			return fmt.Errorf("render ER diagram of %s: %w", t.Name, err)
		}
		site.tableDiagrams[t] = svg
	}
	return nil
}

// prefixIDs prefixes the id attributes of a Graphviz SVG. Graphviz numbers
// elements the same way in every graph ("graph0", "node1", ...). Text and other
// attribute values are XML-escaped, so ` id="` only appears as an attribute.
func prefixIDs(svg []byte, prefix string) []byte {
	return bytes.ReplaceAll(svg, []byte(` id="`), []byte(` id="`+prefix))
}

// Write writes all pages and assets into dir, which must already be prepared.
func (site *Site) Write(dir string) error {
	if site.tableDiagrams == nil {
		// Without this, pages would silently lack their diagrams.
		return errors.New("render: RenderDiagrams must be called before Write")
	}
	if err := writeAssets(dir); err != nil {
		return err
	}
	if err := writeFile(filepath.Join(dir, assetsDir, searchIndexFile), site.writeSearchIndex); err != nil {
		return err
	}
	if site.hasCustomCSS {
		if err := writeFile(filepath.Join(dir, assetsDir, customCSSFile), func(w io.Writer) error {
			_, err := w.Write(site.customCSS)
			return err
		}); err != nil {
			return err
		}
	}
	for _, p := range []struct {
		file string
		tmpl *template.Template
		data any
	}{
		{"index.html", tablesTmpl, site.newTablesPage()},
		{"relationships.html", relationshipsTmpl, site.newRelationshipsPage()},
	} {
		if err := writePage(filepath.Join(dir, p.file), p.tmpl, p.data); err != nil {
			return err
		}
	}
	// Optional pages, linked from the menu only when there is something to list.
	if site.hasFunctions() {
		if err := writePage(filepath.Join(dir, "functions.html"), functionsTmpl, site.newFunctionsPage()); err != nil {
			return err
		}
	}
	if site.hasEnums() {
		if err := writePage(filepath.Join(dir, "enums.html"), enumsTmpl, site.newEnumsPage()); err != nil {
			return err
		}
	}
	tables := filepath.Join(dir, tablesDir)
	if err := output.CreateDir(tables); err != nil {
		return err
	}
	for _, t := range site.schema.Tables {
		if err := writePage(filepath.Join(tables, site.files[t]), tableTmpl, site.newTablePage(t)); err != nil {
			return err
		}
	}
	return nil
}

func writeAssets(dir string) error {
	if err := output.CreateDir(filepath.Join(dir, assetsDir)); err != nil {
		return err
	}
	entries, err := fs.ReadDir(content, assetsDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			// Subdirectories are not used yet; skip rather than fail if one is added without writer support.
			continue
		}
		b, err := content.ReadFile(assetsDir + "/" + e.Name())
		if err != nil {
			return err
		}
		if err := writeFile(filepath.Join(dir, assetsDir, e.Name()), func(w io.Writer) error {
			_, err := w.Write(b)
			return err
		}); err != nil {
			return err
		}
	}
	return nil
}

func writePage(path string, tmpl *template.Template, data any) error {
	return writeFile(path, func(w io.Writer) error {
		return tmpl.ExecuteTemplate(w, "layout", data)
	})
}

func writeFile(path string, write func(io.Writer) error) (err error) {
	f, err := output.CreateFile(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("close %s: %w", path, cerr)
		}
	}()
	if err := write(f); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}
