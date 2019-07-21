package transition

import (
	. "github.com/OnlpLab/yap/alg/transition"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
	// "reflect"
	// "testing"
)

var (
	TEST_EAGER_TRANSITIONS []nlp.DepRel = []nlp.DepRel{
		"SH",
		"LA-ATT",
		"SH",
		"LA-SBJ",
		"SH", // "RA-PRED",
		"SH",
		"LA-ATT",
		"RA-OBJ",
		"RA-ATT",
		"SH",
		"LA-ATT",
		"RA-PC",
		"RE",
		"RE",
		"RE",
		"RA-PU",
		"RE",
		"PR"}
	TEST_EAGER_ENUM_TRANSITIONS []Transition
)

func SetupEagerTransEnum() {
	TRANSITIONS_ENUM = util.NewEnumSet(len(TEST_RELATIONS)*2 + 2)
	_, _ = TRANSITIONS_ENUM.Add("NO")
	iSH, _ := TRANSITIONS_ENUM.Add("SH")
	iRE, _ := TRANSITIONS_ENUM.Add("RE")
	iPR, _ := TRANSITIONS_ENUM.Add("PR")
	SH = Transition(iSH)
	RE = Transition(iRE)
	PR = Transition(iPR)
	LA = PR + 1
	for _, transition := range TEST_RELATIONS {
		TRANSITIONS_ENUM.Add(string("LA-" + transition))
	}
	RA = Transition(TRANSITIONS_ENUM.Len())
	for _, transition := range TEST_RELATIONS {
		TRANSITIONS_ENUM.Add(string("RA-" + transition))
	}
	TEST_EAGER_ENUM_TRANSITIONS = make([]Transition, len(TEST_EAGER_TRANSITIONS))
	for i, transition := range TEST_EAGER_TRANSITIONS {
		index, _ := TRANSITIONS_ENUM.IndexOf(string(transition))
		TEST_EAGER_ENUM_TRANSITIONS[i] = Transition(index)
	}
}

// func SetupEagerEnum() {
// 	SetupEagerTransEnum()
// 	SetupTestEnum()
// }

// // // func TestArcEagerTransitions(t *testing.T) {
// // // 	SetupEagerEnum()
// // // 	conf := &SimpleConfiguration{
// // // 		EWord:  EWord,
// // // 		EPOS:   EPOS,
// // // 		EWPOS:  EWPOS,
// // // 		ERel:   TEST_ENUM_RELATIONS,
// // // 		ETrans: TRANSITIONS_ENUM,
// // // 	}

// // // 	conf.Init(TEST_SENT)

// // // 	var (
// // // 		transition, label int
// // // 		exists            bool
// // // 	)

// // // 	arcEag := &ArcEager{
// // // 		ArcStandard: ArcStandard{
// // // 			SHIFT:       SH,
// // // 			LEFT:        LA,
// // // 			RIGHT:       RA,
// // // 			Relations:   TEST_ENUM_RELATIONS,
// // // 			Transitions: TRANSITIONS_ENUM,
// // // 		},
// // // 		REDUCE:  RE,
// // // 		POPROOT: PR,
// // // 	}
// // // 	// SHIFT
// // // 	transition, exists = TRANSITIONS_ENUM.IndexOf("SH")
// // // 	if !exists {
// // // 		t.Fatal("Can't find transition SH")
// // // 	}
// // // 	shConf := arcEag.Transition(conf, Transition(transition)).(*SimpleConfiguration)
// // // 	if qPeek, qPeekExists := shConf.Queue().Peek(); !qPeekExists || qPeek != 2 {
// // // 		if !qPeekExists {
// // // 			t.Error("Expected N0")
// // // 		} else {
// // // 			t.Error("Expected N0 = 2, got", qPeek)
// // // 		}
// // // 	}
// // // 	if sPeek, sPeekExists := shConf.Stack().Peek(); !sPeekExists || sPeek != 1 {
// // // 		if !sPeekExists {
// // // 			t.Error("Expected S0")
// // // 		} else {
// // // 			t.Error("Expected S0 = 1, got", sPeek)
// // // 		}
// // // 	}
// // // 	// LA
// // // 	transition, exists = TRANSITIONS_ENUM.IndexOf("LA-ATT")
// // // 	if !exists {
// // // 		t.Fatal("Can't find transition LA-ATT")
// // // 	}
// // // 	laConf := arcEag.Transition(shConf, Transition(transition)).(*SimpleConfiguration)
// // // 	if sPeek, sPeekExists := laConf.Stack().Peek(); !sPeekExists || sPeek != 0 {
// // // 		if !sPeekExists {
// // // 			t.Error("Expected N0")
// // // 		} else {
// // // 			t.Error("Expected N0 = 0, got", sPeek)
// // // 		}
// // // 	}
// // // 	label, exists = TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("ATT"))
// // // 	if !exists {
// // // 		t.Fatal("Can't find label ATT")
// // // 	}
// // // 	if arcs := laConf.Arcs().Get(&BasicDepArc{2, label, 1, nlp.DepRel("ATT")}); len(arcs) != 1 {
// // // 		t.Error("Left arc not found, arcs: ", laConf.StringArcs())
// // // 	}
// // // 	recovered := false

