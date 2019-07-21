package featurevector

import "github.com/OnlpLab/yap/util"

// import "fmt"
// import "log"

// import "encoding/gob"

// type GobCodec interface {
// 	gob.GobEncoder
// 	gob.GobDecoder
// }

// type Feature GobCodec

const BASE_SIZE int = 10

var (
	zeroInts   [BASE_SIZE]int
	zeroInt64s [BASE_SIZE]int64
)

type Feature interface{}

type FeatureTransMap map[Feature]map[int]bool

type TAF interface {
	GetTransFeatures() FeatureTransMap
}

type SimpleTAF struct {
	FTMap FeatureTransMap
}

func (s *SimpleTAF) GetTransFeatures() FeatureTransMap {
	return s.FTMap
}

type ScoredStore interface {
	Get(transition int) (int64, bool)
	Set(transition int, score int64)
	SetTransitions(transitions []int)
	IncAll(store TransitionScoreStore, integrated bool)
	Inc(transition int, score int64)
	Len() int
	Clear()
	Init()
}

type ArrayStore struct {
	Generation           int
	Data                 []int64
	DataArray, zeroArray []int64
}

func (s *ArrayStore) Get(transition int) (int64, bool) {
	if transition < len(s.DataArray) {
		return s.DataArray[transition], true
	}
	return 0, false
}
func (s *ArrayStore) Set(transition int, score int64) {
	if len(s.DataArray) < transition {
		s.DataArray[transition] = score
	}
}

func (s *ArrayStore) SetTransitions(transitions []int) {
	slots := util.MaxInt(transitions) + 1
	if len(s.DataArray) <= slots {
		s.Data = make([]int64, slots)
		s.zeroArray = make([]int64, slots)
	}
	s.DataArray = s.Data[:slots]
}

func (s *ArrayStore) IncAll(store TransitionScoreStore, integrated bool) {
	var val *HistoryValue
	// log.Println("\t\tIncrementing for", len(s.DataArray), "transitions")
	for i, _ := range s.DataArray {
		val = store.GetValue(i)
		if val != nil {
			if integrated {
				s.DataArray[i] += val.IntegratedValue(s.Generation)
			} else {
				// log.Println("\t\t\tIncrementing score for transition", i, "to", s.DataArray[i]+val.Value)
				s.DataArray[i] += val.Value
			}
		}
	}
}

func (s *ArrayStore) Inc(transition int, score int64) {
	if len(s.DataArray) < transition {
		s.DataArray[transition] += score
	}
}

func (s *ArrayStore) Len() int {
	return len(s.DataArray)
}

func (s *ArrayStore) Clear() {
	copy(s.Data, s.zeroArray)
}

func (s *ArrayStore) Init() {
	s.Data = make([]int64, 1)
}

type MapStore struct {
	Generation int
	tArray     [BASE_SIZE]int
	sArray     [BASE_SIZE]int64
	// dataMap map[int]int64
	transitions []int
	scores      []int64
}

func (s *MapStore) ScoreMap() map[int]int64 {
	retval := make(map[int]int64, len(s.transitions))
	for i, t := range s.transitions {
		retval[t] = s.scores[i]
	}
	return retval
}
func (s *MapStore) Get(transition int) (int64, bool) {
	// if len(s.scores) != len(s.transitions) {
	// 	panic(fmt.Sprintf("Got different lengths: scores %v vs transitions %v", len(s.scores), len(s.transitions)))
	// }
	for i, val := range s.transitions {
		if val == transition {
			return s.scores[i], true
		}
	}
	return 0, false
	// retVal, exists := s.dataMap[transition]
	// return retVal, exists
}
func (s *MapStore) Set(transition int, score int64) {
	for i, val := range s.transitions {
		if val == transition {
			s.scores[i] = score
		}
	}
	s.scores = append(s.scores, score)
	s.transitions = append(s.transitions, transition)
	// s.dataMap[transition] = score
}

