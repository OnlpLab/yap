package transition

import (
	. "github.com/OnlpLab/yap/nlp/types"
	"fmt"
	// "log"
	"reflect"
	"sort"
	"strings"
)

type ArcSetSimple struct {
	Arcs         []LabeledDepArc
	SeenHead     map[int]bool
	SeenModifier map[int]bool
	SeenArc      map[[2]int]bool
}

var _ ArcSet = &ArcSetSimple{}
var _ sort.Interface = &ArcSetSimple{}

func (s *ArcSetSimple) Less(i, j int) bool {
	if s.Arcs[i].GetHead() < s.Arcs[j].GetHead() {
		return true
	}
	if s.Arcs[i].GetHead() == s.Arcs[j].GetHead() {
		return s.Arcs[i].GetModifier() < s.Arcs[j].GetModifier()
	}
	return false
}

func (s *ArcSetSimple) Swap(i, j int) {
	s.Arcs[i], s.Arcs[j] = s.Arcs[j], s.Arcs[i]
}

func (s *ArcSetSimple) Len() int {
	return s.Size()
}

func (s *ArcSetSimple) ValueComp(i, j int, other *ArcSetSimple) int {
	left := s.Arcs[i]
	right := other.Arcs[j]
	if reflect.DeepEqual(left, right) {
		return 0
	}
	if left.GetModifier() < right.GetModifier() {
		return 1
	}
	return -1
}

func (s *ArcSetSimple) Equal(other ArcSet) bool {
	if s.Size() == 0 && other.Size() == 0 {
		return true
	}
	copyThis := s.Copy().(*ArcSetSimple)
	copyOther := other.Copy().(*ArcSetSimple)
	if copyThis.Len() != copyOther.Len() {
		return false
	}
	sort.Sort(copyThis)
	sort.Sort(copyOther)
	for i, _ := range copyThis.Arcs {
		if !copyThis.Arcs[i].Equal(copyOther.Arcs[i]) {
			return false
		}
	}
	return true
}

func (s *ArcSetSimple) Sorted() *ArcSetSimple {
	copyThis := s.Copy().(*ArcSetSimple)
	sort.Sort(copyThis)
	return copyThis
}

func (s *ArcSetSimple) Diff(other ArcSet) (ArcSet, ArcSet) {
	copyThis := s.Copy().(*ArcSetSimple)
	copyOther := other.Copy().(*ArcSetSimple)
	sort.Sort(copyThis)
	sort.Sort(copyOther)

	leftOnly := NewArcSetSimple(copyThis.Len())
	rightOnly := NewArcSetSimple(copyOther.Len())
	i, j := 0, 0
	for i < copyThis.Len() && j < copyOther.Len() {
		comp := copyThis.ValueComp(i, j, copyOther)
		switch {
		case comp == 0:
			i++
			j++
		case comp < 0:
			leftOnly.Add(copyThis.Arcs[i])
			i++
		case comp > 0:
			rightOnly.Add(copyOther.Arcs[j])
			j++
		}
	}
	return leftOnly, rightOnly
}

func (s *ArcSetSimple) Copy() ArcSet {
	newArcs := make([]LabeledDepArc, len(s.Arcs), cap(s.Arcs))
	// headMap, modMap, arcMap := make(map[int]bool, cap(s.Arcs)), make(map[int]bool, cap(s.Arcs)), make(map[[2]int]bool, cap(s.Arcs))
	// for k, v := range s.SeenArc {
	// 	arcMap[k] = v
	// }
	// for k, v := range s.SeenHead {
	// 	headMap[k] = v
	// }
	// for k, v := range s.SeenModifier {
	// 	modMap[k] = v
	// }
	copy(newArcs, s.Arcs)
	return ArcSet(&ArcSetSimple{Arcs: newArcs})
	// return ArcSet(&ArcSetSimple{newArcs, headMap, modMap, arcMap})
}

func (s *ArcSetSimple) Clear() {
	s.Arcs = s.Arcs[0:0]
}

func (s *ArcSetSimple) Index(i int) LabeledDepArc {
	if i >= len(s.Arcs) {
		return nil
	}
	return s.Arcs[i]
}

func (s *ArcSetSimple) Add(arc LabeledDepArc) {
	// s.SeenHead[arc.GetHead()] = true
	// s.SeenModifier[arc.GetModifier()] = true
	// s.SeenArc[[2]int{arc.GetHead(), arc.GetModifier()}] = true
	s.Arcs = append(s.Arcs, arc)
}

func (s *ArcSetSimple) Get(query LabeledDepArc) []LabeledDepArc {
	var results []LabeledDepArc
	head := query.GetHead()
	modifier := query.GetModifier()
	relation := query.GetRelation()
	for _, arc := range s.Arcs {
		if head >= 0 && head != arc.GetHead() {
			continue
		}
		if modifier >= 0 && modifier != arc.GetModifier() {
			continue
		}
		if string(relation) != "" && relation != arc.GetRelation() {
			continue
		}
		results = append(results, arc)
	}
	return results
}

func (s *ArcSetSimple) Size() int {
	return len(s.Arcs)
}

func (s *ArcSetSimple) Last() LabeledDepArc {
	if s.Size() == 0 {
		return nil
	}
	return s.Arcs[len(s.Arcs)-1]
}

func (s *ArcSetSimple) String() string {
	arcs := make([]string, s.Size())
	for i, arc := range s.Arcs {
		arcs[i] = fmt.Sprintf("%d %d %v", i, arc.ID, arc.String())
	}
	return strings.Join(arcs, "\n")
}

func (s *ArcSetSimple) HasHead(modifier int) bool {
	// _, exists := s.SeenModifier[modifier]
	// return exists
	return len(s.Get(&BasicDepArc{-1, -1, modifier, DepRel("")})) > 0
}

func (s *ArcSetSimple) HasModifiers(head int) bool {
	// _, exists := s.SeenHead[head]
	// return exists
	return len(s.Get(&BasicDepArc{head, -1, -1, DepRel("")})) > 0
}

func (s *ArcSetSimple) HasArc(head, modifier int) bool {
	_, exists := s.SeenArc[[2]int{head, modifier}]
	return exists
}

func NewArcSetSimple(size int) *ArcSetSimple {
	return &ArcSetSimple{
		Arcs: make([]LabeledDepArc, 0, size),
		// SeenHead:     make(map[int]bool, size),
		// SeenModifier: make(map[int]bool, size),
		// SeenArc:      make(map[[2]int]bool, size),
	}
}

func NewArcSetSimpleFromGraph(graph LabeledDependencyGraph) *ArcSetSimple {
	arcSet := NewArcSetSimple(graph.NumberOfEdges())
	// log.Println("Generating new arc set for graph")
	// log.Println(graph)
	for _, edgeNum := range graph.GetEdges() {
		// log.Println("At edge", i)
		arc := graph.GetLabeledArc(edgeNum)
		arcSet.Add(arc)
	}
	return arcSet
}
