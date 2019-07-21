package types

import (
	"fmt"
	"log"
	"reflect"
	"sort"
	"strings"
	"github.com/OnlpLab/yap/alg"
	"github.com/OnlpLab/yap/alg/graph"
	"github.com/OnlpLab/yap/util"
)

type Morpheme struct {
	graph.BasicDirectedEdge
	Form       string
	Lemma      string
	CPOS       string
	POS        string
	Features   map[string]string
	TokenID    int
	FeatureStr string
}

type EMorpheme struct {
	Morpheme
	EForm, ELemma    int
	EFCPOS, EPOS     int
	EFeatures        int
	EMHost, EMSuffix int
}

var _ DepNode = &Morpheme{}
var _ DepNode = &EMorpheme{}

func NewRootMorpheme() *EMorpheme {
	return &EMorpheme{Morpheme: Morpheme{
		graph.BasicDirectedEdge{0, 0, 0},
		ROOT_TOKEN, ROOT_TOKEN, ROOT_TOKEN, ROOT_TOKEN,
		nil, 0, "",
	}}
}

func (m *Morpheme) ID() int {
	return m.BasicDirectedEdge.ID()
}

func (m *Morpheme) From() int {
	return m.BasicDirectedEdge.From()
}

func (m *Morpheme) To() int {
	return m.BasicDirectedEdge.To()
}

func (m *Morpheme) StringNoLemma() string {
	return fmt.Sprintf("%v-%v-%v-%s", m.Form, m.CPOS, m.POS, m.FeatureStr)
}

func (m *Morpheme) String() string {
	return fmt.Sprintf("%v-%v-%v-%v-%s-(%v->%v)", m.Form, m.Lemma, m.CPOS, m.POS, m.FeatureStr, m.From(), m.To())
}

func (m *Morpheme) Copy() *Morpheme {
	newMorph := new(Morpheme)
	*newMorph = *m
	newMorph.Features = make(map[string]string)
	for k, v := range m.Features {
		newMorph.Features[k] = v
	}
	return newMorph
}

func (m *Morpheme) EMorpheme() *EMorpheme {
	newMorph := new(Morpheme)
	*newMorph = *m
	newMorph.Features = make(map[string]string)
	for k, v := range m.Features {
		newMorph.Features[k] = v
	}
	return &EMorpheme{Morpheme: *newMorph}
}

func (m *Morpheme) Equal(otherEq util.Equaler) bool {
	other := otherEq.(*Morpheme)
	featEq := (len(m.Features) == 0 && len(other.Features) == 0) || reflect.DeepEqual(m.Features, other.Features)
	return m.Form == other.Form &&
		m.Lemma == other.Lemma &&
		m.CPOS == other.CPOS &&
		m.POS == other.POS &&
		featEq && m.FeatureStr == other.FeatureStr
}

func (m *EMorpheme) Equal(otherEq util.Equaler) bool {
	other := otherEq.(*EMorpheme)
	return (&m.Morpheme).Equal(&other.Morpheme)
	// return m.EForm == other.EForm &&
	// 	m.Lemma == other.Lemma &&
	// 	m.EPOS == other.EPOS &&
	// 	m.EFCPOS == other.EFCPOS &&
	// 	m.EFeatures == other.EFeatures
}

func (m *EMorpheme) Copy() *EMorpheme {
	newMorph := new(EMorpheme)
	*newMorph = *m
	newMorph.Features = make(map[string]string)
	for k, v := range m.Features {
		newMorph.Features[k] = v
	}
	return newMorph
}

var _ graph.DirectedEdge = &Morpheme{}
var _ graph.DirectedEdge = &EMorpheme{}

type Morphemes []*EMorpheme
type BasicMorphemes []*Morpheme

var _ alg.Index = make(Morphemes, 1)

func (m Morphemes) Len() int {
	return len(m)
}

func (m Morphemes) Less(i, j int) bool {
	return m[i].From() < m[j].From() ||
		(m[i].From() == m[j].From() && m[i].To() < m[j].To())
}

func (m Morphemes) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
	m[i].BasicDirectedEdge[0], m[j].BasicDirectedEdge[0] = m[j].BasicDirectedEdge[0], m[i].BasicDirectedEdge[0]
}

func (m Morphemes) Index(index int) (int, bool) {
	if index >= len(m) {
		return 0, false
	}
	return len(m) - 1 - index, true
}

func (m Morphemes) AsBasic() BasicMorphemes {
	retval := make([]*Morpheme, len(m))
	for i, val := range m {
		retval[i] = &val.Morpheme
	}
	return BasicMorphemes(retval)
}

func (m *Morphemes) Merge(new *EMorpheme) {
	for _, morph := range *m {
		if morph.Equal(new) && new.From() == morph.From() {
			return
		}
	}
	*m = append(*m, new)
}

func (m *BasicMorphemes) Union(others BasicMorphemes) {
	if len(others) != 1 {
		panic("Can't Union with another morpheme set with size other than 1")
	}
	other := others[0]
	for _, cur := range *m {
		if cur.Equal(other) {
			return
		}
	}
	other.BasicDirectedEdge[0] = len(*m)
	*m = append(*m, other)
}

func (m *BasicMorphemes) Equal(others BasicMorphemes) bool {
	if len(*m) != len(others) {
		return false
	}
	for i, cur := range *m {
		if !cur.Equal(others[i]) {
			return false
		}
	}
	return true
}

func (m Morphemes) Standalone() BasicMorphemes {
	// TODO: think of a better name - should mean 'retrieve the
	// raw morphemes, as if they appear by themselves'
	morphs := make(BasicMorphemes, len(m))
	for i, curMorph := range m {
		newMorph := new(Morpheme)
		*newMorph = curMorph.Morpheme
		newMorph.BasicDirectedEdge = [3]int{i, i, i + 1}
		newMorph.TokenID = 0
		morphs[i] = newMorph
	}
	return morphs
}

type Spellout Morphemes

