package search

import (
	"container/heap"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"github.com/OnlpLab/yap/alg/featurevector"
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/rlheap"
	"github.com/OnlpLab/yap/alg/stlheap"
	"github.com/OnlpLab/yap/alg/transition"
	TransitionModel "github.com/OnlpLab/yap/alg/transition/model"
	"github.com/OnlpLab/yap/util"
)

var (
	AgendaOut bool = false
	ShowFeats bool = false
)

type Beam struct {
	// main beam functions and parameters
	Base          transition.Configuration
	TransFunc     transition.TransitionSystem
	FeatExtractor perceptron.FeatureExtractor
	Model         TransitionModel.Interface

	Size                 int
	EstimatedTransitions int
	EarlyUpdateAt        int

	// beam parsing variables
	currentBeamSize int

	// flags
	Averaged           bool
	DecodeTest         bool // set to true when decoding is 'testing' during learning process
	ReturnModelValue   bool
	ReturnSequence     bool
	ShowConsiderations bool
	ConcurrentExec     bool
	Log                bool
	ShortTempAgenda    bool
	NoRecover          bool
	Align              bool

	// used for performance tuning
	lastRoundStart time.Time
	DurTotal       time.Duration

	// used for debug output
	Transitions *util.EnumSet

	candidateScorePool    *sync.Pool
	IntegrationGeneration int
	ScoredStoreDense      bool
}

var _ Interface = &Beam{}
var _ perceptron.EarlyUpdateInstanceDecoder = &Beam{}

func (b *Beam) Name() string {
	notAligned := ""
	if !b.Align {
		notAligned = "Not "
	}
	notAveraged := ""
	if !b.Averaged {
		notAveraged = "Not "
	}
	return "Standard Beam [" + notAligned + "Aligned & " + notAveraged + "Averaged]"
}

func (b *Beam) Concurrent() bool {
	return b.ConcurrentExec
}

func (b *Beam) StartItem(p Problem) []Candidate {
	if b.Base == nil {
		panic("Set Base to a transition.Configuration to parse")
	}
	if b.TransFunc == nil {
		panic("Set Transition to a transition.TransitionSystem to parse")
	}
	if b.Model == nil {
		panic("Set a Model")
	}
	if b.EstimatedTransitions == 0 {
		b.EstimatedTransitions = b.Size
	}

	c := b.Base.Copy()
	c.Clear()
	c.Init(p)

	if b.candidateScorePool == nil {
		if b.ScoredStoreDense {
			b.candidateScorePool = &sync.Pool{New: featurevector.MakeDenseStore}
		} else {
			b.candidateScorePool = &sync.Pool{New: featurevector.MakeMapStore}
		}
	}
	b.currentBeamSize = 0
	firstCandidates := make([]Candidate, 1)
	firstCandidate := &ScoredConfiguration{c, transition.ConstTransition(0), NewScoreState(), nil, 0, 0, true, b.Averaged}
	firstCandidates[0] = firstCandidate
	if AllOut {
		// log.Println("\t\tAgenda post push 0:0 , ")
	}
	return firstCandidates
}

func (b *Beam) Clear(agenda Agenda) Agenda {
	// start := time.Now()
	if agenda == nil {
		newAgenda := NewAgenda(b.Size)
		// newAgenda.HeapReverse = true
		agenda = newAgenda
	} else {
		agenda.Clear()
	}
	// b.DurClearing += time.Since(start)
	return agenda
}

