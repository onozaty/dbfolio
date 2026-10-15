package schema

import (
	"reflect"
	"testing"
)

func TestRelationIndex_IndexesBothDirections(t *testing.T) {
	// Arrange
	users := &Table{Name: "users"}
	orders := &Table{Name: "orders"}
	comments := &Table{Name: "comments"}
	toUsers := &Relation{Table: orders, ParentTable: users}
	commentToUser := &Relation{Table: comments, ParentTable: users, Virtual: true}
	s := &Schema{Tables: []*Table{users, orders, comments}, Relations: []*Relation{toUsers, commentToUser}}

	// Act
	idx := NewRelationIndex(s)

	// Assert
	if got := idx.Children(users); !reflect.DeepEqual(got, []*Relation{toUsers, commentToUser}) {
		t.Errorf("Children(users) = %v, want both relations in input order", got)
	}
	if got := idx.Parents(orders); !reflect.DeepEqual(got, []*Relation{toUsers}) {
		t.Errorf("Parents(orders) = %v, want [toUsers]", got)
	}
	if got := idx.Parents(users); got != nil {
		t.Errorf("Parents(users) = %v, want none", got)
	}
}

func TestRelationIndex_ListsSeveralRelationsBetweenSameTables(t *testing.T) {
	// Arrange: shipments references orders directly and through a composite key of order_items.
	orders := &Table{Name: "orders"}
	items := &Table{Name: "order_items"}
	shipments := &Table{Name: "shipments"}
	toOrder := &Relation{Table: shipments, Columns: []string{"order_id"}, ParentTable: orders}
	toItem := &Relation{Table: shipments, Columns: []string{"order_id", "line_no"}, ParentTable: items}
	s := &Schema{Tables: []*Table{orders, items, shipments}, Relations: []*Relation{toOrder, toItem}}

	// Act
	idx := NewRelationIndex(s)

	// Assert
	if got := idx.Parents(shipments); !reflect.DeepEqual(got, []*Relation{toOrder, toItem}) {
		t.Errorf("Parents(shipments) = %v, want [toOrder toItem]", got)
	}
}

func TestRelationIndex_ListsSelfReferenceAsParentAndChild(t *testing.T) {
	// Arrange
	comments := &Table{Name: "comments"}
	self := &Relation{Table: comments, Columns: []string{"parent_id"}, ParentTable: comments, ParentColumns: []string{"id"}}
	s := &Schema{Tables: []*Table{comments}, Relations: []*Relation{self}}

	// Act
	idx := NewRelationIndex(s)

	// Assert
	if got := idx.Parents(comments); !reflect.DeepEqual(got, []*Relation{self}) {
		t.Errorf("Parents(comments) = %v, want [self]", got)
	}
	if got := idx.Children(comments); !reflect.DeepEqual(got, []*Relation{self}) {
		t.Errorf("Children(comments) = %v, want [self]", got)
	}
}

func TestRelatedTables_ReturnsTablesWithinDistanceInSchemaOrder(t *testing.T) {
	// Arrange: regions <- users <- orders <- items, and users <- comments
	regions := &Table{Name: "regions"}
	users := &Table{Name: "users"}
	orders := &Table{Name: "orders"}
	items := &Table{Name: "items"}
	comments := &Table{Name: "comments"}
	isolated := &Table{Name: "isolated"}
	s := &Schema{
		Tables: []*Table{regions, users, orders, items, comments, isolated},
		Relations: []*Relation{
			{Table: users, ParentTable: regions},
			{Table: orders, ParentTable: users},
			{Table: items, ParentTable: orders},
			{Table: comments, ParentTable: users},
		},
	}
	idx := NewRelationIndex(s)
	tests := []struct {
		name     string
		distance int
		want     []*Table
	}{
		{"distance 0", 0, []*Table{orders}},
		{"distance 1", 1, []*Table{users, orders, items}},
		{"distance 2", 2, []*Table{regions, users, orders, items, comments}},
		{"beyond reach", 10, []*Table{regions, users, orders, items, comments}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			got := idx.RelatedTables(orders, tt.distance)

			// Assert
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RelatedTables(orders, %d) = %v, want %v", tt.distance, names(got), names(tt.want))
			}
		})
	}
}

func TestRelatedTables_HandlesSelfReference(t *testing.T) {
	// Arrange
	comments := &Table{Name: "comments"}
	s := &Schema{Tables: []*Table{comments}, Relations: []*Relation{{Table: comments, ParentTable: comments}}}
	idx := NewRelationIndex(s)

	// Act
	got := idx.RelatedTables(comments, 1)

	// Assert
	if !reflect.DeepEqual(got, []*Table{comments}) {
		t.Errorf("RelatedTables = %v, want [comments]", names(got))
	}
}

func names(tables []*Table) []string {
	var out []string
	for _, t := range tables {
		out = append(out, t.Name)
	}
	return out
}
