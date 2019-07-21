package app

// import (
// 	"github.com/OnlpLab/yap/alg/featurevector"
// 	"github.com/OnlpLab/yap/alg/perceptron"
// 	BeamSearch "github.com/OnlpLab/yap/alg/search"
// 	"github.com/OnlpLab/yap/alg/transition"
// 	transitionmodel "github.com/OnlpLab/yap/alg/transition/model"
// 	"github.com/OnlpLab/yap/nlp/format/conll"
// 	"github.com/OnlpLab/yap/nlp/format/lattice"
// 	"github.com/OnlpLab/yap/nlp/format/segmentation"
// 	"github.com/OnlpLab/yap/nlp/parser/dependency"
// 	. "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
// 	"github.com/OnlpLab/yap/nlp/parser/dependency/transition/morph"
// 	nlp "github.com/OnlpLab/yap/nlp/types"
// 	"github.com/OnlpLab/yap/util"
// 	"github.com/OnlpLab/yap/util/conf"

// 	"fmt"
// 	"log"
// 	"os"
// 	"runtime"

// 	"github.com/gonuts/commander"
// 	"github.com/gonuts/flag"
// )

// func SetupMorphTransEnum(relations []string) {
// 	ETrans = util.NewEnumSet((len(relations)+1)*2 + 2 + APPROX_MORPH_TRANSITIONS)
// 	_, _ = ETrans.Add("NO") // dummy for 0 action
// 	iSH, _ := ETrans.Add("SH")
// 	iRE, _ := ETrans.Add("RE")
// 	_, _ = ETrans.Add("AL") // dummy action transition for zpar equivalence
// 	_, _ = ETrans.Add("AR") // dummy action transition for zpar equivalence
// 	iPR, _ := ETrans.Add("PR")
// 	// iIDLE, _ := ETrans.Add("IDLE")
// 	SH = transition.Transition(iSH)
// 	RE = transition.Transition(iRE)
// 	PR = transition.Transition(iPR)
// 	// IDLE = transition.Transition(iIDLE)
// 	// LA = IDLE + 1
// 	LA = PR + 1
// 	ETrans.Add("LA-" + string(nlp.ROOT_LABEL))
// 	for _, transition := range relations {
// 		ETrans.Add("LA-" + string(transition))
// 	}
// 	RA = transition.Transition(ETrans.Len())
// 	ETrans.Add("RA-" + string(nlp.ROOT_LABEL))
// 	for _, transition := range relations {
// 		ETrans.Add("RA-" + string(transition))
// 	}
// 	log.Println("ETrans Len is", ETrans.Len())
// 	MD = transition.Transition(ETrans.Len())
// }

// func SetupMorphEnum(relations []string) {
// 	SetupRelationEnum(relations)
// 	SetupMorphTransEnum(relations)
// 	EWord, EPOS, EWPOS = util.NewEnumSet(APPROX_WORDS), util.NewEnumSet(APPROX_POS), util.NewEnumSet(APPROX_WORDS*5)
// 	EMorphProp = util.NewEnumSet(130) // random guess of number of possible values
// }

// func TrainingSequences(trainingSet []*morph.BasicMorphGraph, transitionSystem transition.TransitionSystem, extractor perceptron.FeatureExtractor) []perceptron.DecodedInstance {
// 	// verify feature load

// 	mconf := &morph.MorphConfiguration{
// 		SimpleConfiguration: SimpleConfiguration{
// 			EWord:  EWord,
// 			EPOS:   EPOS,
// 			EWPOS:  EWPOS,
// 			ERel:   ERel,
// 			ETrans: ETrans,
// 		},
// 	}

// 	deterministic := &BeamSearch.Deterministic{
// 		TransFunc:          transitionSystem,
// 		FeatExtractor:      extractor,
// 		ReturnModelValue:   false,
// 		ReturnSequence:     true,
// 		ShowConsiderations: false,
// 		Base:               mconf,
// 		TransEnum:          ETrans,
// 		// NoRecover:          true,
// 	}

// 	// decoder := perceptron.EarlyUpdateInstanceDecoder(deterministic)
// 	// updater := new(transitionmodel.AveragedModelStrategy)

// 	// perceptron := &perceptron.LinearPerceptron{Decoder: decoder, Updater: updater}
// 	model := transitionmodel.NewAvgMatrixSparse(NumFeatures, nil, true)

