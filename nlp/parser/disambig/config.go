package disambig

import (
	// "github.com/OnlpLab/yap/alg/graph"
	. "github.com/OnlpLab/yap/alg"
	"github.com/OnlpLab/yap/alg/featurevector"
	. "github.com/OnlpLab/yap/alg/transition"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	"fmt"
	"log"
	// "reflect"
	"sort"
	"strings"
)

var (
	UsePOP           bool
	SwitchFormLemma  bool
	POP_ONLY_VAR_LEN bool = true
	AFFIX_SIZE       int  = 10
)

type MDConfig struct {
	LatticeQueue Queue
	Lattices     nlp.LatticeSentence
	Mappings     nlp.Mappings
	Morphemes    nlp.Morphemes
	Lemmas       []int

	CurrentLatNode int

	InternalPrevious Configuration
	Last             Transition
	ETokens          *util.EnumSet
	Log              bool

	POP         Transition
	Transitions *util.EnumSet
	ParamFunc   nlp.MDParam
	popped      int
}

var _ Configuration = &MDConfig{}

func (c *MDConfig) Init(abstractLattice interface{}) {
	latticeSent := abstractLattice.(nlp.LatticeSentence)
	sentLength := len(latticeSent)

	c.Lattices = latticeSent

	maxSentLength := 0
	var latP *nlp.Lattice
	for _, lat := range c.Lattices {
		latP = &lat
		maxSentLength += latP.MaxPathLen()
	}

	c.LatticeQueue = NewQueueSlice(sentLength)

	// push indexes of statement nodes to *LatticeQueue*, in reverse order (first word at the top of the queue)
	for i := 0; i < sentLength; i++ {
		c.LatticeQueue.Enqueue(i)
	}

	// initialize first mapping structure
	c.Mappings = make([]*nlp.Mapping, 0, len(c.Lattices))
	// c.Mappings[0] = &nlp.Mapping{c.Lattices[0].Token, make(nlp.Spellout, 0, 1)}
	c.Morphemes = make(nlp.Morphemes, 0, len(c.Lattices)*2)
	// explicit resetting of zero-valued properties
	// in case of reuse
	c.Last = ConstTransition(0)
	c.popped = 0
}

func (c *MDConfig) Terminal() bool {
	// return c.Last == Transition(0) && c.Alignment() == 1
	if UsePOP {
		return c.LatticeQueue.Size() == 0 && c.popped == len(c.Mappings)
	} else {
		return c.LatticeQueue.Size() == 0

	}
}

func (c *MDConfig) Copy() Configuration {
	newConf := new(MDConfig)
	c.CopyTo(newConf)
	return newConf
}

func (c *MDConfig) CopyTo(target Configuration) {
	newConf, ok := target.(*MDConfig)
	if !ok {
		panic("Can't copy into non *MDConfig")
	}
	newConf.ETokens = c.ETokens
	newConf.Mappings = make([]*nlp.Mapping, len(c.Mappings), util.Max(cap(c.Mappings), len(c.Lattices)))
	copy(newConf.Mappings, c.Mappings)

	newConf.Morphemes = make(nlp.Morphemes, len(c.Morphemes), cap(c.Morphemes))
	copy(newConf.Morphemes, c.Morphemes)

	// verify initialization (base configurations are not initialized)
	if len(c.Mappings) > 0 {
		// also copy a new spellout of the current mapping
		lastMappingIdx := len(c.Mappings) - 1
		newLastMapping := &nlp.Mapping{
			Token: c.Mappings[lastMappingIdx].Token,
			Spellout: make(nlp.Spellout,
				len(c.Mappings[lastMappingIdx].Spellout),
				cap(c.Mappings[lastMappingIdx].Spellout))}
		copy(newLastMapping.Spellout, c.Mappings[lastMappingIdx].Spellout)
		newConf.Mappings[lastMappingIdx] = newLastMapping
	}

	if c.LatticeQueue != nil {
		newConf.LatticeQueue = c.LatticeQueue.Copy()
	}
	if c.Lemmas != nil && len(c.Lemmas) > 0 {
		newConf.Lemmas = make([]int, len(c.Lemmas))
		copy(newConf.Lemmas, c.Lemmas)
	}
	// lattices slice is read only, no need for copy
	newConf.Lattices = c.Lattices
	newConf.InternalPrevious = c
	newConf.CurrentLatNode = c.CurrentLatNode
	newConf.popped = c.popped
	newConf.POP = c.POP
	newConf.Transitions = c.Transitions
	newConf.ParamFunc = c.ParamFunc
}

