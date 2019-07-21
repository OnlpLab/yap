package transition

import (
	"fmt"
	"log"
	. "github.com/OnlpLab/yap/alg/transition"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
)

var (
	ArcAllOut           = false
	TransitionType byte = 'A'
)

type ArcEager struct {
	ArcStandard
	POPROOT, REDUCE int
}

// Verify that ArcEager is a TransitionSystem
var _ TransitionSystem = &ArcEager{}

func (a *ArcEager) Transition(from Configuration, rawTransition Transition) Configuration {
	conf, ok := from.Copy().(*SimpleConfiguration)
	if !ok {
		panic("Got wrong configuration type")
	}
	transition := rawTransition.Value()
	// Transition System:
	// LA-r	(S|wi,	wj|B,	A) => (S      ,	wj|B,	A+{(wj,r,wi)})	if: (wk,r',wi) notin A; i != 0
	// RA-r	(S|wi,	wj|B,	A) => (S|wi|wj,	   B,	A+{(wi,r,wj)})
	// RE	(S|wi,	   B,	A) => (S      ,	   B,	A)				if: (wk,r',wi) in A
	// SH	(S   ,	wi|B, 	A) => (S|wi   ,	   B,	A)
	// switch transition[:2] {
	switch {
	case transition >= a.LEFT && transition < a.RIGHT:
		wi, wiExists := conf.Stack().Pop()
		// arcs := conf.Arcs().Get(&BasicDepArc{-1, -1, wi, DepRel("")})
		if conf.Arcs().HasHead(wi) {
			panic("Can't create arc for wi, it already has a head")
		}
		wj, wjExists := conf.Queue().Peek()
		if !(wiExists && wjExists) {
			panic("Can't LA, Stack and/or Queue are/is empty")
		}
		// relation := DepRel(transition[3:])
		relation := int(transition - a.LEFT)
		relationValue := a.Relations.ValueOf(relation).(DepRel)
		newArc := &BasicDepArc{wj, relation, wi, relationValue}
		conf.AddArc(newArc)
		conf.Assign(uint16(conf.Nodes[newArc.Modifier].ID()))
		conf.NumHeadStack--
	// case "RA":
	case transition >= a.RIGHT:
		wi, wiExists := conf.Stack().Peek()
		wj, wjExists := conf.Queue().Pop()
		if !(wiExists && wjExists) {
			panic("Can't RA, Stack and/or Queue are/is empty")
		}
		// rel := DepRel(transition[3:])
		rel := int(transition - a.RIGHT)
		relValue := a.Relations.ValueOf(rel).(DepRel)
		newArc := &BasicDepArc{wi, rel, wj, relValue}
		conf.Stack().Push(wj)
		conf.AddArc(newArc)
		conf.Assign(uint16(conf.Nodes[newArc.Modifier].ID()))
	case transition == a.REDUCE:
		if conf.Stack().Size() == 1 {
			panic("Attempted to reduce ROOT (should POPROOT)")
		}
		wi, wiExists := conf.Stack().Pop()
		// arcs := conf.Arcs().Get(&BasicDepArc{-1, -1, wi, DepRel("")})
		if !wiExists {
			panic("Can't reduce, queue is empty")
		}
		_, wjExists := conf.Queue().Peek()
		// if len(arcs) == 0 {
		if !conf.Arcs().HasHead(wi) && wjExists {
			panic(fmt.Sprintf("Can't reduce %d if it doesn't have a head", wi))
		}
		conf.Assign(uint16(conf.Nodes[wi].ID()))

	case transition == a.SHIFT:
		wi, wiExists := conf.Queue().Pop()
		if !wiExists {
			panic("Can't shift, queue is empty")
		}
		conf.Stack().Push(wi)
		conf.Assign(uint16(conf.Nodes[wi].ID()))
		conf.NumHeadStack++
	case transition == a.POPROOT:
		_, wjExists := conf.Queue().Pop()
		if wjExists {
			panic("Can't poproot, queue is not empty")
		}
		stackSize := conf.Stack().Size()
		if stackSize != 1 {
			panic("Can't poproot, stack has doesn't have just 1 value")
		}
		wi, _ := conf.Stack().Pop()
		relID, _ := a.Relations.IndexOf(ROOT_LABEL)
		newArc := &BasicDepArc{0, relID, wi, DepRel(ROOT_LABEL)}
		conf.AddArc(newArc)
		conf.Assign(uint16(conf.Nodes[wi].ID()))
	}
	conf.SetLastTransition(rawTransition)
	return conf
}

