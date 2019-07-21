package app

import (
	// "github.com/OnlpLab/yap/alg/featurevector"
	"fmt"
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/search"
	"github.com/OnlpLab/yap/alg/transition"
	transitionmodel "github.com/OnlpLab/yap/alg/transition/model"
	"github.com/OnlpLab/yap/nlp/format/conll"
	"github.com/OnlpLab/yap/nlp/format/conllu"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	. "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
	"github.com/OnlpLab/yap/util/conf"

	"log"
	"os"
	// "strings"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var (
	DepModelName    string
	DepFeaturesFile string
	DepLabelsFile   string
	DepModelFile   string
	//DepBeamSize   int
	DepArcSystemStr string
)

func SetupDepEnum(relations []string) {
	SetupRelationEnum(relations)
	SetupTransEnum(relations)
	EWord, EPOS, EWPOS = util.NewEnumSet(APPROX_WORDS), util.NewEnumSet(APPROX_POS), util.NewEnumSet(APPROX_WORDS*WORDS_POS_FACTOR)
	EMHost, EMSuffix = util.NewEnumSet(APPROX_MHOSTS), util.NewEnumSet(APPROX_MSUFFIXES)
	EMorphProp = util.NewEnumSet(130) // random guess of number of possible values
	// adding empty string as an element in the morph enum sets so that '0' default values
	// map to empty morphs
	EMHost.Add("")
	EMSuffix.Add("")
}

func EstimatedBeamTransitions() int {
	return ERel.Len()*2 + 2
}

func DepConfigOut(outModelFile string, b search.Interface, t transition.TransitionSystem) {
	log.Println("Configuration")
	log.Printf("Beam:             \t%s", b.Name())
	log.Printf("Transition System:\t%s", t.Name())
	log.Printf("Iterations:\t\t%d", Iterations)
	log.Printf("Beam Size:\t\t%d", BeamSize)
	log.Printf("Beam Concurrent:\t%v", ConcurrentBeam)
	log.Printf("Model file:\t\t%s", outModelFile)
	log.Printf("Use Lemmas:\t\t%v", !lattice.IGNORE_LEMMA)
	log.Printf("Word Type:\t\t%v", conll.WORD_TYPE)

	log.Println()
	log.Printf("Features File:\t%s", DepFeaturesFile)
	if !VerifyExists(DepFeaturesFile) {
		os.Exit(1)
	}
	log.Printf("Labels File:\t\t%s", DepLabelsFile)
	if !VerifyExists(DepLabelsFile) {
		os.Exit(1)
	}
	log.Println()
	log.Println("Data")
	if len(tConll) > 0 {
		log.Printf("Train file (conll):\t\t\t%s", tConll)
		if !VerifyExists(tConll) {
			return
		}
	}
	if len(inputLat) > 0 {
		log.Printf("Input file  (lattice sentences):\t%s", inputLat)
		if !VerifyExists(inputLat) {
			os.Exit(1)
		}
	}
	if len(input) > 0 {
		log.Printf("Input file  (tagged sentences):\t%s", input)
		if !VerifyExists(input) {
			os.Exit(1)
		}

	}
	if len(outConll) > 0 {
		log.Printf("Out (conll) file:\t\t\t%s", outConll)
	}
}

