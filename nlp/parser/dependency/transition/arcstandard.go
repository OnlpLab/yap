package transition

import (
	"fmt"
	. "github.com/OnlpLab/yap/alg/transition"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
	// "log"
)

type ArcStandard struct {
	oracle             Oracle
	Relations          *util.EnumSet
	Transitions        *util.EnumSet
	SHIFT, LEFT, RIGHT int
}

// Verify that ArcStandard is a TransitionSystem
var _ TransitionSystem = &ArcStandard{}

func (a *ArcStandard) Transition(from Configuration, rawTransition Transition) Configuration {
	conf, ok := from.Copy().(*SimpleConfiguration)
	if !ok {
		panic("Got wrong configuration type")
	}
	transition := rawTransition.Value()
	// Transition System:
	// LA-r	(S|wi,	wj|B,	A) => (S   ,	wj|B,	A+{(wj,r,wi)})	if: i != 0
	// RA-r	(S|wi, 	wj|B,	A) => (S   ,	wi|B, 	A+{(wi,r,wj)})
	// SH	(S   ,	wi|B, 	A) => (S|wi,	   B,	A)
	// switch transition[:2] {
	switch {
	// case "LA":
	case transition >= a.LEFT && transition < a.RIGHT:
		wi, wiExists := conf.Stack().Pop()
		// if wi == 0 {
		// 	panic("Attempted to LA the root")
		// }
		wj, wjExists := conf.Queue().Peek()
		if !(wiExists && wjExists) {
			panic(fmt.Sprintf("Can't LA, Stack and/or Queue are/is empty: %v", conf))
		}
		// relation := DepRel(transition[3:])
		relation := int(transition - a.LEFT)
		relationValue := a.Relations.ValueOf(relation).(DepRel)
		newArc := &BasicDepArc{wj, relation, wi, relationValue}

		// conf.Arcs().Add(newArc)
		conf.AddArc(newArc)
		conf.Assign(uint16(conf.Nodes[newArc.Modifier].ID()))
		// we remove a previously head-less element from the stack
		conf.NumHeadStack--
	// case "RA":
	case transition >= a.RIGHT:
		wi, wiExists := conf.Stack().Pop()
		wj, wjExists := conf.Queue().Pop()
		if !(wiExists && wjExists) {
			panic("Can't RA, Stack and/or Queue are/is empty")
		}
		// rel := DepRel(transition[3:])
		rel := int(transition - a.RIGHT)
		relValue := a.Relations.ValueOf(rel).(DepRel)
		newArc := &BasicDepArc{wi, rel, wj, relValue}
		conf.Queue().Push(wi)
		conf.AddArc(newArc)
		conf.Assign(uint16(conf.Nodes[newArc.Modifier].ID()))
		// we push the element on the stack back onto the buffer
		// if it had a head, it would not be on the stack
		conf.NumHeadStack--
		// conf.Arcs().Add(newArc)
	case transition == a.SHIFT:
		wi, wiExists := conf.Queue().Pop()
		if !wiExists {
			panic("Can't shift, queue is empty")
		}
		conf.Assign(uint16(conf.Nodes[wi].ID()))
		conf.Stack().Push(wi)
		// if an element is shifted, it can't have a head
		// in arc standard, in any case of a dependent attached to a head
		// it is popped off the buffer, and not added to the stack
		conf.NumHeadStack++
	default:
		panic(fmt.Sprintf("Unknown transition %v SHIFT is %v", transition, a.SHIFT))
	}
	conf.SetLastTransition(rawTransition)
	return conf
}

func (a *ArcStandard) possibleTransitions(from Configuration, transitions chan int) {
	conf, ok := from.(*SimpleConfiguration)
	if !ok {
		panic("Got wrong configuration type")
	}
	_, qExists := conf.Queue().Peek()
	_, sExists := conf.Stack().Peek()
	if qExists {
		transitions <- a.SHIFT
	}
	// if sExists {
	// 	if sPeek != 0 {
	// 		for rel, _ := range a.Relations.Index {
	// 			// transitions <- Transition("LA-" + rel)
	// 			transitions <- Transition(int(a.LEFT) + rel)
	// 		}
	// 	}
	// }
	if sExists && qExists {
		for rel, _ := range a.Relations.Index {
			// transitions <- Transition("LA-" + rel)
			transitions <- a.LEFT + rel
		}
		for rel, _ := range a.Relations.Index {
			// transitions <- Transition("RA-" + rel)
			transitions <- a.RIGHT + rel
		}
	}
	close(transitions)
}

