package render

import (
	"strings"

	"github.com/onozaty/dbfolio/internal/schema"
)

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
	Parents     relationTable
	Children    relationTable
}

type columnRow struct {
	*schema.Column
	Keys []keyBadge
}

// keyBadge is a PK or FK mark of a column. A column has one FK badge per
// relation it belongs to (it can be in several, some of them composite), each
// linking to that relation's parent table.
type keyBadge struct {
	Label   string
	Href    string
	Title   string
	Virtual bool
}

type relationsPage struct {
	page
	Relations relationTable
}

// relationTable is a list of relations. On a table page one end is the table
// itself, so only the other end is shown.
type relationTable struct {
	Rows       []relationRow
	ShowChild  bool
	ShowParent bool
}

// relationRow is a relation from the child table to the parent table.
type relationRow struct {
	Child       string
	ChildHref   string
	Parent      string
	ParentHref  string
	Columns     string
	Cardinality string
	Virtual     bool
	Def         string
}

func (site *Site) newIndexPage() indexPage {
	s := site.schema
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
			Href:    site.tableHref(p.Root, t),
			Type:    t.Type,
			View:    t.View,
			Columns: len(t.Columns),
			Comment: t.Comment,
		})
	}
	return p
}

func (site *Site) newTablePage(t *schema.Table) tablePage {
	s := site.schema
	p := tablePage{
		page:        page{Root: "../", Title: t.Name + " - " + dbTitle(s), DBName: dbTitle(s)},
		Table:       t,
		Constraints: t.Constraints,
	}
	parents := site.relations.Parents(t)

	pk := make(map[string]bool)
	for _, c := range t.Constraints {
		if c.Kind == schema.ConstraintPrimaryKey {
			for _, col := range c.Columns {
				pk[col] = true
			}
		}
	}
	for _, c := range t.Columns {
		row := columnRow{Column: c}
		if pk[c.Name] {
			row.Keys = append(row.Keys, keyBadge{Label: "PK", Title: "Primary key"})
		}
		// FK marks come from relations rather than FK constraints so that
		// virtual relations are shown too.
		for _, r := range parents {
			if contains(r.Columns, c.Name) {
				row.Keys = append(row.Keys, site.fkBadge(p.Root, r))
			}
		}
		if c.ExtraDef != "" {
			p.ShowExtra = true
		}
		p.Columns = append(p.Columns, row)
	}

	p.Parents = relationTable{ShowParent: true}
	for _, r := range parents {
		p.Parents.Rows = append(p.Parents.Rows, site.relationRow(p.Root, r))
	}
	p.Children = relationTable{ShowChild: true}
	for _, r := range site.relations.Children(t) {
		p.Children.Rows = append(p.Children.Rows, site.relationRow(p.Root, r))
	}
	return p
}

func (site *Site) newRelationsPage() relationsPage {
	s := site.schema
	p := relationsPage{
		page:      page{Title: "Relationships - " + dbTitle(s), DBName: dbTitle(s)},
		Relations: relationTable{ShowChild: true, ShowParent: true},
	}
	for _, r := range s.Relations {
		p.Relations.Rows = append(p.Relations.Rows, site.relationRow(p.Root, r))
	}
	return p
}

func (site *Site) tableHref(root string, t *schema.Table) string {
	return root + tablesDir + "/" + site.files[t]
}

func (site *Site) fkBadge(root string, r *schema.Relation) keyBadge {
	title := "References " + r.ParentTable.Name + " " + columnList(r.ParentColumns)
	if r.Virtual {
		title = "Virtual relation: " + title
	}
	return keyBadge{
		Label:   "FK",
		Href:    site.tableHref(root, r.ParentTable),
		Title:   title,
		Virtual: r.Virtual,
	}
}

func (site *Site) relationRow(root string, r *schema.Relation) relationRow {
	return relationRow{
		Child:       r.Table.Name,
		ChildHref:   site.tableHref(root, r.Table),
		Parent:      r.ParentTable.Name,
		ParentHref:  site.tableHref(root, r.ParentTable),
		Columns:     columnList(r.Columns) + " → " + columnList(r.ParentColumns),
		Cardinality: cardinalityPair(r),
		Virtual:     r.Virtual,
		Def:         r.Def,
	}
}

// columnList formats relation columns; composite ones are parenthesized so
// that "(a, b) → (x, y)" reads as a pairwise mapping.
func columnList(cols []string) string {
	if len(cols) == 1 {
		return cols[0]
	}
	return "(" + strings.Join(cols, ", ") + ")"
}

// cardinalityPair formats "child : parent", e.g. "0..* : 1" for many children per parent.
func cardinalityPair(r *schema.Relation) string {
	child, parent := cardinality(r.Cardinality), cardinality(r.ParentCardinality)
	if child == "" && parent == "" {
		return ""
	}
	return orUnknown(child) + " : " + orUnknown(parent)
}

func cardinality(c schema.Cardinality) string {
	switch c {
	case "zero_or_one":
		return "0..1"
	case "exactly_one":
		return "1"
	case "zero_or_more":
		return "0..*"
	case "one_or_more":
		return "1..*"
	default:
		return string(c)
	}
}

func orUnknown(s string) string {
	if s == "" {
		return "?"
	}
	return s
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func dbTitle(s *schema.Schema) string {
	if s.Name != "" {
		return s.Name
	}
	return "Database"
}