func (a *ArcEager) TransitionTypes() []string {
	standardTypes := a.ArcStandard.TransitionTypes()
	standardTypes = append(standardTypes, "RE")
	standardTypes = append(standardTypes, "PR")
	return standardTypes
}

func (a *ArcEager) possibleTransitions(from Configuration, transitions chan int) {
	conf, ok := from.(*SimpleConfiguration)
	if !ok {
		panic("Got wrong configuration type")
	}
	_, qExists := conf.Queue().Peek()
	sSize := conf.Stack().Size()
	qSize := conf.Queue().Size()
	sPeek, sExists := conf.Stack().Peek()

	if !qExists {
		if sSize == 1 {
			transitions <- a.POPROOT
		}
		if sSize > 1 {
			if ArcAllOut {
				log.Println("REDUCE")
			}
			transitions <- a.REDUCE
		}
	} else {
		if conf.GetLastTransition() == nil || conf.GetLastTransition().Value() != a.REDUCE {
			if !sExists || qSize > 1 {
				if ArcAllOut {
					log.Println("SHIFT")
				}
				transitions <- a.SHIFT
			}
		}

		// sPeekHasModifiers2 := len(conf.Arcs().Get(&BasicDepArc{-1, -1, sPeek, DepRel("")})) > 0
		if sExists {
			sPeekHasHead := conf.Arcs().HasHead(sPeek)
			if ArcAllOut {
				log.Println("Head Stack Size", conf.NumHeadStack)
			}
			if qSize > 1 || conf.NumHeadStack == 1 {
				if ArcAllOut {
					if qSize > 1 {
						log.Println("Queue Len", conf.Graph().NumberOfNodes()-qSize)
					}
					if conf.NumHeadStack == 1 {
						log.Println("Head Stack Size")
					}
					log.Println("ARCRIGHT")
				}
				for rel, _ := range a.Relations.Index {
					// transitions <- Transition("RA-" + rel)
					transitions <- a.RIGHT + rel
				}
			}
			if (sPeekHasHead || !qExists) && sSize > 1 {
				if ArcAllOut {
					log.Println("REDUCE")
				}
				transitions <- a.REDUCE
			}
			if qExists && !sPeekHasHead {
				if ArcAllOut {
					log.Println("ARCLEFT")
				}
				for rel, _ := range a.Relations.Index {
					// transitions <- Transition("LA-" + rel)
					transitions <- a.LEFT + rel
				}
			}
		}
	}
	close(transitions)
}

func (a *ArcEager) GetTransitions(from Configuration) (byte, []int) {
	retval := make([]int, 0, 10)
	tType, transitions := a.YieldTransitions(from)
	for transition := range transitions {
		retval = append(retval, transition)
	}
	return tType, retval
}

func (a *ArcEager) YieldTransitions(from Configuration) (byte, chan int) {
	transitions := make(chan int)
	go a.possibleTransitions(from, transitions)
	return TransitionType, transitions
}

func (a *ArcEager) AddDefaultOracle() {
	a.oracle = Oracle(&ZparArcEagerOracle{Transitions: a.Transitions, LA: int(a.LEFT), RA: int(a.RIGHT)})
}

// type NivreArcEagerOracle struct {
// 	ArcStandardOracle
// 	Transitions *util.EnumSet
// 	LA, RA      int
// }

// var _ Decision = &NivreArcEagerOracle{}