func (s Spellout) EqualCompare(other Spellout, paramFuncName string) bool {
	_, TN, FP, _ := s.Compare(other, paramFuncName)
	return TN == 0 && FP == 0
}
func (s Spellout) Compare(other Spellout, paramFuncName string) (TP, TN, FP, FN int) {
	// log.Println("Comparing", s.AsString(), other.AsString())
	// if s.Equal(other) {
	// 	log.Println("Are Equal")
	// }
	paramFunc, exists := MDParams[paramFuncName]
	if !exists {
		panic("Unsupported parameter function: " + paramFuncName)
	}
	curMorphs, otherMorphs := make(map[string]bool, len(s)), make(map[string]bool, len(other))
	for _, m := range s {
		curMorphs[paramFunc(m)] = true
	}
	for _, m := range other {
		otherMorphs[paramFunc(m)] = true
	}
	for k := range curMorphs {
		if _, exists := otherMorphs[k]; exists {
			TP += 1
		} else {
			FP += 1
		}
	}
	for k := range otherMorphs {
		if _, exists := curMorphs[k]; !exists {
			TN += 1
		}
	}
	// log.Println("Results", TP, TN, FP, FN)
	return
}

type Spellouts []Spellout

func (s Spellouts) Len() int {
	return len(s)
}

func (s Spellouts) Less(i, j int) bool {
	return s[i].AsString() < s[j].AsString()
}

func (s Spellouts) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Adds a new spellout only if it doesn't already exist
func (s *Spellouts) Merge(new Spellout, startNode int) {
	for _, cur := range *s {
		if cur.Equal(new) {
			return
		}
	}
	newSpellout := make(Spellout, len(new))
	// log.Println("Merging", new, "to", s)
	for i, cur := range new {
		newMorph := cur.Copy()
		newMorph.BasicDirectedEdge[1] = startNode + i
		newMorph.BasicDirectedEdge[2] = startNode + i + 1
		newSpellout[i] = newMorph
	}
	*s = append(*s, newSpellout)
}

func (s Spellouts) Intersect(other Spellouts, paramFunc string, startNode int) Spellouts {
	retval := make(Spellouts, 0, util.Min(len(s), len(other)))
outerLoop:
	for _, spelloutA := range s {
		for _, spelloutB := range other {
			if spelloutA.EqualCompare(spelloutB, paramFunc) {
				retval.Merge(spelloutA, startNode)
				continue outerLoop
			}
		}
	}
	return retval
}

func (s Spellouts) UniqueMorphemes() Morphemes {
	m := make(Morphemes, 0, len(s))
	for _, spellout := range s {
		for _, morph := range spellout {
			m.Merge(morph)
		}
	}
	return m
}

func (s Spellout) String() string {
	posStrings := make([]string, len(s))
	for i, morph := range s {
		posStrings[i] = morph.String()
	}
	return strings.Join(posStrings, ":")
}

func (s Spellout) AsString() string {
	strs := make([]string, len(s))
	for i, morph := range s {
		strs[i] = morph.String()
	}
	return strings.Join(strs, ";")
}

func (s Spellout) Equal(other Spellout) bool {
	if len(s) != len(other) {
		return false
	}
	for i, val := range other {
		// if Lemma_POS_Prop(s[i]) != Lemma_POS_Prop(val) {
		// 	return false
		// }
		if !s[i].Equal(val) {
			return false
		}
	}
	return true
}

func (s Spellouts) Find(other Spellout) (int, bool) {
	for i, cur := range s {
		if cur.Equal(other) {
			return i, true
		}
	}
	return 0, false
}

type Mapping struct {
	Token    Token
	Spellout Spellout
}

func (m *Mapping) Equal(other *Mapping) bool {
	return m.Token == other.Token && m.Spellout.Equal(other.Spellout)
}

func (m *Mapping) String() string {
	if len(m.Spellout) > 0 {
		return fmt.Sprintf("%v|%v-%v-%v", m.Token, m.Spellout[len(m.Spellout)-1].Form, m.Spellout[len(m.Spellout)-1].POS, m.Spellout[len(m.Spellout)-1].FeatureStr)
	} else {
		return string(m.Token)
	}
}

type Mappings []*Mapping

var _ alg.Index = make(Mappings, 1)

func (ms Mappings) Equal(otherEq util.Equaler) bool {
	other, ok := otherEq.(Mappings)
	if !ok {
		return false
	}
	if len(ms) != len(other) {
		return false
	}
	for i, m := range ms {
		if !m.Equal(other[i]) {
			return false
		}
	}
	return true
}

func (ms Mappings) Index(i int) (int, bool) {
	if i >= len(ms) {
		return 0, false
	}
	return i, true
}

type Path int

type Lattice struct {
	Token           Token
	Morphemes       Morphemes
	Spellouts       Spellouts
	Next            map[int][]int
	BottomId, TopId int
}

func (l *Lattice) Signature() string {
	return l.Token.Signature()
}

func (l *Lattice) Prefixes(n int) []interface{} {
	return l.Token.Prefixes(n)
}

func (l *Lattice) Suffixes(n int) []interface{} {
	return l.Token.Suffixes(n)
}

func (l *Lattice) MorphsOnGold(other *Lattice) Morphemes {
	retval := make(Morphemes, 0, len(l.Morphemes))
	// curGoldId := other.BottomId
	return retval
}

func (l *Lattice) IsVarLen() bool {
	if l.Spellouts == nil {
		l.GenSpellouts()
	}
	if len(l.Spellouts) <= 1 {
		return false
	}
	len1 := len(l.Spellouts[0])
	for _, spellout2 := range l.Spellouts[1:] {
		if len(spellout2) != len1 {
			return true
		}
	}
	return false
}
func (l *Lattice) Add(morphs BasicMorphemes, start, end, numToken int) {
	nextNode := start
	for i, morph := range morphs {
		newMorph := morph.EMorpheme()
		newMorph.TokenID = numToken
		newMorph.BasicDirectedEdge[0] = len(l.Morphemes)
		if logAddAnalysis {
			log.Println("\t\t\t\t\tSetting first node", nextNode)
		}
		newMorph.BasicDirectedEdge[1] = nextNode
		if i < len(morphs)-1 {
			if logAddAnalysis {
				log.Println("\t\t\t\t\tSearch for outgoing node")
			}
			exists := true
			for exists {
				if logAddAnalysis {
					log.Println("\t\t\t\t\t\tFound outgoing node", nextNode)
					log.Println("\t\t\t\t\t\tIn", l.Next)
				}
				nextNode++
				_, exists = l.Next[nextNode]
			}
			if logAddAnalysis {
				log.Println("\t\t\t\t\tSetting outgoing node", nextNode)
			}
			newMorph.BasicDirectedEdge[2] = nextNode
		} else {
			if logAddAnalysis {
				log.Println("\t\t\t\t\tSetting last node", end)
			}
			newMorph.BasicDirectedEdge[2] = end
		}
		if logAddAnalysis {
			log.Println("\t\t\t\tadding morph", i, morph, "at nodes", newMorph.From(), newMorph.To())
		}
		if _, exists := l.Next[newMorph.From()]; exists {
			if logAddAnalysis {
				log.Println("\t\t\t\tappending morph ID", newMorph.ID(), "to", l.Next[newMorph.From()])
			}
			l.Next[newMorph.From()] = append(l.Next[newMorph.From()], newMorph.ID())
		} else {
			l.Next[newMorph.From()] = []int{newMorph.ID()}
			if logAddAnalysis {
				log.Println("\t\t\t\tcreating new morph next list for", newMorph.ID(), "at", newMorph.From(), ":", l.Next[newMorph.From()])
			}
		}
		if _, exists := newMorph.Features["cliticized"]; exists {
			log.Println("adding morpheme with clitic for token", numToken)
		}
		l.Morphemes = append(l.Morphemes, newMorph)
	}
}

