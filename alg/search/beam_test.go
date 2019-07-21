package search

// import (
// 	"github.com/OnlpLab/yap/alg/featurevector"
// 	"github.com/OnlpLab/yap/alg/perceptron"
// 	"github.com/OnlpLab/yap/alg/transition"
// 	TransitionModel "github.com/OnlpLab/yap/alg/transition/model"
// 	"github.com/OnlpLab/yap/nlp/parser/dependency"
// 	"github.com/OnlpLab/yap/util"
// 	"log"
// 	"runtime"
// 	"sort"
// 	"testing"
// )

// func TestBeam(t *testing.T) {
// 	SetupEagerTransEnum()
// 	SetupTestEnum()
// 	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
// 	// runtime.GOMAXPROCS(1)
// 	runtime.GOMAXPROCS(runtime.NumCPU())
// 	extractor := &GenericExtractor{
// 		EFeatures: util.NewEnumSet(len(TEST_RICH_FEATURES)),
// 	}
// 	extractor.Init()
// 	// verify load
// 	for _, featurePair := range TEST_RICH_FEATURES {
// 		if err := extractor.LoadFeature(featurePair[0], featurePair[1]); err != nil {
// 			t.Error("Failed to load feature", err.Error())
// 			t.FailNow()
// 		}
// 	}
// 	arcSystem := &ArcEager{
// 		ArcStandard: ArcStandard{
// 			SHIFT:       SH,
// 			LEFT:        LA,
// 			RIGHT:       RA,
// 			Relations:   TEST_ENUM_RELATIONS,
// 			Transitions: TRANSITIONS_ENUM,
// 		},
// 		REDUCE:  RE,
// 		POPROOT: PR,
// 	}
// 	arcSystem.AddDefaultOracle()
// 	transitionSystem := transition.TransitionSystem(arcSystem)
// 	conf := &SimpleConfiguration{
// 		EWord:  EWord,
// 		EPOS:   EPOS,
// 		EWPOS:  EWPOS,
// 		ERel:   TEST_ENUM_RELATIONS,
// 		ETrans: TRANSITIONS_ENUM,
// 	}

// 	beam := &Beam{
// 		TransFunc:     transitionSystem,
// 		FeatExtractor: extractor,
// 		Base:          conf,
// 		NumRelations:  arcSystem.Relations.Len(),
// 	}

// 	decoder := perceptron.EarlyUpdateInstanceDecoder(beam)
// 	updater := new(TransitionModel.AveragedModelStrategy)
// 	model := TransitionModel.NewAvgMatrixSparse(extractor.EFeatures.Len(), nil)

// 	perceptron := &perceptron.LinearPerceptron{Decoder: decoder, Updater: updater}
// 	perceptron.Init(model)

// 	// get gold parse
// 	goldModel := dependency.TransitionParameterModel(&PerceptronModel{model})
// 	deterministic := &Deterministic{
// 		TransFunc:          transitionSystem,
// 		FeatExtractor:      extractor,
// 		ReturnModelValue:   true,
// 		ReturnSequence:     true,
// 		ShowConsiderations: false,
// 		Base:               conf,
// 		NoRecover:          true,
// 	}

// 	_, goldParams := deterministic.ParseOracle(GetTestDepGraph(), nil, goldModel)
// 	if goldParams == nil {
// 		t.Fatal("Got nil params from deterministic oracle parsing, can't test beam-perceptron model")
// 	}
// 	seq := goldParams.(*ParseResultParameters).Sequence

// 	goldSequence := make(ScoredConfigurations, len(seq))
// 	var (
// 		lastFeatures *transition.FeaturesList
// 		curFeats     []featurevector.Feature
// 	)
// 	for i := len(seq) - 1; i >= 0; i-- {
// 		val := seq[i]
// 		curFeats = extractor.Features(val)
// 		lastFeatures = &transition.FeaturesList{curFeats, val.GetLastTransition(), lastFeatures}
// 		goldSequence[len(seq)-i-1] = &ScoredConfiguration{val.(DependencyConfiguration), val.GetLastTransition(), 0.0, lastFeatures, 0, 0, true}
// 	}

// 	goldInstances := []perceptron.DecodedInstance{
// 		&perceptron.Decoded{perceptron.Instance(rawTestSent), goldSequence}}

// 	// beam.Log = true
// 	// perceptron.Log = true
// 	beam.ConcurrentExec = true
// 	beam.ReturnSequence = true
// 	// train with increasing iterations
// 	convergenceIterations := []int{1, 2, 4, 8, 20}
// 	beamSizes := []int{1, 2, 4, 16, 64}
// 	// convergenceIterations := []int{2, 4, 8}
// 	// beamSizes := []int{16}
// 	for _, beamSize := range beamSizes {
// 		beam.Size = beamSize
// 		convergenceSharedSequence := make([]int, 0, len(convergenceIterations))
// 		for _, iterations := range convergenceIterations {
// 			perceptron.Iterations = iterations
// 			model = TransitionModel.NewAvgMatrixSparse(extractor.EFeatures.Len(), nil)
// 			perceptron.Init(model)