func (b *Beam) Insert(cs chan Candidate, a Agenda) []Candidate { //Agenda {
	var tempAgendaSize int
	if b.ShortTempAgenda {
		tempAgendaSize = b.Size
	} else {
		tempAgendaSize = b.EstimatedTransitions
	}
	tempAgenda := NewAgenda(tempAgendaSize)
	tempAgendaHeap := heap.Interface(tempAgenda)
	// tempAgenda.HeapReverse = true
	rlheap.Init(tempAgendaHeap)
	// heap.Init(tempAgendaHeap)
	for c := range cs {
		currentScoredConf := c.(*ScoredConfiguration)
		if b.ShortTempAgenda && tempAgenda.Len() == b.Size {
			// if the temp. agenda is the size of the beam
			// there is no reason to add a new one if we can prune
			// some in the beam's Insert function
			if tempAgenda.Peek().Score() > currentScoredConf.Score() {
				// log.Println("\t\tNot pushed onto Beam", b.Transitions.ValueOf(int(currentScoredConf.Transition)))
				// if the current score has a worse score than the
				// worst one in the temporary agenda, there is no point
				// to adding it
				continue
			} else {
				// log.Println("\t\tPopped", tempAgenda.Confs[0].Transition, "from beam")
				rlheap.Pop(tempAgendaHeap)
				// heap.Pop(tempAgendaHeap)
			}
		}
		// log.Println("\t\tPushed onto Beam", b.Transitions.ValueOf(int(currentScoredConf.Transition)))
		rlheap.Push(tempAgendaHeap, currentScoredConf)
		// heap.Push(tempAgendaHeap, currentScoredConf)
		// heaping += time.Since(lastMem)
	}
	// lastMem = time.Now()
	// agenda := a.(*BaseAgenda)
	// agenda.Lock()
	// agenda.HeapReverse = true
	// for _, beamAction := range tempAgenda.Confs {
	// 	agenda.AddCandidate(beamAction)
	// }
	// // agenda.Confs = append(agenda.Confs, tempAgenda.Confs...)
	// agenda.Unlock()
	// agending += time.Since(lastMem)

	// return agenda
	retval := make([]Candidate, len(tempAgenda.Confs))
	for i, cand := range tempAgenda.Confs {
		retval[i] = cand
	}
	return retval
}

func (b *Beam) Expand(c Candidate, p Problem, candidateNum int) chan Candidate {
	var (
		lastMem          time.Time
		featuring        time.Duration
		transitionScore  int64
		transitionExists bool
	)
	// start := time.Now()
	candidate := c.(*ScoredConfiguration)
	conf := candidate.C
	lastMem = time.Now()
	retChan := make(chan Candidate, b.EstimatedTransitions)
	// scores := make([]int64, 0, b.EstimatedTransitions)
	go func(currentConf transition.Configuration, candidateChan chan Candidate) {
		var (
			transNum int
			score    int64
			// feats       []featurevector.Feature
			// newFeatList *transition.FeaturesList
			// score1   int64
			yielded     bool = false
			scores      featurevector.ScoredStore
			transType   byte
			transitions []int
		)
		scores = b.candidateScorePool.Get().(featurevector.ScoredStore)
		// scores.Init()
		scores.Clear()
		transType, transitions = b.TransFunc.GetTransitions(currentConf)
		if AllOut {
			// log.Println("\tSetting transitions to", transitions)
		}
		scores.SetTransitions(transitions)
		scorer := b.Model.(*TransitionModel.AvgMatrixSparse)
		if b.DecodeTest {
			if b.ScoredStoreDense {

				scores.(*featurevector.ArrayStore).Generation = b.IntegrationGeneration
			} else {
				scores.(*featurevector.MapStore).Generation = b.IntegrationGeneration
			}
		}

		if ShowFeats {
			b.FeatExtractor.SetLog(true)
			log.Println("Features")
		}
		feats := b.FeatExtractor.Features(conf, false, transType, transitions)
		b.FeatExtractor.SetLog(false)
		featuring += time.Since(lastMem)

		var newFeatList *transition.FeaturesList
		if b.ReturnModelValue {
			newFeatList = &transition.FeaturesList{feats, conf.GetLastTransition(), candidate.Features}
		} else {
			newFeatList = &transition.FeaturesList{feats, conf.GetLastTransition(), nil}
		}
		scorer.SetTransitionScores(feats, scores, b.DecodeTest)
		// log.Println("\t\tScores set to", scores.(*featurevector.ArrayStore).ScoreMap())
		if AllOut {
			log.Println("\tExpanding candidate", candidateNum+1, "last transition", currentConf.GetLastTransition(), "score", candidate.Score())
			// log.Println("\tCandidate:", candidate.C.GetSequence())
			log.Println("\tCandidate:", candidate)
		}
		for _, curTransition := range transitions {
			yielded = true
			// score1 = b.Model.TransitionModel().TransitionScore(transition, feats)
			if transitionScore, transitionExists = scores.Get(curTransition); transitionExists {
				score = transitionScore
			} else {
				score = 0.0
			}
			// if score != score1 {
			// 	panic(fmt.Sprintf("Got different score for transition %v: %v vs %v", transition, score, score1))
			// }
			// score = b.Model.TransitionModel().TransitionScore(transition, feats)
			// log.Printf("\t\twith transition/score %d/%v\n", curTransition, candidate.Score()+float64(score))
			// at this point, the candidate has it's *previous* score
			// insert will do compute newConf's features and model score
			// this is done to allow for maximum concurrency
			// where candidates are created while others are being scored before
			// adding into the agenda
			scored := &ScoredConfiguration{currentConf, &transition.TypedTransition{transType, curTransition}, candidate.InternalScores.Copy(), newFeatList, candidateNum, transNum, false, candidate.Averaged}
			// log.Println("Scored before", scored.InternalScores)
			scored.AddScore(score, currentConf.Assignment())
			// log.Println("Scored after", scored.InternalScores)
			candidateChan <- scored

			transNum++
		}
		if !yielded {
			if AllOut {
				log.Println("non-yield candidate kept in beam")
			}
			candidateChan <- c
		}
		b.candidateScorePool.Put(scores)
		close(candidateChan)
	}(conf, retChan)
	// b.DurExpanding += time.Since(start)
	return retChan
}