func (l *Lattice) BumpAll(diff int) {
	newNext := make(map[int][]int, len(l.Next))
	for start, nextList := range l.Next {
		newNext[start+diff] = nextList
	}
	for _, morph := range l.Morphemes {
		morph.BasicDirectedEdge[1] += diff
		morph.BasicDirectedEdge[2] += diff
	}
	l.TopId += diff
	l.BottomId += diff
	l.Next = newNext
}

func (l *Lattice) BumpTop(from, to int, upTo int) {
	for _, morph := range l.Morphemes {
		if morph.ID() < upTo && morph.To() == from {
			// log.Println("\t\t\t\t\tBumping top for", morph)
			morph.BasicDirectedEdge[2] = to
		}
	}
	l.TopId = to
}

var logAddAnalysis bool = false

func (l *Lattice) AddAnalysis(prefix BasicMorphemes, hosts []BasicMorphemes, numToken int) {
	if logAddAnalysis {
		log.Println("\t\t\tAdding prefix", prefix, "hosts", hosts)
	}
	startNode := l.BottomId
	oldestId := len(l.Morphemes)
	if prefix != nil {
		maxSameMorphNode := l.BottomId
		lastMatchingMorph := -1
		for i, m := range prefix {
			if logAddAnalysis {
				log.Println("\t\t\t\tSearching for morpheme", m)
			}
			edges, _ := l.Next[maxSameMorphNode]
			for _, edgeId := range edges {
				edge := l.Morphemes[edgeId]
				if edge.Morpheme.Equal(m) {
					maxSameMorphNode = edge.To()
					lastMatchingMorph = i
					break
				}
			}
			if lastMatchingMorph < i {
				break
			}
		}
		if lastMatchingMorph < len(prefix)-1 {
			prefixTail := prefix[lastMatchingMorph+1:]
			if logAddAnalysis {
				log.Println("\t\t\tAdding rest of prefix:", prefixTail)
			}
			addTopOffset := 0
			if val, exists := l.Next[maxSameMorphNode]; exists && len(val) > 0 {
				if logAddAnalysis {
					log.Println("\t\t\tmaxSameMorphNode", maxSameMorphNode, "exists")
				}
				addTopOffset = -1
			}
			endOfPrefix := l.TopId + len(prefixTail) + addTopOffset
			l.Add(prefixTail, maxSameMorphNode, endOfPrefix, numToken)
			if logAddAnalysis {
				log.Println("\t\t\tBump Top to:", l.TopId+len(prefixTail))
			}
			l.BumpTop(l.TopId, l.TopId+len(prefixTail), oldestId)
			if logAddAnalysis {
				log.Println("\t\t\tSetting maxSameMorphNode:", endOfPrefix)
			}
			maxSameMorphNode = endOfPrefix
		}
		startNode = maxSameMorphNode
	}
	if logAddAnalysis {
		log.Println("\t\tadding host")
	}
	for _, host := range hosts {
		maxSameMorphNode := startNode
		lastMatchingMorph := -1
		for i, m := range host {
			if logAddAnalysis {
				log.Println("\t\t\t\tSearching for morpheme", m)
			}
			edges, _ := l.Next[maxSameMorphNode]
			for _, edgeId := range edges {
				edge := l.Morphemes[edgeId]
				if edge.Morpheme.Equal(m) {
					if logAddAnalysis {
						log.Println("\t\t\t\t\tFound", edge)
					}
					if edge.To() == l.Top() && i >= len(host) {
						if logAddAnalysis {
							log.Println("\t\t\t\t\t.. but it ends with the top of the lattice")
							log.Println("\t\t\t\t\t.. ending matching sequences with previous matching edge")
							log.Println("\t\t\t\t\t.. with more morphemes left in the host")
						}
						break
					}
					maxSameMorphNode = edge.To()
					lastMatchingMorph = i
					break
				}
			}
			if lastMatchingMorph < i {
				if logAddAnalysis {
					log.Println("\t\t\t\t\tCurrent edge not matching morph, breaking")
				}
				break
			} else {
				if logAddAnalysis {
					log.Println("\t\t\t\t\tCurrent edge matches morph, continue")
				}
			}
		}
		host = host[lastMatchingMorph+1:]
		if len(host) == 0 {
			if logAddAnalysis {
				log.Println("\t\t\t\tNo edges left in host, continue")
			}
			continue
		}
		newTop := 0
		if len(host) > 1 {
			newTop = l.Top() + len(host) - util.Sign(l.Top()-maxSameMorphNode)
			if logAddAnalysis {
				log.Println("\t\t\t\tnewTop:", newTop, "=", l.Top(), "+", len(host), "- sign(", l.Top(), "-", maxSameMorphNode, ")")
			}
		} else {
			newTop = util.Max(1+maxSameMorphNode, l.Top())
		}
		if logAddAnalysis {
			log.Println("\t\t\tAdding Host:", host, "at", maxSameMorphNode, "to", newTop)
		}
		l.Add(host, maxSameMorphNode, newTop, numToken)

		if newTop >= l.TopId {
			if logAddAnalysis {
				log.Println("\t\t\tBump Top to:", newTop, "from:", l.TopId)
			}
			l.BumpTop(l.TopId, newTop, oldestId)
		}
		oldestId = len(l.Morphemes)
	}
	// optionally regenerate spellout
}

