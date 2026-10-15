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
