package app

import (
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/search"
	"github.com/OnlpLab/yap/alg/transition"
	transitionmodel "github.com/OnlpLab/yap/alg/transition/model"
	"github.com/OnlpLab/yap/nlp/format/conllu"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/format/mapping"

	"github.com/OnlpLab/yap/nlp/parser/dependency/transition/morph"
	"github.com/OnlpLab/yap/nlp/parser/disambig"

	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	"fmt"
	"log"
	"os"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
	"github.com/OnlpLab/yap/nlp/format/conllul"
)

var (
	MdParamFuncName  string
	MdUseWB          bool
	MdCombineGold    bool
	MdNoconverge     bool
	MdModelName    string
	MdModelFile    string
	MdFeaturesFile string
	//MdBeamSize     int
)

func SetupMDEnum() {
	EWord, EPOS, EWPOS = util.NewEnumSet(APPROX_WORDS), util.NewEnumSet(APPROX_POS), util.NewEnumSet(APPROX_WORDS*5)
	EMHost, EMSuffix = util.NewEnumSet(APPROX_MHOSTS), util.NewEnumSet(APPROX_MSUFFIXES)

	ETrans = util.NewEnumSet(10000)
	_, _ = ETrans.Add("IDLE") // dummy no action transition for zpar equivalence
	iPOP, _ := ETrans.Add("POP")

	POP = &transition.TypedTransition{'P', iPOP}

	EMorphProp = util.NewEnumSet(10000) // random guess of number of possible values
	ETokens = util.NewEnumSet(10000)
}

func CombineToGoldMorph(goldLat, ambLat nlp.LatticeSentence) (m *disambig.MDConfig, spelloutsAdded int) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Recovered error", r, "excluding from training corpus")
			m = nil
		}
	}()
	// generate graph

	// generate morph. disambiguation (= mapping) and nodes
	mappings := make([]*nlp.Mapping, len(goldLat))
	for i, lat := range goldLat {
		// log.Println("At lat", i)
		lat.GenSpellouts()
		lat.GenToken()
		if len(lat.Spellouts) == 0 {
			continue
		}
		mapping := &nlp.Mapping{
			lat.Token,
			lat.Spellouts[0],
		}
		// if the gold spellout doesn't exist in the lattice, add it
		if len(ambLat[i].Spellouts) == 0 {
			ambLat[i].GenSpellouts()
		}
		_, exists := ambLat[i].Spellouts.Find(mapping.Spellout)
		if !exists {
			// log.Println(mapping.Spellout, "Spellout not found")
			log.Println(i+1, lat.Spellouts[0].AsString())
			ambLat[i].Spellouts = append(ambLat[i].Spellouts, mapping.Spellout)
			spelloutsAdded++
			prevTop := ambLat[i].Top()
			ambLat[i].AddAnalysis(nil, []nlp.BasicMorphemes{nlp.Morphemes(lat.Spellouts[0]).AsBasic()}, i)
			// log.Println("Lattice is now:")
			// log.Println(ambLat)
			diff := ambLat[i].Top() - prevTop
			if diff > 0 {
				for j := i + 1; j < len(ambLat); j++ {
					ambLat[j].BumpAll(diff)
				}
			}
			// ambLat[i].UnionPath(&lat)
		} else {
			// log.Println(mapping.Spellout, "Spellout found")
		}
		// ambLat[i].BridgeMissingMorphemes()

		mappings[i] = mapping
	}

	m = &disambig.MDConfig{
		Mappings: mappings,
		Lattices: ambLat,
	}
	return m, spelloutsAdded
}

func CombineLatticesCorpus(goldLats, ambLats []interface{}) ([]interface{}, int, int, int) {
	var (
		numSentNoGold, numLatticeNoGold int
		totalLattices                   int
	)
	prefix := log.Prefix()
	configs := make([]interface{}, 0, len(goldLats))
	f := log.Flags()
	log.SetFlags(0)
	for i, goldMap := range goldLats {
		log.SetPrefix(fmt.Sprintf("%d ", i))
		ambLat := ambLats[i].(nlp.LatticeSentence)
		totalLattices += len(ambLat)
		// log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		result, numNoGold := CombineToGoldMorph(goldMap.(nlp.LatticeSentence), ambLat)
		if numNoGold > 0 {
			numSentNoGold += 1
			numLatticeNoGold += numNoGold
		}
		if result != nil {
			configs = append(configs, result)
		}
	}
	log.SetFlags(f)
	log.SetPrefix(prefix)
	return configs, numLatticeNoGold, totalLattices, numSentNoGold
}