func (l *Lattice) Optimize() {
	// removed := make(map[int]bool, len(l.Next))
	// for node, out := range l.Next {
	// 	if _, exists := removed[node]; !exists {
	// 		toRemove := make(map[int]int, len(out))
	// 		for i, outId1 := range out[:len(out)-2] {
	// 			if _, id1Exists := toRemove[outId1]; !id1Exists {
	// 				for _, outId2 := range out[i+1:] {
	// 					if _, id2Exists := toRemove[outId2]; !id2Exists {
	// 						m1, m2 := l.Morphemes[outId1], l.Morphemes[outId2]
	// 						if m1.Equal(m2) {
	// 							toRemove[outId2] = outId1
	// 						}
	// 					}
	// 				}
	// 			}
	// 		}
	// 	}
	// }
}

func (l *Lattice) BridgeMissingMorphemes() {
	for _, m := range l.Morphemes {
		if _, exists := l.Next[m.To()]; !exists && m.To() < l.TopId {
			if _, nextExists := l.Next[m.To()+1]; nextExists {
				// log.Println("Bridging morpheme", m.Form, "from", m.To(), "to", m.To()+1)
				m.BasicDirectedEdge[2] += 1
			} else {
				// log.Println("Morpheme's next does not exist and cannot bridge! (", m.Form, m.From(), m.To(), ")")
			}
		}
	}
}

func (l *Lattice) UnionPath(other *Lattice) {
	// assume other is a "gold" path (only one "next" at each node)
	// add gold lattice path if it is an alternative to existing paths with the
	// same nodes
	// TODO: verify other is a single path (add IsPath to lattice)
	// if !other.IsPath() {
	//	panic(...)
	// }
	formMorphs := make(map[string][]*EMorpheme)
	for _, predMorph := range l.Morphemes {
		// log.Println("Found morpheme", predMorph, "at", predMorph.From(), predMorph.To())
		if cur, exists := formMorphs[predMorph.Form]; exists {
			formMorphs[predMorph.Form] = append(cur, predMorph)
		} else {
			formMorphs[predMorph.Form] = []*EMorpheme{predMorph}
		}
	}
	var found, missingMorpheme bool

	for _, goldMorph := range other.Morphemes {
		// log.Println("At morph", goldMorph)
		if curMorphs, exists := formMorphs[goldMorph.Form]; exists {
			for _, curMorph := range curMorphs {
				// log.Println("\tComparing to morph", curMorph)
				if curMorph.Equal(goldMorph) {
					found = true
				}
			}
		} else {
			// log.Println("Warning: gold morph form", goldMorph.Form, "is not in pred lattice!")
			missingMorpheme = true
			continue
		}
		if !found {
			// log.Println("Getting example for", goldMorph.Form)
			exampleMorphs, _ := formMorphs[goldMorph.Form]
			// log.Println("Examples", exampleMorphs)
			exampleFromTos := make(map[string][]int)
			for _, example := range exampleMorphs {
				exFromToStr := fmt.Sprintf("%v-%v", example.From(), example.To())
				if _, exists := exampleFromTos[exFromToStr]; !exists {
					// log.Println("Found new pair of from to for example", exFromToStr)
					exampleFromTos[exFromToStr] = []int{example.From(), example.To()}
				}
			}
			for _, exampleFromTo := range exampleFromTos {
				// log.Println("Found example pair at", exampleFromTo[0], exampleFromTo[1])
				// log.Println("Adding missing morpheme (form with same POS/properties did not exist)", goldMorph.Form, goldMorph.POS, goldMorph.CPOS, goldMorph.FeatureStr)
				l.InfuseMorph(goldMorph, exampleFromTo[0], exampleFromTo[1], true)
			}
		}
		found = false
	}
	// a gold form was not found in the pred lattice, try to find a trivial attachment
	// fast forward equivalent epilogue morphemes, try to equate missing morpheme to
	// the fusion of the rest of the pred lattice and attach
	if missingMorpheme {
		// the gold lattice should have only one spellout
		var (
			prevPredNodeId    int = -1
			currentPredNodeId int
			currentNode       *EMorpheme
			currentNodes      []int
			nextExists        bool
		)
		currentPredNodeId = l.Bottom()
	GoldLoop:
		for i, goldMorph := range other.Spellouts[0] {
			currentNodes, nextExists = l.Next[currentPredNodeId]
			if !nextExists {
				panic("Lost in pred lattice")
			}
			for _, currentNodeId := range currentNodes {
				if currentNode = l.Morphemes[currentNodeId]; currentNode.Equal(goldMorph) {
					// gold morpheme was found, move on to the next gold morpheme
					prevPredNodeId = currentPredNodeId
					currentPredNodeId = currentNode.BasicDirectedEdge[2]
					// log.Println("Found morpheme at", goldMorph.Form)
					continue GoldLoop
				}
			}
			// log.Println("Failed to find morpheme at", goldMorph.Form)
			// if the previous inner loop did not "continue" the goldloop
			// we found the location of the missing gold morpheme
			// we try to match with the fused morpheme from this point on
			for _, fusedCandidate := range l.AllFusedFrom(currentPredNodeId) {
				if fusedCandidate == goldMorph.Form {
					// if successful, we set the start node to the current node and the end node to the
					// top of the lattice
					// log.Println("Adding missing morpheme (form did not exist)", goldMorph.Form, goldMorph.POS, goldMorph.CPOS, goldMorph.FeatureStr, ";", currentPredNodeId)
					l.InfuseMorph(goldMorph, currentPredNodeId, l.Top(), true)
					break GoldLoop
				}
			}
			log.Println("Failed to find at current morpheme, trying previous", goldMorph.Form)
			// failed to fuse from current node, try to backtrack
			// maybe previous node will succeed
			if prevPredNodeId > -1 {
				for _, fusedCandidate := range l.AllFusedFrom(prevPredNodeId) {
					if fusedCandidate == other.Spellouts[0][i-1].Form {
						// log.Println("Adding missing morpheme (form did not exist); at", goldMorph.Form, goldMorph.POS, goldMorph.CPOS, goldMorph.FeatureStr, ";", currentPredNodeId)
						l.InfuseMorph(goldMorph, currentPredNodeId, l.Top(), true)
					}
				}
			}
		}
		if len(formMorphs) == 1 {
			for formMorph, edges := range formMorphs {
				// will happen exactly once
				// test to see if pred is a concatenation of gold morphs
				// if yes, and a morpheme is missing, assume morphological
				// analysis failure and complete gold path is missing
				goldMorphs := make([]string, len(other.Morphemes))
				for i, goldMorph := range other.Morphemes {
					goldMorphs[i] = goldMorph.Form
				}
				goldConcat := strings.Join(goldMorphs, "")
				if formMorph == goldConcat {
					log.Println("Found morphological analysis failure, adding gold path", strings.Join(goldMorphs, "-"))
					nextFromLatNode := edges[0].From()
					nextToLatNode := edges[0].To() + 1
					for i, goldMorph := range other.Morphemes {
						// log.Println("Need to fuse", goldMorph, "maybe at", nextFromLatNode, nextToLatNode)
						if i == len(other.Morphemes)-1 {
							nextToLatNode = edges[0].To()
							// log.Println("Updating", nextToLatNode)
						}
						l.InfuseMorph(goldMorph, nextFromLatNode, nextToLatNode, false)
						nextFromLatNode = nextToLatNode
						nextToLatNode++
					}
					l.Spellouts = nil
					l.GenSpellouts()
				}
			}
		}
	}
}

