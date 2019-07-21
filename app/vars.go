package app

import (
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/search"
	"github.com/OnlpLab/yap/alg/transition"

	"github.com/OnlpLab/yap/alg/transition/model"
	// dep "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	"github.com/OnlpLab/yap/eval"
	"github.com/OnlpLab/yap/nlp/format/conll"
	"github.com/OnlpLab/yap/nlp/format/mapping"
	"github.com/OnlpLab/yap/nlp/format/raw"
	"github.com/OnlpLab/yap/nlp/format/segmentation"
	"github.com/OnlpLab/yap/nlp/parser/disambig"
	"github.com/OnlpLab/yap/nlp/parser/joint"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	dep "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	"github.com/OnlpLab/yap/nlp/parser/dependency/transition/morph"

	"encoding/gob"
	"fmt"
	"log"
	"os"
	// "runtime"
	"time"
	// "strings"

	"github.com/gonuts/commander"
	"github.com/OnlpLab/yap/nlp/format/conllu"
)

func init() {
	gob.Register(&Serialization{})
	var t nlp.Token
	gob.Register(&t)
}

var (
	allOut   bool = true
	parseOut bool = false

	// processing options
	//Iterations, BeamSize int
	Iterations			int
	BeamSize          int
	ConcurrentBeam       bool
	NumFeatures          int
	UsePOP               bool
	limit                int
	Stream               bool

	// global enumerations
	ERel, ETrans, EWord, EPOS, EWPOS, EMHost, EMSuffix *util.EnumSet
	ETokens                                            *util.EnumSet
	EMorphProp                                         *util.EnumSet
	//DepERel, DepETrans, DepEWord, DepEPOS, DepEWPOS, DepEMHost, DepEMSuffix *util.EnumSet
	//DepETokens                                            *util.EnumSet
	//DepEMorphProp                                         *util.EnumSet

	// enumeration offsets of transitions
	SH, RE, PR, LA, RA, IDLE, POP, MD transition.Transition
	//DepSH, DepRE, DepPR, DepLA, DepRA, DepIDLE, DepPOP, DepMD transition.Transition

	// file names
	tConll           string
	tLatDis, tLatAmb string
	tSeg             string
	input, inputLat  string
	inputGold        string
	test             string
	testGold         string
	outLat, outSeg   string
	outMap           string
	outConll         string
	//modelFile        string
	//modelName        string
	//featuresFile     string
	//labelsFile       string

	AlignBeam             bool
	AverageScores         bool
	alignAverageParseOnly bool

	//ArcSystemStr string

	// string arrays can't be const, so let it be a var
	DEFAULT_MODEL_DIRS = []string{".", "data"}
	DEFAULT_CONF_DIRS  = []string{".", "conf"}
)

// An approximation of the number of different MD-X:Y:Z transitions
// Pre-allocating the enumeration saves frequent reallocation during training and parsing
const (
	APPROX_MORPH_TRANSITIONS        = 100
	APPROX_WORDS, APPROX_POS        = 100, 100
	WORDS_POS_FACTOR                = 5
	APPROX_MHOSTS, APPROX_MSUFFIXES = 128, 16
)

type Serialization struct {
	WeightModel                          *model.AvgMatrixSparseSerialized
	EWord, EPOS, EWPOS, EMHost, EMSuffix *util.EnumSet
	EMorphProp                           *util.EnumSet
	ETrans                               *util.EnumSet
	ETokens                              *util.EnumSet
}