func conllul2Lattices(cls []conllul.ConlluLattice) ([]lattice.Lattice) {
	result := []lattice.Lattice{}
	for _, cl := range cls {
		result = append(result, conllul2Lattice(cl));
	}
	return result;
}

func conllul2Lattice(cl conllul.ConlluLattice) (lattice.Lattice) {
	result := make(lattice.Lattice)
	for _, v := range cl.Edges {
		for _, ce := range v {
			le := lattice.Edge{}
			le.Id = ce.Id
			le.Start = ce.Start
			le.End = ce.End
			le.Lemma = ce.Lemma
			le.Word = ce.Word
			le.CPosTag = ce.UPosTag
			le.PosTag = ce.XPosTag
			le.Feats = ce.Feats
			le.FeatStr = ce.FeatStr
			le.Token = ce.TokenId
			le.TokenStr = cl.Tokens[ce.TokenId-1]
			edges, exists := result[le.Start]
			if exists {
				result[le.Start] = append(edges, le)
			} else {
				result[le.Start] = []lattice.Edge{le}
			}
		}
	}
	return result
}


func MDConfigOut(outModelFile string, b search.Interface, t transition.TransitionSystem) {
	log.Println("Configuration")
	log.Printf("Beam:\t\t%s", b.Name())
	log.Printf("Transition System:\t%s", t.Name())
	log.Printf("Iterations:\t\t%d", Iterations)
	log.Printf("Beam Size:\t\t%d", BeamSize)
	log.Printf("Beam Concurrent:\t%v", ConcurrentBeam)
	log.Printf("Parameter Func:\t%v", MdParamFuncName)
	log.Printf("Use POP:\t\t%v", UsePOP)
	log.Printf("Infuse Gold Dev:\t%v", MdCombineGold)
	log.Printf("Use Lemmas:\t\t%v", !lattice.IGNORE_LEMMA)
	log.Printf("Use CoNLL-U:\t\t%v", useConllU)
	log.Printf("No NNP Feat:\t\t%v", lattice.IGNORE_NNP_FEATS)
	log.Printf("Limit:\t\t%v", limit)
	if len(outModelFile) > 0 {
		log.Printf("Model file:\t\t%s", outModelFile)
	}

	log.Println()
	log.Printf("Features File:\t%s", MdFeaturesFile)
	if !VerifyExists(MdFeaturesFile) {
		os.Exit(1)
	}
	log.Println()
	log.Println("Data")
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
	if len(outMap) > 0 {
		log.Printf("Out (disamb.) file:\t\t\t%s", outMap)
	}
}

