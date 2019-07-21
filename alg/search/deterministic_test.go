package search

import (
	"github.com/OnlpLab/yap/alg/featurevector"

	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/transition"
	TransitionModel "github.com/OnlpLab/yap/alg/transition/model"
	"github.com/OnlpLab/yap/nlp/parser/dependency"
	"github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
	// "fmt"
	"log"
	"runtime"
	"sort"
	"testing"
)

func PrintGraph(graph types.LabeledDependencyGraph) {
	arcIndex := make(map[int]types.LabeledDepArc, graph.NumberOfNodes())
	var (
		// posTag string
		node   types.DepNode
		arc    types.LabeledDepArc
		headID int
		depRel string
	)
	for _, arcID := range graph.GetEdges() {
		arc = graph.GetLabeledArc(arcID)
		if arc == nil {
			// panic("Can't find arc")
		} else {
			arcIndex[arc.GetModifier()] = arc
		}
	}
	for _, nodeID := range graph.GetVertices() {
		node = graph.GetNode(nodeID)
		// posTag = ""

		// taggedToken, ok := node.(*TaggedDepNode)
		// if ok {
		// 	// posTag = taggedToken.RawPOS
		// }

		if node == nil {
			panic("Can't find node")
		}
		arc, exists := arcIndex[node.ID()]
		if exists {
			log.Println("Exists")
			headID = arc.GetHead()
			depRel = string(arc.GetRelation())
			if depRel == types.ROOT_LABEL {
				headID = -1
			}
		} else {
			log.Println("Not Exists")
			headID = -1
			depRel = "None"
		}
		log.Println(node.ID()+1, node.String(), headID+1, depRel)
	}
}

