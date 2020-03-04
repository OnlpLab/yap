package transition

// TODO: fix arc standard for poproot
// import (
// 	. "github.com/OnlpLab/yap/alg/transition"
// 	nlp "github.com/OnlpLab/yap/nlp/types"
// 	"github.com/OnlpLab/yap/util"
// 	"reflect"
// 	"testing"
// )

// var (
// 	TEST_STANDARD_TRANSITIONS []nlp.DepRel = []nlp.DepRel{
// 		"SH",
// 		"LA-ATT",
// 		"SH",
// 		"LA-SBJ",
// 		"SH",
// 		"SH",
// 		"LA-ATT",
// 		"SH",
// 		"SH",
// 		"SH",
// 		"LA-ATT",
// 		"RA-PC",
// 		"RA-ATT",
// 		"RA-OBJ",
// 		"SH",
// 		"RA-PU",
// 		"RA-PRED",
// 		"SH"}
// 	TEST_STANDARD_ENUM_TRANSITIONS []Transition
// )

// func SetupStandardTransEnum() {
// 	TRANSITIONS_ENUM = util.NewEnumSet(len(TEST_RELATIONS)*2 + 2)
// 	iSH, _ := TRANSITIONS_ENUM.Add(nlp.DepRel("SH"))
// 	SH = Transition(iSH)
// 	LA = SH + 1
// 	for _, transition := range TEST_RELATIONS {
// 		TRANSITIONS_ENUM.Add(nlp.DepRel("LA-" + transition))
// 	}
// 	RA = Transition(TRANSITIONS_ENUM.Len())
// 	for _, transition := range TEST_RELATIONS {
// 		TRANSITIONS_ENUM.Add(nlp.DepRel("RA-" + transition))
// 	}

// 	TEST_STANDARD_ENUM_TRANSITIONS = make([]Transition, len(TEST_STANDARD_TRANSITIONS))
// 	for i, transition := range TEST_STANDARD_TRANSITIONS {
// 		index, _ := TRANSITIONS_ENUM.IndexOf(transition)
// 		TEST_STANDARD_ENUM_TRANSITIONS[i] = Transition(index)
// 	}

// }

// func SetupStandardEnum() {
// 	SetupStandardTransEnum()
// 	SetupTestEnum()
// }

// func TestArcStandardTransitions(t *testing.T) {
// 	SetupStandardEnum()
// 	conf := &SimpleConfiguration{
// 		EWord:  EWord,
// 		EPOS:   EPOS,
// 		EWPOS:  EWPOS,
// 		ERel:   TEST_ENUM_RELATIONS,
// 		ETrans: TRANSITIONS_ENUM,
// 	}

// 	conf.Init(TEST_SENT)

// 	var (
// 		transition, label int
// 		exists            bool
// 	)

// 	arcStd := &ArcStandard{
// 		SHIFT:       SH,
// 		LEFT:        LA,
// 		RIGHT:       RA,
// 		Relations:   TEST_ENUM_RELATIONS,
// 		Transitions: TRANSITIONS_ENUM,
// 	}

// 	// SHIFT
// 	transition, exists = TRANSITIONS_ENUM.IndexOf(nlp.DepRel("SH"))
// 	if !exists {
// 		t.Fatal("Can't find transition SH")
// 	}
// 	shConf := arcStd.Transition(conf, Transition(transition)).(*SimpleConfiguration)
// 	if qPeek, qPeekExists := shConf.Queue().Peek(); !qPeekExists || qPeek != 2 {
// 		if !qPeekExists {
// 			t.Error("Expected N0")
// 		} else {
// 			t.Error("Expected N0 = 2, got", qPeek)
// 		}
// 	}
// 	if sPeek, sPeekExists := shConf.Stack().Peek(); !sPeekExists || sPeek != 1 {
// 		if !sPeekExists {
// 			t.Error("Expected S0")
// 		} else {
// 			t.Error("Expected S0 = 1, got", sPeek)
// 		}
// 	}
// 	// LA
// 	transition, exists = TRANSITIONS_ENUM.IndexOf(nlp.DepRel("LA-ATT"))
// 	if !exists {
// 		t.Fatal("Can't find transition for LA-ATT")
// 	}
// 	laConf := arcStd.Transition(shConf, Transition(transition)).(*SimpleConfiguration)
// 	if sPeek, sPeekExists := laConf.Stack().Peek(); !sPeekExists || sPeek != 0 {
// 		if !sPeekExists {
// 			t.Error("Expected N0")
// 		} else {
// 			t.Error("Expected N0 = 0, got", sPeek)
// 		}
// 	}
// 	label, exists = TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("ATT"))
// 	if !exists {
// 		t.Fatal("Can't find label ATT")
// 	}
// 	if arcs := laConf.Arcs().Get(&BasicDepArc{2, label, 1, nlp.DepRel("ATT")}); len(arcs) != 1 {
// 		t.Error("Left arc not found, arcs: ", laConf.StringArcs())
// 	}
// 	recovered := false