func (c *MDConfig) GetSequence() ConfigurationSequence {
	if c.Mappings == nil {
		return make(ConfigurationSequence, 0)
	}
	retval := make(ConfigurationSequence, 0, len(c.Mappings))
	currentConf := c
	for {
		retval = append(retval, currentConf)
		if currentConf.InternalPrevious == nil {
			break
		} else {
			currentConf = currentConf.InternalPrevious.(*MDConfig)
		}
	}
	return retval
}

func (c *MDConfig) SetLastTransition(t Transition) {
	c.Last = t
}

func (c *MDConfig) GetLastTransition() Transition {
	return Transition(c.Last)
}

func (c *MDConfig) State() byte {
	if c.Lemmas != nil && len(c.Lemmas) > 0 {
		// needs lemmatization
		return 'L'
	}
	qTop, qExists := c.LatticeQueue.Peek()
	if UsePOP && ((!qExists && len(c.Mappings) != c.popped) ||
		(qExists && qTop != c.popped)) {
		// can pop
		return 'P'
	}
	// needs morphological disambiguation
	return 'M'
}

func (c *MDConfig) String() string {
	if c.Mappings == nil {
		return fmt.Sprintf("\t=>([],\t[]) - %v", c.Alignment())
	}
	mapLen := len(c.Mappings)
	transStr := "MD"
	if c.Last.Type() == 'L' {
		transStr = "LEX"
	}
	if c.Last.Equal(ConstTransition(0)) {
		transStr = ""
	}
	if c.State() == 'L' && transStr == "MD" {
		transStr = "MD*"
	}
	if c.Last.Equal(c.POP) || c.Last.Type() == 'P' {
		transStr = "POP"
	}
	lemmaStr := ""
	if c.State() == 'L' {
		currentLat, _ := c.LatticeQueue.Peek()
		latticeMorphemes := c.Lattices[currentLat].Morphemes
		lemmas := make([]string, len(c.Lemmas))
		for i, morphID := range c.Lemmas {
			morph := latticeMorphemes[morphID]
			lemmas[i] = morph.Lemma
		}
		lemmaStr = fmt.Sprintf("%v;%s", latticeMorphemes[c.Lemmas[0]].StringNoLemma(), strings.Join(lemmas, ","))
	}

	mapStr := ""
	if mapLen > 0 {
		lastMap := c.Mappings[mapLen-1]
		if len(lastMap.Spellout) > 0 {
			mapStr = lastMap.String()
		} else if mapLen > 1 {
			mapStr = c.Mappings[mapLen-2].String()
		}
	}
	return fmt.Sprintf("%s\t=>([%s],\t[%v],\t[%v]) - %v", transStr, c.StringLatticeQueue(), lemmaStr, mapStr, c.Alignment())
}

func (c *MDConfig) StringLatticeQueue() string {
	if c.LatticeQueue == nil {
		return ""
	}
	queueSize := c.LatticeQueue.Size()
	switch {
	case queueSize > 0 && queueSize <= 3:
		var queueStrings []string = make([]string, 0, 3)
		for i := 0; i < c.LatticeQueue.Size(); i++ {
			atI, _ := c.LatticeQueue.Index(i)
			queueStrings = append(queueStrings, fmt.Sprintf("%v - %v", string(c.Lattices[atI].Token), c.CurrentLatNode))
		}
		return strings.Join(queueStrings, ",")
	case queueSize > 3:
		headID, _ := c.LatticeQueue.Index(0)
		tailID, _ := c.LatticeQueue.Index(c.LatticeQueue.Size() - 1)
		head := c.Lattices[headID]
		tail := c.Lattices[tailID]
		return strings.Join([]string{string(head.Token), "...", string(tail.Token)}, ",")
	default:
		return ""
	}

}
func (c *MDConfig) Equal(otherEq util.Equaler) bool {
	// c.Log = true
	if (otherEq == nil && c != nil) || (c == nil && otherEq != nil) {
		if c.Log {
			log.Println("\tfalse default")
		}
		return false
	}
	// c.Log = true
	switch other := otherEq.(type) {
	case *MDConfig:
		if (other == nil && c != nil) || (c == nil && other != nil) {
			if c.Log {
				log.Println("\tfalse 0")
			}
			return false
		}
		if c.Log {
			log.Println("Comparing", c, "to", other)
			log.Println("Comparing\n", c.GetSequence(), "\n\tto\n", other.GetSequence())
		}
		if !other.Last.Equal(c.Last) {
			if c.Log {
				log.Println("\tfalse 1")
			}
			return false
		}
		if c.InternalPrevious == nil && other.InternalPrevious == nil {
			if c.Log {
				log.Println("\ttrue")
			}
			return true
		}
		if c.InternalPrevious != nil && other.InternalPrevious != nil {
			if c.Log {
				log.Println("\trecurse")
				c.InternalPrevious.(*MDConfig).Log = c.Log
			}
			return c.InternalPrevious.Equal(other.InternalPrevious)
		} else {
			if c.Log {
				log.Println("\tfalse 3: ", c.InternalPrevious, "vs", other.InternalPrevious)
			}
			return false
		}
	default:
		panic("TODO: Figure out what the type of the other is ([]*nlp.Mapping?)")
	}
}