// 	tempModel := model
// 	// perceptron.Init(model)

// 	instances := make([]perceptron.DecodedInstance, 0, len(trainingSet))
// 	var failedTraining int
// 	for i, graph := range trainingSet {
// 		if i%100 == 0 {
// 			log.Println("At line", i)
// 		}
// 		sent := graph.Lattice
// 		deterministic.Model = tempModel
// 		_, goldParams := deterministic.ParseOracle(&perceptron.Decoded{sent, graph})
// 		if goldParams != nil {
// 			seq := goldParams.(*BeamSearch.ParseResultParameters).Sequence
// 			goldSequence := make(BeamSearch.ScoredConfigurations, len(seq))
// 			var (
// 				lastFeatures *transition.FeaturesList
// 				curFeats     []featurevector.Feature
// 			)
// 			for i := len(seq) - 1; i >= 0; i-- {
// 				val := seq[i]
// 				curFeats = extractor.Features(val)
// 				lastFeatures = &transition.FeaturesList{curFeats, val.GetLastTransition(), lastFeatures}
// 				goldSequence[len(seq)-i-1] = &BeamSearch.ScoredConfiguration{val.(DependencyConfiguration), val.GetLastTransition(), 0.0, lastFeatures, 0, 0, true}
// 			}

// 			// log.Println("Gold seq:\n", seq)
// 			decoded := &perceptron.Decoded{sent, goldSequence}
// 			instances = append(instances, decoded)
// 		} else {
// 			failedTraining++
// 		}
// 	}
// 	log.Println("Failed training generation:", failedTraining)
// 	return instances
// }

// func EstimatedBeamTransitions() int {
// 	return ERel.Len()*2 + 2
// }

// func Train(trainingSet []perceptron.DecodedInstance, Iterations, BeamSize int, filename string, model perceptron.Model, transitionSystem transition.TransitionSystem, extractor perceptron.FeatureExtractor) *perceptron.LinearPerceptron {
// 	conf := &morph.MorphConfiguration{
// 		SimpleConfiguration: SimpleConfiguration{
// 			EWord:  EWord,
// 			EPOS:   EPOS,
// 			EWPOS:  EWPOS,
// 			ERel:   ERel,
// 			ETrans: ETrans,
// 		},
// 	}

// 	beam := &BeamSearch.Beam{
// 		TransFunc:            transitionSystem,
// 		FeatExtractor:        extractor,
// 		Base:                 conf,
// 		Size:                 BeamSize,
// 		ConcurrentExec:       ConcurrentBeam,
// 		EstimatedTransitions: EstimatedBeamTransitions(),
// 	}

// 	// varbeam := &VarBeam{beam}
// 	decoder := perceptron.EarlyUpdateInstanceDecoder(beam)
// 	updater := new(transitionmodel.AveragedModelStrategy)

// 	perceptron := &perceptron.LinearPerceptron{
// 		Decoder:   decoder,
// 		Updater:   updater,
// 		Tempfile:  filename,
// 		TempLines: 1000}

// 	perceptron.Iterations = Iterations
// 	perceptron.Init(model)
// 	// perceptron.TempLoad("model.b64.i1")
// 	perceptron.Log = true

// 	perceptron.Train(trainingSet)
// 	log.Println("TRAIN Total Time:", beam.DurTotal)

// 	return perceptron
// }

// func Parse(sents []nlp.LatticeSentence, BeamSize int, model dependency.TransitionParameterModel, transitionSystem transition.TransitionSystem, extractor perceptron.FeatureExtractor) []nlp.MorphDependencyGraph {
// 	conf := &morph.MorphConfiguration{
// 		SimpleConfiguration: SimpleConfiguration{
// 			EWord:  EWord,
// 			EPOS:   EPOS,
// 			EWPOS:  EWPOS,
// 			ERel:   ERel,
// 			ETrans: ETrans,
// 		},
// 	}

// 	beam := BeamSearch.Beam{
// 		TransFunc:            transitionSystem,
// 		FeatExtractor:        extractor,
// 		Base:                 conf,
// 		Size:                 BeamSize,
// 		Model:                model,
// 		ConcurrentExec:       ConcurrentBeam,
// 		ShortTempAgenda:      true,
// 		Transitions:          ETrans,
// 		EstimatedTransitions: EstimatedBeamTransitions(),
// 	}

