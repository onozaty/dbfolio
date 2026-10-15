// Package schema defines dbfolio's internal database model, independent of the input format.
package schema

// Schema is a whole database.
type Schema struct {
	Name      string
	Desc      string
	Tables    []*Table
	Relations []*Relation
}

// Table is a table or a view.
type Table struct {
	// Name is the qualified name as given by the input (e.g. "public.users").
	Name string
	// Type is the table type as given by the input (e.g. "BASE TABLE", "VIEW").
	// Its values differ between databases, so use View to tell views apart.
	Type string
	// View is true for views and materialized views.
	View    bool
	Comment string
	Columns []*Column
	Indexes []*Index
	// Constraints holds keys and checks. Constraints that duplicate column
	// attributes (e.g. NOT NULL) are not included.
	Constraints []*Constraint
	Triggers    []*Trigger
	// Def is the definition SQL when the input provides it (the query of a view,
	// or CREATE TABLE on some databases such as MySQL).
	Def string
}

// Column is a table column.
//
// Key information (PK / FK) is not held here: a column can belong to several
// constraints, some of them composite, so it is derived from Table.Constraints.
type Column struct {
	Name     string
	Type     string
	Nullable bool
	// Default is nil when the column has no default (distinct from an empty string default).
	Default  *string
	ExtraDef string
	Comment  string
}

// Index is a table index.
type Index struct {
	Name    string
	Def     string
	Columns []string
	Comment string
}

// ConstraintKind classifies a constraint.
type ConstraintKind string

const (
	ConstraintPrimaryKey ConstraintKind = "PRIMARY KEY"
	ConstraintForeignKey ConstraintKind = "FOREIGN KEY"
	ConstraintUnique     ConstraintKind = "UNIQUE"
	ConstraintCheck      ConstraintKind = "CHECK"
	// ConstraintOther covers DB-specific kinds (e.g. EXCLUDE); Type keeps the original name.
	ConstraintOther ConstraintKind = "OTHER"
)

// Constraint is a table constraint. Composite keys have several Columns.
type Constraint struct {
	Name string
	Kind ConstraintKind
	// Type is the constraint type as given by the input, for display of ConstraintOther.
	Type    string
	Def     string
	Columns []string
	// ReferencedTable and ReferencedColumns are set for foreign keys only.
	// ReferencedColumns[i] is referenced by Columns[i].
	ReferencedTable   string
	ReferencedColumns []string
	Comment           string
}

// Trigger is a table trigger.
type Trigger struct {
	Name    string
	Def     string
	Comment string
}

// Relation is a parent-child relation between tables, either from a foreign
// key or defined virtually (e.g. in .tbls.yml) without a DB constraint.
//
// Table and ParentTable always point to tables of the same Schema, and every
// column exists in its table; the input layer rejects relations that do not.
// They can be the same table (self reference).
type Relation struct {
	Table *Table
	// Columns[i] references ParentColumns[i]; composite relations have several.
	Columns           []string
	Cardinality       Cardinality
	ParentTable       *Table
	ParentColumns     []string
	ParentCardinality Cardinality
	Def               string
	// Virtual is true for relations without a DB constraint.
	Virtual bool
}

// Cardinality is the number of rows on one side of a relation, using tbls'
// values ("zero_or_one", "exactly_one", "zero_or_more", "one_or_more").
// Other values are kept as given.
type Cardinality string