func (b *Beam) Best(a Agenda) Candidate {
	agenda := a.(*BaseAgenda)
	// agenda.ShowSwap = true
	// log.Println("Agenda before sort")
	// for _, c := range agenda.Confs {
	// 	log.Printf("\t%d %v", c.Score, c.C)
	// }
	// a.HeapReverse = true
	if agenda.Len() == 0 {
		panic("Can't retrieve best candidate from empty agenda")
	}
	if AgendaOut {
		agenda.ReEnumerate()
		log.Println("Agenda pre sort")
		log.Println(agenda.ConfStr())
	}
	agenda.HeapReverse = true
	stlheap.Sort(agenda)
	agenda.HeapReverse = false
	// rlheap.RegularSort(agenda)
	// log.Println("Sorting")
	// j := 0
	// rlheap.Logging = true
	// for i := agenda.Len() - 1; i > 0; i-- {
	// 	// log.Println(j)
	// 	// Pop without reslicing
	// 	// i := agenda.Len() - 1
	// 	// agenda.Swap(0, i)
	// 	cur := agenda.Confs[i]
	// 	agenda.Copy(0, i)
	// 	stlheap.Adjust(agenda, i, cur)
	// 	// rlheap.RegularDown(agenda, 0, i)
	// 	// rlheap.Down(agenda, 0, i)
	// 	log.Println(agenda.ConfStr())
	// 	// j++
	// }
	// agenda.HeapReverse = false
	// if agenda.Len() > 1 && agenda.Less(0, 1) { // (agenda.Less(0, 1) || (agenda.Less(0, 1) == agenda.Less(1, 0))) {
	// 	agenda.Swap(0, 1)
	// 	log.Println(agenda.ConfStr())
	// }
	// rlheap.Logging = false

	// for _, c := range agenda.Confs {
	// 	c.Expand(b.TransFunc)
	// }
	if AgendaOut {
		log.Println("Agenda after sort")
		log.Println(agenda.ConfStr())
	}
	// for _, c := range agenda.Confs {
	// 	log.Printf("\t%d %v", c.Score, c.C)
	// }
	agenda.Confs[0].Expand(b.TransFunc)
	// agenda.ShowSwap = false
	return agenda.Confs[0]
}

func (b *Beam) Top(a Agenda) Candidate {
	// start := time.Now()
	agenda := a.(*BaseAgenda)
	if agenda.Len() == 0 {
		panic("Got empty agenda!")
	}
	// return nil
	// agendaHeap := heap.Interface(agenda)
	// agenda.HeapReverse = true
	// // heapify agenda
	// rlheap.Init(agendaHeap)
	// peeking into an initialized (heapified) array
	if len(agenda.Confs) == 0 {
		panic("Got empty agenda")
	}
	var bestCandidate *ScoredConfiguration
	for _, candidate := range agenda.Confs {
		if bestCandidate == nil || candidate.Score() > bestCandidate.Score() {
			bestCandidate = candidate
		}
	}
	bestCandidate.Expand(b.TransFunc)
	// log.Println("Beam's Best:\n", bestCandidate)
	// sort.Sort(agendaHeap)
	// b.DurTop += time.Since(start)
	return bestCandidate
}

func (b *Beam) SetEarlyUpdate(i int) {
	b.EarlyUpdateAt = i
}

func (b *Beam) GoalTest(p Problem, c Candidate, rounds int) bool {
	if c != nil {
		c.(*ScoredConfiguration).Expand(b.TransFunc)
		conf := c.(*ScoredConfiguration).C
		return conf.Terminal()
	} else {
		return false
	}
}