// 	// varbeam := &VarBeam{beam}

// 	parsedGraphs := make([]nlp.MorphDependencyGraph, len(sents))
// 	for i, sent := range sents {
// 		// if i%100 == 0 {
// 		runtime.GC()
// 		log.Println("Parsing sent", i)
// 		// }
// 		graph, _ := beam.Parse(sent)
// 		labeled := graph.(nlp.MorphDependencyGraph)
// 		parsedGraphs[i] = labeled
// 	}
// 	log.Println("PARSE Total Time:", beam.DurTotal)

// 	return parsedGraphs
// }

// func CombineTrainingInputs(graphs []nlp.LabeledDependencyGraph, goldLats, ambLats []nlp.LatticeSentence) ([]*morph.BasicMorphGraph, int) {
// 	if len(graphs) != len(goldLats) || len(graphs) != len(ambLats) {
// 		panic(fmt.Sprintf("Got mismatched training slice inputs (graphs, gold lattices, ambiguous lattices):", len(graphs), len(goldLats), len(ambLats)))
// 	}
// 	morphGraphs := make([]*morph.BasicMorphGraph, len(graphs))
// 	var (
// 		numLatticeNoGold int
// 		noGold           bool
// 	)
// 	prefix := log.Prefix()
// 	for i, goldGraph := range graphs {
// 		goldLat := goldLats[i]
// 		ambLat := ambLats[i]
// 		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
// 		morphGraphs[i], noGold = morph.CombineToGoldMorph(goldGraph, goldLat, ambLat)
// 		if noGold {
// 			numLatticeNoGold++
// 		}
// 	}
// 	log.SetPrefix(prefix)
// 	return morphGraphs, numLatticeNoGold
// }

// func MorphConfigOut(outModelFile string) {
// 	log.Println("Configuration")
// 	// log.Printf("Beam:             \tVariable Length")
// 	log.Printf("Beam:             \tStatic Length")
// 	// log.Printf("Transition System:\tIDLE + Morph + ArcEager")
// 	log.Printf("Transition System:\tMorph + ArcEager")
// 	log.Printf("Iterations:\t\t%d", Iterations)
// 	log.Printf("Beam Size:\t\t%d", BeamSize)
// 	log.Printf("Beam Concurrent:\t%v", ConcurrentBeam)
// 	// log.Printf("Model file:\t\t%s", outModelFile)

// 	log.Println()
// 	log.Printf("Features File:\t%s", featuresFile)
// 	if !VerifyExists(featuresFile) {
// 		os.Exit(1)
// 	}
// 	log.Printf("Labels File:\t\t%s", labelsFile)
// 	if !VerifyExists(labelsFile) {
// 		os.Exit(1)
// 	}
// 	log.Println()
// 	log.Println("Data")
// 	log.Printf("Train file (conll):\t\t\t%s", tConll)
// 	if !VerifyExists(tConll) {
// 		return
// 	}
// 	log.Printf("Train file (disamb. lattice):\t%s", tLatDis)
// 	if !VerifyExists(tLatDis) {
// 		return
// 	}
// 	log.Printf("Train file (ambig.  lattice):\t%s", tLatAmb)
// 	if !VerifyExists(tLatAmb) {
// 		return
// 	}
// 	log.Printf("Test file  (ambig.  lattice):\t%s", input)
// 	if !VerifyExists(input) {
// 		return
// 	}
// 	log.Printf("Out (disamb.) file:\t\t\t%s", outLat)
// 	log.Printf("Out (segmt.) file:\t\t\t%s", outSeg)
// 	log.Printf("Out Train (segmt.) file:\t\t%s", tSeg)
// }

// func MorphTrainAndParse(cmd *commander.Command, args []string) {
// 	REQUIRED_FLAGS := []string{"it", "tc", "td", "tl", "in", "oc", "os", "ots", "f", "l"}

// 	VerifyFlags(cmd, REQUIRED_FLAGS)
// 	// RegisterTypes()

// 	outModelFile := fmt.Sprintf("%s.b%d.i%d", modelFile, BeamSize, Iterations)

// 	MorphConfigOut(outModelFile)

