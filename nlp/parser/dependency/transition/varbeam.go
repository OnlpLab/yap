package transition

import (
	"github.com/OnlpLab/yap/alg/perceptron"
	// "github.com/OnlpLab/yap/nlp/parser/dependency"

	"github.com/OnlpLab/yap/alg/search"
)

type VarBeam struct {
	search.Beam
}

var _ search.Interface = &VarBeam{}
var _ perceptron.EarlyUpdateInstanceDecoder = &VarBeam{}

// var _ dependency.DependencyParser = &VarBeam{}

type NoCandidate struct{}

var _ search.Candidate = &NoCandidate{}

func (c *NoCandidate) Copy() search.Candidate {
	return c
}

func (c *NoCandidate) Score() float64 {
	return 0.0
}

func (c *NoCandidate) Equal(other search.Candidate) bool {
	_, ok := other.(*NoCandidate)
	return ok
}

func (c *NoCandidate) Len() int {
	return 0
}

func (c *NoCandidate) Terminal() bool {
	return false
}

func (v *VarBeam) Top(a search.Agenda) search.Candidate {
	agenda := a.(*search.BaseAgenda)
	for _, conf := range agenda.Confs {
		if !conf.C.Terminal() {
			return &NoCandidate{}
		}
	}
	return v.Beam.Top(a)
}

func (v *VarBeam) GoalTest(p search.Problem, c search.Candidate, rounds int) bool {
	_, isNoCandidate := c.(*NoCandidate)
	if isNoCandidate {
		return false
	} else {
		return v.Beam.GoalTest(p, c, rounds)
	}
}