// // // 	// LA checks conditions
// // // 	panicFunc := func() {
// // // 		defer func() {
// // // 			r := recover()
// // // 			recovered = r != nil
// // // 		}()
// // // 		_ = arcEag.Transition(laConf, Transition(LA))
// // // 	}
// // // 	panicFunc()
// // // 	if !recovered {
// // // 		t.Error("Did not panic when trying to Left-Arc with root as stack head")
// // // 	}
// // // 	// fast forward to RA
// // // 	interimTransitions := TEST_EAGER_ENUM_TRANSITIONS[2:4]
// // // 	c := Configuration(laConf)
// // // 	for _, transition := range interimTransitions {
// // // 		if transition >= RA {
// // // 			panic("Shouldn't execute untested transition")
// // // 		}
// // // 		c = arcEag.Transition(c, Transition(transition))
// // // 	}
// // // 	// TODO: update transitions for POPROOT
// // // 	// RA
// // // 	transition, exists = TRANSITIONS_ENUM.IndexOf("RA-PRED")
// // // 	if !exists {
// // // 		t.Fatal("Can't find transition RA-PRED")
// // // 	}
// // // 	raConf := arcEag.Transition(c, Transition(transition)).(*SimpleConfiguration)
// // // 	if qPeek, qPeekExists := raConf.Queue().Peek(); !qPeekExists || qPeek != 4 {
// // // 		if !qPeekExists {
// // // 			t.Error("Expected N0")
// // // 		} else {
// // // 			t.Error("Expected N0 == 4, got", qPeek)
// // // 		}
// // // 	}
// // // 	if sPeek, sPeekExists := raConf.Stack().Peek(); !sPeekExists || sPeek != 3 {
// // // 		if !sPeekExists {
// // // 			t.Error("Expected S0")
// // // 		} else {
// // // 			t.Error("Expected S0 == 3, got", sPeek)
// // // 		}
// // // 	}
// // // 	label, exists = TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("PRED"))
// // // 	if arcs := raConf.Arcs().Get(&BasicDepArc{0, label, 3, nlp.DepRel("PRED")}); len(arcs) != 1 {
// // // 		t.Error("Right arc not found")
// // // 	}
// // // }

// // func TestArcEagerOracle(t *testing.T) {
// // 	SetupEagerEnum()

// // 	goldGraph := GetTestDepGraph()

// // 	conf := Configuration(&SimpleConfiguration{
// // 		EWord:  EWord,
// // 		EPOS:   EPOS,
// // 		EWPOS:  EWPOS,
// // 		ERel:   TEST_ENUM_RELATIONS,
// // 		ETrans: TRANSITIONS_ENUM,
// // 	})

// // 	conf.Init(TEST_SENT)

// // 	arcEag := &ArcEager{
// // 		ArcStandard: ArcStandard{
// // 			SHIFT:       SH,
// // 			LEFT:        LA,
// // 			RIGHT:       RA,
// // 			Relations:   TEST_ENUM_RELATIONS,
// // 			Transitions: TRANSITIONS_ENUM,
// // 		},
// // 		REDUCE:  RE,
// // 		POPROOT: PR,
// // 	}
// // 	arcEag.AddDefaultOracle()
// // 	oracle := arcEag.Oracle()
// // 	oracle.SetGold(goldGraph)
// // 	for i, expected := range TEST_EAGER_ENUM_TRANSITIONS {
// // 		transition := oracle.Transition(conf)

// // 		if transition != expected {
// // 			t.Error("Oracle failed at transition", i, "expected", TRANSITIONS_ENUM.ValueOf(int(expected)).(string), "got", TRANSITIONS_ENUM.ValueOf(int(transition)).(string))
// // 		}
// // 		conf = arcEag.Transition(conf, Transition(transition))
// // 	}
// // 	if !conf.Terminal() {
// // 		t.Error("Configuration should be terminal at end of expected transition sequence")
// // 	}

// // 	expectedArcSet := NewArcSetSimpleFromGraph(goldGraph)
// // 	if !expectedArcSet.Equal(conf.(*SimpleConfiguration).Arcs()) {
// // 		t.Error("Oracle/Gold parsing resulted in wrong dependency graph")
// // 	}
// // }

// // func TestArcEagerEsotericFunctions(t *testing.T) {
// // 	arcEag := new(ArcEager)
// // 	transitions := arcEag.TransitionTypes()
// // 	if !reflect.DeepEqual(transitions, []string{"LA-*", "RA-*", "SH", "RE", "PR"}) {
// // 		t.Error("Wrong transition types")
// // 	}

// // }