func (b *Beam) TopB(a Agenda, B int) ([]Candidate, bool) {
	// start := time.Now()
	agenda := a.(*BaseAgenda).Confs
	candidates := make([]Candidate, len(agenda))
	allTerminal := true
	for i, candidate := range agenda {
		candidates[i] = candidate
		allTerminal = allTerminal && candidate.Terminal()
	}
	// assume agenda heap is already size of beam
	// agendaHeap := a.(heap.Interface)
	// rlheap.Init(agendaHeap)
	// for i := 0; i < B; i++ {
	// 	if len(a.(*BaseAgenda).Confs) > 0 {
	// 		candidate := rlheap.Pop(agendaHeap).(Candidate)
	// 		candidates = append(candidates, candidate)
	// 	} else {
	// 		break
	// 	}
	// }

	// concurrent expansion
	var wg sync.WaitGroup
	for _, candidate := range candidates {
		wg.Add(1)
		go func(c Candidate) {
			defer wg.Done()
			c.(*ScoredConfiguration).Expand(b.TransFunc)
		}(candidate)
		if !b.Concurrent() {
			wg.Wait()
		}
	}
	wg.Wait()
	// b.DurTopB += time.Since(start)
	return candidates, allTerminal
}

func (b *Beam) Parse(problem Problem) (transition.Configuration, interface{}) {
	start := time.Now()
	prefix := log.Prefix()
	// log.SetPrefix("Parsing ")
	// log.Println("Starting parse")
	beamScored := Search(b, problem, b.Size).(*ScoredConfiguration)
	// build result parameters
	var resultParams *ParseResultParameters
	if b.ReturnModelValue || b.ReturnSequence {
		resultParams = new(ParseResultParameters)
		if b.ReturnModelValue {
			resultParams.ModelValue = beamScored.Features
		}
		if b.ReturnSequence {
			resultParams.Sequence = beamScored.C.GetSequence()
		}
	}

	// log.Println("Time Expanding (pct):\t", b.DurExpanding.Nanoseconds(), 100*b.DurExpanding/b.DurTotal)
	// log.Println("Time Inserting (pct):\t", b.DurInserting.Nanoseconds(), 100*b.DurInserting/b.DurTotal)
	// log.Println("Time Inserting-Feat (pct):\t", b.DurInsertFeat.Nanoseconds(), 100*b.DurInsertFeat/b.DurTotal)
	// log.Println("Time Inserting-Scor (pct):\t", b.DurInsertScor.Nanoseconds(), 100*b.DurInsertScor/b.DurTotal)
	// log.Println("Total Time:", b.DurTotal.Nanoseconds())
	// log.Println("\n", beamScored.C.GetSequence())
	log.SetPrefix(prefix)
	b.DurTotal += time.Since(start)
	return beamScored.C, resultParams
}