// func (o *NivreArcEagerOracle) Transition(conf Configuration) Transition {
// 	c := conf.(*SimpleConfiguration)

// 	if o.gold == nil {
// 		panic("Oracle needs gold reference, use SetGold")
// 	}
// 	// # http://www.cs.bgu.ac.il/~yoavg/publications/coling2012dynamic.pdf
// 	// Given Gd=(Vd,Ad) # gold dependencies
// 	// o(c = (S,B,A)) =
// 	// LA-r	if	(B[0],r,S[0]) in Ad
// 	// RA-r	if	(S[0],r,B[0]) in Ad
// 	// RE	if	i=S[0],  j=B[0], exists k<i and exists r: (B[0],r,k) in Ad or (k,r,B[0]) in Ad
// 	// SH	otherwise
// 	bTop, bExists := c.Queue().Peek()
// 	sTop, sExists := c.Stack().Peek()
// 	sSize := c.Stack().Size()
// 	var (
// 		index  int
// 		exists bool
// 		arcs   []LabeledDepArc
// 	)

// 	if sExists && !bExists {
// 		sSize := c.Stack().Size()
// 		if sSize == 1 {
// 			index, exists = o.Transitions.IndexOf("PR")
// 			if !exists {
// 				panic("PR not found in trans enum")
// 			}
// 			return Transition(index)
// 		}
// 	}

// 	if bExists && sExists {
// 		// test if should Left-Attach
// 		arcs = o.arcSet.Get(&BasicDepArc{bTop, -1, sTop, DepRel("")})
// 		// log.Println("LA test returned", len(arcs))
// 		if len(arcs) > 0 {
// 			arc := arcs[0]
// 			index, exists = o.Transitions.IndexOf("LA-" + string(arc.GetRelation()))
// 			if !exists {
// 				panic("LA-" + string(arc.GetRelation()) + " not found in trans enum")
// 			}
// 			// log.Println("Oracle", o.Transitions.ValueOf(index))
// 			return Transition(index)
// 		}

// 		// test if should Right-Attach
// 		arcs = o.arcSet.Get(&BasicDepArc{sTop, -1, bTop, DepRel("")})
// 		// log.Println("RA test returned", len(arcs))
// 		if len(arcs) > 0 {
// 			arc := arcs[0]
// 			index, exists = o.Transitions.IndexOf("RA-" + string(arc.GetRelation()))
// 			if !exists {
// 				panic("RA-" + string(arc.GetRelation()) + " not found in trans enum")
// 			}
// 			// log.Println("Oracle", o.Transitions.ValueOf(index))
// 			return Transition(index)
// 		}
// 	}
// 	// test if should reduce
// 	if sExists && sSize > 1 {
// 		// if modifier < sTop, REDUCE
// 		arcs = o.arcSet.Get(&BasicDepArc{bTop, -1, -1, DepRel("")})
// 		for _, arc := range arcs {
// 			if arc.GetModifier() < sTop {
// 				index, exists = o.Transitions.IndexOf("RE")
// 				if !exists {
// 					panic("RE not found in trans enum")
// 				}
// 				// log.Println("Oracle", o.Transitions.ValueOf(index), "arc", arc.String())
// 				return Transition(index)
// 			}
// 		}
// 		// if head < sTop, REDUCE
// 		arcs = o.arcSet.Get(&BasicDepArc{-1, -1, bTop, DepRel("")})
// 		for _, arc := range arcs {
// 			if arc.GetHead() < sTop {
// 				index, exists = o.Transitions.IndexOf("RE")
// 				if !exists {
// 					panic("RE not found in trans enum")
// 				}
// 				// log.Println("Oracle2", o.Transitions.ValueOf(index), "arc", arc.String())
// 				return Transition(index)
// 			}
// 		}
// 	}
// 	if bExists {
// 		index, exists = o.Transitions.IndexOf("SH")
// 		if !exists {
// 			panic("SH not found in trans enum")
// 		}
// 		return Transition(index)
// 	}
// 	panic(fmt.Sprintf("Oracle cannot take any action when both stack and queue are empty (%v,%v)", sExists, bExists))
// }