func (l *Lattice) InfuseMorph(morph *EMorpheme, from, to int, genSpellout bool) {
	// log.Println("Infusing", morph, "at", from, to)
	newMorph := morph.Copy()
	newMorph.Morpheme.BasicDirectedEdge[1] = from
	newMorph.Morpheme.BasicDirectedEdge[2] = to
	id := len(l.Morphemes)
	newMorph.Morpheme.BasicDirectedEdge[0] = id
	l.Morphemes = append(l.Morphemes, newMorph)
	mList, _ := l.Next[newMorph.From()]
	l.Next[newMorph.From()] = append(mList, id)

	if genSpellout {
		l.Spellouts = nil
		l.GenSpellouts()
	}
}

func NewRootLattice() Lattice {
	morphs := make(Morphemes, 1)
	morphs[0] = NewRootMorpheme()
	lat := &Lattice{
		ROOT_TOKEN,
		morphs,
		nil,
		make(map[int][]int),
		0,
		0,
	}
	return *lat
}

type LatticeSentence []Lattice

var _ Sentence = LatticeSentence{}

func (ls LatticeSentence) TaggedSentence() TaggedSentence {
	// assume morphs ~= 2x tokens (or say ~80%)
	var res BasicETaggedSentence = make([]EnumTaggedToken, 0, len(ls)*2)
	for _, lat := range ls {
		lat.GenSpellouts()
		if len(lat.Spellouts) == 0 {
			log.Fatalln("Lattice has no spellout, check input")
		}
		if len(lat.Spellouts) > 1 {
			log.Fatalln("Lattice is not morphologically unambiguous")
		}
		sp := lat.Spellouts[0]
		for _, m := range sp {
			res = append(res, EnumTaggedToken{
				TaggedToken{m.Form, m.Lemma, m.POS},
				m.EForm, 0, m.EPOS, m.EFCPOS, m.EMHost, m.EMSuffix})
		}
	}
	return res
}

func (ls LatticeSentence) Tokens() []string {
	res := make([]string, len(ls))
	for i, val := range ls {
		res[i] = string(val.Token)
	}
	return res
}

func (ls LatticeSentence) Equal(otherEq util.Equaler) bool {
	otherSent := otherEq.(Sentence)
	if len(otherSent.Tokens()) != len(ls) {
		return false
	}
	otherToks := otherSent.Tokens()
	curToks := ls.Tokens()
	return reflect.DeepEqual(curToks, otherToks)
}

type UpdateMorph struct {
	Type          string
	Tok, From, To int
}

