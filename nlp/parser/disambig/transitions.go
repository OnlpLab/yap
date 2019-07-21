package disambig

import (
	. "github.com/OnlpLab/yap/alg/transition"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	"fmt"
	"log"
	// "strings"
)

const TSAllOut bool = false

var LEMMAS bool = true

type MDTrans struct {
	ParamFunc MDParam
	POP       Transition

	Transitions *util.EnumSet
	oracle      Oracle

	Log    bool
	UsePOP bool
}

var _ TransitionSystem = &MDTrans{}

func (t *MDTrans) Transition(from Configuration, transition Transition) Configuration {
	c := from.Copy().(*MDConfig)

	if transition.Type() == 'L' {
		lemma := t.Transitions.ValueOf(transition.Value()).(string)
		if TSAllOut || t.Log {
			log.Println("Lexicalizing", lemma)
			log.Println("Pre copy:", from.(*MDConfig).Lemmas)
			log.Println("Post copy:", c.Lemmas)
		}
		c.SetLastTransition(transition)
		c.ChooseLemma(lemma)
		return c
	}
	if t.UsePOP && (transition.Type() == 'P' || transition == t.POP) {
		c.Pop()
		c.SetLastTransition(transition)
		if TSAllOut || t.Log {
			log.Println("POPing")
		}
		return c
	}
	if transition.Equal(ConstTransition(0)) {
		c.SetLastTransition(transition)
		if TSAllOut || t.Log {
			log.Println("Idling")
		}
		return c
	}
	paramStr := t.Transitions.ValueOf(transition.Value())
	qTop, _ := c.LatticeQueue.Peek()
	// if !qExists {
	// 	panic("Lattice queue is empty! Whatcha doin'?!")
	// }

	if TSAllOut || t.Log {
		log.Println("\tQtop:", qTop, "currentNode", c.CurrentLatNode)
	}
	lattice := c.Lattices[qTop]
	if TSAllOut || t.Log {
		log.Println("\tAt lattice", qTop, "-", lattice.Token)
		log.Println("\tCurrent lat node", c.CurrentLatNode)
	}
	nexts, _ := lattice.Next[c.CurrentLatNode]
	if TSAllOut || t.Log {
		log.Println("\tNexts are", nexts)
		log.Println("\tMorphemes are", lattice.Morphemes)
		log.Println("\tLattice is", lattice)
	}
	var (
		ambLemmas  []int
		foundMorph *EMorpheme
	)
	for _, next := range nexts {
		morph := lattice.Morphemes[next]
		if TSAllOut || t.Log {
			log.Println("\tComparing morpheme param val", t.ParamFunc(morph), "to", paramStr, t.ParamFunc(morph) == paramStr)
		}
		if t.ParamFunc(morph) == paramStr {
			if foundMorph == nil {
				// log.Println("\t\tSetting morph", morph)
				c.SetLastTransition(transition)
				foundMorph = morph
			} else if ambLemmas == nil {
				// log.Println("\t\tSetting amb lemmas", foundMorph, morph)
				ambLemmas = make([]int, 2, 3)
				ambLemmas[0] = foundMorph.ID()
				ambLemmas[1] = morph.ID()
			} else {
				// log.Println("\t\tAppending to amb lemmas", morph)
				ambLemmas = append(ambLemmas, morph.ID())
			}
		}
	}
	if foundMorph != nil {
		if LEMMAS && ambLemmas != nil && len(ambLemmas) > 1 {
			if TSAllOut || t.Log {
				log.Println("Add lemma ambiguity", ambLemmas)
			}
			c.AddLemmaAmbiguity(ambLemmas)
		} else {
			if TSAllOut || t.Log {
				log.Println("Adding morph", foundMorph)
			}
			c.AddMapping(foundMorph)
		}
		return c
	}
	var panicStr string
	panicStr = "transition did not match a given morpheme :`( -- "
	panicStr += fmt.Sprintf("failed to transition to %v", paramStr)
	panic(panicStr)
}

func (t *MDTrans) TransitionTypes() []string {
	return []string{"MD:M-*", "MD:L-*", "MD:P-*"}
}

