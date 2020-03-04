package transition

import (
	. "github.com/OnlpLab/yap/nlp/types"
	"testing"
)

type StackArrayTest struct {
	stack *StackArray
	t     *testing.T
}

func (s *StackArrayTest) Clear() {
	s.stack.Array = []int{1, 2, 3}
	s.stack.Clear()
	if s.stack.Size() != 0 {
		s.t.Error("After clear got size != 0")
	}
	_, peekExists := s.stack.Peek()
	if peekExists {
		s.t.Error("Stack reported peek exists when it is cleared")
	}
	_, popExists := s.stack.Pop()
	if popExists {
		s.t.Error("Stack reported pop exists when it is cleared")
	}
}

func (s *StackArrayTest) Push() {
	const VAL = 1
	s.stack.Push(VAL)
	if s.stack.Array[len(s.stack.Array)-1] != 1 {
		s.t.Error("Pushed 1, not found at the end of the array")
	}
}

func (s *StackArrayTest) Pop() {
	const VAL = 2
	s.stack.Clear()

	s.stack.Push(VAL)
	popped, exists := s.stack.Pop()
	if !exists {
		s.t.Error("Failed to pop after push")
	}
	if popped != VAL {
		s.t.Error("Pop returned wrong value")
	}
	if s.stack.Size() != 0 {
		s.t.Error("Pop failed to remove value")
	}
}

func (s *StackArrayTest) Index() {
	s.stack.Clear()
	s.stack.Array = []int{4, 8, 10}
	idx0, idx0Exists := s.stack.Index(0)
	if !idx0Exists {
		s.t.Error("Index 0 not found")
	}
	if idx0 != 10 {
		s.t.Error("Got wrong value for index 0")
	}
	idx1, idx1Exists := s.stack.Index(1)
	if !idx1Exists {
		s.t.Error("Index 0 not found")
	}
	if idx1 != 8 {
		s.t.Error("Got wrong value for index 1")
	}
	idx2, idx2Exists := s.stack.Index(2)
	if !idx2Exists {
		s.t.Error("Index 0 not found")
	}
	if idx2 != 4 {
		s.t.Error("Got wrong value for index 2")
	}
	_, idx3Exists := s.stack.Index(3)
	if idx3Exists {
		s.t.Error("Index found for non existent index 3")
	}
}

func (s *StackArrayTest) Peek() {
	const VAL = 3
	s.stack.Clear()

	s.stack.Push(VAL)
	peeked, exists := s.stack.Peek()
	if !exists {
		s.t.Error("Failed to peek after push")
	}
	if peeked != VAL {
		s.t.Error("Peek returned wrong value")
	}
	if s.stack.Array[len(s.stack.Array)-1] != VAL {
		s.t.Error("Pushed 3, not found at the end of the array after peek")
	}
}

func (s *StackArrayTest) Size() {
	s.stack.Clear()
	if s.stack.Size() != 0 {
		s.t.Error("Cleared stack reported size != 0")
	}
	const VAL = 3
	s.stack.Push(VAL)
	if s.stack.Size() != 1 {
		s.t.Error("Stack after push reported size != 1")
	}
}

func (s *StackArrayTest) Copy() {
	s.stack.Clear()
	s.stack.Push(1)
	s.stack.Push(4)
	s.stack.Push(3)
	s.stack.Push(2)
	newStack := s.stack.Copy().(*StackArray)
	if len(newStack.Array) != len(s.stack.Array) {
		s.t.Error("Stack copy failed to produce copy of same length")
	}
	for i, val := range newStack.Array {
		if s.stack.Array[i] != val {
			s.t.Error("Stack copy failed to produce copy - differing values")
		}
	}
	newStack.Array[2] = 5
	if newStack.Array[2] == s.stack.Array[2] {
		s.t.Error("Copy was shallow, changing a value in copied stack should not affect original")
	}
}

func (test *StackArrayTest) All() {
	test.Clear()
	test.Push()
	test.Pop()
	test.Index()
	test.Peek()
	test.Size()
	test.Copy()
}

