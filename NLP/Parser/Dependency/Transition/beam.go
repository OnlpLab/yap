package Transition

import (
	"chukuparser/Algorithm/Model/Perceptron"
	BeamSearch "chukuparser/Algorithm/Search"
	"chukuparser/Algorithm/Transition"
	"chukuparser/NLP"
	"chukuparser/NLP/Parser/Dependency"
	"container/heap"
	"sync"
)

type Beam struct {
	Base             DependencyConfiguration
	TransFunc        Transition.TransitionSystem
	FeatExtractor    Perceptron.FeatureExtractor
	Model            Dependency.ParameterModel
	Size             int
	NumRelations     int
	ReturnModelValue bool
	ReturnSequence   bool
}

var _ BeamSearch.Interface = &Beam{}
var _ Perceptron.EarlyUpdateInstanceDecoder = &Beam{}
var _ Dependency.DependencyParser = &Beam{}

func (b *Beam) StartItem(p BeamSearch.Problem) BeamSearch.Candidates {
	sent, ok := p.(NLP.TaggedSentence)
	if !ok {
		panic("Problem should be an NLP.TaggedSentence")
	}
	if b.Base == nil {
		panic("Set Base to a DependencyConfiguration to parse")
	}
	if b.TransFunc == nil {
		panic("Set Transition to a Transition.TransitionSystem to parse")
	}
	if b.Model == nil {
		panic("Set Model to Dependency.ParameterModel to parse")
	}
	b.Base.Conf().Init(sent)

	firstCandidates := make([]BeamSearch.Candidate, 1)
	firstCandidates[0] = &ScoredConfiguration{b.Base, 0.0}

	return firstCandidates
}

func (b *Beam) Clear() BeamSearch.Agenda {
	// beam size * # of transitions
	return NewAgenda(b.Size * b.estimatedTransitions())
}

func (b *Beam) Insert(cs chan BeamSearch.Candidate, a BeamSearch.Agenda) BeamSearch.Agenda {
	tempAgenda := NewAgenda(b.estimatedTransitions())
	tempAgendaHeap := heap.Interface(tempAgenda)
	heap.Init(tempAgendaHeap)
	for c := range cs {
		candidate := c.(*ScoredConfiguration)
		conf := candidate.C
		feats := b.FeatExtractor.Features(conf)
		featsAsWeights := b.Model.ModelValue(feats)
		currentScore := b.Model.NewModelValue().ScoreWith(b.Model, featsAsWeights)
		candidate.Score += currentScore
		if tempAgenda.Len() == b.Size {
			// if the temp. agenda is the size of the beam
			// there is no reason to add a new one if we can prune
			// some in the beam's Insert function
			if tempAgenda.confs[0].Score > candidate.Score {
				// if the current score has a worse score than the
				// worst one in the temporary agenda, there is no point
				// to adding it
				continue
			} else {
				heap.Pop(tempAgendaHeap)
			}
		}
		heap.Push(tempAgendaHeap, candidate)
	}
	agenda := a.(*Agenda)
	agenda.Lock()
	agenda.confs = append(agenda.confs, tempAgenda.confs...)
	agenda.Unlock()
	return agenda
}

func (b *Beam) estimatedTransitions() int {
	return b.NumRelations*2 + 2
}

func (b *Beam) Expand(c BeamSearch.Candidate, p BeamSearch.Problem) chan BeamSearch.Candidate {
	candidate := c.(*ScoredConfiguration)
	conf := candidate.C
	retChan := make(chan BeamSearch.Candidate, b.estimatedTransitions())
	go func(currentConf DependencyConfiguration, candidateChan chan BeamSearch.Candidate) {
		for transition := range b.TransFunc.YieldTransitions(currentConf.Conf()) {
			newConf := b.TransFunc.Transition(currentConf.Conf(), transition)

			// at this point, the candidate has it's *previous* score
			// insert will do compute newConf's features and model score
			// this is done to allow for maximum concurrency
			// where candidates are created while others are being scored before
			// adding into the agenda
			candidateChan <- &ScoredConfiguration{newConf.(DependencyConfiguration), candidate.Score}
		}
		close(candidateChan)
	}(conf, retChan)
	return retChan
}

