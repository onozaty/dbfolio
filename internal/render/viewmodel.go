package render

import "github.com/onozaty/dbfolio/internal/schema"

// page is the data common to all pages.
type page struct {
	// Root is the relative path from the page to the output root ("" or "../"),
	// so that links work when opened via file://.
	Root   string
	Title  string
	DBName string
}

type indexPage struct {
	page
	Desc   string
	Stats  stats
	Tables []tableRow
}

type stats struct {
	Tables    int
	Views     int
	Columns   int
	Relations int
	Indexes   int
}

type tableRow struct {
	Name    string
	Href    string
	Type    string
	View    bool
	Columns int
	Comment string
}

type tablePage struct {
	page
	Table       *schema.Table
	Columns     []columnRow
	ShowExtra   bool
	Constraints []*schema.Constraint
}

type columnRow struct {
	*schema.Column
	// Keys lists "PK" and/or "FK". A column in several (or composite) foreign
	// keys still shows a single "FK"; the details are in the constraints section.
	Keys []string
}

func newIndexPage(s *schema.Schema, files map[*schema.Table]string) indexPage {
	p := indexPage{
		page: page{Title: dbTitle(s), DBName: dbTitle(s)},
		Desc: s.Desc,
	}
	p.Stats.Relations = len(s.Relations)
	for _, t := range s.Tables {
		if t.View {
			p.Stats.Views++
		} else {
			p.Stats.Tables++
		}
		p.Stats.Columns += len(t.Columns)
		p.Stats.Indexes += len(t.Indexes)
		p.Tables = append(p.Tables, tableRow{
			Name:    t.Name,
			Href:    tablesDir + "/" + files[t],
			Type:    t.Type,
			View:    t.View,
			Columns: len(t.Columns),
			Comment: t.Comment,
		})
	}
	return p
}

func newTablePage(s *schema.Schema, t *schema.Table) tablePage {
	p := tablePage{
		page:        page{Root: "../", Title: t.Name + " - " + dbTitle(s), DBName: dbTitle(s)},
		Table:       t,
		Constraints: t.Constraints,
	}
	pk := make(map[string]bool)
	fk := make(map[string]bool)
	for _, c := range t.Constraints {
		switch c.Kind {
		case schema.ConstraintPrimaryKey:
			for _, col := range c.Columns {
				pk[col] = true
			}
		case schema.ConstraintForeignKey:
			for _, col := range c.Columns {
				fk[col] = true
			}
		}
	}
	for _, c := range t.Columns {
		row := columnRow{Column: c}
		if pk[c.Name] {
			row.Keys = append(row.Keys, "PK")
		}
		if fk[c.Name] {
			row.Keys = append(row.Keys, "FK")
		}
		if c.ExtraDef != "" {
			p.ShowExtra = true
		}
		p.Columns = append(p.Columns, row)
	}
	return p
}

func dbTitle(s *schema.Schema) string {
	if s.Name != "" {
		return s.Name
	}
	return "Database"
}