func TestStackArray(t *testing.T) {
	const CAPACITY = 5
	stack := NewStackArray(CAPACITY)
	if cap(stack.Array) != CAPACITY {
		t.Error("NewStackArray has wrong capacity")
	}

	test := StackArrayTest{stack, t}
	test.All()
}

type ArcSetSimpleTest struct {
	set *ArcSetSimple
	t   *testing.T
}

func (a *ArcSetSimpleTest) Clear() {
	a.set.Arcs = []LabeledDepArc{&BasicDepArc{}, &BasicDepArc{}}
	a.set.Clear()
	if a.set.Size() != 0 {
		a.t.Error("After clear got size != 0")
	}
	arc0 := a.set.Index(0)
	if arc0 != nil {
		a.t.Error("Index of 0 returned not nil after clear")
	}
	lastArc := a.set.Last()
	if lastArc != nil {
		a.t.Error("Last returned not nil after clear")
	}
	arcs := a.set.Get(&BasicDepArc{-1, -1, -1, ""})
	if len(arcs) != 0 {
		a.t.Error("Got non-empty slice of arcs for * query after clear")
	}
}

func (a *ArcSetSimpleTest) Add() {
	a.set.Clear()
	arc := &BasicDepArc{2, 1, 1, "a"}
	a.set.Add(arc)
	if a.set.Size() != 1 {
		a.t.Error("After clear and add, size is not 1")
	}
	if a.set.Arcs[0] != arc {
		a.t.Error("Pointer in set is not the added pointer")
	}
}

func (a *ArcSetSimpleTest) Index() {
	a.set.Clear()
	arc := a.set.Index(0)
	if arc != nil {
		a.t.Error("Got non-nil result for index 0 of cleared set")
	}
	arcs := []*BasicDepArc{&BasicDepArc{1, 1, 1, "a"}, &BasicDepArc{2, 2, 2, "b"}}
	a.set.Add(arcs[0])
	a.set.Add(arcs[1])
	if a.set.Index(0) != arcs[0] {
		a.t.Error("Couldn't find first added arc")
	}
	if a.set.Index(1) != arcs[1] {
		a.t.Error("Couldn't find first added arc")
	}
	if a.set.Index(2) != nil {
		a.t.Error("Got non-nil result for index 2 when only 2 arcs were added")
	}
}

func (a *ArcSetSimpleTest) Get() {
	a.set.Clear()
	a.set.Arcs = []LabeledDepArc{
		&BasicDepArc{1, 1, 2, "a"},
		&BasicDepArc{1, 2, 3, "b"},
		&BasicDepArc{2, 3, 4, "c"},
		&BasicDepArc{3, 1, 5, "a"},
	}
	// get all
	allArcs := a.set.Get(&BasicDepArc{-1, -1, -1, ""})
	if len(allArcs) != a.set.Size() {
		a.t.Error("Get all failed, retrieved less arcs than in the set")
	}
	// get arc that doesn't exist
	noArcs := a.set.Get(&BasicDepArc{1, 1, 8, "a"})
	if len(noArcs) != 0 {
		a.t.Error("Found an arc that doesn't exist")
	}
	// get modifiers
	modArcs := a.set.Get(&BasicDepArc{1, -1, -1, ""})
	if len(modArcs) != 2 {
		a.t.Error("Got wrong number of modifiers for head 1")
	}
	if len(modArcs) > 0 && modArcs[0] != a.set.Arcs[0] {
		a.t.Error("Got wrong first modifier arc for head 1")
	}
	if len(modArcs) > 1 && modArcs[1] != a.set.Arcs[1] {
		a.t.Error("Got wrong second modifier arc for head 1")
	}
	// get specific modifier by relation
	relModArcs := a.set.Get(&BasicDepArc{1, 1, -1, "a"})
	if len(relModArcs) != 1 {
		a.t.Error("Got wrong number of modifiers of type 'a' for head 1")
	}
	if len(relModArcs) > 0 && relModArcs[0] != a.set.Arcs[0] {
		a.t.Error("Got wrong arc")
	}
	// get head by modifier
	headArcs := a.set.Get(&BasicDepArc{-1, -1, 2, ""})
	if len(headArcs) != 1 {
		a.t.Error("Got wrong number of head arcs")
	}
	if len(headArcs) > 0 && headArcs[0] != a.set.Arcs[0] {
		a.t.Error("Got wrong head arc")
	}
	// get arcs by relation
	relArcs := a.set.Get(&BasicDepArc{-1, 1, -1, "a"})
	if len(relArcs) != 2 {
		a.t.Error("Got wrong number of arcs by relation")
	}
	if len(relArcs) > 0 && relArcs[0] != a.set.Arcs[0] {
		a.t.Error("Got wrong first arc")
	}
	if len(relArcs) > 1 && relArcs[1] != a.set.Arcs[3] {
		a.t.Error("Got wrong second arc")
	}
}

