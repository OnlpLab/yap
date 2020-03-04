package types

import (
	"github.com/OnlpLab/yap/alg/graph"
	"github.com/OnlpLab/yap/util"
)

type DepNode interface {
	graph.Vertex
	String() string
}

type DepArc interface {
	graph.DirectedEdge
	GetModifier() int
	GetHead() int
	String() string
}

type DepRel string

func (d DepRel) String() string {
	return string(d)
}

type LabeledDepArc interface {
	DepArc
	GetRelation() DepRel
}

type Labeled interface {
	GetLabeledArc(int) LabeledDepArc
}

type DependencyGraph interface {
	graph.DirectedGraph
	GetNode(int) DepNode
	GetArc(int) DepArc
	NumberOfNodes() int
	NumberOfArcs() int
	Equal(otherEq util.Equaler) bool
	Sentence() Sentence
	TaggedSentence() TaggedSentence
}

type LabeledDependencyGraph interface {
	DependencyGraph
	Labeled
}