func (b *Beam) Top(a BeamSearch.Agenda) BeamSearch.Candidate {
	agenda := a.(*Agenda)
	agendaHeap := heap.Interface(agenda)
	heap.Init(agendaHeap)
	// peeking into an initalized heap
	best := agenda.confs[0]
	return best
}

func (b *Beam) GoalTest(p BeamSearch.Problem, c BeamSearch.Candidate) bool {
	conf := c.(DependencyConfiguration)
	return conf.Conf().Terminal()
}

func (b *Beam) TopB(a BeamSearch.Agenda, B int) BeamSearch.Candidates {
	candidates := make([]BeamSearch.Candidate, B)
	agendaHeap := a.(heap.Interface)
	// assume agenda heap is already heapified
	// heap.Init(agendaHeap)
	for i := 0; i < B; i++ {
		candidates[i] = heap.Pop(agendaHeap)
	}
	return candidates
}

func (b *Beam) Parse(sent NLP.Sentence, constraints Dependency.ConstraintModel, model Dependency.ParameterModel) (NLP.DependencyGraph, interface{}) {
	b.Model = model

	return nil, nil
}

// Perceptron function
func (b *Beam) DecodeEarlyUpdate(goldInstance Perceptron.DecodedInstance, m Perceptron.Model) (Perceptron.DecodedInstance, *Perceptron.SparseWeightVector, *Perceptron.SparseWeightVector) {
	sent := goldInstance.Instance().(NLP.Sentence)
	b.Model = Dependency.ParameterModel(&PerceptronModel{m.(*Perceptron.LinearPerceptron)})

	// abstract casting >:-[
	rawGoldSequence := goldInstance.Decoded().(Transition.ConfigurationSequence)
	goldSequence := make([]interface{}, len(rawGoldSequence))
	for i, val := range rawGoldSequence {
		goldSequence[i] = val
	}

	goldGraph := goldSequence[0].(*SimpleConfiguration).Graph()
	b.ReturnModelValue = true
	beamResult, goldResult := BeamSearch.SearchEarlyUpdate(b, sent, b.Size, goldSequence)
	parsedGraph := beamResult.(NLP.DependencyGraph)

	if parsedGraph.NumberOfEdges() == goldGraph.NumberOfEdges() && !goldGraph.Equal(parsedGraph) {
		panic("Oracle parse result does not equal gold")
	}
	parseParams := parseParamsInterface.(*ParseResultParameters)
	weights := parseParams.modelValue.(*PerceptronModelValue).vector
	return &Perceptron.Decoded{goldInstance.Instance(), parsedGraph}, weights, goldWeights.(*Perceptron.SparseWeightVector)
}

type ScoredConfiguration struct {
	C     DependencyConfiguration
	Score float64
}

type Agenda struct {
	sync.Mutex
	confs []*ScoredConfiguration
}

func (a *Agenda) Len() int {
	return len(a.confs)
}

func (a *Agenda) Less(i, j int) bool {
	scoredI := a.confs[i]
	scoredJ := a.confs[j]
	// less in reverse, we want the highest scoring to be first in the heap
	return scoredI.Score > scoredJ.Score
}

func (a *Agenda) Swap(i, j int) {
	a.confs[i], a.confs[j] = a.confs[j], a.confs[i]
}

func (a *Agenda) Push(x interface{}) {
	scored := x.(*ScoredConfiguration)
	a.confs = append(a.confs, scored)
}

func (a *Agenda) Pop() interface{} {
	n := len(a.confs)
	scored := a.confs[n-1]
	a.confs = a.confs[0 : n-1]
	return scored
}

func (a *Agenda) Contains(goldCandidate BeamSearch.Candidate) bool {
	for _, candidate := range a.confs {
		if candidate.C.Equal(goldCandidate.(DependencyConfiguration)) {
			return true
		}
	}
	return false
}

var _ BeamSearch.Agenda = &Agenda{}
var _ heap.Interface = &Agenda{}

func NewAgenda(size int) *Agenda {
	newAgenda := new(Agenda)
	newAgenda.confs = make([]*ScoredConfiguration, 0, size)
	return newAgenda
}