func (b *Beam) DecodeEarlyUpdate(goldInstance perceptron.DecodedInstance, m perceptron.Model) (perceptron.DecodedInstance, interface{}, interface{}, int, int, float64) {
	b.EarlyUpdateAt = -1
	start := time.Now()
	prefix := log.Prefix()
	// log.SetPrefix("Training ")
	// log.Println("Starting decode")
	if goldInstance == nil {
		return nil, nil, nil, 0, 0, 0
	}
	sent := goldInstance.Instance()
	b.Model = m.(TransitionModel.Interface)

	// abstract casting >:-[

	goldSequence := goldInstance.Decoded().(ScoredConfigurations)
	// log.Println("Gold sequence")
	// log.Println(goldSequence[len(goldSequence)-1].C.GetSequence())
	b.ReturnModelValue = true

	// log.Println("Begin search..")
	beamResult, goldResult := SearchEarlyUpdate(b, sent, b.Size, goldSequence)
	// log.Println("Search ended")

	beamScored := beamResult.(*ScoredConfiguration)
	beamScore := beamScored.Score()
	var (
		goldFeatures, parsedFeatures *transition.FeaturesList
		goldScored                   *ScoredConfiguration
	)
	if goldResult != nil {
		goldScored = goldResult.(*ScoredConfiguration)
		goldFeatures = goldScored.Features
		parsedFeatures = beamScored.Features
		beamLastFeatures := b.FeatExtractor.Features(beamScored.C, false, beamScored.Transition.Type(), nil) //maybe wrong, what if it was idle?
		parsedFeatures = &transition.FeaturesList{beamLastFeatures, beamScored.Transition, beamScored.Features}
		// log.Println("Finding first wrong transition")
		// log.Println("Beam Conf")
		// log.Println(beamScored.C.GetSequence())
		// log.Println("Gold Conf")
		// log.Println(goldScored.C.GetSequence())
		parsedSeq, goldSeq := beamScored.C.GetSequence(), goldScored.C.GetSequence()
		var (
			i                  int
			parsedLen, goldLen int = len(parsedSeq), len(goldSeq)
			diffParsedGold     int = parsedLen - goldLen
		)

		for i = parsedLen - 1; i >= 0; i-- {
			// log.Println("At transition", parsedLen-i, "of", parsedLen-1)
			// log.Println(parsedSeq[i])
			// log.Println(goldSeq[i-diffParsedGold])
			if !parsedSeq[i].GetLastTransition().Equal(goldSeq[i-diffParsedGold].GetLastTransition()) {
				break
			}
		}
		// log.Println("Found", i)

		// log.Println("Rewinding")
		curBeamFeatures := parsedFeatures
		for j := 0; j <= i; j++ {
			// log.Println("At reverse transition", j)
			// log.Println(b.Transitions.ValueOf(int(curBeamFeatures.Transition)))
			// log.Println("\tFirst 6 features")
			// for k := 0; k < 6; k++ {
			// 	feat := b.Model.TransitionModel().(*TransitionModel.AvgMatrixSparse).Formatters[k]
			// 	log.Println("\t\t", feat, "=", feat.Format(curBeamFeatures.Previous.Features[k]))
			// }
			// log.Println(curGoldConf)
			// log.Println("\tFirst 6 features")
			// for k := 0; k < 6; k++ {
			// 	feat := b.Model.TransitionModel().(*TransitionModel.AvgMatrixSparse).Formatters[k]
			// 	log.Println("\t\t", feat, "=", feat.Format(curGoldFeatures.Previous.Features[k]))
			// }
			if curBeamFeatures != nil {
				curBeamFeatures = curBeamFeatures.Previous
			} else {
				break
			}
		}
		if curBeamFeatures != nil {
			curBeamFeatures.Previous = nil
		}
	}

	// parsedFeatures := beamScored.ModelValue.(*PerceptronModelValue).vector

	if b.Log {
		log.Println("Beam Sequence")
		log.Println("\n", beamScored.C.GetSequence().String())
		// log.Println("\n", parsedFeatures)
		if goldScored != nil {
			log.Println("Gold")
			log.Println("\n", goldScored.C.GetSequence().String())
			// log.Println("\n", goldFeatures)
		}
	}

	// if b.Log {
	// 	log.Println("Beam Weights")
	// 	log.Println(parsedFeatures)
	// 	log.Println("Gold Weights")
	// 	log.Println(goldFeatures)
	// }

	log.SetPrefix(prefix)
	b.DurTotal += time.Since(start)
	return &perceptron.Decoded{goldInstance.Instance(), beamScored.C}, parsedFeatures, goldFeatures, b.EarlyUpdateAt, len(goldSequence) - 1, beamScore
}

func (b *Beam) Aligned() bool {
	return b.Align
}

func (b *Beam) Idle(c Candidate, candidateNum int) Candidate {
	candidate := c.(*ScoredConfiguration)
	conf := candidate.C
	feats := b.FeatExtractor.Features(conf, true, 'I', nil)

	var newFeatList *transition.FeaturesList
	if b.ReturnModelValue {
		newFeatList = &transition.FeaturesList{feats, transition.IDLE, candidate.Features}
	} else {
		newFeatList = &transition.FeaturesList{feats, transition.IDLE, nil}
	}
	scores := b.candidateScorePool.Get().(featurevector.ScoredStore)
	scores.Clear()
	scores.SetTransitions([]int{transition.IDLE.Value()})
	scorer := b.Model.(*TransitionModel.AvgMatrixSparse)
	if b.DecodeTest {
		if b.ScoredStoreDense {
			scores.(*featurevector.ArrayStore).Generation = b.IntegrationGeneration
		} else {
			scores.(*featurevector.MapStore).Generation = b.IntegrationGeneration
		}
	}

	scorer.SetTransitionScores(feats, scores, b.DecodeTest)
	score, _ := scores.Get(transition.IDLE.Value())
	newConf := conf.Copy()
	newConf.SetLastTransition(transition.IDLE)
	scored := &ScoredConfiguration{newConf, transition.Transition(transition.IDLE), candidate.InternalScores.Copy(), newFeatList, 0, 0, true, candidate.Averaged}

	scored.AddScore(score, conf.Assignment())
	return scored
}