func (ls LatticeSentence) FindFixDisHost(gold Mappings, pf MDParam) []*UpdateMorph {
	retval := make([]*UpdateMorph, 0, len(ls))
	var (
		projections map[string]map[string][]int
		projection  string
		morph       *EMorpheme
		mapping     *Mapping
	)
	for token, l := range ls {
		// log.Println("At token", token, l.Token)
		mapping = gold[token]
		curNode := l.Bottom()
		// first go through gold
		for goldNum, goldMorph := range mapping.Spellout {
			// log.Println("\tAt node", curNode, "goldMorph", goldMorph, "pf:", pf(goldMorph))
			nextMorphs := l.Next[curNode]
			// projections are a map of param function projections to lemmas
			// where lemmas is a map of lemma strings to a list of morphemes
			projections = make(map[string]map[string][]int, len(nextMorphs))
			// gather lemmas by projection
			for _, m := range nextMorphs {
				morph = l.Morphemes[m]
				projection = pf(morph)
				// log.Println("\t\tGot lattice morph", morph, "pf:", projection)
				if curLemmas, exists := projections[projection]; exists {
					// log.Println("\t\t\tprojection exists")
					if morphs, exists := curLemmas[morph.Lemma]; exists {
						// log.Println("\t\t\t\tadding to existing lemma", morph.Lemma)
						curLemmas[morph.Lemma] = append(morphs, m)
					} else {
						// log.Println("\t\t\t\tadding new lemma", morph.Lemma)
						curLemmas[morph.Lemma] = []int{m}
					}
				} else {
					// log.Println("\t\t\tadding new projection")
					newLemmas := make(map[string][]int, 3)
					newLemmas[morph.Lemma] = []int{m}
					projections[projection] = newLemmas
				}
			}
			if lemmas, exists := projections[pf(goldMorph)]; exists {
				// log.Println("\tfound existing lemmas for gold")
				if len(lemmas) > 1 {
					// log.Println("\t\tfound", len(lemmas), "ambiguous lemmas; >1")
					// found ambiguous lemmas for gold morpheme
					nextNodes := map[int]bool{}
					lastNextNode := 0
					for _, morphemes := range lemmas {
						// log.Println("\t\t\tat lemma", lemma)
						for _, m := range morphemes {
							morph = l.Morphemes[m]
							// log.Println("\t\t\t\tadding morpheme", morph)
							nextNodes[morph.To()] = true
							lastNextNode = morph.To()
						}
					}
					if len(nextNodes) == 1 {
						// log.Println("\t\t\t\tcontinuing to next")
						// we can continue, there is only one "next" node for
						// the gold path
						curNode = lastNextNode
					} else {
						// log.Println("\t\t\t\tcan't continue, breaking")
						// we don't know what the next node is, we can't follow
						// the gold path for this token
						break
					}
				} else if len(lemmas) == 1 {
					// log.Println("\t\tfound 1 lemma for morpheme")
					for _, morphs := range lemmas {
						// there is only one lemma
						curNode = l.Morphemes[morphs[0]].To()
						if len(morphs) == 1 {
							// log.Println("\t\t\t\tadvancing to next token")
							// we can continue, there is only one "next" node for
							// the gold path
							// only one morpheme
						} else {
							log.Println("Multiple morphemes for the same lemma at token", token, "advancing to next token")
							if len(morphs) == 2 {
								// there are two morphs, check if one of them is
								// ends the lattice while the other has a
								// postfix
								morph1, morph2 := l.Morphemes[morphs[0]], l.Morphemes[morphs[1]]
								if morph2.To() == l.Top() && morph1.To() != l.Top() {
									morph2, morph1 = morph1, morph2
								}
								// canonical case
								if morph1.To() == l.Top() && morph2.To() != l.Top() && goldNum < len(mapping.Spellout)-1 {
									log.Println("Found update case for gold at token", token)
									newGoldUpdate := &UpdateMorph{
										"Gold", token, goldMorph.From(), goldMorph.To(),
									}
									retval = append(retval, newGoldUpdate)
								}
							}
							break
						}
					}
				} else {
					log.Println("No lemmas at all at token", token)
				}
			}
		}
		// now go through all lattice nodes
		for i := l.Bottom(); i < l.Top(); i++ {
			nextMorphs := l.Next[i]
			// gather morphemes incl. lemma
			for _, m := range nextMorphs {
				morph = l.Morphemes[m]
				projection = Funcs_All_WLemma(morph)
				// log.Println("\t\tGot lattice morph", morph, "pf:", projection)
				if curProjs, exists := projections[projection]; exists {
					// log.Println("\t\t\tprojection exists")
					if morphs, exists := curProjs[morph.Lemma]; exists {
						// log.Println("\t\t\t\tadding to existing lemma", morph.Lemma)
						curProjs[morph.Lemma] = append(morphs, m)
					} else {
						// log.Println("\t\t\t\tadding new lemma", morph.Lemma)
						curProjs[morph.Lemma] = []int{m}
					}
				} else {
					// log.Println("\t\t\tadding new projection")
					newLemmas := make(map[string][]int, 3)
					newLemmas[morph.Lemma] = []int{m}
					projections[projection] = newLemmas
				}
			}
		}
	}
	return retval
}

type AmbLemma struct {
	Token    int
	Lemmas   []string
	PrevGold []string
}
type AmbMorphs []*AmbLemma

func (ls LatticeSentence) FindGoldAmbMorphs(gold Mappings, pf MDParam) AmbMorphs {
	retval := make(AmbMorphs, 0, len(ls))
	var (
		projections map[string]map[string][]int
		projection  string
		morph       *EMorpheme
		mapping     *Mapping
		ambMorphCur *AmbLemma
		prevGold    []string
	)
	for token, l := range ls {
		// log.Println("At token", token, l.Token)
		mapping = gold[token]
		curNode := l.Bottom()
		prevGold = make([]string, 0, len(mapping.Spellout))
		// iterate through the gold morphemes of the current token's lattice
		// foreach node (while possible)
		// 1. gather paramfunc's projections and their lemmas
		// 2. check lemmas of gold projection
		// 2.1 if more than one lemmas, gather and return
		// 2.2 else continue
		for _, goldMorph := range mapping.Spellout {
			prevGold = append(prevGold, goldMorph.String())
			// log.Println("\tAt node", curNode, "goldMorph", goldMorph, "pf:", pf(goldMorph))
			nextMorphs := l.Next[curNode]
			// projections are a map of param function projections to lemmas
			// where lemmas is a map of lemma strings to a list of morphemes
			projections = make(map[string]map[string][]int, len(nextMorphs))
			// gather lemmas by projection
			for _, m := range nextMorphs {
				morph = l.Morphemes[m]
				projection = pf(morph)
				// log.Println("\t\tGot lattice morph", morph, "pf:", projection)
				if curLemmas, exists := projections[projection]; exists {
					// log.Println("\t\t\tprojection exists")
					if morphs, exists := curLemmas[morph.Lemma]; exists {
						// log.Println("\t\t\t\tadding to existing lemma", morph.Lemma)
						curLemmas[morph.Lemma] = append(morphs, m)
					} else {
						// log.Println("\t\t\t\tadding new lemma", morph.Lemma)
						curLemmas[morph.Lemma] = []int{m}
					}
				} else {
					// log.Println("\t\t\tadding new projection")
					newLemmas := make(map[string][]int, 3)
					newLemmas[morph.Lemma] = []int{m}
					projections[projection] = newLemmas
				}
			}
			if lemmas, exists := projections[pf(goldMorph)]; exists {
				// log.Println("\tfound existing lemmas for gold")
				if len(lemmas) > 1 {
					// log.Println("\t\tfound", len(lemmas), "ambiguous lemmas; >1")
					// found ambiguous lemmas for gold morpheme
					ambMorphCur = &AmbLemma{token, make([]string, 0, len(lemmas)), prevGold[:]}
					nextNodes := map[int]bool{}
					lastNextNode := 0
					for lemma, morphemes := range lemmas {
						// log.Println("\t\t\tat lemma", lemma)
						for _, m := range morphemes {
							morph = l.Morphemes[m]
							// log.Println("\t\t\t\tadding morpheme", morph)
							nextNodes[morph.To()] = true
							lastNextNode = morph.To()
						}
						ambMorphCur.Lemmas = append(ambMorphCur.Lemmas, lemma)
					}
					retval = append(retval, ambMorphCur)
					if len(nextNodes) == 1 {
						// log.Println("\t\t\t\tcontinuing to next")
						// we can continue, there is only one "next" node for
						// the gold path
						curNode = lastNextNode
					} else {
						// log.Println("\t\t\t\tcan't continue, breaking")
						// we don't know what the next node is, we can't follow
						// the gold path for this token
						break
					}
				} else if len(lemmas) == 1 {
					// log.Println("\t\tfound 1 lemma for morpheme")
					for _, morphs := range lemmas {
						// there is only one lemma
						curNode = l.Morphemes[morphs[0]].To()
						if len(morphs) == 1 {
							// log.Println("\t\t\t\tadvancing to next token")
							// we can continue, there is only one "next" node for
							// the gold path
							// only one morpheme
						} else {
							log.Println("Multiple morphemes for the same lemma at token", token, "advancing to next token")
							break
						}
					}
				} else {
					log.Println("No lemmas at all at token", token)
				}
			}
		}
	}
	return retval
}