// 			// log.Println("Starting training", iterations, "iterations")
// 			// perceptron.Log = true
// 			// if j > 0 {
// 			// beam.Log = true
// 			// }
// 			beam.ClearTiming()
// 			perceptron.Train(goldInstances)
// 			// log.Println("TRAIN Time Expanding (pct):\t", beam.DurExpanding.Seconds(), 100*beam.DurExpanding/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting (pct):\t", beam.DurInserting.Seconds(), 100*beam.DurInserting/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Feat (pct):\t", beam.DurInsertFeat.Seconds(), 100*beam.DurInsertFeat/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Modl (pct):\t", beam.DurInsertModl.Seconds(), 100*beam.DurInsertModl/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-ModA (pct):\t", beam.DurInsertModA.Seconds(), 100*beam.DurInsertModA/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-ModB (pct):\t", beam.DurInsertModB.Seconds(), 100*beam.DurInsertModB/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-ModC (pct):\t", beam.DurInsertModC.Seconds(), 100*beam.DurInsertModC/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Scrp (pct):\t", beam.DurInsertScrp.Seconds(), 100*beam.DurInsertScrp/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Scrm (pct):\t", beam.DurInsertScrm.Seconds(), 100*beam.DurInsertScrm/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Heap (pct):\t", beam.DurInsertHeap.Seconds(), 100*beam.DurInsertHeap/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Agen (pct):\t", beam.DurInsertAgen.Seconds(), 100*beam.DurInsertAgen/beam.DurTotal)
// 			// log.Println("TRAIN Time Inserting-Init (pct):\t", beam.DurInsertInit.Seconds(), 100*beam.DurInsertInit/beam.DurTotal)
// 			// log.Println("TRAIN Total Time:", beam.DurTotal.Seconds())
// 			// log.Println("Finished training", iterations, "iterations")

// 			trainedModel := dependency.TransitionParameterModel(&PerceptronModel{model})
// 			beam.ReturnModelValue = false
// 			beam.ClearTiming()
// 			_, params := beam.Parse(TEST_SENT, nil, trainedModel)
// 			// log.Println("PARSE Time Expanding (pct):\t", beam.DurExpanding.Seconds(), 100*beam.DurExpanding/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting (pct):\t", beam.DurInserting.Seconds(), 100*beam.DurInserting/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Feat (pct):\t", beam.DurInsertFeat.Seconds(), 100*beam.DurInsertFeat/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Modl (pct):\t", beam.DurInsertModl.Seconds(), 100*beam.DurInsertModl/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-ModA (pct):\t", beam.DurInsertModA.Seconds(), 100*beam.DurInsertModA/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-ModB (pct):\t", beam.DurInsertModB.Seconds(), 100*beam.DurInsertModB/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-ModC (pct):\t", beam.DurInsertModC.Seconds(), 100*beam.DurInsertModC/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Scrp (pct):\t", beam.DurInsertScrp.Seconds(), 100*beam.DurInsertScrp/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Scrm (pct):\t", beam.DurInsertScrm.Seconds(), 100*beam.DurInsertScrm/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Heap (pct):\t", beam.DurInsertHeap.Seconds(), 100*beam.DurInsertHeap/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Agen (pct):\t", beam.DurInsertAgen.Seconds(), 100*beam.DurInsertAgen/beam.DurTotal)
// 			// log.Println("PARSE Time Inserting-Init (pct):\t", beam.DurInsertInit.Seconds(), 100*beam.DurInsertInit/beam.DurTotal)
// 			// log.Println("PARSE Total Time:", beam.DurTotal.Seconds())
// 			sharedSteps := 0
// 			if params != nil {
// 				seq := params.(*ParseResultParameters).Sequence
// 				sharedSteps = goldSequence[len(goldSequence)-1].C.GetSequence().SharedTransitions(seq)
// 			}
// 			convergenceSharedSequence = append(convergenceSharedSequence, sharedSteps)
// 		}
// 		if len(convergenceSharedSequence) != len(convergenceIterations) {
// 			t.Error("Not enough examples in shared sequence samples")
// 		}
// 		// verify convergence
// 		log.Println("Shared Sequence For Beam", beamSize, convergenceSharedSequence)
// 		if !sort.IntsAreSorted(convergenceSharedSequence) || convergenceSharedSequence[len(convergenceSharedSequence)-1] == 0 {
// 			t.Error("Model not converging, shared sequences lengths:", convergenceSharedSequence)
// 		}
// 	}
// 	t.Error("bla")
// }
