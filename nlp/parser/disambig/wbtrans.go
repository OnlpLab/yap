package disambig

import (
	"fmt"
	"log"
	. "github.com/OnlpLab/yap/alg/transition"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
)

type MDWBTrans struct {
	ParamFunc MDParam
	POP       Transition

	Transitions *util.EnumSet
	oracle      Oracle

	Log    bool
	UsePOP bool
}

var _ TransitionSystem = &MDWBTrans{}

func (t *MDWBTrans) Transition(from Configuration, transition Transition) Configuration {
	c := from.Copy().(*MDConfig)

	if transition.Equal(t.POP) && t.UsePOP {
		c.Pop()
		c.SetLastTransition(transition)
		if TSAllOut || t.Log {
			log.Println("POPing")
		}
		return c
	}
	// if transition == Transition(0) {
	// 	c.SetLastTransition(transition)
	// 	if TSAllOut || t.Log {
	// 		log.Println("Idling")
	// 	}
	// 	return c
	// }
	paramStr := t.Transitions.ValueOf(transition.Value()).(string)
	qTop, qExists := c.LatticeQueue.Peek()
	if !qExists {
		panic("Lattice queue is empty! Whatcha doin'?!")
	}

	if TSAllOut || t.Log {
		log.Println("Qtop:", qTop, "currentNode", c.CurrentLatNode)
	}
	lattice := c.Lattices[qTop]
	if TSAllOut || t.Log {
		log.Println("At lattice", qTop, "-", lattice.Token)
		log.Println("Current lat node", c.CurrentLatNode)
	}
	if TSAllOut || t.Log {
		log.Println("Nexts are", lattice.Spellouts)
	}
	if success := c.AddSpellout(paramStr, t.ParamFunc); success {
		if TSAllOut || t.Log {
			log.Println("Adding spellout", paramStr)
		}
		c.SetLastTransition(transition)
		if !t.UsePOP {
			c.Pop()
		}
		return c
	}
	var panicStr string
	panicStr = "transition did not match a given spellout :`( -- "
	panicStr += fmt.Sprintf("failed to transition to %v", paramStr)
	panic(panicStr)
}

func (t *MDWBTrans) TransitionTypes() []string {
	return []string{"MD-*"}
}

func (t *MDWBTrans) possibleTransitions(from Configuration, transitions chan int) {
	var transition int

	conf, ok := from.(*MDConfig)
	if !ok {
		panic("Got wrong configuration type")
	}
	qTop, qExists := conf.LatticeQueue.Peek()
	if t.UsePOP && conf.State() == 'P' {
		transitions <- t.POP.Value()
	} else {
		if qExists {
			lat := conf.Lattices[qTop]
			for _, s := range lat.Spellouts {
				transition, _ = t.Transitions.Add(ProjectSpellout(s, t.ParamFunc))
				transitions <- transition
			}
		} else {
			// if t.Log {
			// 	log.Println("\t\tpossible transitions IDLE")
			// }
			// transitions <- Transition(0)
		}
	}
	close(transitions)
}

func (a *MDWBTrans) GetTransitions(from Configuration) (byte, []int) {
	retval := make([]int, 0, 10)
	tType, transitions := a.YieldTransitions(from)
	for transition := range transitions {
		retval = append(retval, int(transition))
	}
	return tType, retval
}

func (t *MDWBTrans) YieldTransitions(c Configuration) (byte, chan int) {
	conf, ok := c.(*MDConfig)
	if !ok {
		panic("Got wrong configuration type")
	}
	transitions := make(chan int)
	go t.possibleTransitions(conf, transitions)
	return conf.State(), transitions
}

func (t *MDWBTrans) Oracle() Oracle {
	return t.oracle
}

func (t *MDWBTrans) AddDefaultOracle() {
	t.oracle = &MDWBOracle{
		Transitions: t.Transitions,
		ParamFunc:   t.ParamFunc,
		UsePOP:      t.UsePOP,
	}
}

func (t *MDWBTrans) Name() string {
	return "Word-Based Morphological Disambiguator"
}

type MDWBOracle struct {
	Transitions *util.EnumSet
	gold        Mappings
	ParamFunc   MDParam
	UsePOP      bool
}

var _ Decision = &MDWBOracle{}

func (o *MDWBOracle) SetGold(g interface{}) {
	mappings, ok := g.(Mappings)
	if !ok {
		panic("Gold is not an array of mappings")
	}
	o.gold = mappings
}

func (o *MDWBOracle) Transition(conf Configuration) Transition {
	c := conf.(*MDConfig)

	if o.gold == nil {
		panic("Oracle needs gold reference, use SetGold")
	}

	qTop, qExists := c.LatticeQueue.Peek()
	if o.UsePOP && ((!qExists && len(c.Mappings) != c.popped) ||
		(qExists && qTop != c.popped)) {
		return c.POP
	}
	if !qExists {
		// oracle forces a single final idle
		// if c.Last != Transition(0) {
		// 	return Transition(0)
		// }
		panic("No lattices in given configuration to disambiguate")
	}
	if len(o.gold) <= qTop {
		panic("Gold has less mappings than given configuration")
	}
	goldSpellout := o.gold[qTop].Spellout

	// log.Println("Confspellout")
	// log.Println(confSpellout)
	// log.Println("At lattice", qTop, "mapping", len(confSpellout))
	// log.Println("GoldSpellout", goldSpellout)
	// log.Println("len(confSpellout)", len(confSpellout))
	// currentMorph := goldSpellout[len(confSpellout)]
	// log.Println("Gold morpheme", currentMorph.Form)
	paramVal := ProjectSpellout(goldSpellout, o.ParamFunc)
	// log.Println("Gold transition", paramVal)
	transition, _ := o.Transitions.Add(paramVal)
	return &TypedTransition{conf.State(), transition}
}

func (o *MDWBOracle) Name() string {
	return "Word-Based MD Spellout Param Func Match"
}