type AssignmentScore struct {
	Total  int64
	Number uint16
}

func (a *AssignmentScore) Add(value int64) {
	a.Total += value
	a.Number += 1
}

func (a *AssignmentScore) Average() float64 {
	if a.Number == 0 {
		return 0.0
	}
	return float64(a.Total) / float64(a.Number)
}

type ScoreState []AssignmentScore

func (s ScoreState) Copy() ScoreState {
	result := make([]AssignmentScore, len(s), len(s)+1)
	copy(result, s)
	return result
}

func (s ScoreState) Average() float64 {
	var result float64
	for _, value := range s {
		result += value.Average()
	}
	// log.Println("For Array", s)
	// log.Println("Returning average", result)
	return result
}

func (s ScoreState) Total() float64 {
	var result int64
	for _, value := range s {
		result += value.Total
	}
	// log.Println("For Array", s)
	// log.Println("Returning average", result)
	return float64(result)
}

func (s *ScoreState) Add(score int64, assignment uint16) {
	// log.Println("Adding", score, "to", assignment)
	assignmentInt := int(assignment)
	sLen := len(*s)
	if assignmentInt < sLen {
		(*s)[assignment].Add(score)
	} else if assignmentInt == sLen {
		*s = append(*s, AssignmentScore{score, 1})
	} else {
		scoreTail := make([]AssignmentScore, assignmentInt-sLen+1)
		for i, _ := range scoreTail {
			if i+sLen == assignmentInt {
				scoreTail[i].Add(score)
			}
		}
		*s = append(*s, scoreTail...)
	}
	// log.Println("After adding", s)
}

func NewScoreState() ScoreState {
	return []AssignmentScore{}
}

type ScoredConfiguration struct {
	C              transition.Configuration
	Transition     transition.Transition
	InternalScores ScoreState
	Features       *transition.FeaturesList

	CandidateNum, TransNum int
	Expanded               bool
	Averaged               bool
}

var _ Candidate = &ScoredConfiguration{}

type ScoredConfigurations []*ScoredConfiguration

var _ util.Equaler = ScoredConfigurations{}

func (scs ScoredConfigurations) Len() int {
	return len(scs)
}

func (scs ScoredConfigurations) Get(i int) Candidate {
	return scs[i]
}

func (scs ScoredConfigurations) Equal(otherEq util.Equaler) bool {
	switch other := otherEq.(type) {
	case Candidate:
		return scs[0].Equal(other)
	default:
		// log.Println("Equating", scs[len(scs)-1].C, "and", otherEq)
		// log.Println(scs[len(scs)-1].C.GetSequence())
		// log.Println(otherEq.GetSequence())
		return otherEq.Equal(scs[len(scs)-1].C)
		panic("Cannot compare to other")
	}
}

func (s *ScoredConfiguration) AddScore(newScore int64, assignment uint16) {
	(&(s.InternalScores)).Add(newScore, assignment)
}

func (s *ScoredConfiguration) Score() float64 {
	if s.Averaged {
		return s.InternalScores.Average()
	} else {
		return s.InternalScores.Total()
	}
}

func (s *ScoredConfiguration) Equal(otherEq Candidate) bool {
	other := otherEq.(*ScoredConfiguration)
	if s.Expanded {
		if !other.Expanded {
			return other.Equal(s)
		}
		return s.C.Equal(other.C)
	} else {
		if !other.Expanded {
			panic("Can't compare two unexpanded scored configurations")
		}
		log.Println("Checking last transition")
		return s.Transition.Equal(other.C.GetLastTransition()) && s.C.Equal(other.C.Previous())
	}
}

func (s *ScoredConfiguration) Clear() {
	s.C.Clear()
	s.C = nil
}

func (s *ScoredConfiguration) Copy() Candidate {
	newCand := &ScoredConfiguration{s.C, s.Transition, s.InternalScores.Copy(), s.Features, s.CandidateNum, s.TransNum, true, s.Averaged}
	return newCand
}

func (s *ScoredConfiguration) Expand(t transition.TransitionSystem) {
	if !s.Expanded {
		// if s.Transition != transition.IDLE {
		s.C = t.Transition(s.C, s.Transition)
		// }
		s.Expanded = true
	}
}

func (s *ScoredConfiguration) String() string {
	return s.C.String()
}