func WriteModel(file string, data *Serialization) {
	fObj, err := os.Create(file)
	if err != nil {
		log.Fatalln("Failed creating model file", file, err)
		return
	}
	defer func() {
		fObj.Close()
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
	//defer fObj.Close()
	writer := gob.NewEncoder(fObj)
	err = writer.Encode(data)
	if err != nil {
		log.Fatalln("Failed writing model model to", file, err)
		panic("Failed to write model")
	}
}

func ReadModel(file string) *Serialization {
	data := &Serialization{}
	fObj, err := os.Open(file)
	if err != nil {
		log.Fatalln("Failed reading model from", file, err)
		return nil
	}
	defer fObj.Close()
	reader := gob.NewDecoder(fObj)
	reader.Decode(data)
	return data
}

func SetupRelationEnum(labels []string) {
	if ERel != nil {
		return
	}
	ERel = util.NewEnumSet(len(labels) + 1)
	ERel.Add(nlp.DepRel(nlp.ROOT_LABEL))
	for _, label := range labels {
		ERel.Add(nlp.DepRel(label))
	}
	ERel.Frozen = true
}

func SetupTransEnum(relations []string) {
	ETrans = util.NewEnumSet((len(relations)+1)*2 + 2)
	_, _ = ETrans.Add("IDLE") // dummy no action transition for zpar equivalence
	iSH, _ := ETrans.Add("SH")
	iRE, _ := ETrans.Add("RE")
	_, _ = ETrans.Add("AL") // dummy action transition for zpar equivalence
	_, _ = ETrans.Add("AR") // dummy action transition for zpar equivalence
	iPR, _ := ETrans.Add("PR")
	SH = transition.ConstTransition(iSH)
	RE = transition.ConstTransition(iRE)
	PR = transition.ConstTransition(iPR)
	LA = transition.ConstTransition(iPR + 1)
	ETrans.Add("LA-" + string(nlp.ROOT_LABEL))
	for _, transition := range relations {
		ETrans.Add("LA-" + string(transition))
	}
	RA = transition.ConstTransition(ETrans.Len())
	ETrans.Add("RA-" + string(nlp.ROOT_LABEL))
	for _, transition := range relations {
		ETrans.Add("RA-" + string(transition))
	}
}

func SetupMorphTransEnum(relations []string) {
	ETrans = util.NewEnumSet((len(relations)+1)*2 + 2 + APPROX_MORPH_TRANSITIONS)
	_, _ = ETrans.Add("NO") // dummy for 0 action
	iSH, _ := ETrans.Add("SH")
	iRE, _ := ETrans.Add("RE")
	_, _ = ETrans.Add("AL") // dummy action transition for zpar equivalence
	_, _ = ETrans.Add("AR") // dummy action transition for zpar equivalence
	iPR, _ := ETrans.Add("PR")
	// iIDLE, _ := ETrans.Add("IDLE")
	SH = transition.ConstTransition(iSH)
	RE = transition.ConstTransition(iRE)
	PR = transition.ConstTransition(iPR)
	// IDLE = transition.Transition(iIDLE)
	// LA = IDLE + 1
	LA = transition.ConstTransition(iPR + 1)
	ETrans.Add("LA-" + string(nlp.ROOT_LABEL))
	for _, transition := range relations {
		ETrans.Add("LA-" + string(transition))
	}
	RA = transition.ConstTransition(ETrans.Len())
	ETrans.Add("RA-" + string(nlp.ROOT_LABEL))
	for _, transition := range relations {
		ETrans.Add("RA-" + string(transition))
	}
	log.Println("ETrans Len is", ETrans.Len())
	iPOP, _ := ETrans.Add("POP")
	POP = &transition.TypedTransition{'P', iPOP}
	MD = transition.ConstTransition(ETrans.Len())
}

func VerifyExists(filename string) bool {
	_, err := os.Stat(filename)
	if err != nil {
		//log.Println("Error accessing file", filename)
		//log.Println(err)
		return false
	}
	return true
}

func VerifyFlags(cmd *commander.Command, required []string) {
	for _, flag := range required {
		f := cmd.Flag.Lookup(flag)
		if f.Value.String() == "" {
			log.Printf("Required flag %s not set", f.Name)
			cmd.Usage()
			os.Exit(1)
		}
	}
}

func SetupExtractor(setup *transition.FeatureSetup, transTypes []byte) *transition.GenericExtractor {
	extractor := &transition.GenericExtractor{
		EFeatures:  util.NewEnumSet(setup.NumFeatures()),
		Concurrent: false,
		EWord:      EWord,
		EPOS:       EPOS,
		EWPOS:      EWPOS,
		ERel:       ERel,
		EMHost:     EMHost,
		EMSuffix:   EMSuffix,
		EMorphProp: EMorphProp,
		EToken:     ETokens,
		POPTrans:   POP,
		// Log:        true,
	}
	if transTypes == nil {
		extractor.Init()
	} else {
		extractor.InitTypes(transTypes)
	}
	extractor.LoadFeatureSetup(setup)

	NumFeatures = setup.NumFeatures()
	return extractor
}

type InstanceFunc func(interface{}) util.Equaler
type GoldFunc func(interface{}) util.Equaler

func Limit(instances []interface{}, limit int) []interface{} {
	if limit > 0 && len(instances) > limit {
		return instances[:limit]
	}
	return instances
}

func TrainingSequences(trainingSet []interface{}, instFunc InstanceFunc, goldFunc GoldFunc) []perceptron.DecodedInstance {
	instances := make([]perceptron.DecodedInstance, 0, len(trainingSet))

	for _, instance := range trainingSet {
		// log.Println("At training", i)

		decoded := &perceptron.Decoded{instFunc(instance), goldFunc(instance)}
		instances = append(instances, decoded)
	}
	return instances
}

// Assumes sorted inputs of equal length
func DepEval(test, gold interface{}) *eval.Result {
	testConf, testOk := test.(*dep.SimpleConfiguration)
	// testGraph, _ := test.(*dep.BasicDepGraph)
	goldGraph, _ := gold.(*dep.BasicDepGraph)
	// log.Println(testMorph.GetSequence())
	// log.Println(goldMorph.GetSequence())
	if !testOk {
		panic("Test argument should be MDConfig")
	}
	// if !goldOk {
	// 	panic("Gold argument should be nlp.Mappings")
	// }
	testArcs := testConf.Arcs().(*dep.ArcSetSimple).Arcs
	// testArcs := testGraph.Arcs
	goldArcs := goldGraph.Arcs
	retval := &eval.Result{ // retval is LAS
		Other: &eval.Result{}, // Other is UAS evaluation
	}
	// log.Println("Test is:")
	// log.Println(testArcs)
	// log.Println("Gold is:")
	// log.Println(goldArcs)
	var unlabeledAttached, labeledAttached, modifierExists bool
	for _, curTestArc := range testArcs {
		unlabeledAttached, labeledAttached = false, false
		for _, curGoldArc := range goldArcs {
			if curTestArc.GetHead() == curGoldArc.GetHead() &&
				curTestArc.GetModifier() == curGoldArc.GetModifier() {
				unlabeledAttached = true
				retval.Other.(*eval.Result).TP += 1
				if curTestArc.GetRelation() == curGoldArc.GetRelation() {
					labeledAttached = true
					retval.TP += 1
				}
				break
			}
		}
		if !labeledAttached {
			retval.FP += 1
		}
		if !unlabeledAttached {
			retval.Other.(*eval.Result).FP += 1
		}
	}
	for _, curGoldArc := range goldArcs {
		unlabeledAttached, labeledAttached, modifierExists = false, false, false
		for _, curTestArc := range testArcs {
			if curGoldArc.GetModifier() == curTestArc.GetModifier() {
				modifierExists = true
			}
			if curTestArc.GetHead() == curGoldArc.GetHead() &&
				curTestArc.GetModifier() == curGoldArc.GetModifier() {
				unlabeledAttached = true
				if curTestArc.GetRelation() == curGoldArc.GetRelation() {
					labeledAttached = true
				}
				break
			}
		}
		if !modifierExists {
			retval.FP += 1
		}
		if !labeledAttached {
			retval.TN += 1
		}
		if !modifierExists {
			retval.Other.(*eval.Result).FP += 1
		}
		if !unlabeledAttached {
			retval.Other.(*eval.Result).TN += 1
		}
	}
	return retval
}

// Assumes sorted inputs of equal length
func JointEval(test, gold interface{}, metric string) *eval.Result {
	testMorph, testOk := test.(*joint.JointConfig)
	if !testOk {
		panic("Test argument should be JointConfig")
	}
	return MorphEval(&testMorph.MDConfig, gold, metric)
}

// Assumes sorted inputs of equal length
func MorphEval(test, gold interface{}, metric string) *eval.Result {
	var result string
	testMorph, testOk := test.(*disambig.MDConfig)
	goldMappings, goldOk := gold.(nlp.Mappings)
	// log.Println(testMorph.GetSequence())
	// log.Println(goldMorph.GetSequence())
	if !testOk {
		log.Println("Got test:", test)
		panic("Test argument should be MDConfig")
	}
	if !goldOk {
		panic("Gold argument should be nlp.Mappings")
	}
	testMappings := testMorph.Mappings
	retval := &eval.Result{Other: make(nlp.BasicSentence, len(testMappings))}
	// log.Println("Test is:")
	// log.Println(testMappings)
	// log.Println("Gold is:")
	// log.Println(goldMappings)
	for i, testMapping := range testMappings {
		goldMapping := goldMappings[i]
		// if testMapping.Token != goldMapping.Token {
		// 	panic(fmt.Sprintf("Mappings #%v are not equal: %v %v", i, testMapping.Token, goldMapping.Token))
		// }
		testSpellout := testMapping.Spellout
		goldSpellout := goldMapping.Spellout
		TP, TN, FP, FN := testSpellout.Compare(goldSpellout, metric)
		retval.TP += TP
		retval.TN += TN
		retval.FP += FP
		retval.FN += FN
		if FP == 0 {
			result = "Success"
		} else {
			result = "Error"
		}
		retval.Other.(nlp.BasicSentence)[i] = nlp.Token(result)
	}
	return retval
}

func Train(trainingSet []perceptron.DecodedInstance, Iterations int, filename string, paramModel perceptron.Model, decoder perceptron.EarlyUpdateInstanceDecoder, goldDecoder perceptron.InstanceDecoder, converge perceptron.StopCondition) *perceptron.LinearPerceptron {
	updater := new(model.AveragedModelStrategy)

	perceptron := &perceptron.LinearPerceptron{
		Decoder:     decoder,
		GoldDecoder: goldDecoder,
		Updater:     updater,
		Continue:    converge,
		Tempfile:    filename,
		TempLines:   500}

	perceptron.Iterations = Iterations
	perceptron.Init(paramModel)
	// perceptron.TempLoad("model.b64.i1")
	perceptron.Log = true
	// beam.Log = true
	startTime := time.Now()
	perceptron.Train(trainingSet)
	if allOut {
		trainTime := time.Since(startTime)
		log.Println("TRAIN Total Time:", trainTime)
	}
	return perceptron
}

type Parser interface {
	Parse(search.Problem) (transition.Configuration, interface{})
}

func ParseStream(instances chan interface{}, writeStream chan interface{}, parser Parser) {
	// runtime.GOMAXPROCS(1)
	// Search.AllOut = true
	startTime := time.Now()

	// prevGC := debug.SetGCPercent(-1)
	var i int
	for instance := range instances {
		// if i%50 == 0 {
		// 	debug.SetGCPercent(100)
		// 	runtime.GC()
		// 	debug.SetGCPercent(-1)
		// }
		log.Println("Parsing instance", i) //, "len", len(sent.Tokens()))
		// }
		result, _ := parser.Parse(instance)
		writeStream <- result
		i++
	}
	if allOut {
		parseTime := time.Since(startTime)
		log.Println("PARSE Total Time:", parseTime)
	}
	// debug.SetGCPercent(prevGC)
	close(writeStream)
}

func Parse(instances []interface{}, parser Parser) []interface{} {
	// runtime.GOMAXPROCS(1)
	// Search.AllOut = true
	startTime := time.Now()

	// prevGC := debug.SetGCPercent(-1)
	parsed := make([]interface{}, len(instances))
	for i, instance := range instances {
		// if i%50 == 0 {
		// 	debug.SetGCPercent(100)
		// 	runtime.GC()
		// 	debug.SetGCPercent(-1)
		// }
		log.Println("Parsing instance", i) //, "len", len(sent.Tokens()))
		// }
		result, _ := parser.Parse(instance)
		parsed[i] = result
	}
	if allOut {
		parseTime := time.Since(startTime)
		log.Println("PARSE Total Time:", parseTime)
	}
	// debug.SetGCPercent(prevGC)
	return parsed
}

func GetMDConfigAsLattices(instance interface{}) util.Equaler {
	return instance.(*disambig.MDConfig).Lattices
}

func GetMDConfigAsMappings(instance interface{}) util.Equaler {
	return instance.(*disambig.MDConfig).Mappings
}

func GetMorphGraphAsLattices(instance interface{}) util.Equaler {
	return instance.(*morph.BasicMorphGraph).Lattice
}

func GetMorphGraph(instance interface{}) util.Equaler {
	return instance.(*morph.BasicMorphGraph)
}

func GetAsTaggedSentence(instance interface{}) util.Equaler {
	return instance.(nlp.LabeledDependencyGraph).TaggedSentence()
}

func GetAsLabeledDepGraph(instance interface{}) util.Equaler {
	return instance.(nlp.LabeledDependencyGraph)
}

func GetJointMDConfig(instance interface{}) util.Equaler {
	return &instance.(*joint.JointConfig).MDConfig
}

// func GetJointMDConfigAsMappings(instance interface{}) util.Equaler {
// 	return &instance.(*joint.JointConfig).MDConfig.Mappings
// }
//
// func GetJointMDConfigAsLattices(instance interface{}) util.Equaler {
// 	return &instance.(*joint.JointConfig).MDConfig.Lattices
// }

func GetInstances(instances []interface{}, getFunc InstanceFunc) []interface{} {
	retval := make([]interface{}, len(instances))
	for i, val := range instances {
		retval[i] = getFunc(val)
	}
	return retval
}

func MakeMorphEvalStopCondition(instances []interface{}, goldInstances []interface{}, testInstances []interface{}, testGoldInstances []interface{}, parser Parser, goldDecoder perceptron.InstanceDecoder, beamSize int) perceptron.StopCondition {
	var (
		equalIterations int
		prevResult      float64
	)
	return func(curIteration, iterations, generations int, model perceptron.Model) bool {
		// first write current model
		serialize(model, curIteration, generations)
		// log.Println("Eval starting for iteration", curIteration)
		var total = &eval.Total{
			Results: make([]*eval.Result, 0, len(instances)),
		}
		var posonlytotal = &eval.Total{
			Results: make([]*eval.Result, 0, len(instances)),
		}
		// Don't test before initial run
		if curIteration == 0 {
			return true
		}
		var curResult float64
		var curPosResult float64
		// TODO: fix this leaky abstraction :(
		// log.Println("Temp integration using", generations)
		parser.(*search.Beam).IntegrationGeneration = generations
		parsed := Parse(instances, parser)
		goldInstances := TrainingSequences(goldInstances, GetMDConfigAsLattices, GetMDConfigAsMappings)
		log.Println("START Evaluation")
		if len(goldInstances) != len(instances) {
			panic("Evaluation instance lengths are different")
		}
		for i, instance := range parsed {
			// log.Println("Evaluating", i)
			goldInstance := goldInstances[i]
			if goldInstance != nil {
				result := MorphEval(instance, goldInstance.Decoded(), "Form_POS_Prop")
				posresult := MorphEval(instance, goldInstance.Decoded(), "Form_POS")
				// log.Println("Correct: ", result.TP)
				total.Add(result)
				posonlytotal.Add(posresult)
			}
		}
		curResult = total.F1()
		curPosResult = posonlytotal.F1()
		// Break out of edge case where result remains the same
		if curResult == prevResult {
			equalIterations += 1
		}
		retval := (curIteration >= iterations) && (curResult < prevResult || equalIterations > 2)
		// retval := curIteration >= iterations
		log.Println("Result (F1): ", curResult, "Exact:", total.Exact, "TruePos:", total.TP, "in", total.Population, "POS F1:", curPosResult)
		if retval {
			log.Println("Stopping")
		} else {
			log.Println("Continuing")
		}
		prevResult = curResult
		log.Println("Writing interm results to", fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outMap))
		mapping.WriteFile(fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outMap), parsed)
		if testInstances != nil {
			// Test output
			testTotal := &eval.Total{
				Results: make([]*eval.Result, 0, len(instances)),
			}
			testposonlytotal := &eval.Total{
				Results: make([]*eval.Result, 0, len(instances)),
			}
			testParsed := Parse(testInstances, parser)
			testGoldInstances := TrainingSequences(testGoldInstances, GetMDConfigAsLattices, GetMDConfigAsMappings)
			log.Println("START Test Evaluation")
			if len(testGoldInstances) != len(testInstances) {
				panic("Evaluation instance lengths are different")
			}
			testErrorVectors := make([]interface{}, len(testParsed))
			testPOSErrorVectors := make([]interface{}, len(testParsed))
			for i, instance := range testParsed {
				// log.Println("Evaluating", i)
				testInstance := testGoldInstances[i]
				if testInstance != nil {
					result := MorphEval(instance, testInstance.Decoded(), "Form_POS_Prop")
					posresult := MorphEval(instance, testInstance.Decoded(), "Form_POS")
					testErrorVectors[i] = result.Other
					testPOSErrorVectors[i] = posresult.Other
					// log.Println("Correct: ", result.TP)
					testTotal.Add(result)
					testposonlytotal.Add(posresult)

				}
			}
			log.Println("Test Result (F1): ", testTotal.F1(), "Exact:", testTotal.Exact, "TruePos:", testTotal.TP, "in", testTotal.Population, "POS F1:", testposonlytotal.F1())
			log.Println("Writing test results to", fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outMap))
			mapping.WriteFile(fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outMap), testParsed)
			raw.WriteFile(fmt.Sprintf("err.test.i%v.b%v.%v.raw", curIteration, beamSize, outMap), testErrorVectors)
			raw.WriteFile(fmt.Sprintf("errpos.test.i%v.b%v.%v.raw", curIteration, beamSize, outMap), testPOSErrorVectors)
		}
		return !retval
	}
}