func (a *ArcSetSimpleTest) Size() {
	a.set.Clear()
	if a.set.Size() != 0 {
		a.t.Error("Got non-zero size for cleared set")
	}
	arcSet := []LabeledDepArc{&BasicDepArc{1, 1, 1, "a"}, &BasicDepArc{2, 2, 2, "b"}}
	a.set.Arcs = arcSet
	if a.set.Size() != len(arcSet) {
		a.t.Error("Got incorrect size for injected set")
	}
}

func (a *ArcSetSimpleTest) Last() {
	a.set.Clear()
	result := a.set.Last()
	if result != nil {
		a.t.Error("Got non-nil last arc for empty set")
	}
	arc := &BasicDepArc{2, 1, 1, "a"}
	a.set.Add(&BasicDepArc{3, 2, 4, "b"})
	a.set.Add(&BasicDepArc{4, 3, 5, "c"})
	a.set.Add(arc)
	if a.set.Last() != arc {
		a.t.Error("Got wrong last arc")
	}
}

func (a *ArcSetSimpleTest) Copy() {
	a.set.Clear()
	arcSet := []LabeledDepArc{&BasicDepArc{1, 1, 1, "a"}, &BasicDepArc{2, 2, 2, "b"}}
	a.set.Arcs = arcSet
	newSet := a.set.Copy()
	if newSet.Size() != a.set.Size() {
		a.t.Error("Copied set has non-matching size")
	}
	for i, val := range a.set.Arcs {
		if newSet.Index(i) != val {
			a.t.Error("Found non-matching set element in copy")
		}
	}
	newSet.Add(&BasicDepArc{0, -1, 0, ""})
	if !(newSet.Size() == 3 && a.set.Size() == 2) {
		a.t.Error("Copy is shallow")
	}
}

func (a *ArcSetSimpleTest) Equal() {
	a.set.Clear()
	arcSet := []LabeledDepArc{&BasicDepArc{1, 1, 1, "a"}, &BasicDepArc{2, 2, 2, "b"}}
	a.set.Arcs = arcSet
	otherSet := a.set.Copy().(*ArcSetSimple)
	if !otherSet.Equal(a.set) {
		a.t.Error("Unequal sets using same ordering")
	}
	otherSet.Swap(0, 1)
	if !otherSet.Equal(a.set) {
		a.t.Error("Unequal sets using different ordering")
	}
}

func (a *ArcSetSimpleTest) String() {
	if len(a.set.String()) == 0 {
		a.t.Error("Got empty String representation")
	}
}

func (test *ArcSetSimpleTest) All() {
	test.Clear()
	test.Index()
	test.Add()
	test.Get()
	test.Size()
	test.Last()
	test.Copy()
	test.Equal()
	test.String()
}

func TestArcSetSimple(t *testing.T) {
	const CAPACITY = 5
	arcSet := NewArcSetSimple(CAPACITY)
	if cap(arcSet.Arcs) != CAPACITY {
		t.Error("NewArcSetSimple has wrong capacity")
	}
	test := ArcSetSimpleTest{arcSet, t}
	test.All()
}