// 	relations, err := conf.ReadFile(labelsFile)
// 	if err != nil {
// 		log.Println("Failed reading dependency labels configuration file:", labelsFile)
// 		log.Fatalln(err)
// 	}
// 	if allOut {
// 		log.Println()
// 		// start processing - setup enumerations
// 		log.Println("Setup enumerations")
// 	}
// 	SetupMorphEnum(relations.Values)

// 	if allOut {
// 		log.Println()
// 		log.Println("Loading features")
// 	}
// 	features, err := conf.ReadFile(featuresFile)
// 	if err != nil {
// 		log.Println("Failed reading feature configuration file:", featuresFile)
// 		log.Fatalln(err)
// 	}
// 	extractor := SetupExtractor(features.Values)

// 	if allOut {
// 		log.Println("Generating Gold Sequences For Training")
// 		log.Println("Conll:\tReading training conll sentences from", tConll)
// 	}
// 	s, e := conll.ReadFile(tConll)
// 	if e != nil {
// 		log.Println(e)
// 		return
// 	}
// 	if allOut {
// 		log.Println("Conll:\tRead", len(s), "sentences")
// 		log.Println("Conll:\tConverting from conll to internal structure")
// 	}
// 	goldConll := conll.Conll2GraphCorpus(s, EWord, EPOS, EWPOS, ERel, nil, nil)

// 	if allOut {
// 		log.Println("Dis. Lat.:\tReading training disambiguated lattices from", tLatDis)
// 	}
// 	lDis, lDisE := lattice.ReadFile(tLatDis)
// 	if lDisE != nil {
// 		log.Println(lDisE)
// 		return
// 	}
// 	if allOut {
// 		log.Println("Dis. Lat.:\tRead", len(lDis), "disambiguated lattices")
// 		log.Println("Dis. Lat.:\tConverting lattice format to internal structure")
// 	}
// 	goldDisLat := lattice.Lattice2SentenceCorpus(lDis, EWord, EPOS, EWPOS, EMorphProp)

// 	if allOut {
// 		log.Println("Amb. Lat:\tReading ambiguous lattices from", tLatAmb)
// 	}
// 	lAmb, lAmbE := lattice.ReadFile(tLatAmb)
// 	if lAmbE != nil {
// 		log.Println(lAmbE)
// 		return
// 	}
// 	if allOut {
// 		log.Println("Amb. Lat:\tRead", len(lAmb), "ambiguous lattices")
// 		log.Println("Amb. Lat:\tConverting lattice format to internal structure")
// 	}
// 	goldAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp)
// 	if allOut {
// 		log.Println("Combining train files into gold morph graphs with original lattices")
// 	}
// 	combined, missingGold := CombineTrainingInputs(goldConll, goldDisLat, goldAmbLat)

// 	if allOut {
// 		log.Println("Combined", len(combined), "graphs, with", missingGold, "missing at least one gold path in lattice")

// 		log.Println()

// 	}

// 	morphArcSystem := &morph.ArcEagerMorph{
// 		ArcEager: ArcEager{
// 			ArcStandard: ArcStandard{
// 				SHIFT:       SH,
// 				LEFT:        LA,
// 				RIGHT:       RA,
// 				Relations:   ERel,
// 				Transitions: ETrans,
// 			},
// 			REDUCE:  RE,
// 			POPROOT: PR},
// 		MD: MD,
// 	}
// 	morphArcSystem.AddDefaultOracle()

// 	// arcSystem := &morph.Idle{morphArcSystem, IDLE}
// 	transitionSystem := transition.TransitionSystem(morphArcSystem)

// 	if allOut {
// 		log.Println()

// 		log.Println("Parsing with gold to get training sequences")
// 	}
// 	// const NUM_SENTS = 20
// 	// combined = combined[:NUM_SENTS]
// 	goldSequences := TrainingSequences(combined, transitionSystem, extractor)
// 	if allOut {
// 		log.Println("Generated", len(goldSequences), "training sequences")
// 		log.Println()
// 		// util.LogMemory()
// 		log.Println("Training", Iterations, "iteration(s)")
// 	}
// 	formatters := make([]util.Format, len(extractor.FeatureTemplates))
// 	for i, formatter := range extractor.FeatureTemplates {
// 		formatters[i] = formatter
// 	}
// 	model := transitionmodel.NewAvgMatrixSparse(NumFeatures, formatters, true)
// 	_ = Train(goldSequences, Iterations, BeamSize, modelFile, model, transitionSystem, extractor)
// 	if allOut {
// 		log.Println("Done Training")
// 		// util.LogMemory()
// 		log.Println()
// 		// log.Println("Writing final model to", outModelFile)
// 		// WriteModel(model, outModelFile)
// 		// log.Println()
// 		log.Print("Parsing test")