func (c *MDConfig) Previous() Configuration {
	return c.InternalPrevious
}

func (c *MDConfig) SetPrevious(prev Configuration) {
	c.InternalPrevious = prev
}

func (c *MDConfig) Clear() {
	c.InternalPrevious = nil
}

func (c *MDConfig) AddSpellout(spellout string, paramFunc nlp.MDParam) bool {
	// log.Println("\tAdding spellout")
	if curLatticeId, exists := c.LatticeQueue.Pop(); exists {
		curLattice := c.Lattices[curLatticeId]
		if UsePOP && POP_ONLY_VAR_LEN {
			poppedLat := c.Lattices[curLatticeId]
			// only need to pop variable length
			if !poppedLat.IsVarLen() {
				c.Pop()
			}
		}
		// log.Println("\tAt Lattice", curLattice.Token)
		for _, s := range curLattice.Spellouts {
			if nlp.ProjectSpellout(s, paramFunc) == spellout {
				c.CurrentLatNode = curLattice.Top()
				c.Mappings = append(c.Mappings, &nlp.Mapping{Token: curLattice.Token, Spellout: s})
				// log.Println("\tPost mappings:", c.Mappings)
				return true
			}
		}
		return false
	}
	panic("No lattices left in queue")
}

func (c *MDConfig) AddLemmaAmbiguity(morphIDs []int) {
	c.Lemmas = morphIDs
	// lemmas := make([]string, len(morphIDs))
	// currentLat, _ := c.LatticeQueue.Peek()
	// latticeMorphemes := c.Lattices[currentLat].Morphemes
	// for i, morphID := range c.Lemmas {
	// 	morph := latticeMorphemes[morphID]
	// 	lemmas[i] = morph.Lemma
	// }
	// log.Println("Adding ambiguous lemmas", strings.Join(lemmas, "|"))
}

func (c *MDConfig) ChooseLemma(lemma string) {
	currentLat, exists := c.LatticeQueue.Peek()
	if !exists {
		panic("Can't choose lemma if no lattices are in the queue")
	}
	if c.Lemmas == nil || len(c.Lemmas) == 0 {
		panic("Can't disambiguate lemmas if no ambiguous lemmas exist")
	}
	latticeMorphemes := c.Lattices[currentLat].Morphemes
	lemmas := make([]string, len(c.Lemmas))
	for i, morphID := range c.Lemmas {
		morph := latticeMorphemes[morphID]
		if morph.Lemma == lemma {
			c.AddMapping(morph)
			c.Lemmas = nil
			return
		}
		lemmas[i] = morph.Lemma
	}
	panic(fmt.Sprintf("Lemma not found in ambiguous morphemes: (%v, %v)", lemma, strings.Join(lemmas, "|")))
}