func (t *MDTrans) possibleTransitions(conf *MDConfig, transitions chan int) {
	var (
		transition int
		morph      *EMorpheme
	)

	if conf.State() == 'L' {
		currentLat, exists := conf.LatticeQueue.Peek()
		if !exists {
			panic("Can't choose lemma if no lattices are in the queue")
		}
		latticeMorphemes := conf.Lattices[currentLat].Morphemes
		for _, m := range conf.Lemmas {
			morph = latticeMorphemes[m]
			transition, _ = t.Transitions.Add(morph.Lemma)
			transitions <- transition
		}
	}
	if t.UsePOP && conf.State() == 'P' {
		transitions <- t.POP.Value()
	}
	if conf.State() == 'M' {
		qTop, qExists := conf.LatticeQueue.Peek()
		if qExists {
			lat := conf.Lattices[qTop]
			if conf.CurrentLatNode < lat.Top() {
				nextList, _ := lat.Next[conf.CurrentLatNode]
				if t.Log {
					log.Println("\t\tpossible transitions", nextList)
				}
				for _, next := range nextList {
					transition, _ = t.Transitions.Add(t.ParamFunc(lat.Morphemes[next]))
					transitions <- transition
				}
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

func (a *MDTrans) GetTransitions(from Configuration) (byte, []int) {
	retval := make([]int, 0, 10)
	tType, transitions := a.YieldTransitions(from)
	for transition := range transitions {
		retval = append(retval, int(transition))
	}
	return tType, retval
}

func (t *MDTrans) YieldTransitions(c Configuration) (byte, chan int) {
	conf, ok := c.(*MDConfig)
	if !ok {
		panic("Got wrong configuration type")
	}
	transitions := make(chan int)
	go t.possibleTransitions(conf, transitions)
	return conf.State(), transitions
}

func (t *MDTrans) Oracle() Oracle {
	return t.oracle
}

func (t *MDTrans) AddDefaultOracle() {
	t.oracle = &MDOracle{
		Transitions: t.Transitions,
		ParamFunc:   t.ParamFunc,
		UsePOP:      t.UsePOP,
	}
}

func (t *MDTrans) Name() string {
	return "Morpheme-Based Morphological Disambiguator"
}

type MDOracle struct {
	Transitions *util.EnumSet
	gold        Mappings
	ParamFunc   MDParam
	UsePOP      bool
}

var _ Decision = &MDOracle{}

func (o *MDOracle) SetGold(g interface{}) {
	mappings, ok := g.(Mappings)
	if !ok {
		panic("Gold is not an array of mappings")
	}
	o.gold = mappings
}

func (o *MDOracle) CountMatchingTrans(c *MDConfig, pf MDParam, testTrans string) (matches int, matching string) {
	qTop, _ := c.LatticeQueue.Peek()
	lat := c.Lattices[qTop]
	if c.CurrentLatNode >= lat.Top() {
		panic("current lat node >= lattice's top :s")
	}
	nextList, _ := lat.Next[c.CurrentLatNode]
	// log.Println("\t\tpossible transitions", nextList, "for", testTrans)
	for _, next := range nextList {
		transStr := pf(lat.Morphemes[next])
		if transStr == testTrans {
			// log.Println("\t\t\t", transStr, "matches")
			matching = o.ParamFunc(lat.Morphemes[next])
			matches++
			continue
		}
		// log.Println("\t\t\t", transStr)
	}
	return
}

func (o *MDOracle) Transition(conf Configuration) Transition {
	c := conf.(*MDConfig)

	if o.gold == nil {
		panic("Oracle needs gold reference, use SetGold")
	}

	qTop, qExists := c.LatticeQueue.Peek()
	if o.UsePOP && c.State() == 'P' {
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
	// log.Println("Current gold")
	// log.Println(o.gold)
	// log.Println("Top", qTop)
	// log.Println("Yielding", goldSpellout)

	// log.Println("Current mappings")
	// log.Println(c.Mappings)
	var spellOutMorph int
	if len(c.Mappings) > 0 {
		confSpellout := c.Mappings[len(c.Mappings)-1].Spellout
		spellOutMorph = len(confSpellout)
		// log.Println("Confspellout")
		// log.Println(confSpellout)
		// log.Println("At lattice", qTop, "mapping", len(confSpellout))
		// log.Println("GoldSpellout", goldSpellout)
		// log.Println("len(confSpellout)", len(confSpellout))
		// currentMorph := goldSpellout[len(confSpellout)]
		// log.Println("Gold morpheme", currentMorph.Form)
	}
	if c.State() == 'L' {
		// need lexicalization
		morph := goldSpellout[spellOutMorph]

		lemmas := make([]string, 1, len(c.Lemmas))
		lemmas[0] = morph.Lemma
		// latticeMorphemes := c.Lattices[qTop].Morphemes
		// log.Println("Lattice:")
		// log.Println("Printing Morphemes by order")
		// for _, m := range latticeMorphemes {
		// 	log.Println("\t", m.ID(), m)
		// }
		// for _, morphID := range c.Lemmas {
		// 	otherMorph := latticeMorphemes[morphID]
		// 	// log.Println("Comparing", morph, otherMorph)
		// 	if morph.Lemma != otherMorph.Lemma {
		// 		lemmas = append(lemmas, otherMorph.Lemma)
		// 	}
		// }
		// log.Println("Lex options", morph.TokenID-1, spellOutMorph, strings.Join(lemmas, "|"))
		transition, _ := o.Transitions.Add(morph.Lemma)
		return &TypedTransition{'L', transition}
	}
	// need morphological disambiguation
	var paramVal string
	if spellOutMorph < len(goldSpellout) {
		paramVal = o.ParamFunc(goldSpellout[spellOutMorph])
	} else {
		qTop, _ := c.LatticeQueue.Peek()
		lat := c.Lattices[qTop]
		nextList, _ := lat.Next[c.CurrentLatNode]
		if len(nextList) == 1 {
			morph := lat.Morphemes[nextList[0]]
			if morph.CPOS == "NNP" && lat.Top() == morph.To() {
				log.Println("\t\tOracle has no gold, using only possible morpheme, possible OOV; token", qTop)
			} else {
				log.Println("\t\tOracle has no gold, using only possible morpheme", morph)
			}
		} else {
			log.Println("\t\tOracle has no gold, arbitrarily attempting to use first possible morpheme")
		}
		paramVal = o.ParamFunc(lat.Morphemes[nextList[0]])
	}

	// failoverPFStr := []string{"Lemma_POS_Prop", "Funcs_Lemma_Main_POS", "Funcs_Main_POS", "POS_Prop", "POS", "Form"}
	// failoverPFs := []MDParam{Lemma_POS_Prop, Funcs_Lemma_Main_POS, Funcs_Main_POS, POS_Prop, POS, Form}
	failoverPFStr := []string{"Funcs_Main_POS", "POS_Prop", "POS", "Form"}
	failoverPFs := []MDParam{Funcs_Main_POS, POS_Prop, POS, Form}
	verifyPossibleTransition := true
	if verifyPossibleTransition {
		matches, matching := o.CountMatchingTrans(c, o.ParamFunc, paramVal)
		if matches == 0 {
			// log.Println("\tmatch not found, trying to match relaxed param func for gold morph", goldSpellout[spellOutMorph])
			for i, _ := range failoverPFStr {
				relaxedPF := failoverPFs[i]
				// log.Println("\t\tTrying pf", relaxedPFStr)
				paramVal = relaxedPF(goldSpellout[spellOutMorph])
				matches, matching = o.CountMatchingTrans(c, relaxedPF, paramVal)
				if matches >= 1 {
					paramVal = matching
					break
				}
			}
		}
		if matches > 1 {
			// log.Println("\t\tOracle found too many matches, arbitrarily designating last found match for token", qTop, ":", matching)
			// panic("found too many matches, can't distinguish gold morpheme")
		} else {
			// log.Println("\t\tMatch found for '", paramVal, "'")
		}
		paramVal = matching
		if matches == 0 {
			qTop, _ := c.LatticeQueue.Peek()
			lat := c.Lattices[qTop]
			nextList, _ := lat.Next[c.CurrentLatNode]
			if len(nextList) == 1 && lat.Morphemes[nextList[0]].CPOS == "NNP" {
				log.Println("\t\tOracle found no matches, only morpheme is NNP, assuming OOV for token", qTop, ":", matching, "for", len(goldSpellout), "gold morphs")
				paramVal = o.ParamFunc(lat.Morphemes[nextList[0]])
				// log.Println("\t\tUsing transition", paramVal)
			} else {
				panic(fmt.Sprintf("failed to find gold match for gold morph %v", goldSpellout[spellOutMorph]))
			}
		} else {
			// log.Println("\tmatch found")
		}
	}

	transition, _ := o.Transitions.Add(paramVal)
	return &TypedTransition{'M', transition}
}

func (o *MDOracle) Name() string {
	return "MD Exact Match"
}