func MakeDepEvalStopCondition(instances []interface{}, goldInstances []interface{}, testInstances []interface{}, morphInstances []interface{}, goldMorphInstances []interface{}, testMorphInstances []interface{}, parser Parser, goldDecoder perceptron.InstanceDecoder, beamSize int) perceptron.StopCondition {
	var (
		equalIterations     int
		prevResult          float64
		continuousDecreases int
	)
	return func(curIteration, iterations, generations int, model perceptron.Model) bool {
		// first write current model
		serialize(model, curIteration, generations)
		// log.Println("Eval starting for iteration", curIteration)
		var total = &eval.Total{
			Results: make([]*eval.Result, 0, len(instances)),
		}
		var utotal = &eval.Total{
			Results: make([]*eval.Result, 0, len(instances)),
		}
		// Don't test before initial run
		if curIteration == 0 {
			return true
		}
		var curResult float64
		// TODO: fix this leaky abstraction :(
		// log.Println("Temp integration using", generations)
		parser.(*search.Beam).IntegrationGeneration = generations
		oldparseOut := parseOut
		parseOut = true
		parsed := Parse(instances, parser)
		parseOut = oldparseOut
		goldInstances := TrainingSequences(goldInstances, GetAsTaggedSentence, GetAsLabeledDepGraph)
		// log.Println("START Evaluation")
		if len(goldInstances) != len(instances) {
			panic("Evaluation instance lengths are different")
		}
		for i, instance := range parsed {
			// log.Println("Evaluating", i)
			goldInstance := goldInstances[i]
			if goldInstance != nil {
				result := DepEval(instance, goldInstance.Decoded())
				// log.Println("Correct: ", result.TP)
				total.Add(result)
				utotal.Add(result.Other.(*eval.Result))
			}
		}
		curResult = total.Precision()
		// Break out of edge case where result remains the same
		if curResult == prevResult {
			equalIterations += 1
		}
		retval := (Iterations < curIteration) && ((continuousDecreases > 1 && curResult < prevResult) || equalIterations > 3)
		// retval := curIteration >= iterations
		log.Println("Result (UAS, LAS, UEM #, UEM %): ", utotal.Precision(), total.Precision(), utotal.Exact, float64(utotal.Exact)/float64(total.Population), "TruePos:", total.TP, "in", total.Population)
		if retval {
			log.Println("Stopping")
		} else {
			log.Println("Continuing")
		}
		if curResult < prevResult {
			continuousDecreases += 1
		} else {
			continuousDecreases = 0
		}
		prevResult = curResult
		if useConllU {
			graphs := conllu.Graph2ConllUCorpus(parsed, EMHost, EMSuffix)
			morphGraphs := conllu.MergeGraphAndMorphCorpus(graphs, morphInstances)
			conllu.WriteFile(fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outConll), morphGraphs)
		} else {
			graphs := conll.Graph2ConllCorpus(parsed, EMHost, EMSuffix)
			conll.WriteFile(fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outConll), graphs)
		}
		if testInstances != nil {
			log.Println("Parsing test")
			testParsed := Parse(testInstances, parser)
			log.Println("Writing test results to", fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, test))
			if useConllU {
				testGraphs := conllu.Graph2ConllUCorpus(testParsed, EMHost, EMSuffix)
				testMorphGraphs := conllu.MergeGraphAndMorphCorpus(testGraphs, morphInstances)
				conllu.WriteFile(fmt.Sprintf("test.i%v.b%v.conll", curIteration, beamSize), testMorphGraphs)
			} else {
				testGraphs := conll.Graph2ConllCorpus(testParsed, EMHost, EMSuffix)
				conll.WriteFile(fmt.Sprintf("test.i%v.b%v.conll", curIteration, beamSize), testGraphs)
			}
		}
		return !retval
	}
}