func (a *ArcStandard) GetTransitions(from Configuration) (byte, []int) {
	retval := make([]int, 0, 10)
	tType, transitions := a.YieldTransitions(from)
	for transition := range transitions {
		retval = append(retval, int(transition))
	}
	return tType, retval
}

func (a *ArcStandard) YieldTransitions(from Configuration) (byte, chan int) {
	transitions := make(chan int)
	go a.possibleTransitions(from, transitions)
	return TransitionType, transitions
}

func (a *ArcStandard) TransitionTypes() []string {
	return []string{"LA-*", "RA-*", "SH"}
}

func (a *ArcStandard) Projective() bool {
	return true
}

func (a *ArcStandard) Labeled() bool {
	return true
}

func (a *ArcStandard) Oracle() Oracle {
	return a.oracle
}

func (a *ArcStandard) AddDefaultOracle() {
	a.oracle = Oracle(&ArcStandardOracle{Transitions: a.Transitions, LA: int(a.LEFT), RA: int(a.RIGHT)})
}

func (a *ArcStandard) Name() string {
	return "Arc Standard"
}

type ArcStandardOracle struct {
	LA, RA      int
	Transitions *util.EnumSet
	gold        LabeledDependencyGraph
	arcSet      *ArcSetSimple
}

var _ Decision = &ArcStandardOracle{}

func (o *ArcStandardOracle) SetGold(g interface{}) {
	labeledGold, ok := g.(LabeledDependencyGraph)
	if !ok {
		panic("Gold is not a labeled dependency graph")
	}
	o.gold = labeledGold
	o.arcSet = NewArcSetSimpleFromGraph(o.gold)
}

func (o *ArcStandardOracle) Transition(conf Configuration) Transition {
	c := conf.(*SimpleConfiguration)

	if o.gold == nil {
		panic("Oracle needs gold reference, use SetGold")
	}
	// Given Gd=(Vd,Ad) # gold dependencies
	// o(c = (S,B,A)) =
	// LA-r	if	(B[0],r,S[0]) in Ad
	// RA-r	if	(S[0],r,B[0]) in Ad; and for all w,r', if (B[0],r',w) in Ad then (B[0],r',w) in A
	// SH	otherwise
	bTop, bExists := c.Queue().Peek()
	sTop, sExists := c.Stack().Peek()
	var (
		index int
		// exists bool
	)
	if bExists {
		if sExists {
			// test if should Left-Attach
			arcs := o.arcSet.Get(&BasicDepArc{bTop, -1, sTop, DepRel("")})
			if len(arcs) > 0 {
				arc := arcs[0]
				index, _ = o.Transitions.IndexOf("LA-" + string(arc.GetRelation()))
				return &TypedTransition{TransitionType, index}
			}

			// test if should Right-Attach
			arcs = o.arcSet.Get(&BasicDepArc{sTop, -1, bTop, DepRel("")})
			if len(arcs) > 0 {
				reverseArcs := o.arcSet.Get(&BasicDepArc{bTop, -1, -1, DepRel("")})
				// for all w,r', if (B[0],r',w) in Ad then (B[0],r',w) in A
				// otherwise, return SH
				for _, arc := range reverseArcs {
					revArcs := c.Arcs().Get(arc)
					if len(revArcs) == 0 {
						index, _ = o.Transitions.IndexOf("SH")
						return &TypedTransition{TransitionType, index}
					}
				}
				arc := arcs[0]
				// return Transition("RA-" + string(arc.GetRelation()))
				index, _ = o.Transitions.IndexOf("RA-" + string(arc.GetRelation()))
				return &TypedTransition{TransitionType, index}
			}
		}
		index, _ = o.Transitions.IndexOf("SH")
		return &TypedTransition{TransitionType, index}
	}
	panic(fmt.Sprintf("Got empty configuration %v", c))

}

func (o *ArcStandardOracle) Name() string {
	return "Arc Standard"
}