type DisAmbLemma struct {
	Token     int
	GoldMorph int
	CPOS      string
	Lemma     string
}
type DisAmbMorphs []*DisAmbLemma

func (ls LatticeSentence) FindGoldDisAmbMorphs(gold Mappings, mdParam MDParam) DisAmbMorphs {
	retval := make(DisAmbMorphs, 0, len(ls))
	var (
		projections map[string]map[string][]int
		projection  string
		morph       *EMorpheme
		mapping     *Mapping
		prevGold    []string
	)
	failoverPFs := []MDParam{mdParam, Funcs_Main_POS, POS_Prop, POS, Form}
	for token, l := range ls {
		// log.Println("At token", token, l.Token)
		mapping = gold[token]
		curNode := l.Bottom()
		prevGold = make([]string, 0, len(mapping.Spellout))
		// iterate through the gold morphemes of the current token's lattice
		// foreach node (while possible)
		// 1. gather paramfunc's projections and their lemmas
		// 2. check lemmas of gold projection
		// 2.1 if more than one lemmas, gather and return
		// 2.2 else continue
	goldMorphs:
		for goldToken, goldMorph := range mapping.Spellout {
			for _, pf := range failoverPFs {
				prevGold = append(prevGold, goldMorph.String())
				// log.Println("\tAt node", curNode, "goldMorph", goldMorph, "pf:", pf(goldMorph))
				nextMorphs := l.Next[curNode]
				// projections are a map of param function projections to lemmas
				// where lemmas is a map of lemma strings to a list of morphemes
				projections = make(map[string]map[string][]int, len(nextMorphs))
				// gather lemmas by projection
				for _, m := range nextMorphs {
					morph = l.Morphemes[m]
					projection = pf(morph)
					// log.Println("\t\tGot lattice morph", morph, "pf:", projection)
					if curLemmas, exists := projections[projection]; exists {
						// log.Println("\t\t\tprojection exists")
						if morphs, exists := curLemmas[morph.Lemma]; exists {
							// log.Println("\t\t\t\tadding to existing lemma", morph.Lemma)
							curLemmas[morph.Lemma] = append(morphs, m)
						} else {
							// log.Println("\t\t\t\tadding new lemma", morph.Lemma)
							curLemmas[morph.Lemma] = []int{m}
						}
					} else {
						// log.Println("\t\t\tadding new projection")
						newLemmas := make(map[string][]int, 3)
						newLemmas[morph.Lemma] = []int{m}
						projections[projection] = newLemmas
					}
				}
				if lemmas, exists := projections[pf(goldMorph)]; exists {
					// log.Println("\tfound existing lemmas for gold")
					if len(lemmas) > 1 {
						// log.Println("\t\tfound", len(lemmas), "ambiguous lemmas; >1")
						// found ambiguous lemmas for gold morpheme
						// ambMorphCur = &AmbLemma{token, make([]string, 0, len(lemmas)), prevGold[:]}
						nextNodes := map[int]bool{}
						lastNextNode := 0
						for _, morphemes := range lemmas {
							// log.Println("\t\t\tat lemma", lemma)
							for _, m := range morphemes {
								morph = l.Morphemes[m]
								// log.Println("\t\t\t\tadding morpheme", morph)
								nextNodes[morph.To()] = true
								lastNextNode = morph.To()
							}
							// ambMorphCur.Lemmas = append(ambMorphCur.Lemmas, lemma)
						}
						// retval = append(retval, ambMorphCur)
						if len(nextNodes) == 1 {
							// log.Println("\t\t\t\tcontinuing to next")
							// we can continue, there is only one "next" node for
							// the gold path
							curNode = lastNextNode
							continue goldMorphs
						} else {
							// log.Println("\t\t\t\tcan't continue, breaking")
							// we don't know what the next node is, we can't follow
							// the gold path for this token
							break goldMorphs
						}
					} else if len(lemmas) == 1 {
						// log.Println("\t\tfound 1 lemma for morpheme")
						for _, morphs := range lemmas {
							// there is only one lemma
							morph := l.Morphemes[morphs[0]]
							curNode = morph.To()
							if len(morphs) == 1 {
								// log.Println("\t\t\t\tadvancing to next token")
								// we can continue, there is only one "next" node for
								// the gold path
								if len(morph.Lemma) > 0 && morph.Lemma != "_" {
									retval = append(retval, &DisAmbLemma{
										token,
										goldToken,
										morph.CPOS,
										morph.Lemma,
									})
									if morph.CPOS == "BN" || morph.CPOS == "IN" {
										if goldToken+1 < len(mapping.Spellout) {
											log.Println("Found IN/BN with a suffix in gold(?)", token, goldToken)
										}
									}
								}
							} else {
								log.Println("Multiple morphemes for the same lemma at token", token, "trying to back off")
							}
						}
					} else {
						log.Println("No lemmas at all at token", token)
						continue goldMorphs
					}
				}
			}
		}
	}
	return retval
}

func (l *Lattice) GetDirectedEdge(i int) graph.DirectedEdge {
	return graph.DirectedEdge(l.Morphemes[i])
}

func (l *Lattice) GetEdge(i int) graph.Edge {
	return graph.Edge(l.Morphemes[i])
}

func (l *Lattice) GetEdges() []int {
	res := make([]int, len(l.Morphemes))
	for i, _ := range l.Morphemes {
		res[i] = i
	}
	return res
}

func (l *Lattice) GetVertices() []int {
	vSet := make(map[int]bool)
	for _, edge := range l.Morphemes {
		vSet[edge.From()] = true
		vSet[edge.To()] = true
	}
	res := make([]int, 0, len(vSet))
	for k, _ := range vSet {
		res = append(res, k)
	}
	return res
}

