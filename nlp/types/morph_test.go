package types

import (
	G "github.com/OnlpLab/yap/alg/graph"
	"testing"
)

var testLat *Lattice = &Lattice{
	"KFHM",
	[]*Morpheme{
		&Morpheme{G.BasicDirectedEdge{0, 7, 8}, "K", "ADVERB", "ADVERB", nil, 6},
		&Morpheme{G.BasicDirectedEdge{1, 7, 9}, "KF", "TEMP", "TEMP", nil, 6},
		&Morpheme{G.BasicDirectedEdge{2, 8, 10}, "FHM", "NNP", "NNP", nil, 6},
		&Morpheme{G.BasicDirectedEdge{3, 9, 10}, "HM", "PRP", "PRP", map[string]string{"gen": "M", "num": "P", "per": "3"}, 6},
		&Morpheme{G.BasicDirectedEdge{4, 9, 10}, "HM", "COP", "COP", map[string]string{"gen": "M", "num": "P", "per": "3", "polar": "pos"}, 6},
	},
	nil,
}

func TestLattice(t *testing.T) {
	testLat.GenSpellouts()
}