func (c *MDConfig) AddMapping(m *nlp.EMorpheme) {
	// log.Println("\tAdding mapping to spellout")
	c.CurrentLatNode = m.To()

	currentLatIdx, _ := c.LatticeQueue.Peek()

	if len(c.Mappings) == 0 || len(c.Mappings) < currentLatIdx {
		// log.Println("\tAdding new mapping because", len(c.Mappings), currentLatIdx)
		c.Mappings = append(c.Mappings, &nlp.Mapping{Token: c.Lattices[currentLatIdx].Token, Spellout: make(nlp.Spellout, 0, 1)})
	}

	currentMap := c.Mappings[len(c.Mappings)-1]
	currentMap.Spellout = append(currentMap.Spellout, m)

	// log.Println("\tNode bumped to", c.CurrentLatNode, "of lattice", currentLatIdx)
	// debugLat := c.Lattices[currentLatIdx]
	// log.Println("\tCurrent lattice token bottom/top", debugLat.Token, debugLat.Bottom(), debugLat.Top())
	// if current lattice node is the last of current lattice
	// then pop lattice and make new mapping struct
	if currentLat := c.Lattices[currentLatIdx]; c.CurrentLatNode == currentLat.Top() {
		// log.Println("\tPopping lattice queue")
		poppedIndex, _ := c.LatticeQueue.Pop()
		if UsePOP && POP_ONLY_VAR_LEN {
			poppedLat := c.Lattices[poppedIndex]
			// only need to pop variable length
			if !poppedLat.IsVarLen() {
				c.Pop()
			}
		}
		val, exists := c.LatticeQueue.Peek()
		// log.Println("\tNow at lattice (exists)", val, exists)
		if exists {
			c.Mappings = append(c.Mappings, &nlp.Mapping{Token: c.Lattices[val].Token, Spellout: make(nlp.Spellout, 0, 1)})
			// log.Println("\tSetting token to", c.Lattices[val].Token)
		}
	}
	c.Morphemes = append(c.Morphemes, m)
}

func (c *MDConfig) Address(location []byte, sourceOffset int) (int, bool, bool) {
	source := c.GetSource(location[0])
	if source == nil {
		return 0, false, false
	}
	var (
		atAddress int
		exists    bool
	)
	// test if feature address is a generator of feature (e.g. for each child..)
	locationLen := len(location)
	if location[0] == 'L' && locationLen >= 4 {
		if string(location[2:4]) == "Ci" {
			return atAddress, true, true
		}
	}
	sourceOffsetInt := int(sourceOffset)
	// log.Println("\tUsing sourceOffset", sourceOffset, "computed as", sourceOffsetInt, "for", location)
	// hack for lattices to retrieve previously seen lattices
	if location[0] == 'L' && sourceOffsetInt < 0 {
		// assumes lattice indices are continuous in lattice queue
		atAddress, exists = source.Index(0)
		// log.Println("\tFound base", atAddress)
		atAddress = atAddress + sourceOffsetInt
		if exists {
			exists = atAddress >= 0
			// log.Println("\tExists:", exists)
		} else {
			// special override for POP features existing after last lattice
			// removed from queue
			if c.LatticeQueue.Size() == 0 {
				exists = true
				atAddress = len(c.Lattices) - 1
			}
		}
	} else {
		atAddress, exists = source.Index(sourceOffsetInt)
	}
	if !exists {
		return 0, false, false
	}

	if sourceOffsetInt < 0 {
		location = location[3:]
	} else {
		location = location[2:]
	}
	if len(location) == 0 {
		// log.Println("\tAddress Success")
		return atAddress, true, false
	}
	// log.Println("\tAddress Fail, location was", location)
	return 0, false, false
}

