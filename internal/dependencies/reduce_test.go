package dependencies

import (
	"reflect"
	"testing"
)

func TestReduceDropsImpliedEdge(t *testing.T) {
	got := Reduce(map[string][]string{
		"api":       {"go-shared", "nested"},
		"go-shared": {"nested"},
		"nested":    {},
	})
	want := map[string][]string{
		"api":       {"go-shared"},
		"go-shared": {"nested"},
		"nested":    {},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Reduce = %v, want %v", got, want)
	}
}

func TestReduceKeepsIndependentEdges(t *testing.T) {
	graph := map[string][]string{
		"api": {"a", "b"},
		"a":   {},
		"b":   {},
	}
	if got := Reduce(graph); len(got["api"]) != 2 {
		t.Errorf("Reduce dropped an independent edge: %v", got)
	}
}

func TestReduceLeavesCyclicGraphAlone(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}
	got := Reduce(graph)
	if !reflect.DeepEqual(got, graph) {
		t.Errorf("Reduce = %v, want the input unchanged", got)
	}
}
