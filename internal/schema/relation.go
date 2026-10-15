package schema

// RelationIndex looks up relations by table in both directions, so that pages
// do not scan all relations each time.
type RelationIndex struct {
	parents  map[*Table][]*Relation
	children map[*Table][]*Relation
}

// NewRelationIndex indexes the relations of s, keeping their order in s.Relations.
func NewRelationIndex(s *Schema) *RelationIndex {
	idx := &RelationIndex{
		parents:  make(map[*Table][]*Relation),
		children: make(map[*Table][]*Relation),
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