func (s *ScoredConfiguration) Alignment() int {
	if alignedConfiguration, aligned := s.C.(Aligned); aligned {
		return alignedConfiguration.Alignment()
	} else {
		panic("Configuration not aligned")
	}
}

func (s *ScoredConfiguration) Len() int {
	return s.C.Len()
}

func (s *ScoredConfiguration) Terminal() bool {
	return s.C.Terminal()
}

type BaseAgenda struct {
	sync.Mutex
	HeapReverse bool
	BeamSize    int
	Confs       []*ScoredConfiguration
	ShowSwap    bool
}

func (a *BaseAgenda) String() string {
	retval := make([]string, len(a.Confs))
	for i, conf := range a.Confs {
		retval[i] = fmt.Sprintf("%v:%v", conf.Transition, conf.Score())
	}
	return strings.Join(retval, ",")
}

func (a *BaseAgenda) AddCandidates(cs []Candidate, curBest Candidate, minAlignment int) (Candidate, int) {
	var (
		aligned   Aligned
		alignment int
		ok        bool
	)
	for _, c := range cs {
		curBest = a.AddCandidate(c, curBest)
		if aligned, ok = c.(*ScoredConfiguration).C.(Aligned); ok {
			if alignment = aligned.Alignment(); minAlignment < 0 || alignment < minAlignment {
				minAlignment = alignment
			}
		}
	}
	if a.Len() > a.BeamSize {
		log.Println("Agenda exceeded beam size")
	}
	return curBest, minAlignment
}

func (a *BaseAgenda) AddCandidate(c, best Candidate) Candidate {
	scored := c.(*ScoredConfiguration)
	if best != nil {
		bestScored := best.(*ScoredConfiguration)
		if scored.Score() > bestScored.Score() {
			best = scored
		}
	} else {
		best = c
	}
	if len(a.Confs) < a.BeamSize {
		if AgendaOut {
			log.Println("\t\tSpace left on Agenda, current size:", len(a.Confs))
			if len(a.Confs) > 0 {
				log.Println("\t\tFront was:", a.Confs[0].Transition, "score", a.Confs[0].Score())
			}
		}
		rlheap.Push(a, scored)
		// heap.Push(a, scored)
		if AgendaOut {
			if len(a.Confs) > 1 {
				log.Println("\t\tPushed onto Agenda", scored.Transition, "score", scored.Score())
			}
			// log.Println("\t\tAgenda post push", a.ConfStr(), ", ")
		}
		return best
	}
	peekScore := a.Peek()
	if !(peekScore.Score() < scored.Score()) {
		if AgendaOut {
			log.Println("\t\tNot pushed onto Agenda", scored.Transition, "score", scored.Score())
			log.Println("\t\tKeeping Current", peekScore.Transition, "score", peekScore.Score())
		}
		return best
	}

	if AgendaOut {
		// log.Println("\t\tAgenda pre pop", a.ConfStr(), ", ")
	}
	popped := rlheap.Pop(a).(*ScoredConfiguration)
	// popped := heap.Pop(a).(*ScoredConfiguration)
	if AgendaOut {
		log.Println("\t\tPopped off Agenda", popped.Transition, "score", popped.Score())
		// log.Println("\t\tAgenda post pop", a.ConfStr(), ", ")
	}
	// _ = rlheap.Pop(a).(*ScoredConfiguration)
	rlheap.Push(a, scored)
	// heap.Push(a, scored)
	if AgendaOut {
		log.Println("\t\tPushed onto Agenda", scored.Transition, "score", scored.Score())
		// log.Println("\t\tAgenda post push", a.ConfStr(), ", ")
	}
	return best
}

func (a *BaseAgenda) ReEnumerate() {
	for i, val := range a.Confs {
		val.CandidateNum = i
	}
}

func (a *BaseAgenda) CandidateStr(c *ScoredConfiguration) string {
	return fmt.Sprintf("%v:%v:%v", c.CandidateNum, c.Transition, c.Score())
}

func (a *BaseAgenda) ConfStr() string {
	// retval := make([]string, len(a.Confs)+1)
	// retval[0] = fmt.Sprintf("%v", len(a.Confs))
	retval := make([]string, len(a.Confs))
	// retval[0] = fmt.Sprintf("%v", len(a.Confs))
	for i, val := range a.Confs {
		// retval[i+1] = fmt.Sprintf("%v:%v", val.Transition, val.Score())
		retval[i] = a.CandidateStr(val)
	}
	return strings.Join(retval, " , ")
}

