package schema

import "slices"

// RelationIndex looks up relations by table in both directions, so that pages
// do not scan all relations each time.
type RelationIndex struct {
	parents  map[*Table][]*Relation
	children map[*Table][]*Relation
	// order is the position of each table in Schema.Tables, for stable output.
	order map[*Table]int
}

// NewRelationIndex indexes the relations of s, keeping their order in s.Relations.
func NewRelationIndex(s *Schema) *RelationIndex {
	idx := &RelationIndex{
		parents:  make(map[*Table][]*Relation),
		children: make(map[*Table][]*Relation),
		order:    make(map[*Table]int, len(s.Tables)),
	}
	for i, t := range s.Tables {
		idx.order[t] = i
	}
	for _, r := range s.Relations {
		idx.parents[r.Table] = append(idx.parents[r.Table], r)
		idx.children[r.ParentTable] = append(idx.children[r.ParentTable], r)
	}
	return idx
}

// Parents returns the relations in which t is the child, i.e. t references their ParentTable.
func (idx *RelationIndex) Parents(t *Table) []*Relation {
	return idx.parents[t]
}

// Children returns the relations in which t is the parent, i.e. their Table references t.
func (idx *RelationIndex) Children(t *Table) []*Relation {
	return idx.children[t]
}

// RelatedTables returns t and the tables reachable from it within distance
// relations, in either direction, ordered as in Schema.Tables.
func (idx *RelationIndex) RelatedTables(t *Table, distance int) []*Table {
	seen := map[*Table]bool{t: true}
	frontier := []*Table{t}
	for d := 0; d < distance && len(frontier) > 0; d++ {
		var next []*Table
		visit := func(other *Table) {
			if !seen[other] {
				seen[other] = true
				next = append(next, other)
			}
		}
		for _, cur := range frontier {
			for _, r := range idx.parents[cur] {
				visit(r.ParentTable)
			}
			for _, r := range idx.children[cur] {
				visit(r.Table)
			}
		}
		frontier = next
	}

	tables := make([]*Table, 0, len(seen))
	for other := range seen {
		tables = append(tables, other)
	}
	slices.SortFunc(tables, func(a, b *Table) int { return idx.order[a] - idx.order[b] })
	return tables
}
