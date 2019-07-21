package graph

import "github.com/OnlpLab/yap/util"

type BasicVertex int
type BasicDirectedEdge [3]int
type BasicGraph struct {
	Vertices []BasicVertex
	Edges    []BasicDirectedEdge
}

var _ Vertex = *new(BasicVertex)
var _ DirectedEdge = BasicDirectedEdge{}
var _ DirectedGraph = &BasicGraph{}

func (b BasicVertex) ID() int {
	return int(b)
}

func (b BasicVertex) Equal(otherEq util.Equaler) bool {
	other := otherEq.(BasicVertex)
	return b == other
}

func (e BasicDirectedEdge) ID() int {
	return e[0]
}

func (e BasicDirectedEdge) From() int {
	return e[1]
}

func (e BasicDirectedEdge) To() int {
	return e[2]
}

func (e BasicDirectedEdge) Vertices() []int {
	return []int{e[1], e[2]}
}

func (e BasicDirectedEdge) Equal(otherEq util.Equaler) bool {
	other := otherEq.(BasicDirectedEdge)
	return e[1] == other[1] && e[2] == other[2]
}

func (g *BasicGraph) GetVertices() []int {
	vertices := make([]int, len(g.Vertices))
	for i, _ := range g.Vertices {
		vertices[i] = i
	}
	return vertices
}

func (g *BasicGraph) GetEdges() []int {
	edges := make([]int, len(g.Edges))
	for i, _ := range g.Edges {
		edges[i] = i
	}
	return edges
}

func (g *BasicGraph) GetVertex(i int) Vertex {
	return g.Vertices[i]
}

func (g *BasicGraph) GetEdge(i int) Edge {
	return Edge(g.Edges[i])
}

func (g *BasicGraph) NumberOfVertices() int {
	return len(g.Vertices)
}

func (g *BasicGraph) NumberOfEdges() int {
	return len(g.Edges)
}

func (g *BasicGraph) GetDirectedEdge(i int) DirectedEdge {
	return g.Edges[i]
}