func MakeJointEvalStopCondition(instances []interface{}, goldInstances []interface{}, testInstances []interface{}, testGoldInstances []interface{}, parser Parser, goldDecoder perceptron.InstanceDecoder, beamSize int) perceptron.StopCondition {
	var (
		equalIterations             int
		prevResult, bestResult      float64
		continuousDecreases         int
		bestIteration               int
		bestModelFile, curModelFile string
	)
	return func(curIteration, iterations, generations int, model perceptron.Model) bool {
		// log.Println("Eval starting for iteration", curIteration)
		var total = &eval.Total{
			Results: make([]*eval.Result, 0, len(instances)),
		}
		var posonlytotal = &eval.Total{
			Results: make([]*eval.Result, 0, len(instances)),
		}
		// Don't test before initial run
		if curIteration == 0 {
			return true
		}
		curModelFile = serialize(model, curIteration, generations)
		var curResult float64
		var curPosResult float64
		// TODO: fix this leaky abstraction :(
		// log.Println("Temp integration using", generations)
		parser.(*search.Beam).IntegrationGeneration = generations
		parsedGraphs := Parse(instances, parser)
		goldInstances := TrainingSequences(goldInstances, GetMDConfigAsLattices, GetMDConfigAsMappings)
		log.Println("START Evaluation Joint Eval")
		if len(goldInstances) != len(instances) {
			panic("Evaluation instance lengths are different")
		}
		for i, instance := range parsedGraphs {
			// log.Println("Evaluating", i)
			goldInstance := goldInstances[i]
			if goldInstance != nil {
				result := JointEval(instance, goldInstance.Decoded(), "Form_POS_Prop")
				posresult := JointEval(instance, goldInstance.Decoded(), "Form_POS")
				// log.Println("Correct: ", result.TP)
				total.Add(result)
				posonlytotal.Add(posresult)
			}
		}
		curResult = total.F1()
		curPosResult = posonlytotal.F1()
		// Break out of edge case where result remains the same
		if curResult == prevResult {
			equalIterations += 1
		}
		if curResult < prevResult {
			continuousDecreases += 1
		} else {
			continuousDecreases = 0
		}
		if bestResult < curResult {
			bestResult = curResult
			bestIteration = curIteration
			bestModelFile = curModelFile
		}
		retval := (Iterations < curIteration) && ((continuousDecreases > 1 && curResult < prevResult) || equalIterations > 3)
		log.Println("It", Iterations, "CurIt", curIteration, "Continuous", continuousDecreases, "CurResult", curResult, "PrevResult", prevResult, "Comp", curResult < prevResult, "Retval", retval)
		// retval := curIteration >= iterations
		log.Println("Result (F1): ", curResult, "Exact:", total.Exact, "TruePos:", total.TP, "in", total.Population, "POS F1:", curPosResult)
		if retval {
			log.Println("Stopping")
			log.Println("Best iteration was", bestIteration)
			log.Println("Best model file", bestModelFile)

			file, err := os.Create("bestmodelname")
			defer file.Close()
			if err != nil {
				log.Println("Failed to write name of best model:", err)
			} else {
				file.Write([]byte(bestModelFile))
			}
		} else {
			log.Println("Continuing")
		}
		prevResult = curResult
		graphs := conll.MorphGraph2ConllCorpus(parsedGraphs)
		log.Println("Writing interm results to conll:", fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outConll))
		conll.WriteFile(fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outConll), graphs)
		log.Println("Writing interm results to segmentation:", fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outSeg))
		segmentation.WriteFile(fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outSeg), parsedGraphs)
		log.Println("Writing interm results to mapping:", fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outMap))
		mapping.WriteFile(fmt.Sprintf("interm.i%v.b%v.%v", curIteration, beamSize, outMap), GetInstances(parsedGraphs, GetJointMDConfig))
		if testInstances != nil {
			// Test output
			testTotal := &eval.Total{
				Results: make([]*eval.Result, 0, len(instances)),
			}
			testposonlytotal := &eval.Total{
				Results: make([]*eval.Result, 0, len(instances)),
			}
			testParsed := Parse(testInstances, parser)
			testGoldInstances := TrainingSequences(testGoldInstances, GetMDConfigAsLattices, GetMDConfigAsMappings)
			log.Println("START Test Evaluation")
			if len(testGoldInstances) != len(testInstances) {
				panic("Evaluation instance lengths are different")
			}
			for i, instance := range testParsed {
				// log.Println("Evaluating", i)
				testInstance := testGoldInstances[i]
				if testInstance != nil {
					result := JointEval(instance, testInstance.Decoded(), "Form_POS_Prop")
					posresult := JointEval(instance, testInstance.Decoded(), "Form_POS")
					// log.Println("Correct: ", result.TP)
					testTotal.Add(result)
					testposonlytotal.Add(posresult)
				}
			}
			graphs := conll.MorphGraph2ConllCorpus(testParsed)
			log.Println("Test Result (F1): ", testTotal.F1(), "Exact:", testTotal.Exact, "TruePos:", testTotal.TP, "in", testTotal.Population, "POS F1:", testposonlytotal.F1())
			log.Println("Writing test results to conll:", fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outConll))
			conll.WriteFile(fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outConll), graphs)
			log.Println("Writing test results to segmentation:", fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outSeg))
			segmentation.WriteFile(fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outSeg), testParsed)
			log.Println("Writing test results to mapping", fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outMap))
			mapping.WriteFile(fmt.Sprintf("test.i%v.b%v.%v", curIteration, beamSize, outMap), GetInstances(testParsed, GetJointMDConfig))
		}
		return !retval
	}
}

func serialize(perceptronModel perceptron.Model, iteration, generations int) string {
	serialization := &Serialization{
		perceptronModel.(*model.AvgMatrixSparse).Serialize(generations),
		EWord, EPOS, EWPOS, EMHost, EMSuffix, EMorphProp, ETrans, ETokens,
	}
	modelFile := fmt.Sprintf("model.temp.i%d", iteration)
	WriteModel(modelFile, serialization)
	return modelFile
}