func (a *ArcEager) Name() string {
	return "Arc Zeager (zpar acl '11) [a.k.a. ArcZEager]"
}

type ZparArcEagerOracle struct {
	ArcStandardOracle
	Transitions *util.EnumSet
	LA, RA      int
}

var _ Decision = &ZparArcEagerOracle{}

func (o *ZparArcEagerOracle) Transition(conf Configuration) Transition {
	c := conf.(*SimpleConfiguration)
	// log.Println("Oracle at", c)
	if o.gold == nil {
		panic("Oracle needs gold reference, use SetGold")
	}
	// zpar oracle
	bTop, bExists := c.Queue().Peek()
	if !bExists {
		bTop = -1
	}
	sTop, sExists := c.Stack().Peek()
	sSize := c.Stack().Size()
	var (
		index  int
		exists bool
	)

	if !bExists {
		if !sExists {
			panic("Queue empty while stack empty too")
		}
		if sSize > 1 {
			index, exists = o.Transitions.IndexOf("RE")
			if !exists {
				panic("RE not found in trans enum")
			}
			// log.Println("Oracle 1", o.Transitions.ValueOf(index))
			return &TypedTransition{TransitionType, index}
		} else {
			index, exists = o.Transitions.IndexOf("PR")
			if !exists {
				panic("PR not found in trans enum")
			}
			// log.Println("Oracle 2", o.Transitions.ValueOf(index))
			return &TypedTransition{TransitionType, index}
		}
	}

	if sExists {
		top := sTop
		for c.GetLabeledArc(top) != nil && c.GetLabeledArc(top).GetHead() != -1 {
			top = c.GetLabeledArc(top).GetHead()
		}
		arc := o.gold.GetLabeledArc(top)
		if arc.GetHead() == bTop {
			if top == sTop {
				index, exists = o.Transitions.IndexOf("LA-" + string(arc.GetRelation()))
				if !exists {
					panic("LA-" + string(arc.GetRelation()) + " not found in trans enum")
				}
				// log.Println("Oracle 3", o.Transitions.ValueOf(index))
				return &TypedTransition{TransitionType, index}
			} else {
				index, exists = o.Transitions.IndexOf("RE")
				if !exists {
					panic("RE not found in trans enum")
				}
				// log.Println("Oracle 4", o.Transitions.ValueOf(index), "arc", arc.String())
				return &TypedTransition{TransitionType, index}
			}
		}
	}

	goldHead := o.gold.GetLabeledArc(bTop).GetHead()
	if goldHead == -1 || goldHead > bTop {
		index, exists = o.Transitions.IndexOf("SH")
		if !exists {
			panic("SH not found in trans enum")
		}
		// log.Println("Oracle 5", o.Transitions.ValueOf(index))
		return &TypedTransition{TransitionType, index}
	} else {
		arc := o.gold.GetLabeledArc(bTop)
		if arc.GetHead() == sTop {
			index, exists = o.Transitions.IndexOf("RA-" + string(arc.GetRelation()))
			if !exists {
				panic("RA-" + string(arc.GetRelation()) + " not found in trans enum")
			}
			// log.Println("Oracle 6", o.Transitions.ValueOf(index))
			return &TypedTransition{TransitionType, index}
		} else {
			index, exists = o.Transitions.IndexOf("RE")
			if !exists {
				panic("RE not found in trans enum")
			}
			// log.Println("Oracle 7", o.Transitions.ValueOf(index), "arc", arc.String())
			return &TypedTransition{TransitionType, index}
		}
	}
	panic(fmt.Sprintf("Oracle cannot take any action when both stack and queue are empty (%v,%v)", sExists, bExists))
}

func (o *ZparArcEagerOracle) Name() string {
	return "Zpar Arc Eager Oracle (zpar acl '11) [a.k.a. ArcZEager]"
}
