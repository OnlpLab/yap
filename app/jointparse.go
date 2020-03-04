package app

import (
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/search"
	"github.com/OnlpLab/yap/alg/transition"
	transitionmodel "github.com/OnlpLab/yap/alg/transition/model"
	"github.com/OnlpLab/yap/nlp/format/conll"
	"github.com/OnlpLab/yap/nlp/format/conllu"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/format/mapping"
	"github.com/OnlpLab/yap/nlp/format/segmentation"
	. "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	"github.com/OnlpLab/yap/nlp/parser/dependency/transition/morph"
	"github.com/OnlpLab/yap/nlp/parser/disambig"
	"github.com/OnlpLab/yap/nlp/parser/joint"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
	"github.com/OnlpLab/yap/util/conf"

	"fmt"
	"log"
	"os"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var (
	JointFeaturesFile			string
	JointModelFile				string
	JointStrategy, OracleStrategy string
	limitdev                      int
	hebMACompat                   bool
)

func SetupEnum(relations []string) {
	SetupRelationEnum(relations)
	SetupMorphTransEnum(relations)
	EWord, EPOS, EWPOS = util.NewEnumSet(APPROX_WORDS), util.NewEnumSet(APPROX_POS), util.NewEnumSet(APPROX_WORDS*5)
	EMHost, EMSuffix = util.NewEnumSet(APPROX_MHOSTS), util.NewEnumSet(APPROX_MSUFFIXES)
	EMorphProp = util.NewEnumSet(130) // random guess of number of possible values
	ETokens = util.NewEnumSet(10000)  // random guess of number of possible values
	// adding empty string as an element in the morph enum sets so that '0' default values
	// map to empty morphs
	EMHost.Add("")
	EMSuffix.Add("")
}

func CombineJointCorpus(graphs, goldLats, ambLats []interface{}) ([]interface{}, int) {
	if len(graphs) != len(goldLats) || len(graphs) != len(ambLats) {
		panic(fmt.Sprintf("Got mismatched training slice inputs (graphs, gold lattices, ambiguous lattices):", len(graphs), len(goldLats), len(ambLats)))
	}
	morphGraphs := make([]interface{}, len(graphs))
	var (
		numLatticeNoGold int
		noGold           int
	)
	prefix := log.Prefix()
	for i, goldGraph := range graphs {
		goldLat := goldLats[i].(nlp.LatticeSentence)
		ambLat := ambLats[i].(nlp.LatticeSentence)
		_, noGold = CombineToGoldMorph(goldLat, ambLat)
		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		morphGraphs[i], _ = morph.CombineToGoldMorph(goldGraph.(nlp.LabeledDependencyGraph), goldLat, ambLat)
		numLatticeNoGold += noGold
	}
	log.SetPrefix(prefix)
	return morphGraphs, numLatticeNoGold
}

func CombineToGoldMorphs(goldLats, ambLats []interface{}) ([]interface{}, int) {
	if len(goldLats) != len(ambLats) {
		panic(fmt.Sprintf("Got mismatched training slice inputs (gold lattices, ambiguous lattices):", len(goldLats), len(ambLats)))
	}
	morphGraphs := make([]interface{}, len(goldLats))
	var (
		numLatticeNoGold, numSentNoGold int
		numNoGold                       int
	)
	prefix := log.Prefix()
	for i, goldLat := range goldLats {
		ambLat := ambLats[i].(nlp.LatticeSentence)
		log.SetPrefix(fmt.Sprintf("%v lattice# %v ", prefix, i))
		morphGraphs[i], numNoGold = CombineToGoldMorph(goldLat.(nlp.LatticeSentence), ambLat)
		if numNoGold > 0 {
			numSentNoGold++
			numLatticeNoGold += numNoGold
		}
	}
	log.SetPrefix(prefix)
	return morphGraphs, numSentNoGold
}

func JointConfigOut(outModelFile string, b search.Interface, t transition.TransitionSystem) {
	log.Println("*** CONFIGURATION ***")
	log.Printf("Beam:             \t%s", b.Name())
	log.Printf("Transition System:\t%s", t.Name())
	log.Printf("Transition Oracle:\t%s", t.Oracle().Name())
	log.Printf("Iterations:\t\t%d", Iterations)
	log.Printf("Beam Size:\t\t%d", BeamSize)
	log.Printf("Beam Concurrent:\t%v", ConcurrentBeam)
	log.Printf("Parameter Func:\t%v", MdParamFuncName)
	log.Printf("Use Lemmas:\t\t%v", !lattice.IGNORE_LEMMA)
	log.Printf("Use POP:\t\t%v", UsePOP)
	log.Printf("Infuse Gold Dev:\t%v", MdCombineGold)
	log.Printf("Limit (thousands):\t%v", limit)
	log.Printf("Use CoNLL-U:\t\t%v", useConllU)
	// log.Printf("Model file:\t\t%s", outModelFile)

	log.Println()
	log.Printf("Features File:\t%s", JointFeaturesFile)
	outFeaturesFile := JointFeaturesFile
	featuresExists := VerifyExists(outFeaturesFile)
	if !featuresExists {
		outFeaturesFile, featuresExists = util.LocateFile(outFeaturesFile, DEFAULT_CONF_DIRS)
	}
	if !featuresExists {
		os.Exit(1)
	}
	JointFeaturesFile = outFeaturesFile
	log.Printf("Labels File:\t\t%s", DepLabelsFile)
	outLabelsFile := DepLabelsFile
	labelsExists := VerifyExists(outLabelsFile)
	if !labelsExists {
		outLabelsFile, labelsExists = util.LocateFile(outLabelsFile, DEFAULT_CONF_DIRS)
	}
	if !labelsExists {
		os.Exit(1)
	}
	DepLabelsFile = outLabelsFile
	log.Println()
	log.Println("Data")
	if len(tConll) > 0 {
		log.Printf("Train file (conll):\t\t\t%s", tConll)
		if !VerifyExists(tConll) {
			return
		}
	}
	if len(tLatDis) > 0 {
		log.Printf("Train file (disamb. lattice):\t%s", tLatDis)
		if !VerifyExists(tLatDis) {
			return
		}
	}
	if len(tLatAmb) > 0 {
		log.Printf("Train file (ambig.  lattice):\t%s", tLatAmb)
		if !VerifyExists(tLatAmb) {
			return
		}
	}
	if len(input) > 0 {
		log.Printf("Test file  (ambig.  lattice):\t%s", input)
		if !VerifyExists(input) {
			return
		}
	}
	if len(inputGold) > 0 {
		log.Printf("Test file  (disambig.  lattice):\t%s", inputGold)
		if !VerifyExists(inputGold) {
			return
		}
	}
	if len(outConll) > 0 {
		log.Printf("Out (disamb.) file:\t\t\t%s", outConll)
	}
	if len(outSeg) > 0 {
		log.Printf("Out (segmt.) file:\t\t\t%s", outSeg)
	}
	if len(outMap) > 0 {
		log.Printf("Out (mapping.) file:\t\t\t%s", outMap)
	}
	if len(tSeg) > 0 {
		log.Printf("Out Train (segmt.) file:\t\t%s", tSeg)
	}
}

func JointTrainAndParse(cmd *commander.Command, args []string) error {
	// *** SETUP ***
	paramFunc, exists := nlp.MDParams[MdParamFuncName]
	if !exists {
		log.Fatalln("Param Func", MdParamFuncName, "does not exist")
	}

	mdTrans := &disambig.MDTrans{
		ParamFunc: paramFunc,
		UsePOP:    UsePOP,
	}

	var (
		arcSystem     transition.TransitionSystem
		model         *transitionmodel.AvgMatrixSparse = &transitionmodel.AvgMatrixSparse{}
		terminalStack int
	)

	switch DepArcSystemStr {
	case "standard":
		arcSystem = &ArcStandard{}
		terminalStack = 1
	case "eager":
		arcSystem = &ArcEager{
			ArcStandard: ArcStandard{},
		}
		terminalStack = 0
	default:
		panic("Unknown arc system")
	}

	arcSystem.AddDefaultOracle()

	jointTrans := &joint.JointTrans{
		MDTrans:       mdTrans,
		ArcSys:        arcSystem,
		JointStrategy: JointStrategy,
	}
	jointTrans.AddDefaultOracle()
	jointTrans.Oracle().(*joint.JointOracle).OracleStrategy = OracleStrategy
	transitionSystem := transition.TransitionSystem(jointTrans)

	outModelFile := JointModelFile
	modelExists := VerifyExists(outModelFile)
	if !modelExists {
		outModelFile, modelExists = util.LocateFile(outModelFile, DEFAULT_MODEL_DIRS)
	}
	REQUIRED_FLAGS := []string{"in", "oc", "om", "os"}
	VerifyFlags(cmd, REQUIRED_FLAGS)

	if !modelExists {
		REQUIRED_FLAGS = []string{"it", "tc", "td", "tl", "in", "oc", "om", "os", "ots", "f", "l", "jointstr", "oraclestr"}
		VerifyFlags(cmd, REQUIRED_FLAGS)
	}

	// RegisterTypes()

	confBeam := &search.Beam{}
	if !alignAverageParseOnly {
		confBeam.Align = AlignBeam
		confBeam.Averaged = AverageScores
	}

	JointConfigOut(outModelFile, confBeam, transitionSystem)

	relations, err := conf.ReadFile(DepLabelsFile)
	if err != nil {
		log.Println("Failed reading dependency labels configuration file:", DepLabelsFile)
		log.Fatalln(err)
	}
	if allOut {
		log.Println()
		// start processing - setup enumerations
		log.Println("Setup enumerations")
	}
	SetupEnum(relations.Values)

	// after calling SetupEnum, enums are instantiated and set according to the relations
	// therefore we re-instantiate the arc system with the right parameters
	// DON'T REMOVE!!
	switch DepArcSystemStr {
	case "standard":
		arcSystem = &ArcStandard{
			SHIFT:       SH.Value(),
			LEFT:        LA.Value(),
			RIGHT:       RA.Value(),
			Transitions: ETrans,
			Relations:   ERel,
		}
	case "eager":
		arcSystem = &ArcEager{
			ArcStandard: ArcStandard{
				SHIFT:       SH.Value(),
				LEFT:        LA.Value(),
				RIGHT:       RA.Value(),
				Relations:   ERel,
				Transitions: ETrans,
			},
			REDUCE:  RE.Value(),
			POPROOT: PR.Value(),
		}
	default:
		panic("Unknown arc system")
	}
	arcSystem.AddDefaultOracle()
	jointTrans.ArcSys = arcSystem
	jointTrans.Transitions = ETrans
	mdTrans.Transitions = ETrans
	mdTrans.UsePOP = UsePOP
	mdTrans.POP = POP
	disambig.UsePOP = UsePOP
	disambig.SwitchFormLemma = !lattice.IGNORE_LEMMA
	disambig.LEMMAS = !lattice.IGNORE_LEMMA
	mdTrans.AddDefaultOracle()
	jointTrans.MDTransition = MD
	jointTrans.JointStrategy = JointStrategy
	jointTrans.AddDefaultOracle()
	jointTrans.Oracle().(*joint.JointOracle).OracleStrategy = OracleStrategy

	transitionSystem = transition.TransitionSystem(jointTrans)

	if allOut {
		log.Println()
		log.Println("Loading features")
	}

	featureSetup, err := transition.LoadFeatureConfFile(JointFeaturesFile)
	if err != nil {
		log.Println("Failed reading feature configuration file:", JointFeaturesFile)
		log.Fatalln(err)
	}
	// M - MD
	// P - POP
	// L - Lemma (not in use right now)
	// A - Arc (syntactic)
	groups := []byte("MPLA")
	extractor := SetupExtractor(featureSetup, groups)

	log.Println()
	if useConllU {
		nlp.InitOpenParamFamily("UD")
		conllu.IGNORE_LEMMA = lattice.IGNORE_LEMMA
		if hebMACompat {
			conllu.STRIP_VOICE = true
		}
	} else {
		nlp.InitOpenParamFamily("HEBTB")
	}
	log.Println()

	if !modelExists {
		log.Println("")
		log.Println("*** TRAINING ***")
		// *** TRAINING ***

		if allOut {
			log.Println("Generating Gold Sequences For Training")
			log.Println("Conll:\tReading training conll sentences from", tConll)
		}
		var goldConll []interface{}
		if useConllU {
			s, _, e := conllu.ReadFile(tConll, limit)
			if e != nil {
				log.Println(e)
				return e
			}
			if allOut {
				log.Println("Conll:\tRead", len(s), "sentences")
				log.Println("Conll:\tConverting from conll to internal structure")
			}
			goldConll = conllu.ConllU2MorphGraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
		} else {
			s, e := conll.ReadFile(tConll, limit)
			if e != nil {
				log.Println(e)
				return e
			}
			if allOut {
				log.Println("Conll:\tRead", len(s), "sentences")
				log.Println("Conll:\tConverting from conll to internal structure")
			}
			goldConll = conll.Conll2GraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
		}

		var goldDisLat []interface{}
		if !useConllU {
			if allOut {
				log.Println("Dis. Lat.:\tReading training disambiguated lattices from", tLatDis)
			}
			lDis, lDisE := lattice.ReadFile(tLatDis, limit)
			if lDisE != nil {
				log.Println(lDisE)
				return lDisE
			}
			if allOut {
				log.Println("Dis. Lat.:\tRead", len(lDis), "disambiguated lattices")
				log.Println("Dis. Lat.:\tConverting lattice format to internal structure")
			}
			goldDisLat = lattice.Lattice2SentenceCorpus(lDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
		} else {
			goldDisLat = make([]interface{}, len(goldConll))
			for i, sent := range goldConll {
				goldDisLat[i] = sent.(*morph.BasicMorphGraph).Lattice
			}
		}

		if allOut {
			log.Println("Amb. Lat:\tReading ambiguous lattices from", tLatAmb)
		}
		var (
			lAmb  []lattice.Lattice
			lAmbE error
		)
		if useConllU {
			lAmb, lAmbE = lattice.ReadUDFile(tLatAmb, limit)
		} else {
			lAmb, lAmbE = lattice.ReadFile(tLatAmb, limit)
		}
		if lAmbE != nil {
			log.Println(lAmbE)
			return lAmbE
		}
		if allOut {
			log.Println("Amb. Lat:\tRead", len(lAmb), "ambiguous lattices")
			log.Println("Amb. Lat:\tConverting lattice format to internal structure")
		}
		goldAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
		if allOut {
			log.Println("Combining train files into gold morph graphs with original lattices")
		}
		combined, missingGold := CombineJointCorpus(goldConll, goldDisLat, goldAmbLat)

		if allOut {
			log.Println("Combined", len(combined), "graphs, with", missingGold, "missing at least one gold path in lattice")

			log.Println()

		}

		if allOut {
			log.Println()

			log.Println("Parsing with gold to get training sequences")
		}

		// const NUM_SENTS = 20
		// combined = combined[:NUM_SENTS]
		goldSequences := TrainingSequences(combined, GetMorphGraphAsLattices, GetMorphGraph)
		if allOut {
			log.Println("Generated", len(goldSequences), "training sequences")
			log.Println()
			// util.LogMemory()
			log.Println("Training", Iterations, "iteration(s)")
		}
		formatters := make([]util.Format, 0, 100)
		for _, g := range groups {
			group, _ := extractor.TransTypeGroups[g]
			for _, formatter := range group.FeatureTemplates {
				formatters = append(formatters, formatter)
			}
		}
		model := transitionmodel.NewAvgMatrixSparse(NumFeatures, formatters, false)
		model.Extractor = extractor
		// model.Classifier = func(t transition.Transition) string {
		// 	if t.Value() < MD.Value() {
		// 		return "Arc"
		// 	} else {
		// 		return "MD"
		// 	}
		// }

		conf := &joint.JointConfig{
			SimpleConfiguration: SimpleConfiguration{
				EWord:         EWord,
				EPOS:          EPOS,
				EWPOS:         EWPOS,
				EMHost:        EMHost,
				EMSuffix:      EMSuffix,
				ERel:          ERel,
				ETrans:        ETrans,
				TerminalStack: terminalStack,
				TerminalQueue: 0,
			},
			MDConfig: disambig.MDConfig{
				ETokens:     ETokens,
				POP:         POP,
				Transitions: ETrans,
				ParamFunc:   paramFunc,
			},
			MDTrans: MD,
		}

		beam := &search.Beam{
			TransFunc:            transitionSystem,
			FeatExtractor:        extractor,
			Base:                 conf,
			Size:                 BeamSize,
			ConcurrentExec:       ConcurrentBeam,
			Transitions:          ETrans,
			EstimatedTransitions: 1000,
			NoRecover:            false,
		}

		if !alignAverageParseOnly {
			beam.Align = AlignBeam
			beam.Averaged = AverageScores
		}

		deterministic := &search.Deterministic{
			TransFunc:          transitionSystem,
			FeatExtractor:      extractor,
			ReturnModelValue:   false,
			ReturnSequence:     true,
			ShowConsiderations: false,
			Base:               conf,
			NoRecover:          false,
			DefaultTransType:   'M',
		}

		var evaluator perceptron.StopCondition
		if len(inputGold) > 0 && !MdNoconverge {
			var (
				convCombined []interface{}
				convDisLat   []interface{}
				convAmbLat   []interface{}
			)
			if allOut {
				log.Println("Setting convergence tester")
			}
			decodeTestBeam := &search.Beam{}
			*decodeTestBeam = *beam
			decodeTestBeam.Model = model
			decodeTestBeam.DecodeTest = true
			decodeTestBeam.ShortTempAgenda = true

			if useConllU {

				s, _, e := conllu.ReadFile(inputGold, limitdev)
				if e != nil {
					log.Println(e)
					return e
				}
				if allOut {
					log.Println("Convergence Dev Gold Dis. Lat.:\tRead", len(s), "disambiguated lattices")
					log.Println("Convergence Dev Gold Dis. Lat.:\tConverting lattice format to internal structure")
				}
				asGraph := conllu.ConllU2MorphGraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
				convDisLat = make([]interface{}, len(asGraph))
				for i, sent := range asGraph {
					convDisLat[i] = sent.(*morph.BasicMorphGraph).Lattice
				}
			} else {

				lConvDis, lConvDisE := lattice.ReadFile(inputGold, limitdev)
				if lConvDisE != nil {
					log.Println(lConvDisE)
					return lConvDisE
				}
				if allOut {
					log.Println("Convergence Dev Gold Dis. Lat.:\tRead", len(lConvDis), "disambiguated lattices")
					log.Println("Convergence Dev Gold Dis. Lat.:\tConverting lattice format to internal structure")
				}

				convDisLat = lattice.Lattice2SentenceCorpus(lConvDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
			}

			if allOut {
				log.Println("Reading dev test ambiguous lattices (for convergence testing) from", input)
			}

			var (
				lConvAmb  []lattice.Lattice
				lConvAmbE error
			)
			if useConllU {
				lConvAmb, lConvAmbE = lattice.ReadUDFile(input, limitdev)
			} else {
				lConvAmb, lConvAmbE = lattice.ReadFile(input, limitdev)
			}
			// lConvAmb = lConvAmb[:NUM_SENTS]
			if lConvAmbE != nil {
				log.Println(lConvAmbE)
				return lConvAmbE
			}
			// lAmb = lAmb[:NUM_SENTS]
			if allOut {
				log.Println("Read", len(lConvAmb), "ambiguous lattices from", input)
				log.Println("Converting lattice format to internal structure")
			}
			convAmbLat = lattice.Lattice2SentenceCorpus(lConvAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
			if MdCombineGold {
				var devMissingGold, devSentMissingGold, devLattices int
				convCombined, devMissingGold, devLattices, devSentMissingGold = CombineLatticesCorpus(convDisLat, convAmbLat)
				log.Println("Combined", len(convCombined), "graphs, with", devMissingGold, "lattices of", devLattices, "missing at least one gold path in lattice in", devSentMissingGold, "sentences")
			} else {
				convCombined, _, _, _ = CombineLatticesCorpus(convDisLat, convDisLat)
			}
			if allOut {
				log.Println("Setting convergence tester")
			}
			var testCombined []interface{}
			var testDisLat []interface{}
			var testAmbLat []interface{}

			if len(test) > 0 {
				if len(testGold) > 0 {
					log.Println("Reading test disambiguated lattice (for convergence testing) from", testGold)
					lConvDis, lConvDisE := lattice.ReadFile(testGold, limitdev)
					if lConvDisE != nil {
						log.Println(lConvDisE)
						return lConvDisE
					}
					if allOut {
						log.Println("Convergence Test Gold Dis. Lat.:\tRead", len(lConvDis), "disambiguated lattices")
						log.Println("Convergence Test Gold Dis. Lat.:\tConverting lattice format to internal structure")
					}

					testDisLat = lattice.Lattice2SentenceCorpus(lConvDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
				}
				if allOut {
					log.Println("Reading test ambiguous lattices from", test)
				}

				lConvAmb, lConvAmbE := lattice.ReadFile(test, limitdev)
				// lConvAmb = lConvAmb[:NUM_SENTS]
				if lConvAmbE != nil {
					log.Println(lConvAmbE)
					return lConvAmbE
				}
				// lAmb = lAmb[:NUM_SENTS]
				if allOut {
					log.Println("Read", len(lConvAmb), "ambiguous lattices from", test)
					log.Println("Converting lattice format to internal structure")
				}
				testAmbLat = lattice.Lattice2SentenceCorpus(lConvAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
				if MdCombineGold {
					var devMissingGold, devSentMissingGold, devLattices int
					testCombined, devMissingGold, devLattices, devSentMissingGold = CombineLatticesCorpus(testDisLat, testAmbLat)
					log.Println("Combined", len(testCombined), "graphs, with", devMissingGold, "lattices of", devLattices, "missing at least one gold path in lattice in", devSentMissingGold, "sentences")
				} else {
					testCombined, _, _, _ = CombineLatticesCorpus(testDisLat, testDisLat)
				}
				// if limit > 0 {
				// 	testCombined = Limit(testCombined, limit*1000)
				// 	testAmbLat = Limit(testAmbLat, limit*1000)
				// }
				// convCombined = convCombined[:100]
			}
			// TODO: replace nil param with test sentences
			evaluator = MakeJointEvalStopCondition(convAmbLat, convCombined, testAmbLat, testCombined, decodeTestBeam, perceptron.InstanceDecoder(deterministic), BeamSize)
		}
		_ = Train(goldSequences, Iterations, JointModelFile, model, perceptron.EarlyUpdateInstanceDecoder(beam), perceptron.InstanceDecoder(deterministic), evaluator)
		search.AllOut = false
		if allOut {
			log.Println("Done Training")
			// util.LogMemory()
			// log.Println()
			// serialization := &Serialization{
			// 	model.Serialize(-1),
			// 	EWord, EPOS, EWPOS, EMHost, EMSuffix, EMorphProp, ETrans, ETokens,
			// }
			// log.Println("Writing final model to", outModelFile)
			// WriteModel(outModelFile, serialization)
			// if allOut {
			// 	log.Println("Done writing model")
			// }
		}
		return nil
	} else {
		if allOut && !parseOut {
			log.Println("Found model file", outModelFile, " ... loading model")
		}
		serialization := ReadModel(outModelFile)
		model.Deserialize(serialization.WeightModel)
		EWord, EPOS, EWPOS, EMHost, EMSuffix, EMorphProp, ETrans, ETokens = serialization.EWord, serialization.EPOS, serialization.EWPOS, serialization.EMHost, serialization.EMSuffix, serialization.EMorphProp, serialization.ETrans, serialization.ETokens
		if allOut && !parseOut {
			log.Println("Loaded model")
		}
		switch DepArcSystemStr {
		case "standard":
			arcSystem = &ArcStandard{
				SHIFT:       SH.Value(),
				LEFT:        LA.Value(),
				RIGHT:       RA.Value(),
				Transitions: ETrans,
				Relations:   ERel,
			}
		case "eager":
			arcSystem = &ArcEager{
				ArcStandard: ArcStandard{
					SHIFT:       SH.Value(),
					LEFT:        LA.Value(),
					RIGHT:       RA.Value(),
					Relations:   ERel,
					Transitions: ETrans,
				},
				REDUCE:  RE.Value(),
				POPROOT: PR.Value(),
			}
		default:
			panic("Unknown arc system")
		}
		arcSystem.AddDefaultOracle()
		jointTrans.ArcSys = arcSystem
		jointTrans.Transitions = ETrans
		mdTrans.Transitions = ETrans
		mdTrans.UsePOP = UsePOP
		mdTrans.POP = POP
		disambig.UsePOP = UsePOP
		disambig.SwitchFormLemma = !lattice.IGNORE_LEMMA
		disambig.LEMMAS = !lattice.IGNORE_LEMMA
		mdTrans.AddDefaultOracle()
		jointTrans.MDTransition = MD
		jointTrans.JointStrategy = JointStrategy

		transitionSystem = transition.TransitionSystem(jointTrans)
	}

	// *** PARSING ***
	log.Println()
	log.Println("*** PARSING ***")
	log.Print("Parsing test")

	log.Println("Reading ambiguous lattices from", input)

	var (
		lAmb  []lattice.Lattice
		lAmbE error
	)
	if useConllU {
		lAmb, lAmbE = lattice.ReadUDFile(input, limit)
	} else {
		lAmb, lAmbE = lattice.ReadFile(input, limit)
	}
	if lAmbE != nil {
		log.Println(lAmbE)
		return lAmbE
	}
	// lAmb = lAmb[:NUM_SENTS]
	if allOut {
		log.Println("Read", len(lAmb), "ambiguous lattices from", input)
		log.Println("Converting lattice format to internal structure")
	}
	predAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)

	if len(inputGold) > 0 {
		log.Println("Reading test disambiguated lattice (for test ambiguous infusion)")
		var (
			lDis  []lattice.Lattice
			lDisE error
		)
		if useConllU {
			lDis, lDisE = lattice.ReadUDFile(inputGold, limit)
		} else {
			lDis, lDisE = lattice.ReadFile(inputGold, limit)
		}
		if lDisE != nil {
			log.Println(lDisE)
			return lDisE
		}
		if allOut {
			log.Println("Dev Gold Dis. Lat.:\tRead", len(lDis), "disambiguated lattices")
			log.Println("Dev Gold Dis. Lat.:\tConverting lattice format to internal structure")
		}

		predDisLat := lattice.Lattice2SentenceCorpus(lDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)

		if allOut {
			log.Println("Infusing test's dev disambiguation into ambiguous lattice")
		}

		combined, missingGold := CombineToGoldMorphs(predDisLat, predAmbLat)

		if allOut {
			log.Println("Combined", len(combined), "graphs, with", missingGold, "missing at least one gold path in lattice")

			log.Println()
		}
	}
	conf := &joint.JointConfig{
		SimpleConfiguration: SimpleConfiguration{
			EWord:         EWord,
			EPOS:          EPOS,
			EWPOS:         EWPOS,
			EMHost:        EMHost,
			EMSuffix:      EMSuffix,
			ERel:          ERel,
			ETrans:        ETrans,
			TerminalStack: terminalStack,
			TerminalQueue: 0,
		},
		MDConfig: disambig.MDConfig{
			ETokens:     ETokens,
			POP:         POP,
			Transitions: ETrans,
			ParamFunc:   paramFunc,
		},
		MDTrans: MD,
	}
	beam := &search.Beam{
		TransFunc:            transitionSystem,
		FeatExtractor:        extractor,
		Base:                 conf,
		Size:                 BeamSize,
		ConcurrentExec:       ConcurrentBeam,
		Transitions:          ETrans,
		EstimatedTransitions: 1000, // chosen by random dice roll
	}
	beam.Model = model
	beam.ShortTempAgenda = true
	parsedGraphs := Parse(predAmbLat, beam)

	if allOut {
		log.Println("Converting", len(parsedGraphs), "to conll")
	}
	if allOut {
		log.Println("Writing to output file")
	}
	var graphAsConll []interface{}
	if useConllU {
		graphAsConll = conllu.MorphGraph2ConllCorpus(parsedGraphs)
		conllu.WriteFile(outConll, graphAsConll)
	} else {
		graphAsConll = conll.MorphGraph2ConllCorpus(parsedGraphs)
		conll.WriteFile(outConll, graphAsConll)
	}
	if allOut {
		log.Println("Wrote", len(graphAsConll), "in conll format to", outConll)

		log.Println("Writing to segmentation file")
	}
	segmentation.WriteFile(outSeg, parsedGraphs)
	if allOut {
		log.Println("Wrote", len(parsedGraphs), "in segmentation format to", outSeg)

		log.Println("Writing to mapping file")
	}
	mapping.WriteFile(outMap, GetInstances(parsedGraphs, GetJointMDConfig))
	if allOut {
		log.Println("Wrote", len(parsedGraphs), "in mapping format to", outMap)

		log.Println("Writing to gold segmentation file")
	}
	return nil
}

func JointCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       JointTrainAndParse,
		UsageLine: "joint <file options> [arguments]",
		Short:     "runs joint morpho-syntactic training and parsing",
		Long: `
runs morpho-syntactic training and parsing

	$ ./yap joint -tc <conll> -td <train disamb. lat> -tl <train amb. lat> -in <input lat> -oc <out lat> -om <out map> -os <out seg> -ots <out train seg> -jointstr <joint strategy> -oraclestr <oracle strategy> [options]

`,
		Flag: *flag.NewFlagSet("joint", flag.ExitOnError),
	}
	cmd.Flag.BoolVar(&ConcurrentBeam, "bconc", true, "Concurrent Beam")
	cmd.Flag.IntVar(&Iterations, "it", 1, "Number of Perceptron Iterations")
	cmd.Flag.IntVar(&BeamSize, "b", 64, "Beam Size")
	cmd.Flag.StringVar(&JointModelFile, "m", "joint_arc_zeager_model_temp_i33.b64", "Joint model name")
	cmd.Flag.StringVar(&DepArcSystemStr, "a", "eager", "Optional - Arc System [standard, eager]")

	cmd.Flag.StringVar(&tConll, "tc", "", "Training Conll File")
	cmd.Flag.StringVar(&tLatDis, "td", "", "Training Disambiguated Lattices File")
	cmd.Flag.StringVar(&tLatAmb, "tl", "", "Training Ambiguous Lattices File")
	cmd.Flag.StringVar(&input, "in", "", "Dev Ambiguous Lattices File")
	cmd.Flag.StringVar(&inputGold, "ing", "", "Optional - Gold Dev Lattices File (for infusion/convergence into dev ambiguous)")
	cmd.Flag.StringVar(&test, "test", "", "Test Ambiguous Lattices File")
	cmd.Flag.StringVar(&testGold, "testgold", "", "Optional - Gold Test Lattices File (for infusion into test ambiguous)")
	cmd.Flag.StringVar(&outConll, "oc", "", "Output Conll File")
	cmd.Flag.StringVar(&outSeg, "os", "", "Output Segmentation File")
	cmd.Flag.StringVar(&outMap, "om", "", "Output Mapping File")
	cmd.Flag.StringVar(&tSeg, "ots", "", "Output Training Segmentation File")
	cmd.Flag.StringVar(&JointFeaturesFile, "f", "jointzeager.yaml", "Features Configuration File")
	cmd.Flag.StringVar(&DepLabelsFile, "l", "hebtb.labels.conf", "Dependency Labels Configuration File")
	cmd.Flag.StringVar(&MdParamFuncName, "p", "Funcs_Main_POS_Both_Prop", "Param Func types: ["+nlp.AllParamFuncNames+"]")
	cmd.Flag.StringVar(&JointStrategy, "jointstr", "ArcGreedy", "Joint Strategy: ["+joint.JointStrategies+"]")
	cmd.Flag.StringVar(&OracleStrategy, "oraclestr", "ArcGreedy", "Oracle Strategy: ["+joint.OracleStrategies+"]")
	cmd.Flag.BoolVar(&search.AllOut, "showbeam", false, "Show candidates in beam")
	cmd.Flag.BoolVar(&search.SHOW_ORACLE, "showoracle", false, "Show oracle transitions")
	cmd.Flag.BoolVar(&search.ShowFeats, "showfeats", false, "Show features of candidates in beam")
	cmd.Flag.BoolVar(&MdCombineGold, "infusedev", false, "Infuse gold morphs into lattices for test corpus")
	cmd.Flag.BoolVar(&UsePOP, "pop", true, "Add POP operation to MD")
	cmd.Flag.BoolVar(&lattice.IGNORE_LEMMA, "nolemma", true, "Ignore lemmas")
	cmd.Flag.BoolVar(&MdNoconverge, "noconverge", false, "don't test convergence (run -it number of iterations)")
	cmd.Flag.IntVar(&limit, "limit", 0, "limit training set (in thousands)")
	cmd.Flag.IntVar(&limitdev, "limitdev", 0, "limit dev set (in thousands)")
	cmd.Flag.BoolVar(&useConllU, "conllu", false, "use CoNLL-U-format input file (for disamb lattices)")
	cmd.Flag.BoolVar(&hebMACompat, "hebcompat", false, "Read CoNLLu with hebrew lexicon compatability")
	// cmd.Flag.BoolVar(&AlignBeam, "align", false, "Use Beam Alignment")
	// cmd.Flag.BoolVar(&AverageScores, "average", false, "Use Average Scoring")
	// cmd.Flag.BoolVar(&alignAverageParseOnly, "parseonly", false, "Use Alignment & Average Scoring in parsing only")
	return cmd
}
