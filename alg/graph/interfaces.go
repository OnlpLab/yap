package graph

import "github.com/OnlpLab/yap/util"

type Vertex interface {
	util.Equaler
	ID() int
}

type Edge interface {
	util.Equaler
	Vertices() []int
	ID() int
}

type DirectedEdge interface {
	Edge
	From() int
	To() int
}

type Graph interface {
	GetVertices() []int
	GetEdges() []int
	GetVertex(int) Vertex
	GetEdge(int) Edge
	NumberOfVertices() int
	NumberOfEdges() int
}

type DirectedGraph interface {
	Graph
	GetDirectedEdge(int) DirectedEdge
}

type Lattice interface {
	Inf(int, int) int
	Sup(int, int) int
}

type BoundedLattice interface {
	Lattice
	// Top() >= Sup(x,y) \forall x,y
	Top() int
	// Bottom() <= Inf(x,y) \forall x,y
	Bottom() int
}
