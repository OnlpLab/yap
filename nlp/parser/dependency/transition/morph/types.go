package morph

import (
	"github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	nlp "github.com/OnlpLab/yap/nlp/types"
	// "log"
)

type BasicMorphGraph struct {
	transition.BasicDepGraph
	Mappings nlp.Mappings
	Lattice  nlp.LatticeSentence
}

var _ nlp.MorphDependencyGraph = &BasicMorphGraph{}

func (m *BasicMorphGraph) GetMappings() nlp.Mappings {
	return m.Mappings
}

func (m *BasicMorphGraph) GetMorpheme(i int) *nlp.EMorpheme {
	return m.Nodes[i].(*nlp.EMorpheme)
}

func (m *BasicMorphGraph) Sentence() nlp.Sentence {
	return m.Lattice
}

func (m *BasicMorphGraph) TaggedSentence() nlp.TaggedSentence {
	sent := make([]nlp.TaggedToken, m.NumberOfNodes()-1)
	for _, node := range m.Nodes {
		taggedNode := node.(*nlp.EMorpheme)
		if taggedNode.Form == nlp.ROOT_TOKEN {
			continue
		}
		target := taggedNode.ID() - 1
		if target < 0 {
			panic("Too small")
		}
		if target >= len(sent) {
			panic("Too large")
		}
		sent[target] = nlp.TaggedToken{taggedNode.Form, taggedNode.Lemma, taggedNode.POS}
	}
	return nlp.TaggedSentence(nlp.BasicTaggedSentence(sent))
}

func CombineToGoldMorph(graph nlp.LabeledDependencyGraph, goldLat, ambLat nlp.LatticeSentence) (*BasicMorphGraph, bool) {
	var addedMissingSpellout bool
	// generate graph
	mGraph := new(transition.BasicDepGraph)

	mGraph.Nodes = make([]nlp.DepNode, 0, graph.NumberOfNodes())

	// generate morph. disambiguation (= mapping) and nodes
	mappings := make([]*nlp.Mapping, len(goldLat))
	for i, lat := range goldLat {
		lat.GenSpellouts()
		lat.GenToken()
		if len(lat.Spellouts) == 0 {
			continue
		}
		mapping := &nlp.Mapping{
			lat.Token,
			lat.Spellouts[0],
		}
		// if the gold spellout doesn't exist in the lattice, add it
		_, exists := ambLat[i].Spellouts.Find(mapping.Spellout)
		if !exists {
			ambLat[i].Spellouts = append(ambLat[i].Spellouts, mapping.Spellout)
			addedMissingSpellout = true
			ambLat[i].UnionPath(&lat)
		}

		ambLat[i].BridgeMissingMorphemes()
		mappings[i] = mapping

		// add the morpheme as a node
		for _, morpheme := range mapping.Spellout {
			mGraph.Nodes = append(mGraph.Nodes, morpheme)
		}
	}

	// copy arcs
	mGraph.Arcs = make([]*transition.BasicDepArc, graph.NumberOfArcs())
	for i, arcId := range graph.GetEdges() {
		arc := graph.GetLabeledArc(arcId)
		// TODO: fix this ugly casting
		mGraph.Arcs[i] = arc.(*transition.BasicDepArc)
	}

	m := &BasicMorphGraph{
		*mGraph,
		mappings,
		ambLat,
	}

	return m, addedMissingSpellout
}