func DepTrainAndParse(cmd *commander.Command, args []string) error {
	// instantiate the arc system for config output only
	// it will be reinstantiated later on with struct values

	var (
		arcSystem     transition.TransitionSystem
		terminalStack int
	)
	switch DepArcSystemStr {
	case "standard":
		arcSystem = &ArcStandard{}
		terminalStack = 1
	case "eager":
		arcSystem = &ArcEager{}
		terminalStack = 0
	default:
		panic("Unknown arc system")
	}

	arcSystem.AddDefaultOracle()

	transitionSystem := transition.TransitionSystem(arcSystem)
	REQUIRED_FLAGS := []string{"oc"}

	featuresLocation, found := util.LocateFile(DepFeaturesFile, DEFAULT_CONF_DIRS)
	if found {
		DepFeaturesFile = featuresLocation
	} else {
		REQUIRED_FLAGS = append(REQUIRED_FLAGS, "f")
	}
	labelsLocation, found := util.LocateFile(DepLabelsFile, DEFAULT_CONF_DIRS)
	if found {
		DepLabelsFile = labelsLocation
	} else {
		REQUIRED_FLAGS = append(REQUIRED_FLAGS, "l")
	}
	if VerifyExists(inputLat) {
		REQUIRED_FLAGS = append(REQUIRED_FLAGS, "inl")
	} else {
		REQUIRED_FLAGS = append(REQUIRED_FLAGS, "in")
	}

	// RegisterTypes()
	var (
		outModelFile string                           = fmt.Sprintf("%s.b%d", DepModelFile, BeamSize)
		model        *transitionmodel.AvgMatrixSparse = &transitionmodel.AvgMatrixSparse{}
		modelExists  bool
	)
	// search for model file locally or in data/ path
	modelLocation, found := util.LocateFile(DepModelName, DEFAULT_MODEL_DIRS)
	if found {
		modelExists = true
		outModelFile = modelLocation
	} else {
		log.Println("Pre-trained model not found in default directories, looking for", outModelFile)
		modelExists = VerifyExists(outModelFile)
	}
	if !modelExists {
		log.Println("No model found, training")
		REQUIRED_FLAGS = []string{"it", "tc"}
		VerifyFlags(cmd, REQUIRED_FLAGS)
	}
	if allOut && !parseOut {
		DepConfigOut(outModelFile, &search.Beam{}, transitionSystem)
	}
	// modelExists := false
	relations, err := conf.ReadFile(DepLabelsFile)
	if err != nil {
		log.Println("Failed reading dependency labels configuration file:", DepLabelsFile)
		log.Fatalln(err)
	}
	if allOut && !parseOut {
		log.Println()
		// start processing - setup enumerations
		log.Println("Setup enumerations")
	}
	SetupDepEnum(relations.Values)

	// after calling SetupDepEnum, enums are instantiated and set according to the relations
	// therefore we re-instantiate the arc system with the right parameters
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

	transitionSystem = transition.TransitionSystem(arcSystem)

	if allOut && !parseOut {
		log.Println()
		log.Println("Loading features")
	}

	// features, err := conf.ReadFile(featuresFile)
	if err != nil {
		log.Println("Failed reading feature configuration file:", DepFeaturesFile)
		log.Fatalln(err)
	}
	featureSetup, err := transition.LoadFeatureConfFile(DepFeaturesFile)
	if err != nil {
		log.Println("Failed reading feature configuration file:", DepFeaturesFile)
		log.Fatalln(err)
	}
	extractor := SetupExtractor(featureSetup, []byte("A"))
	// extractor.Log = true
	group, _ := extractor.TransTypeGroups['A']
	formatters := make([]util.Format, len(group.FeatureTemplates))
	for i, formatter := range group.FeatureTemplates {
		formatters[i] = formatter
	}

	var (
		sents       []interface{}
		sentsStream chan interface{}
	)
	if !modelExists {
		var (
			asMorphGraphs []interface{}
			//goldMorphGraphs []interface{}
		)
		if allOut {
			log.Println("Model file", outModelFile, "not found, training")
		}
		var asGraphs []interface{}
		if useConllU {
			devi, _, e2 := conllu.ReadFile(input, limit)
			if e2 != nil {
				log.Fatalln(e2)
			}
			// const NUM_SENTS = 20

			// s = s[:NUM_SENTS]
			if allOut {
				log.Println("Read", len(devi), "sentences from", input)
				log.Println("Converting from conllu to internal format")
			}
			asGraphs = conllu.ConllU2GraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
			asMorphGraphs = conllu.ConllU2MorphGraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
		} else {
			devi, e2 := conll.ReadFile(input, limit)
			if e2 != nil {
				log.Fatalln(e2)
			}
			// const NUM_SENTS = 20

			// s = s[:NUM_SENTS]
			if allOut {
				log.Println("Read", len(devi), "sentences from", input)
				log.Println("Converting from conll to internal format")
			}
			asGraphs = conll.Conll2GraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
		}

		//check tagged returns morph
		sents = make([]interface{}, len(asGraphs))
		for i, instance := range asGraphs {
			sents[i] = GetAsTaggedSentence(instance)
		}
		if allOut {
			log.Println()

			log.Println("Generating Gold Sequences For Training")
			log.Println("Reading training sentences from", tConll)
		}
		var goldGraphs []interface{}
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
			goldGraphs = conllu.ConllU2GraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
			//goldMorphGraphs = conllu.ConllU2MorphGraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)

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
			goldGraphs = conll.Conll2GraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
		}
		if allOut {
			log.Println()

			log.Println("Parsing with gold to get training sequences")
		}
		// goldGraphs = goldGraphs[:NUM_SENTS]
		goldSequences := TrainingSequences(goldGraphs, GetAsTaggedSentence, GetAsLabeledDepGraph)
		if allOut {
			log.Println("Generated", len(goldSequences), "training sequences")
			log.Println()
			log.Println("Training", Iterations, "iteration(s)")
		}
		model = transitionmodel.NewAvgMatrixSparse(featureSetup.NumFeatures(), formatters, true)
		// model.Log = true

		conf := &SimpleConfiguration{
			EWord:         EWord,
			EPOS:          EPOS,
			EWPOS:         EWPOS,
			EMHost:        EMHost,
			EMSuffix:      EMSuffix,
			ERel:          ERel,
			ETrans:        ETrans,
			TerminalStack: terminalStack,
			TerminalQueue: 0,
		}

		deterministic := &search.Deterministic{
			TransFunc:          transitionSystem,
			FeatExtractor:      extractor,
			ReturnModelValue:   false,
			ReturnSequence:     true,
			ShowConsiderations: false,
			Base:               conf,
			NoRecover:          false,
			DefaultTransType:   'A', // use Arc as default transition type
		}

		beam := &search.Beam{
			TransFunc:            transitionSystem,
			FeatExtractor:        extractor,
			Base:                 conf,
			Size:                 BeamSize,
			ConcurrentExec:       ConcurrentBeam,
			EstimatedTransitions: EstimatedBeamTransitions(),
			ScoredStoreDense:     true,
		}

		var evaluator perceptron.StopCondition

		if len(inputGold) > 0 {
			if allOut {
				log.Println("Setting convergence tester")
			}
			decodeTestBeam := &search.Beam{}
			*decodeTestBeam = *beam
			decodeTestBeam.Model = model
			decodeTestBeam.DecodeTest = true
			decodeTestBeam.ShortTempAgenda = true
			var asGoldGraphs []interface{}
			var asMorphGoldGraphs []interface{}
			if useConllU {
				s, _, e := conllu.ReadFile(inputGold, limit)
				if e != nil {
					log.Println(e)
					return e
				}
				if allOut {
					log.Println("Conll:\tRead", len(s), "sentences")
					log.Println("Conll:\tConverting from conll to internal structure")
				}
				asGoldGraphs = conllu.ConllU2GraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
				asMorphGoldGraphs = conllu.ConllU2MorphGraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
			} else {
				s, e := conll.ReadFile(inputGold, limit)
				if e != nil {
					log.Println(e)
					return e
				}
				if allOut {
					log.Println("Conll:\tRead", len(s), "sentences")
					log.Println("Conll:\tConverting from conll to internal structure")
				}
				asGoldGraphs = conll.Conll2GraphCorpus(s, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
			}

			goldSents := make([]interface{}, len(asGoldGraphs))
			for i, instance := range asGraphs {
				goldSents[i] = GetAsLabeledDepGraph(instance)
			}
			var testAsGraphs []interface{}
			var testAsMorphGraphs []interface{}
			var testSents []interface{}
			if len(test) > 0 {
				if allOut {
					log.Println("Reading test file for per iteration parse")
				}
				if useConllU {
					testi, _, e3 := conllu.ReadFile(test, limit)
					if e3 != nil {
						log.Fatalln(e3)
					}
					if allOut {
						log.Println("Read", len(testi), "sentences from", test)
						log.Println("Converting from conll to internal format")
					}
					testAsGraphs = conllu.ConllU2GraphCorpus(testi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
					testAsMorphGraphs = conllu.ConllU2MorphGraphCorpus(testi, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
				} else {
					testi, e3 := conll.ReadFile(test, limit)
					if e3 != nil {
						log.Fatalln(e3)
					}
					if allOut {
						log.Println("Read", len(testi), "sentences from", test)
						log.Println("Converting from conll to internal format")
					}
					testAsGraphs = conll.Conll2GraphCorpus(testi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
				}
				testSents = make([]interface{}, len(testAsGraphs))
				for i, instance := range testAsGraphs {
					testSents[i] = GetAsTaggedSentence(instance)
				}
			}
			evaluator = MakeDepEvalStopCondition(sents, goldSents, testSents, asMorphGraphs, asMorphGoldGraphs, testAsMorphGraphs, decodeTestBeam, perceptron.InstanceDecoder(deterministic), BeamSize)
		}
		_ = Train(goldSequences, Iterations, DepModelFile, model, perceptron.EarlyUpdateInstanceDecoder(beam), perceptron.InstanceDecoder(deterministic), evaluator)
		if allOut {
			log.Println("Done Training")
			log.Println()
			log.Println("Writing model to", outModelFile)
		}
		serialization := &Serialization{
			model.Serialize(-1),
			EWord, EPOS, EWPOS, EMHost, EMSuffix, EMorphProp, ETrans, ETokens,
		}
		WriteModel(outModelFile, serialization)
		if allOut {
			log.Println("Done writing model")
		}
	} else {
		if allOut && !parseOut {
			log.Println("Found model file", outModelFile, " ... loading model")
		}
		serialization := ReadModel(outModelFile)
		model.Deserialize(serialization.WeightModel)
		EWord, EPOS, EWPOS, EMHost, EMSuffix = serialization.EWord, serialization.EPOS, serialization.EWPOS, serialization.EMHost, serialization.EMSuffix
		if allOut && !parseOut {
			log.Println("Loaded model")
		}
		// model.Log = true
	}
	if allOut {
		log.Println()
	}

	// group, _ = extractor.TransTypeGroups[transition.ConstTransition(0).Type()]
	// formatters = make([]util.Format, len(group.FeatureTemplates))
	// for i, _ := range group.FeatureTemplates {
	// 	group.FeatureTemplates[i].EWord, group.FeatureTemplates[i].EPOS, group.FeatureTemplates[i].EWPOS = EWord, EPOS, EWPOS
	// 	formatters[i] = &(group.FeatureTemplates[i])
	// }
	//
	// model.Formatters = formatters
	// sents = sents[:NUM_SENTS]
	var asMorphGraphs, asGraphs []interface{}
	if len(inputLat) > 0 {
		if Stream {
			lDisamb, lDisambE := lattice.StreamFile(inputLat, limit)
			if lDisambE != nil {
				log.Fatalln(lDisambE)
			}
			if allOut {
				log.Println("Streaming lattice conversion to sentence")
			}
			internalStream := lattice.Lattice2SentenceStream(lDisamb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
			sentsStream = make(chan interface{}, 2)
			if allOut {
				log.Println("Streaming sentence conversion to taggged sentence")
			}
			go func() {
				for instance := range internalStream {
					converted := instance.(nlp.LatticeSentence).TaggedSentence()
					sentsStream <- converted
				}
				close(sentsStream)
			}()
		} else {
			lDisamb, lDisambE := lattice.ReadFile(inputLat, limit)
			if lDisambE != nil {
				log.Fatalln(lDisambE)
			}
			if allOut {
				log.Println("Read", len(lDisamb), "disambiguated lattices from", inputLat)
				log.Println("Converting lattice format to TaggedSentence internal structure")
				log.Println("\tlattice format to sentence")
			}
			internalSents := lattice.Lattice2SentenceCorpus(lDisamb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
			if allOut {
				log.Println("\tsentence to TaggedSentence")
			}
			sents = make([]interface{}, len(internalSents))
			for i, instance := range internalSents {
				sents[i] = instance.(nlp.LatticeSentence).TaggedSentence()
			}
		}
	} else {
		if useConllU {
			devi, _, e2 := conllu.ReadFile(input, limit)
			if e2 != nil {
				log.Fatalln(e2)
			}
			// const NUM_SENTS = 20

			// s = s[:NUM_SENTS]
			if allOut {
				log.Println("Read", len(devi), "sentences from", input)
				log.Println("Converting from conllu to internal format")
			}
			asGraphs = conllu.ConllU2GraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
			asMorphGraphs = conllu.ConllU2MorphGraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
		} else {
			devi, e2 := conll.ReadFile(input, limit)
			if e2 != nil {
				log.Fatalln(e2)
			}
			// const NUM_SENTS = 20

			// s = s[:NUM_SENTS]
			if allOut {
				log.Println("Read", len(devi), "sentences from", input)
				log.Println("Converting from conll to internal format")
			}
			asGraphs = conll.Conll2GraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)
		}
		sents = make([]interface{}, len(asGraphs))
		for i, instance := range asGraphs {
			sents[i] = GetAsTaggedSentence(instance)
		}
	}

	conf := &SimpleConfiguration{
		EWord:         EWord,
		EPOS:          EPOS,
		EWPOS:         EWPOS,
		EMHost:        EMHost,
		EMSuffix:      EMSuffix,
		ERel:          ERel,
		ETrans:        ETrans,
		TerminalStack: terminalStack,
		TerminalQueue: 0,
	}

	beam := &search.Beam{
		TransFunc:            transitionSystem,
		FeatExtractor:        extractor,
		Base:                 conf,
		Model:                model,
		Size:                 BeamSize,
		ConcurrentExec:       ConcurrentBeam,
		ShortTempAgenda:      true,
		EstimatedTransitions: EstimatedBeamTransitions(),
		ScoredStoreDense:     true,
	}
	if Stream {
		parsedStream := make(chan interface{}, 2)
		if allOut {
			log.Println("Starting parser")
		}
		go ParseStream(sentsStream, parsedStream, beam)
		log.Println("Streaming conversion to conll")
		graphAsConllStream := conll.Graph2ConllStream(parsedStream, EMHost, EMSuffix)
		if allOut {
			log.Println("Creating writer stream to", outConll)
		}
		conll.WriteStreamToFile(outConll, graphAsConllStream)
		return nil
	}
	if allOut {
		if parseOut {
			log.SetPrefix("")
			log.SetFlags(0)
			log.Print("Parsing started")
		} else {
			log.Print("Parsing")
		}

		parsedGraphs := Parse(sents, beam)
		if !parseOut {
			log.Println("Converting to conll")
		}
		if useConllU {
			graphAsConll := conllu.Graph2ConllUCorpus(parsedGraphs, EMHost, EMSuffix)
			morphGraphs := conllu.MergeGraphAndMorphCorpus(graphAsConll, asMorphGraphs)
			conllu.WriteFile(outConll, morphGraphs)
			if !parseOut {
				log.Println("Wrote", len(parsedGraphs), "in conllu format to", outConll)
			}
		} else {
			graphAsConll := conll.Graph2ConllCorpus(parsedGraphs, EMHost, EMSuffix)
			conll.WriteFile(outConll, graphAsConll)
			if !parseOut {
				log.Println("Wrote", len(parsedGraphs), "in conll format to", outConll)
			}
		}
	} else {
		search.AllOut = true
		// runtime.GOMAXPROCS(1)
		model.Log = true
		search.AllOut = true
		log.SetPrefix("")
		log.SetFlags(0)
		log.Print("Parsing started")
		parsedGraphs := Parse(sents, beam)
		graphAsConll := conll.Graph2ConllCorpus(parsedGraphs, EMHost, EMSuffix)
		conll.WriteFile(outConll, graphAsConll)
		log.Println("Wrote", len(parsedGraphs), "in conll format to", outConll)
	}
	return nil
}

func DepCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       DepTrainAndParse,
		UsageLine: "dep <file options> [arguments]",
		Short:     "runs dependency training/parsing",
		Long: `
runs dependency training/parsing

	$ ./yap dep -f <features> -l <labels> -tc <conll> -in <input tagged> -oc <out conll> [-a eager|standard] [options]

`,
		Flag: *flag.NewFlagSet("dep", flag.ExitOnError),
	}
	cmd.Flag.BoolVar(&ConcurrentBeam, "bconc", true, "Concurrent Beam")
	cmd.Flag.IntVar(&Iterations, "it", 1, "Number of Perceptron Iterations")
	cmd.Flag.IntVar(&BeamSize, "b", 64, "Dependency Beam Size")
	cmd.Flag.StringVar(&DepModelFile, "m", "model", "Prefix for model file ({m}.b{b}.i{it}.model)")
	cmd.Flag.StringVar(&DepModelName, "mn", "dep.b64", "Modelfile")
	cmd.Flag.StringVar(&DepArcSystemStr, "a", "eager", "Optional - Arc System [standard, eager]")

	cmd.Flag.StringVar(&tConll, "tc", "", "Training Conll File")
	cmd.Flag.StringVar(&input, "in", "", "Dev Tagged Sentences File")
	cmd.Flag.StringVar(&inputLat, "inl", "", "Input Lattice Disambiguated Sentences File")
	cmd.Flag.StringVar(&inputGold, "ing", "", "Optional - Dev Gold Parsed Sentences (for convergence)")
	cmd.Flag.StringVar(&test, "test", "", "Test Conll File")
	cmd.Flag.StringVar(&outConll, "oc", "", "Output Conll File")
	cmd.Flag.StringVar(&DepFeaturesFile, "f", "zhangnivre2011.yaml", "Features Configuration File")
	cmd.Flag.StringVar(&DepLabelsFile, "l", "hebtb.labels.conf", "Dependency Labels Configuration File")
	//cmd.Flag.BoolVar(&conll.IGNORE_LEMMA, "nolemma", false, "Ignore lemmas")
	cmd.Flag.BoolVar(&lattice.IGNORE_LEMMA, "nolemma", true, "Ignore lemmas")
	cmd.Flag.StringVar(&conll.WORD_TYPE, "wordtype", "form", "Word type [form, lemma, lemma+f (=lemma if present else form)]")
	cmd.Flag.IntVar(&limit, "limit", 0, "limit training set")
	cmd.Flag.BoolVar(&search.SHOW_ORACLE, "showoracle", false, "Show oracle transitions")
	cmd.Flag.BoolVar(&search.AllOut, "showbeam", false, "Show candidates in beam")
	cmd.Flag.BoolVar(&useConllU, "conllu", false, "use CoNLL-U-format input file (for disamb lattices)")
	cmd.Flag.BoolVar(&Stream, "stream", false, "Stream data from input through parser to output")
	return cmd
}
