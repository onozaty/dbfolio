// Package render generates the HTML documentation site.
package render

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"

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
	indexTmpl = parsePage("templates/index.html")
	tableTmpl = parsePage("templates/table.html")
)

func parsePage(name string) *template.Template {
	return template.Must(template.ParseFS(content, "templates/layout.html", name))
}

// Site is a planned documentation site. Planning is separated from writing so
// that problems are detected before the output directory is touched.
type Site struct {
	schema *schema.Schema
	// files maps each table to its page file name under tablesDir.
	files map[*schema.Table]string
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
	return &Site{schema: s, files: files}, nil
}

// Write writes all pages and assets into dir, which must already be prepared.
func (site *Site) Write(dir string) error {
	if err := writeAssets(dir); err != nil {
		return err
	}
	if err := writePage(filepath.Join(dir, "index.html"), indexTmpl, newIndexPage(site.schema, site.files)); err != nil {
		return err
	}
	tables := filepath.Join(dir, tablesDir)
	if err := output.CreateDir(tables); err != nil {
		return err
	}
	for _, t := range site.schema.Tables {
		if err := writePage(filepath.Join(tables, site.files[t]), tableTmpl, newTablePage(site.schema, t)); err != nil {
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