func (c *MDConfig) Attribute(source byte, nodeID int, attribute []byte, transitions []int) (att interface{}, exists bool, isGenerator bool) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in MDConfig::Attribute", r)
		}
	}()
	exists = true
	switch source {
	case 'M':
		morpheme := c.Morphemes[nodeID]
		switch attribute[0] {
		case 'm':
			if len(attribute) > 1 && attribute[1] == 'p' {
				att = morpheme.EFCPOS
				return
			} else {
				if SwitchFormLemma {
					att = morpheme.Lemma
				} else {
					att = morpheme.EForm
				}
				return
			}
		case 'p':
			att = morpheme.EPOS
			return
		case 'f':
			att = morpheme.EFeatures
			return
		case 't':
			lat := c.Lattices[morpheme.TokenID]
			// tokId, _ := c.ETokens.Add(lat.Token)
			// att = tokId
			att = lat.Token
			return
		case 'i': // path of lattice of last morpheme
			result := make([]string, 0, 5) // assume most lattice lengths are <= 5
			// log.Println("Generating idle feature starting with morpheme")
			// log.Println(" mappings are")
			// log.Println(c.Mappings)
			// log.Println(" morphemes are (current nodeID is:", nodeID, ")")
			// log.Println(c.Morphemes)
			// log.Println(morpheme)
			curTokenId := morpheme.TokenID
			// log.Println("Token is", curTokenId, c.Lattices[curTokenId-1].Token)
			for {
				// log.Println("Adding morph string", nlp.Funcs_Main_POS_Both_Prop(morpheme))
				result = append(result, nlp.Funcs_Main_POS_Both_Prop(morpheme))
				// get the next morpheme
				// break if reached end of morpheme stack or reached
				// next token (== lattice)
				nodeID--
				if nodeID < 0 {
					break
				}
				morpheme = c.Morphemes[nodeID]
				if morpheme.TokenID != curTokenId {
					break
				}
			}
			// log.Println("Idle feature", fmt.Sprintf("%v", result))
			att = fmt.Sprintf("%v", result)
			return
		}
	case 'L':
		if nodeID >= len(c.Lattices) {
			exists = false
			return
		}
		lat := c.Lattices[nodeID]
		// log.Println("At lattice", lat)
		switch attribute[0] {
		case 'c':
			if lat.Top() == c.CurrentLatNode {
				exists = false
				return
			}
			if len(attribute) < 2 {
				panic("(c)urrent morphemes attribute needs a sub-attribute")
			}
			curTransMap := make(map[int]bool, len(transitions))
			for _, val := range transitions {
				curTransMap[val] = true
			}
			if nextEdges, nextExists := lat.Next[c.CurrentLatNode]; nextExists {
				retval := &featurevector.SimpleTAF{FTMap: make(featurevector.FeatureTransMap, len(nextEdges))}
				var (
					feature    interface{}
					transition int
					curMap     map[int]bool
					mapExists  bool
				)
				for _, edgeId := range nextEdges {
					curEdge := lat.Morphemes[edgeId]
					switch string(attribute[1:]) {
					case "q": // generate token|feature per feature
						if lat.Top()-lat.Bottom() != 1 {
							exists = false
							return
						}
						for k, v := range curEdge.Features {
							f := fmt.Sprintf("%s|%s", k, v)
							transition, _ = c.Transitions.Add(c.ParamFunc(curEdge))
							if _, tExists := curTransMap[transition]; tExists {
								if curMap, mapExists = retval.FTMap[f]; !mapExists {
									curMap = make(map[int]bool, 4) // TODO: compute better constant?
								}
								curMap[transition] = true
								retval.FTMap[feature] = curMap
							}
						}
						continue
					case "mq": // generate token|feature per feature
						if lat.Top()-lat.Bottom() != 1 {
							exists = false
							return
						}
						for k, v := range curEdge.Features {
							f := fmt.Sprintf("%s-%s|%s", curEdge.Form, k, v)
							transition, _ = c.Transitions.Add(c.ParamFunc(curEdge))
							if _, tExists := curTransMap[transition]; tExists {
								if curMap, mapExists = retval.FTMap[f]; !mapExists {
									curMap = make(map[int]bool, 4) // TODO: compute better constant?
								}
								curMap[transition] = true
								retval.FTMap[feature] = curMap
							}
						}
						continue
					case "r":
						feature = c.ParamFunc(curEdge)
					case "mp":
						feature = curEdge.EFCPOS
					case "mp2":
						if len(lat.Morphemes) > 0 {
							lastMorph := lat.Morphemes[len(lat.Morphemes)-1]
							feature = [2]string{curEdge.Form, lastMorph.CPOS}
						} else {
							exists = false
						}
					case "m":
						feature = curEdge.EForm
					case "m2":
						if len(lat.Morphemes) > 0 {
							lastMorph := lat.Morphemes[len(lat.Morphemes)-1]
							feature = [2]string{curEdge.Form, lastMorph.Form}
						} else {
							exists = false
						}
					case "p":
						feature = curEdge.EPOS
					case "p2":
						if len(lat.Morphemes) > 0 {
							lastMorph := lat.Morphemes[len(lat.Morphemes)-1]
							feature = [2]string{curEdge.CPOS, lastMorph.CPOS}
						} else {
							exists = false
						}
					case "f":
						feature = curEdge.EFeatures
					case "g":
						feature = util.Signature(curEdge.Form)
					case "pg":
						feature = [2]string{curEdge.CPOS, util.Signature(curEdge.Form)}
					case "fg":
						feature = [2]string{curEdge.FeatureStr, util.Signature(curEdge.Form)}
					case "fp":
						feature = [2]string{curEdge.FeatureStr, curEdge.CPOS}
					case "fpg":
						feature = [3]string{curEdge.FeatureStr, curEdge.CPOS, util.Signature(curEdge.Form)}
					default:
						panic("Don't know what this feature is")
					}
					transition, _ = c.Transitions.Add(c.ParamFunc(curEdge))
					if _, tExists := curTransMap[transition]; tExists {
						if curMap, mapExists = retval.FTMap[feature]; !mapExists {
							curMap = make(map[int]bool, 4) // TODO: compute better constant?
						}
						curMap[transition] = true
						retval.FTMap[feature] = curMap
					}
				}
				att = retval
			} else {
				exists = false
			}
			return
		case 'a': // current lattice represented as all projected paths (spellouts)
			result := make([]string, len(lat.Spellouts))
			for i, s := range lat.Spellouts {
				result[i] = nlp.ProjectSpellout(s, nlp.Funcs_Main_POS_Both_Prop)
			}
			att = fmt.Sprintf("%v", result)
			return
		case 't': // token of last lattice
			// att, _ = c.ETokens.Add(lat.Token)
			att = lat.Token
			return
		case 'g': // signature
			att = lat.Signature()
			return
		case 'e': // prefix
			isGenerator = true
			att = lat.Prefixes(AFFIX_SIZE)
			return
		case 'x': // suffix
			isGenerator = true
			att = lat.Suffixes(AFFIX_SIZE)
			return
		case 'n': // next edges of current lattice node
			if nextEdges, nextExists := lat.Next[c.CurrentLatNode]; nextExists {
				retval := make([]string, 0, len(nextEdges))
				for _, edgeId := range nextEdges {
					curEdge := lat.Morphemes[edgeId]
					retval = append(retval, nlp.Funcs_Main_POS_Both_Prop(curEdge))
				}
				sort.StringSlice(retval).Sort()
				att = fmt.Sprintf("%v", retval)
				return
			}
		case 'i': // path of lattice
			// log.Println("Generating feature starting with morpheme")
			// log.Println(" mappings are")
			// log.Println(c.Mappings)
			// log.Println(" morphemes are (current nodeID is:", nodeID, ")")
			// log.Println(c.Morphemes)
			if nodeID >= 0 && nodeID < len(c.Mappings) {
				latMapping := c.Mappings[nodeID]
				result := make([]string, len(latMapping.Spellout)) // assume most lattice lengths are <= 5
				for i, morpheme := range latMapping.Spellout {
					// log.Println("Adding morph string", nlp.Funcs_Main_POS_Both_Prop(morpheme))
					result[i] = nlp.Funcs_Main_POS_Both_Prop(morpheme)
					// get the next morpheme
					// break if reached end of morpheme stack or reached
					// next token (== lattice)
				}
				att = fmt.Sprintf("%v", result)
				return
			}
		}
	}
	exists = false
	att = 0
	return
}

func (c *MDConfig) GenerateAddresses(nodeID int, location []byte) (nodeIDs []int) {
	return util.RangeInt(len(c.Lattices))
}

func (c *MDConfig) GetSource(location byte) Index {
	switch location {
	case 'M':
		return c.Morphemes
	case 'L':
		return c.LatticeQueue
	}
	return nil
}

func (c *MDConfig) Alignment() int {
	return c.popped
	// if c.popped == len(c.Mappings) && c.LatticeQueue.Size() > 0 {
	// if c.popped == len(c.Mappings) && c.LatticeQueue.Size() == 0 {
	// 	return 1
	// } else {
	// 	return 0
	// }
	// return len(c.Mappings)
}

func (c *MDConfig) Assignment() uint16 {
	return uint16(len(c.Mappings))
}

func (c *MDConfig) Len() int {
	if c == nil {
		return 0
	}
	if c.Previous() != nil {
		return 1 + c.Previous().Len()
	} else {
		return 1
	}
}

func (c *MDConfig) Pop() {
	c.popped += 1
}