func TestDeterministic(t *testing.T) {
	SetupTestEnum()
	SetupEagerTransEnum()
	runtime.GOMAXPROCS(runtime.NumCPU())
	extractor := &GenericExtractor{
		EFeatures: util.NewEnumSet(len(TEST_RICH_FEATURES)),
		EWord:     EWord,
		EPOS:      EPOS,
		EWPOS:     EWPOS,
		ERel:      TEST_ENUM_RELATIONS,
	}
	extractor.Init()
	// verify load
	for _, featurePair := range TEST_RICH_FEATURES {
		if err := extractor.LoadFeature(featurePair[0], featurePair[1]); err != nil {
			t.Error("Failed to load feature", err.Error())
			t.FailNow()
		}
	}
	arcSystem := &ArcStandard{
		SHIFT:       SH,
		LEFT:        LA,
		RIGHT:       RA,
		Relations:   TEST_ENUM_RELATIONS,
		Transitions: TRANSITIONS_ENUM,
	}

	// arcSystem := &ArcEager{
	// 	ArcStandard: ArcStandard{
	// 		SHIFT:       SH,
	// 		LEFT:        LA,
	// 		RIGHT:       RA,
	// 		Relations:   TEST_ENUM_RELATIONS,
	// 		Transitions: TRANSITIONS_ENUM,
	// 	},
	// 	REDUCE:  RE,
	// 	POPROOT: PR,
	// }
	arcSystem.AddDefaultOracle()
	transitionSystem := transition.TransitionSystem(arcSystem)

	conf := &SimpleConfiguration{
		EWord:  EWord,
		EPOS:   EPOS,
		EWPOS:  EWPOS,
		ERel:   TEST_ENUM_RELATIONS,
		ETrans: TRANSITIONS_ENUM,
	}

	deterministic := &Deterministic{
		TransFunc:          transitionSystem,
		FeatExtractor:      extractor,
		ReturnModelValue:   true,
		ReturnSequence:     true,
		ShowConsiderations: false,
		Base:               conf,
		NoRecover:          true,
	}
	decoder := perceptron.EarlyUpdateInstanceDecoder(deterministic)
	goldDecoder := perceptron.InstanceDecoder(deterministic)
	updater := new(TransitionModel.AveragedModelStrategy)

	model := TransitionModel.NewAvgMatrixSparse(extractor.EFeatures.Len(), nil)
	perceptronInstance := &perceptron.LinearPerceptron{Decoder: decoder, GoldDecoder: goldDecoder, Updater: updater}
	perceptronInstance.Init(model)
	goldModel := dependency.TransitionParameterModel(&PerceptronModel{model})

	goldGraph, goldParams := deterministic.ParseOracle(GetTestDepGraph(), nil, goldModel)
	if goldParams == nil {
		t.Fatal("Got nil params from deterministic oracle parsing, can't test deterministic-perceptron model")
	}
	seq := goldParams.(*ParseResultParameters).Sequence
	log.Println("\n", seq.String())
	goldSequence := make(ScoredConfigurations, len(seq))
	var (
		lastFeatures *transition.FeaturesList
		curFeats     []featurevector.Feature
	)
	// extractor.Log = true
	for i := len(seq) - 1; i >= 0; i-- {
		// for i := 0; i < len(seq); i++ {
		val := seq[i]
		// log.Println("Conf:", val)
		curFeats = extractor.Features(val)
		// log.Printf("\t%d %s %v\n", i, "Features:", curFeats)
		lastFeatures = &transition.FeaturesList{curFeats, val.GetLastTransition(), lastFeatures}
		goldSequence[len(seq)-i-1] = &ScoredConfiguration{val.(DependencyConfiguration), val.GetLastTransition(), 0.0, lastFeatures, 0, 0, true}
	}
	t.Errorf("bla")
	goldDirected := goldGraph.(types.LabeledDependencyGraph)
	for i := 0; i <= goldDirected.NumberOfArcs(); i++ {
		arc := goldDirected.GetLabeledArc(i)
		log.Println("Arc", i, arc)
	}

	goldInstances := []perceptron.DecodedInstance{
		&perceptron.Decoded{perceptron.Instance(rawTestSent), GetTestDepGraph()}}
	// log.Println(goldSequence)
	// train with increasing iterations
	// convergenceIterations := []int{1, 8, 16, 24, 32}
	// deterministic.ShowConsiderations = true
	convergenceIterations := []int{32}
	convergenceSharedSequence := make([]int, 0, len(convergenceIterations))
	for _, iterations := range convergenceIterations {
		perceptronInstance.Iterations = iterations
		// perceptron.Log = true
		model = TransitionModel.NewAvgMatrixSparse(extractor.EFeatures.Len(), nil)
		perceptronInstance.Init(model)

		// deterministic.ShowConsiderations = true
		perceptronInstance.Train(goldInstances)

		parseModel := dependency.TransitionParameterModel(&PerceptronModel{model})
		deterministic.ShowConsiderations = false
		graph, params := deterministic.Parse(TEST_SENT, nil, parseModel)
		labeledGraph := graph.(types.LabeledDependencyGraph)
		seq := params.(*ParseResultParameters).Sequence
		log.Println("\n", seq.String())
		PrintGraph(labeledGraph)
		sharedSteps := goldSequence[len(goldSequence)-1].C.GetSequence().SharedTransitions(seq)
		convergenceSharedSequence = append(convergenceSharedSequence, sharedSteps)
	}

	// verify convergence
	log.Println(convergenceSharedSequence)
	if !sort.IntsAreSorted(convergenceSharedSequence) || convergenceSharedSequence[0] == convergenceSharedSequence[len(convergenceSharedSequence)-1] {
		t.Error("Model not converging, shared sequences lengths:", convergenceSharedSequence)
	}
}

func TestArrayDiff(t *testing.T) {
	left := []featurevector.Feature{"def", "abc"}
	right := []featurevector.Feature{"def", "ghi"}
	oLeft, oRight := ArrayDiff(left, right)
	if len(oLeft) != 1 {
		t.Error("Wrong len for oLeft", oLeft)
	}
	if len(oRight) != 1 {
		t.Error("Wrong len for oRight", oRight)
	}
	if len(oLeft) > 0 && oLeft[0] != "abc" {
		t.Error("Didn't get abc for oLeft")
	}
	if len(oRight) > 0 && oRight[0] != "ghi" {
		t.Error("Didn't get ghi for oRight")
	}
}