// 		log.Println("Reading ambiguous lattices from", input)
// 	}
// 	lAmb, lAmbE = lattice.ReadFile(input)
// 	if lAmbE != nil {
// 		log.Println(lAmbE)
// 		return
// 	}
// 	// lAmb = lAmb[:NUM_SENTS]
// 	if allOut {
// 		log.Println("Read", len(lAmb), "ambiguous lattices from", input)
// 		log.Println("Converting lattice format to internal structure")
// 	}
// 	predAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp)

// 	parsedGraphs := Parse(predAmbLat, BeamSize, model, transitionSystem, extractor)

// 	if allOut {
// 		log.Println("Converting", len(parsedGraphs), "to conll")
// 	}
// 	graphAsConll := conll.MorphGraph2ConllCorpus(parsedGraphs)
// 	if allOut {
// 		log.Println("Writing to output file")
// 	}
// 	conll.WriteFile(outLat, graphAsConll)
// 	if allOut {
// 		log.Println("Wrote", len(graphAsConll), "in conll format to", outLat)

// 		log.Println("Writing to segmentation file")
// 	}
// 	segmentation.WriteFile(outSeg, parsedGraphs)
// 	if allOut {
// 		log.Println("Wrote", len(parsedGraphs), "in segmentation format to", outSeg)

// 		log.Println("Writing to gold segmentation file")
// 	}
// 	segmentation.WriteFile(tSeg, ToMorphGraphs(combined))
// 	if allOut {
// 		log.Println("Wrote", len(combined), "in segmentation format to", tSeg)
// 	}
// }

// func ToMorphGraphs(graphs []*morph.BasicMorphGraph) []nlp.MorphDependencyGraph {
// 	morphs := make([]nlp.MorphDependencyGraph, len(graphs))
// 	for i, g := range graphs {
// 		morphs[i] = nlp.MorphDependencyGraph(g)
// 	}
// 	return morphs
// }

// func MorphCmd() *commander.Command {
// 	cmd := &commander.Command{
// 		Run:       MorphTrainAndParse,
// 		UsageLine: "morph <file options> [arguments]",
// 		Short:     "runs morpho-syntactic training and parsing",
// 		Long: `
// runs morpho-syntactic training and parsing

// 	$ ./yap morph -tc <conll> -td <train disamb. lat> -tl <train amb. lat> -in <input lat> -oc <out lat> -os <out seg> -ots <out train seg> [options]

// `,
// 		Flag: *flag.NewFlagSet("morph", flag.ExitOnError),
// 	}
// 	cmd.Flag.BoolVar(&ConcurrentBeam, "bconc", false, "Concurrent Beam")
// 	cmd.Flag.IntVar(&Iterations, "it", 1, "Number of Perceptron Iterations")
// 	cmd.Flag.IntVar(&BeamSize, "b", 4, "Beam Size")
// 	cmd.Flag.StringVar(&modelFile, "m", "model", "Prefix for model file ({m}.b{b}.i{it}.model)")

// 	cmd.Flag.StringVar(&tConll, "tc", "", "Training Conll File")
// 	cmd.Flag.StringVar(&tLatDis, "td", "", "Training Disambiguated Lattices File")
// 	cmd.Flag.StringVar(&tLatAmb, "tl", "", "Training Ambiguous Lattices File")
// 	cmd.Flag.StringVar(&input, "in", "", "Test Ambiguous Lattices File")
// 	cmd.Flag.StringVar(&outLat, "oc", "", "Output Conll File")
// 	cmd.Flag.StringVar(&outSeg, "os", "", "Output Segmentation File")
// 	cmd.Flag.StringVar(&tSeg, "ots", "", "Output Training Segmentation File")
// 	cmd.Flag.StringVar(&featuresFile, "f", "", "Features Configuration File")
// 	cmd.Flag.StringVar(&labelsFile, "l", "", "Dependency Labels Configuration File")
// 	return cmd
// }