func (a *BaseAgenda) Len() int {
	return len(a.Confs)
}

func (a *BaseAgenda) Less(i, j int) bool {
	scoredI := a.Confs[i]
	scoredJ := a.Confs[j]
	// if a.ShowSwap {
	// 	log.Println("\tComparing ", a.CandidateStr(a.Confs[i]), a.CandidateStr(a.Confs[j]))
	// }
	return CompareConf(scoredI, scoredJ, a.HeapReverse)
}

func (a *BaseAgenda) Swap(i, j int) {
	// if a.ShowSwap {
	// 	log.Println("Swapping ", a.CandidateStr(a.Confs[j]), a.CandidateStr(a.Confs[i]))
	// }
	a.Confs[i], a.Confs[j] = a.Confs[j], a.Confs[i]
}

func (a *BaseAgenda) Push(x interface{}) {
	scored := x.(*ScoredConfiguration)
	a.Confs = append(a.Confs, scored)
}

func (a *BaseAgenda) Pop() interface{} {
	n := len(a.Confs)
	scored := a.Confs[n-1]
	a.Confs = a.Confs[0 : n-1]
	return scored
}

func (a *BaseAgenda) Peek() *ScoredConfiguration {
	return a.Confs[0]
}

func (a *BaseAgenda) Contains(goldCandidate Candidate) bool {
	goldScored := goldCandidate.(*ScoredConfiguration)
	for _, candidate := range a.Confs {
		if candidate.Equal(goldScored) {
			return true
		}
	}
	return false
}

func (a *BaseAgenda) Clear() {
	if a.Confs != nil {
		// nullify all pointers
		// for _, candidate := range a.Confs {
		// 	candidate.Clear()
		// 	candidate = nil
		// }
		a.Confs = a.Confs[0:0]
	}
	a.HeapReverse = false
}

var _ Agenda = &BaseAgenda{}
var _ heap.Interface = &BaseAgenda{}

func NewAgenda(size int) *BaseAgenda {
	newAgenda := new(BaseAgenda)
	newAgenda.BeamSize = size
	newAgenda.Confs = make([]*ScoredConfiguration, 0, size)
	return newAgenda
}

func CompareConf(confA, confB *ScoredConfiguration, reverse bool) bool {
	// less in reverse, we want the highest scoring to be first in the heap
	if reverse {
		return confA.Score() > confB.Score()
	}
	return confA.Score() < confB.Score() // less in reverse, we want the highest scoring to be first in the heap
	var retval bool
	if reverse {
		return confA.Score() > confB.Score()
		// if confA.Score() > confB.Score() {
		// 	retval = true
		// }
		// if confA.Score() == confB.Score() {
		// 	if confA.CandidateNum > confB.CandidateNum {
		// 		retval = true
		// 	}
		// 	if confA.CandidateNum == confB.CandidateNum {
		// 		if confA.TransNum > confB.TransNum {
		// 			retval = true
		// 		}
		// 	}
		// }
	} else {
		return confA.Score() < confB.Score()
		// if confA.Score() < confB.Score() {
		// 	retval = true
		// }
		// if confA.Score() == confB.Score() {
		// 	if confA.CandidateNum > confB.CandidateNum {
		// 		retval = true
		// 	}
		// 	if confA.CandidateNum == confB.CandidateNum {
		// 		if confA.TransNum > confB.TransNum {
		// 			retval = true
		// 		}
		// 	}
		// }
	}
	// if reverse {
	return retval
	// } else {
	// 	return !retval
	// }
}

type ParseResultParameters struct {
	ModelValue interface{}
	Sequence   transition.ConfigurationSequence
}

func (a *BaseAgenda) Copy(i, j int) {
	a.Confs[j] = a.Confs[i]
}

func (a *BaseAgenda) Set(i int, x interface{}) {
	a.Confs[i] = x.(*ScoredConfiguration)
}

func (a *BaseAgenda) Get(i int) interface{} {
	return a.Confs[i]
}

func (a *BaseAgenda) LessValue(i int, x interface{}) bool {
	scoredI := a.Confs[i]
	return CompareConf(scoredI, x.(*ScoredConfiguration), a.HeapReverse)
}

func (a *BaseAgenda) Reverse() {
	l := len(a.Confs)
	for i := l/2 - 1; i >= 0; i-- {
		opp := l - 1 - i
		a.Swap(i, opp)
	}
}
