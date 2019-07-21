package morph

//
// import (
// 	. "github.com/OnlpLab/yap/alg/transition"
// 	. "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
// 	nlp "github.com/OnlpLab/yap/nlp/types"
//
// 	"fmt"
// 	// "log"
// )
//
// type ArcEagerMorph struct {
// 	ArcEager
// 	oracle Oracle
// 	MD     Transition
// }
//
// var _ TransitionSystem = &ArcEagerMorph{}
//
// func (a *ArcEagerMorph) Transition(from Configuration, transition Transition) Configuration {
// 	originalConf, ok := from.(*MorphConfiguration)
// 	if !ok {
// 		panic("Got wrong configuration type")
// 	}
// 	// if transition[:2] == "MD" {
// 	if transition >= a.MD {
// 		conf := originalConf.Copy().(*MorphConfiguration)
// 		lID, lExists := conf.LatticeQueue.Pop()
// 		lattice := conf.Lattices[lID]
// 		if !lExists {
// 			panic("Can't MD, Lattice Queue is empty")
// 		}
// 		// _, qExists := conf.Queue().Peek()
// 		// if qExists {
// 		// 	log.Println("Got transition", transition, a.Transitions.ValueOf(int(transition)))
// 		// 	panic("Can't MD, Queue is not empty")
// 		// }
// 		spelloutStr := a.Transitions.ValueOf(int(transition)).(string)[3:]
// 		// spelloutNum, err := strconv.Atoi(string(transition[3:]))
// 		// if err != nil {
// 		// 	panic("Error converting MD transition # to int:\n" + err.Error())
// 		// }
// 		lattice.GenSpellouts()
// 		var spellout nlp.Spellout
// 		for _, curSpellout := range lattice.Spellouts {
// 			if curSpellout.String() == spelloutStr {
// 				spellout = curSpellout
// 			}
// 		}
// 		token := lattice.Token
// 		conf.Mappings = append(conf.Mappings, &nlp.Mapping{token, spellout})
// 		numNodes := len(conf.Nodes)
// 		var id int
// 		for i, morpheme := range spellout {
// 			id = numNodes + i
// 			panic("Fix line below, should be enqueue")
// 			conf.Queue().Push(id)
// 			m := new(nlp.EMorpheme)
// 			*m = *morpheme
// 			m.BasicDirectedEdge[0] = len(conf.Nodes)
// 			conf.Nodes = append(conf.Nodes, NewArcCachedDepNode(nlp.DepNode(m)))
// 		}
// 		transitionIndex, _ := a.Transitions.Add("MD-" + spellout.String())
// 		conf.SetLastTransition(Transition(transitionIndex))
// 		return conf
// 	} else {
// 		copyconf := originalConf.Copy().(*MorphConfiguration)
// 		copyconf.SimpleConfiguration = *a.ArcEager.Transition(&originalConf.SimpleConfiguration, transition).(*SimpleConfiguration)
// 		return copyconf
// 	}
// }
//
// func (a *ArcEagerMorph) TransitionTypes() []string {
// 	eagerTypes := a.ArcEager.TransitionTypes()
// 	eagerTypes = append(eagerTypes, "MD-*")
// 	return eagerTypes
// }
//
// func (a *ArcEagerMorph) YieldTransitions(from Configuration) chan Transition {
// 	conf, ok := from.(*MorphConfiguration)
// 	if !ok {
// 		panic("Got wrong configuration type")
// 	}
// 	qSize := conf.Queue().Size()
// 	latticeID, lExists := conf.LatticeQueue.Peek()
// 	lattice := conf.Lattices[latticeID]
// 	var (
// 		spellout nlp.Spellout
// 		transID  int
// 	)
// 	if lExists && qSize < 3 {
// 		morphChan := make(chan Transition)
// 		go func() {
// 			for path := range lattice.YieldPaths() {
// 				spellout = lattice.Spellouts[path]
// 				transID, _ = a.Transitions.Add("MD-" + spellout.String())
// 				morphChan <- Transition(transID)
// 			}
// 			close(morphChan)
// 		}()
// 		return morphChan
// 	} else {
// 		return a.ArcEager.YieldTransitions(&conf.SimpleConfiguration)
// 	}
// }
//
// func (a *ArcEagerMorph) AddDefaultOracle() {
// 	a.oracle = Oracle(&ArcEagerMorphOracle{
// 		ZparArcEagerOracle: ZparArcEagerOracle{
// 			Transitions: a.Transitions,
// 		},
// 		MD: int(a.MD),
// 	})
// 	a.ArcEager.AddDefaultOracle()
// }
//
// func (a *ArcEagerMorph) Oracle() Oracle {
// 	return a.oracle
// }
//
// type ArcEagerMorphOracle struct {
// 	ZparArcEagerOracle
// 	morphGold []*nlp.Mapping
// 	MD        int
// }
//
// var _ Decision = &ArcEagerMorphOracle{}
//
// func (o *ArcEagerMorphOracle) SetGold(g interface{}) {
// 	morphGold, ok := g.(nlp.MorphDependencyGraph)
// 	if !ok {
// 		panic("Gold is not a morph dependency graph")
// 	}
// 	o.morphGold = morphGold.GetMappings()
// 	o.ZparArcEagerOracle.SetGold(g)
// }
//
// func (o *ArcEagerMorphOracle) Transition(conf Configuration) Transition {
// 	c := conf.(*MorphConfiguration)
// 	if o.morphGold == nil {
// 		panic("Oracle neds gold reference, use SetGold")
// 	}
// 	latticeID, lExists := c.LatticeQueue.Peek()
// 	bSize := c.Queue().Size()
// 	// log.Println("Oracle got Conf:", c)
// 	if lExists && bSize < 3 {
// 		lattice := c.Lattices[latticeID]
// 		mapping := o.morphGold[len(c.Mappings)]
// 		lattice.GenSpellouts()
// 		pathId, exists := lattice.Spellouts.Find(mapping.Spellout)
// 		if !exists {
// 			panic(fmt.Sprintf("Oracle can't find oracle spellout in instance lattice %v", latticeID))
// 		}
// 		transStr := "MD-" + lattice.Spellouts[pathId].String()
// 		// log.Println("Oracle:", transStr)
// 		transEnum, _ := o.Transitions.Add(transStr)
// 		// log.Println("Oracle", transStr)
// 		return Transition(transEnum)
// 	} else {
// 		oracleTrans := o.ZparArcEagerOracle.Transition(&c.SimpleConfiguration)
// 		// log.Println("Oracle:", o.Transitions.ValueOf(int(oracleTrans)))
// 		return oracleTrans
// 	}
// }
