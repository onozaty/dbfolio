package render

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// searchIndexFile is written under assetsDir.
const searchIndexFile = "search-index.js"

// searchEntry is a table or a column in the search index. Field names are
// short because the index grows with the schema.
type searchEntry struct {
	// Kind is "t" for tables and "c" for columns.
	Kind string `json:"k"`
	// Name is the table name, or the column name.
	Name string `json:"n"`
	// Table is the table of a column.
	Table string `json:"t,omitempty"`
	// Type is the kind label of a table (e.g. "VIEW") or the type of a column.
	Type    string `json:"y,omitempty"`
	Comment string `json:"c,omitempty"`
	// Href is relative to the output root; pages in subdirectories prefix it.
	Href string `json:"h"`
}

// columnAnchor is the id of a column row on a table page. Column names can
// contain any character, so the position is used.
func columnAnchor(i int) string {
	return fmt.Sprintf("column-%d", i)
}

func (site *Site) searchEntries() []searchEntry {
	var entries []searchEntry
	for _, t := range site.schema.Tables {
		href := site.tableHref("", t)
		entries = append(entries, searchEntry{Kind: "t", Name: t.Name, Type: t.KindLabel(), Comment: t.Comment, Href: href})
		for i, c := range t.Columns {
			entries = append(entries, searchEntry{
				Kind:    "c",
				Name:    c.Name,
				Table:   t.Name,
				Type:    c.Type,
				Comment: c.Comment,
				Href:    href + "#" + columnAnchor(i),
			})
		}
	}
	return entries
}

// writeSearchIndex writes the index as a script that assigns a global, not as
// JSON: pages opened via file:// cannot fetch() local files.
func (site *Site) writeSearchIndex(w io.Writer) error {
	// json.Marshal escapes <, > and & (so "</script>" in a comment is harmless)
	// as well as U+2028 / U+2029, which are line terminators in older JavaScript.
	b, err := json.Marshal(site.searchEntries())
	if err != nil {
		return err
	}
	if bytes.Equal(b, []byte("null")) {
		b = []byte("[]")
	}
	_, err = fmt.Fprintf(w, "window.DBFOLIO_SEARCH_INDEX = %s;\n", b)
	return err
}