func (l *Lattice) GetVertex(i int) graph.Vertex {
	return graph.BasicVertex(i)
}

func (l *Lattice) NumberOfEdges() int {
	return len(l.Morphemes)
}

func (l *Lattice) NumberOfVertices() int {
	return len(l.GetVertices())
}

var _ graph.BoundedLattice = &Lattice{}
var _ graph.DirectedGraph = &Lattice{}

// untested..
func (l *Lattice) Inf(i, j int) int {
	iReachable := make(map[int]int)
	for path := range graph.YieldAllPaths(graph.DirectedGraph(l), l.Bottom(), i) {
		for i, el := range path {
			dist := len(path) - i - 1
			iReachable[el.ID()] = dist
		}
	}
	var bestVal, bestDist int = -1, -1
	for path := range graph.YieldAllPaths(graph.DirectedGraph(l), l.Bottom(), j) {
		for i, _ := range path {
			el := path[len(path)-i-1]
			dist, exists := iReachable[el.ID()]
			if exists {
				if bestDist == -1 || bestDist > dist {
					bestVal = el.ID()
					bestDist = dist
					break
				}
			}
		}
	}
	return bestVal
}

// untested..
func (l *Lattice) Sup(i, j int) int {
	iReachable := make(map[int]int)
	for path := range graph.YieldAllPaths(graph.DirectedGraph(l), i, l.Top()) {
		for dist, el := range path {
			iReachable[el.ID()] = dist
		}
	}
	var bestVal, bestDist int = -1, -1
	for path := range graph.YieldAllPaths(graph.DirectedGraph(l), j, l.Top()) {
		for _, el := range path {
			dist, exists := iReachable[el.ID()]
			if exists {
				if bestDist == -1 || bestDist > dist {
					bestVal = el.ID()
					bestDist = dist
					break
				}
			}
		}
	}
	return bestVal
}

func (l *Lattice) Top() int {
	return l.TopId
}

func (l *Lattice) Bottom() int {
	return l.BottomId
}

func (l *Lattice) MaxPathLen() int {
	if len(l.Morphemes) == 0 {
		return 0
	}
	return l.Top() - l.Bottom()
}

func (l *Lattice) SortMorphemes() {
	sort.Sort(l.Morphemes)
	l.GenNexts(true)
}

func (l *Lattice) GenNexts(panicMismatch bool) {
	l.Next = make(map[int][]int, l.NumberOfVertices())
	for i, m := range l.Morphemes {
		if i != m.ID() && panicMismatch {
			panic("index != ID")
		}
		if cur, exists := l.Next[m.From()]; exists {
			cur = append(cur, i)
			l.Next[m.From()] = cur
		} else {
			l.Next[m.From()] = []int{i}
		}
	}

	for _, next := range l.Next {
		sort.Ints(next)
	}
}

func (l *Lattice) GenToken() {
	if l.Spellouts == nil {
		panic("Can't generate token without a spellout")
	}
	if len(l.Spellouts) == 0 {
		l.Token = Token("")
		return
	}
	spellout := l.Spellouts[0]
	strs := make([]string, len(spellout))
	for i, morph := range spellout {
		strs[i] = morph.Form
	}
	l.Token = Token(strings.Join(strs, ""))
}

func (l *Lattice) GenSpellouts() {
	if l.Spellouts != nil {
		return
	}
	if len(l.Morphemes) == 0 {
		l.Spellouts = make(Spellouts, 0)
		return
	}
	var (
		pathId   int
		from, to int = l.Bottom(), l.Top()
	)
	l.Spellouts = make(Spellouts, 0, to-from)
	for path := range graph.YieldAllPaths(graph.DirectedGraph(l), from, to) {
		spellout := make(Spellout, len(path))
		for i, el := range path {
			spellout[i] = el.(*EMorpheme)
		}
		l.Spellouts = append(l.Spellouts, spellout)

		pathId++
	}
	sort.Sort(l.Spellouts)
}

func (l *Lattice) YieldPaths() chan Path {
	l.GenSpellouts()
	pathChan := make(chan Path)
	go func() {
		for i, _ := range l.Spellouts {
			pathChan <- Path(i)
		}
		close(pathChan)
	}()
	return pathChan
}

func (l *Lattice) Path(i int) Spellout {
	return l.Spellouts[i]
}

func (l *Lattice) AllFusedFrom(from int) []string {
	var (
		curMorphemeIds []int
		curMorpheme    *EMorpheme
		curNode        int = from
		exists         bool
	)
	forms := make([]string, 0, 3)
	allNext := []string{""}
	// log.Println("Fusing from", from, "in lattice", l.Token)
	for curNode < l.Top() {
		curMorphemeIds, exists = l.Next[curNode]
		if !exists {
			panic(fmt.Sprintf("Lattice does not have outgoing node Id %v", curNode))
		}
		if len(curMorphemeIds) > 1 {
			// log.Println("\tFound multiple outgoing")
			for _, curMorphemeId := range curMorphemeIds {
				curMorpheme = l.Morphemes[curMorphemeId]
				// log.Println("\tRecursing at", curMorpheme)
				for _, curStr := range l.AllFusedFrom(curMorpheme.To()) {
					allNext = append(allNext, curMorpheme.Form+curStr)
				}
			}
			break
		} else {
			curMorpheme = l.Morphemes[curMorphemeIds[0]]
			// log.Println("\tAt morpheme", curMorpheme)
			curNode = curMorpheme.To()
			if curMorpheme.From() > l.Bottom() && curMorpheme.Form == "H" {
				// log.Println("\tSkipping fused H")
				// fuse any H encountered mid lattice
				continue
			}
			forms = append(forms, curMorpheme.Form)
		}
	}
	if len(allNext) > 0 {
		results := make([]string, len(allNext))
		for i, val := range allNext {
			results[i] = strings.Join(forms, "") + val
		}
		// log.Println("Returning", results)
		return results
	} else {
		// log.Println("Returning", forms)
		return []string{strings.Join(forms, "")}
	}
}

type MorphDependencyGraph interface {
	LabeledDependencyGraph
	GetMappings() Mappings
	GetMorpheme(int) *EMorpheme
}