func MDTrainAndParse(cmd *commander.Command, args []string) error {
	//BeamSize = MdBeamSize
	paramFunc, exists := nlp.MDParams[MdParamFuncName]
	if !exists {
		log.Fatalln("Param Func", MdParamFuncName, "does not exist")
	}
	var (
		mdTrans transition.TransitionSystem
		model   *transitionmodel.AvgMatrixSparse = &transitionmodel.AvgMatrixSparse{}
	)
	if MdUseWB {
		mdTrans = &disambig.MDWBTrans{
			ParamFunc: paramFunc,
			UsePOP:    UsePOP,
		}
	} else {
		mdTrans = &disambig.MDTrans{
			ParamFunc: paramFunc,
			UsePOP:    UsePOP,
		}
	}
	disambig.UsePOP = UsePOP

	// arcSystem := &morph.Idle{morphArcSystem, IDLE}
	transitionSystem := transition.TransitionSystem(mdTrans)

	REQUIRED_FLAGS := []string{"in", "om"}

	featuresLocation, found := util.LocateFile(MdFeaturesFile, DEFAULT_CONF_DIRS)
	if found {
		MdFeaturesFile = featuresLocation
	} else {
		REQUIRED_FLAGS = append(REQUIRED_FLAGS, "f")
	}
	VerifyFlags(cmd, REQUIRED_FLAGS)

	var (
		outModelFile string = fmt.Sprintf("%s.b%d", MdModelFile, BeamSize)
		modelExists  bool
	)
	// search for model file locally or in data/ path
	modelLocation, found := util.LocateFile(MdModelName, DEFAULT_MODEL_DIRS)
	if found {
		modelExists = true
		outModelFile = modelLocation
	} else {
		log.Println("Pre-trained model not found in default directories, looking for", outModelFile)
		modelExists = VerifyExists(outModelFile)
	}

	if !modelExists {
		log.Println("No model found, training")
		REQUIRED_FLAGS = []string{"it", "td", "tl"}
		VerifyFlags(cmd, REQUIRED_FLAGS)
	}

	// RegisterTypes()

	confBeam := &search.Beam{}
	if !alignAverageParseOnly {
		confBeam.Align = AlignBeam
		confBeam.Averaged = AverageScores
	}

	MDConfigOut(outModelFile, confBeam, transitionSystem)

	disambig.SwitchFormLemma = !lattice.IGNORE_LEMMA
	if allOut {
		log.Println()
		// start processing - setup enumerations
		log.Println("Setup enumerations")
	}
	SetupMDEnum()
	if MdUseWB {
		mdTrans.(*disambig.MDWBTrans).POP = POP
		mdTrans.(*disambig.MDWBTrans).Transitions = ETrans
	} else {
		mdTrans.(*disambig.MDTrans).POP = POP
		mdTrans.(*disambig.MDTrans).Transitions = ETrans
	}
	mdTrans.AddDefaultOracle()
	if allOut {
		log.Println()
		log.Println("Loading features")
	}
	featureSetup, err := transition.LoadFeatureConfFile(MdFeaturesFile)
	if err != nil {
		log.Println("Failed reading feature configuration file:", MdFeaturesFile)
		log.Fatalln(err)
	}
	extractor := SetupExtractor(featureSetup, []byte("MPL"))

	log.Println()
	if useConllU {
		nlp.InitOpenParamFamily("UD")
	} else {
		nlp.InitOpenParamFamily("HEBTB")
	}
	log.Println()

	if !modelExists {
		if allOut {
			log.Println("Generating Gold Sequences For Training")
		}

		const NUM_SENTS = 10
		var goldDisLat, goldAmbLat []interface{}
		if useConllU {
			conllu.IGNORE_LEMMA = lattice.IGNORE_LEMMA
			if allOut {
				log.Println("Dis. Lat.:\tReading training disambiguated lattices from (conllU)", tLatDis)
			}
			conllus, hasSegmentation, err := conllu.ReadFile(tLatDis, limit)
			if err != nil {
				log.Println(err)
				return err
			}
			if allOut {
				if hasSegmentation {
					log.Println("Dis. Lat.:\tRead", len(conllus), "disambiguated lattices (conllU) WITH SEGMENTATION")
				} else {
					log.Println("Dis. Lat.:\tRead", len(conllus), "disambiguated lattices (conllU) WITHOUT SEGMENTATION")
				}
				log.Println("Dis. Lat.:\tConverting lattice format to internal structure")
			}
			ERel = util.NewEnumSet(100)
			morphGraphs := conllu.ConllU2MorphGraphCorpus(conllus, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
			goldDisLat = make([]interface{}, len(morphGraphs))
			for i, val := range morphGraphs {
				basicMorphGraph := val.(*morph.BasicMorphGraph)
				goldDisLat[i] = basicMorphGraph.Lattice
			}
			if allOut {
				log.Println("Amb. Lat:\tReading ambiguous conllu lattices from", tLatAmb)
			}
			//lAmb, lAmbE := lattice.ReadUDFile(tLatAmb, limit)
			lAmb, lAmbE := lattice.ReadULFile(tLatAmb, limit)
			if lAmbE != nil {
				log.Println(lAmbE)
				return lAmbE
			}
			//clAmb, clAmbE := conllul.ReadFile(tLatAmb, limit)
			//if clAmbE != nil {
			//	log.Println(clAmbE)
			//	return clAmbE
			//}
			//lAmb := conllul2Lattices(clAmb)
			if allOut {
				log.Println("Amb. Lat:\tRead", len(lAmb), "ambiguous lattices")
				log.Println("Amb. Lat:\tConverting lattice format to internal structure")
			}
			goldAmbLat = lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
		} else {
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
			if allOut {
				log.Println("Amb. Lat:\tReading ambiguous lattices from", tLatAmb)
			}
			lAmb, lAmbE := lattice.ReadFile(tLatAmb, limit)
			if lAmbE != nil {
				log.Println(lAmbE)
				return lAmbE
			}
			if allOut {
				log.Println("Amb. Lat:\tRead", len(lAmb), "ambiguous lattices")
				log.Println("Amb. Lat:\tConverting lattice format to internal structure")
			}
			goldAmbLat = lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
		}
		if allOut {
			log.Println("Combining train files into gold morph graphs with original lattices")
		}
		combined, missingGold, numLattices, sentMissingGold := CombineLatticesCorpus(goldDisLat, goldAmbLat)
		if limit > 0 {
			combined = Limit(combined, limit*1000)
		}

		if allOut {
			log.Println("Combined", len(combined), "graphs, with", missingGold, "lattices of", numLattices, "missing at least one gold path in lattice in", sentMissingGold, "sentences")
			log.Println()
		}

		if allOut {
			log.Println()

			log.Println("Parsing with gold to get training sequences")
		}
		// combined = combined[:NUM_SENTS]
		goldSequences := TrainingSequences(combined, GetMDConfigAsLattices, GetMDConfigAsMappings)
		if allOut {
			log.Println("Generated", len(goldSequences), "training sequences")
			log.Println()
			// util.LogMemory()
			log.Println("Training", Iterations, "iteration(s)")
		}
		group, _ := extractor.TransTypeGroups['M']
		formatters := make([]util.Format, len(group.FeatureTemplates))
		for i, formatter := range group.FeatureTemplates {
			formatters[i] = formatter
		}
		model = transitionmodel.NewAvgMatrixSparse(NumFeatures, formatters, false)

		conf := &disambig.MDConfig{
			ETokens:     ETokens,
			POP:         POP,
			Transitions: ETrans,
			ParamFunc:   paramFunc,
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

		// old research stuff
		// if !alignAverageParseOnly {
		// 	beam.Align = AlignBeam
		// 	beam.Averaged = AverageScores
		// }

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

		var (
			lConvAmb []lattice.Lattice
			lConvAmbE error
			convCombined []interface{}
			convDisLat []interface{}
			convAmbLat []interface{}
		)

		if len(inputGold) > 0 {
			log.Println("Reading dev test disambiguated lattice (for convergence testing) from", inputGold)
			if useConllU {
				conllus, _, err := conllu.ReadFile(inputGold, limit)
				if err != nil {
					log.Println(err)
					return err
				}
				// conllus = conllus[:NUM_SENTS]
				if allOut {
					log.Println("Dev Gold Dis. Lat.:\tRead", len(conllus), "disambiguated lattices")
					log.Println("Dev Gold Dis. Lat.:\tConverting lattice format to internal structure")
				}
				morphGraphs := conllu.ConllU2MorphGraphCorpus(conllus, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
				convDisLat = make([]interface{}, len(morphGraphs))
				for i, val := range morphGraphs {
					basicMorphGraph := val.(*morph.BasicMorphGraph)
					convDisLat[i] = basicMorphGraph.Lattice
				}
			} else {
				lConvDis, lConvDisE := lattice.ReadFile(inputGold, limit)
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

			if useConllU {
				//lConvAmb, lConvAmbE = lattice.ReadUDFile(input, limit)
				lConvAmb, lConvAmbE = lattice.ReadULFile(input, limit)
				if lConvAmbE != nil {
					log.Println(lConvAmbE)
					return lConvAmbE
				}
				//clAmb, clAmbE := conllul.ReadFile(input, limit)
				//if clAmbE != nil {
				//	log.Println(clAmbE)
				//	return clAmbE
				//}
				//lConvAmb = conllul2Lattices(clAmb)
			} else {
				lConvAmb, lConvAmbE = lattice.ReadFile(input, limit)
				if lConvAmbE != nil {
					log.Println(lConvAmbE)
					return lConvAmbE
				}
			}
			//lConvAmb, lConvAmbE := lattice.ReadFile(input, limit)
			// lConvAmb = lConvAmb[:NUM_SENTS]
			//if lConvAmbE != nil {
			//	log.Println(lConvAmbE)
			//	return lConvAmbE
			//}
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

			// if limit > 0 {
			// 	convCombined = Limit(convCombined, limit*1000)
			// 	convAmbLat = Limit(convAmbLat, limit*1000)
			// 	log.Println("Limited to", limit*1000)
			// }
			// convCombined = convCombined[:100]
		}

		var testCombined []interface{}
		var testDisLat []interface{}
		var testAmbLat []interface{}

		if len(test) > 0 {
			log.Println("Reading test disambiguated lattice (for convergence testing) from", testGold)
			if useConllU {
				conllus, _, err := conllu.ReadFile(testGold, 0)
				if err != nil {
					log.Println(err)
					return err
				}
				// conllus = conllus[:NUM_SENTS]
				if allOut {
					log.Println("Test Gold Dis. Lat.:\tRead", len(conllus), "disambiguated lattices")
					log.Println("Test Gold Dis. Lat.:\tConverting lattice format to internal structure")
				}
				morphGraphs := conllu.ConllU2MorphGraphCorpus(conllus, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
				testDisLat = make([]interface{}, len(morphGraphs))
				for i, val := range morphGraphs {
					basicMorphGraph := val.(*morph.BasicMorphGraph)
					testDisLat[i] = basicMorphGraph.Lattice
				}
			} else {
				lConvDis, lConvDisE := lattice.ReadFile(testGold, 0)
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

			lConvAmb, lConvAmbE := lattice.ReadFile(test, 0)
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
		decodeTestBeam := &search.Beam{}
		*decodeTestBeam = *beam
		decodeTestBeam.Model = model
		decodeTestBeam.DecodeTest = true
		decodeTestBeam.ShortTempAgenda = true
		log.Println("Parse beam alignment:", AlignBeam)
		decodeTestBeam.Align = AlignBeam
		log.Println("Parse beam averaging:", AverageScores)
		decodeTestBeam.Averaged = AverageScores
		var evaluator perceptron.StopCondition
		if len(inputGold) > 0 {
			if !MdNoconverge {
				if allOut {
					log.Println("Setting convergence tester")
				}
				evaluator = MakeMorphEvalStopCondition(convAmbLat, convCombined, testAmbLat, testCombined, decodeTestBeam, perceptron.InstanceDecoder(deterministic), BeamSize)
			}
		}
		_ = Train(goldSequences, Iterations, MdModelFile, model, perceptron.EarlyUpdateInstanceDecoder(beam), perceptron.InstanceDecoder(deterministic), evaluator)

		if allOut {
			log.Println("Done Training")
			// util.LogMemory()
			log.Println()
			log.Println("Writing final model to", outModelFile)
			serialization := &Serialization{
				model.Serialize(-1),
				EWord, EPOS, EWPOS, EMHost, EMSuffix, EMorphProp, ETrans, ETokens,
			}
			WriteModel(outModelFile, serialization)
			log.Println("Done")
			// log.Print("Parsing test")
		}
		return nil
	}
	if allOut {
		log.Println("Found model file", outModelFile, " ... loading model")
	}
	serialization := ReadModel(outModelFile)
	model.Deserialize(serialization.WeightModel)
	EWord, EPOS, EWPOS, EMHost, EMSuffix, EMorphProp, ETrans, ETokens = serialization.EWord, serialization.EPOS, serialization.EWPOS, serialization.EMHost, serialization.EMSuffix, serialization.EMorphProp, serialization.ETrans, serialization.ETokens

	if MdUseWB {
		mdTrans = &disambig.MDWBTrans{
			ParamFunc:   paramFunc,
			UsePOP:      UsePOP,
			POP:         POP,
			Transitions: ETrans,
		}
	} else {
		mdTrans = &disambig.MDTrans{
			ParamFunc:   paramFunc,
			UsePOP:      UsePOP,
			POP:         POP,
			Transitions: ETrans,
		}
	}

	transitionSystem = transition.TransitionSystem(mdTrans)
	extractor = SetupExtractor(featureSetup, []byte("MPL"))

	// setup configuration and beam
	conf := &disambig.MDConfig{
		ETokens:     ETokens,
		POP:         POP,
		Transitions: ETrans,
		ParamFunc:   paramFunc,
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
	if Stream {

		if allOut {
			log.Println("Amb. Lat:\tReading ambiguous conllu lattices from", input)
		}
		lAmb, lAmbE := lattice.StreamFile(input, limit)
		if lAmbE != nil {
			log.Println(lAmbE)
			return lAmbE
		}
		if allOut {
			log.Println("Streaming to lattice conversion")
		}
		predAmbLatStream := lattice.Lattice2SentenceStream(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
		beam.ShortTempAgenda = true
		beam.Model = model
		mappings := make(chan interface{}, 2)
		if allOut {
			log.Println("Starting parser")
		}
		go ParseStream(predAmbLatStream, mappings, beam)
		if allOut {
			log.Println("Creating writer stream to", outMap)
		}
		mapping.WriteStreamToFile(outMap, mappings)

		return nil
	}
	var (
		lAmb  lattice.Lattices
		lAmbE error
		clAmb []conllul.ConlluLattice
		clAmbE error
	)
	if useConllU {

		if allOut {
			log.Println("Amb. Lat:\tReading ambiguous conllu lattices from", input)
		}
		//lAmb, lAmbE = lattice.ReadUDFile(input, limit)
		clAmb, clAmbE = conllul.ReadFile(input, limit)
		if clAmbE != nil {
			log.Println(clAmbE)
			return clAmbE
		}
		lAmb = conllul2Lattices(clAmb)
		if allOut {
			log.Println("Amb. Lat:\tRead", len(lAmb), "ambiguous lattices")
			log.Println("Amb. Lat:\tConverting lattice format to internal structure")
		}
	} else {
		if allOut {
			log.Println("Reading ambiguous lattices from", input)
		}

		lAmb, lAmbE = lattice.ReadFile(input, limit)
		if lAmbE != nil {
			log.Println(lAmbE)
			return lAmbE
		}
		// lAmb = lAmb[:NUM_SENTS]
		if allOut {
			log.Println("Read", len(lAmb), "ambiguous lattices from", input)
			log.Println("Converting lattice format to internal structure")
		}
	}
	predAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)

	if len(inputGold) > 0 {
		log.Println("Reading test disambiguated lattice (for test ambiguous infusion)")
		var predDisLat []interface{}
		if useConllU {
			conllus, _, err := conllu.ReadFile(tLatDis, limit)
			if err != nil {
				log.Println(err)
				return err
			}
			// conllus = conllus[:NUM_SENTS]
			if allOut {
				log.Println("Test Gold Dis. Lat.:\tRead", len(conllus), "disambiguated lattices")
				log.Println("Test Gold Dis. Lat.:\tConverting lattice format to internal structure")
			}
			morphGraphs := conllu.ConllU2MorphGraphCorpus(conllus, EWord, EPOS, EWPOS, ERel, EMorphProp, EMHost, EMSuffix)
			predDisLat = make([]interface{}, len(morphGraphs))
			for i, val := range morphGraphs {
				basicMorphGraph := val.(*morph.BasicMorphGraph)
				predDisLat[i] = basicMorphGraph.Lattice
			}
		} else {
			lDis, lDisE := lattice.ReadFile(inputGold, limit)
			if lDisE != nil {
				log.Println(lDisE)
				return lDisE
			}
			if allOut {
				log.Println("Test Gold Dis. Lat.:\tRead", len(lDis), "disambiguated lattices")
				log.Println("Test Gold Dis. Lat.:\tConverting lattice format to internal structure")
			}

			predDisLat = lattice.Lattice2SentenceCorpus(lDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
		}

		if allOut {
			log.Println("Infusing test's gold disambiguation into ambiguous lattice")
		}

		_, missingGold, numLattices, sentMissingGold := CombineLatticesCorpus(predDisLat, predAmbLat)

		if allOut {
			log.Println("Combined", len(predAmbLat), "graphs, with", missingGold, "lattices of", numLattices, "missing at least one gold path in lattice in", sentMissingGold, "sentences")
			log.Println()
		}
	}
	beam.ShortTempAgenda = true
	beam.Model = model

	mappings := Parse(predAmbLat, beam)

	/*	if allOut {
			log.Println("Converting", len(parsedGraphs), "to conll")
		}
	*/ // // // graphAsConll := conll.MorphGraph2ConllCorpus(parsedGraphs)
	// // // if allOut {
	// // // 	log.Println("Writing to output file")
	// // // }
	// // conll.WriteFile(outLat, graphAsConll)
	// if allOut {
	// 	log.Println("Wrote", len(graphAsConll), "in conll format to", outLat)

	// 	log.Println("Writing to segmentation file")
	// }
	// segmentation.WriteFile(outSeg, parsedGraphs)
	// if allOut {
	// 	log.Println("Wrote", len(parsedGraphs), "in segmentation format to", outSeg)

	// 	log.Println("Writing to gold segmentation file")
	// }
	// segmentation.WriteFile(tSeg, ToMorphGraphs(combined))

	if allOut {
		log.Println("Writing to mapping file")
	}
	if useConllU {
		mapping.UDWriteFile(outMap, mappings, clAmb)
	} else {
		mapping.WriteFile(outMap, mappings)
	}

	if allOut {
		log.Println("Wrote", len(mappings), "in mapping format to", outMap)
	}
	return nil
}

func MdCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       MDTrainAndParse,
		UsageLine: "md <file options> [arguments]",
		Short:     "runs standalone morphological disambiguation training and parsing",
		Long: `
runs standalone morphological disambiguation training and parsing

	$ ./yap md -td <train disamb. lat> -tl <train amb. lat> -in <input lat> [-ing <input lat>] -om <out disamb> -f <feature file> [-p <param func>] [options]

`,
		Flag: *flag.NewFlagSet("md", flag.ExitOnError),
	}
	cmd.Flag.BoolVar(&ConcurrentBeam, "bconc", true, "Concurrent Beam")
	cmd.Flag.IntVar(&Iterations, "it", 1, "Minimum Number of Perceptron Iterations")
	cmd.Flag.IntVar(&BeamSize, "b", 32, "Beam Size")
	cmd.Flag.StringVar(&MdModelFile, "m", "model", "Prefix for model file ({m}.b{b}.model)")
	cmd.Flag.StringVar(&MdModelName, "mn", "hebmd.b32", "Modelfile")

	cmd.Flag.StringVar(&tLatDis, "td", "", "Training Disambiguated Lattices File")
	cmd.Flag.StringVar(&tLatAmb, "tl", "", "Training Ambiguous Lattices File")
	cmd.Flag.StringVar(&input, "in", "", "Dev-Test Ambiguous Lattices File")
	cmd.Flag.StringVar(&inputGold, "ing", "", "Optional - Gold Dev-Test Lattices File (for infusion into dev-test ambiguous)")
	cmd.Flag.StringVar(&test, "test", "", "Test Ambiguous Lattices File")
	cmd.Flag.StringVar(&testGold, "testgold", "", "Optional - Gold Test Lattices File (for infusion into test ambiguous)")
	cmd.Flag.StringVar(&outMap, "om", "", "Output Mapping File")
	cmd.Flag.StringVar(&MdFeaturesFile, "f", "standalone.md.yaml", "Features Configuration File")
	cmd.Flag.StringVar(&MdParamFuncName, "p", "Funcs_Main_POS_Both_Prop", "Param Func types: ["+nlp.AllParamFuncNames+"]")
	cmd.Flag.BoolVar(&AlignBeam, "align", false, "Use Beam Alignment")
	cmd.Flag.BoolVar(&AverageScores, "average", false, "Use Average Scoring")
	cmd.Flag.BoolVar(&alignAverageParseOnly, "parseonly", false, "Use Alignment & Average Scoring in parsing only")
	cmd.Flag.BoolVar(&UsePOP, "pop", true, "Add POP operation to MD")
	cmd.Flag.BoolVar(&lattice.IGNORE_LEMMA, "nolemma", true, "Ignore lemmas")
	cmd.Flag.BoolVar(&lattice.IGNORE_NNP_FEATS, "stripnnpfeats", false, "Strip all NNPs of features")
	cmd.Flag.BoolVar(&MdUseWB, "wb", false, "Word Based MD")
	cmd.Flag.BoolVar(&search.AllOut, "showbeam", false, "Show candidates in beam")
	cmd.Flag.BoolVar(&search.SHOW_ORACLE, "showoracle", false, "Show oracle transitions")
	cmd.Flag.BoolVar(&search.ShowFeats, "showfeats", false, "Show features of candidates in beam")
	cmd.Flag.BoolVar(&MdCombineGold, "infusedev", false, "Infuse gold morphs into lattices for test corpus")
	cmd.Flag.BoolVar(&useConllU, "conllu", false, "use CoNLL-U-format input file (for disamb lattices)")
	cmd.Flag.IntVar(&limit, "limit", 0, "limit training set")
	cmd.Flag.BoolVar(&MdNoconverge, "noconverge", false, "don't test convergence (run -it number of iterations)")
	cmd.Flag.BoolVar(&Stream, "stream", false, "Stream data from input through parser to output")
	return cmd
}