// 	// LA checks conditions
// 	panicFunc := func() {
// 		defer func() {
// 			r := recover()
// 			recovered = r != nil
// 		}()
// 		_ = arcStd.Transition(laConf, Transition(LA))
// 	}
// 	panicFunc()
// 	if !recovered {
// 		t.Error("Did not panic when trying to Left-Arc with root as stack head")
// 	}
// 	// fast forward to RA
// 	interimTransitions := TEST_STANDARD_ENUM_TRANSITIONS[2:11]
// 	c := Configuration(laConf)
// 	for _, transition := range interimTransitions {
// 		if transition >= RA {
// 			panic("Shouldn't execute untested transition")
// 		}
// 		c = arcStd.Transition(c, Transition(transition))
// 	}
// 	// RA
// 	transition, exists = TRANSITIONS_ENUM.IndexOf(nlp.DepRel("RA-PC"))
// 	raConf := arcStd.Transition(c, Transition(transition)).(*SimpleConfiguration)
// 	if qPeek, qPeekExists := raConf.Queue().Peek(); !qPeekExists || qPeek != 6 {
// 		if !qPeekExists {
// 			t.Error("Expected N0")
// 		} else {
// 			t.Error("Expected N0 == 6, to", qPeek)
// 		}
// 	}
// 	if sPeek, sPeekExists := raConf.Stack().Peek(); !sPeekExists || sPeek == 6 {
// 		if !sPeekExists {
// 			t.Error("Expected N0")
// 		} else {
// 			t.Error("Expected N0 != 6")
// 		}
// 	}
// 	label, exists = TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("PC"))
// 	if arcs := raConf.Arcs().Get(&BasicDepArc{6, label, 8, nlp.DepRel("PC")}); len(arcs) != 1 {
// 		t.Error("Right arc not found")
// 	}
// }

// func TestArcStandardOracle(t *testing.T) {
// 	goldGraph := GetTestDepGraph()

// 	conf := Configuration(&SimpleConfiguration{
// 		EWord:  EWord,
// 		EPOS:   EPOS,
// 		EWPOS:  EWPOS,
// 		ERel:   TEST_ENUM_RELATIONS,
// 		ETrans: TRANSITIONS_ENUM,
// 	})
// 	conf.Init(TEST_SENT)

// 	arcStd := &ArcStandard{
// 		SHIFT:       SH,
// 		LEFT:        LA,
// 		RIGHT:       RA,
// 		Relations:   TEST_ENUM_RELATIONS,
// 		Transitions: TRANSITIONS_ENUM,
// 	}

// 	arcStd.AddDefaultOracle()
// 	oracle := arcStd.Oracle()
// 	oracle.SetGold(goldGraph)
// 	for i, expected := range TEST_STANDARD_ENUM_TRANSITIONS {
// 		transition := oracle.Transition(conf)
// 		if transition != expected {
// 			t.Error("Oracle failed at transition", i, "expected", TRANSITIONS_ENUM.ValueOf(int(expected)).(nlp.DepRel), "got", TRANSITIONS_ENUM.ValueOf(int(transition)).(nlp.DepRel))
// 		}
// 		conf = arcStd.Transition(conf, Transition(transition))
// 	}
// 	if !conf.Terminal() {
// 		t.Error("Configuration should be terminal at end of expected transition sequence")
// 	}

// 	expectedArcSet := NewArcSetSimpleFromGraph(goldGraph)
// 	if !expectedArcSet.Equal(conf.(*SimpleConfiguration).Arcs()) {
// 		t.Error("Oracle/Gold parsing resulted in wrong dependency graph")
// 	}
// }

// func TestArcStandardEsotericFunctions(t *testing.T) {
// 	arcStd := new(ArcStandard)
// 	if !arcStd.Projective() {
// 		t.Error("Not projective")
// 	}
// 	if !arcStd.Labeled() {
// 		t.Error("Not labeled")
// 	}
// 	transitions := arcStd.TransitionTypes()
// 	if !reflect.DeepEqual(transitions, []string{"LA-*", "RA-*", "SH"}) {
// 		t.Error("Wrong transition types")
// 	}
// }