func (s *MapStore) SetTransitions(transitions []int) {
	if cap(s.transitions) < len(transitions) {
		s.transitions = make([]int, len(transitions))
		s.scores = make([]int64, len(transitions))
	} else {
		s.transitions = s.tArray[:len(transitions)]
		s.scores = s.sArray[:len(transitions)]
	}
	copy(s.transitions, transitions)
}
func (s *MapStore) Inc(transition int, score int64) {
	// if len(s.scores) != len(s.transitions) {
	// 	panic(fmt.Sprintf("Got different lengths: scores %v vs transitions %v", len(s.scores), len(s.transitions)))
	// }
	for i, val := range s.transitions {
		if val == transition {
			s.scores[i] += score
		}
	}
	// if cur, exists := s.dataMap[transition]; exists {
	// 	s.dataMap[transition] = cur + score
	// } else {
	// 	s.Set(transition, score)
	// }
}

func (s *MapStore) IncAll(store TransitionScoreStore, integrated bool) {
	var val *HistoryValue
	for i, transition := range s.transitions {
		val = store.GetValue(transition)
		if val != nil {
			if integrated {
				s.scores[i] += val.IntegratedValue(s.Generation)
			} else {
				s.scores[i] += val.Value
			}
		}
	}
}

func (s *MapStore) Len() int {
	return len(s.scores)
}

func (s *MapStore) Clear() {
	copy(s.tArray[0:BASE_SIZE], zeroInts[:BASE_SIZE])
	copy(s.sArray[0:BASE_SIZE], zeroInt64s[:BASE_SIZE])
	s.transitions = s.tArray[0:0]
	s.scores = s.sArray[0:0]
	// s.dataMap = make(map[int]int64, 5)
}

func (s *MapStore) Init() {
	// s.transitions = make([]int, 0, 5)
	// s.scores = make([]int64, 0, 5)
	s.transitions = s.tArray[0:0]
	s.scores = s.sArray[0:0]
}

type HybridStore struct {
	cutoff int
	ArrayStore
	MapStore
}

func (s *HybridStore) Get(transition int) (int64, bool) {
	if transition < s.cutoff {
		return s.ArrayStore.Get(transition)
	} else {
		return s.MapStore.Get(transition)
	}
}
func (s *HybridStore) Set(transition int, score int64) {
	if transition < s.cutoff {
		s.ArrayStore.Set(transition, score)
	} else {
		s.MapStore.Set(transition, score)
	}
}

func (s *HybridStore) SetTransitions(transitions []int) {

	arrayTransitions := make([]int, 0, len(transitions))
	mapTransitions := make([]int, 0, 5)
	for _, transition := range transitions {
		if transition < s.cutoff {
			arrayTransitions = append(arrayTransitions, transition)
		} else {
			mapTransitions = append(mapTransitions, transition)
		}
	}
	s.ArrayStore.SetTransitions(arrayTransitions)
	s.MapStore.SetTransitions(mapTransitions)
}

func (s *HybridStore) Inc(transition int, score int64) {
	if transition < s.cutoff {
		s.ArrayStore.Inc(transition, score)
	} else {
		s.MapStore.Inc(transition, score)
	}
}

func (s *HybridStore) IncAll(store TransitionScoreStore, integrated bool) {
	s.ArrayStore.IncAll(store, integrated)
	s.MapStore.IncAll(store, integrated)
}
func (s *HybridStore) Len() int {
	return s.ArrayStore.Len() + s.MapStore.Len()
}

func (s *HybridStore) Clear() {
	s.ArrayStore.Clear()
	s.MapStore.Clear()
}

func (s *HybridStore) Init() {
	s.ArrayStore.Init()
	s.MapStore.Init()
}

var (
	_ ScoredStore = &ArrayStore{}
	_ ScoredStore = &MapStore{}
	_ ScoredStore = &HybridStore{}
)

func MakeScoredStore(dense bool) interface{} {
	var s ScoredStore
	if dense {
		s = &ArrayStore{}
	} else {
		s = &MapStore{}
	}
	s.Init()
	return s
}

func MakeDenseStore() interface{} {
	return MakeScoredStore(true)
}
func MakeMapStore() interface{} {
	return MakeScoredStore(false)
}
