package app

import (
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/search"
	"github.com/OnlpLab/yap/alg/transition"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/parser/disambig"

	nlp "github.com/OnlpLab/yap/nlp/types"

	"fmt"
	"log"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var (
	transitionSystem transition.TransitionSystem
	vmaParamFuncName string
)

func ValidMAGoldConfigOut(t transition.TransitionSystem) {
	log.Println("Configuration")
	log.Printf("Transition System:\t%s", t.Name())
	log.Printf("Infuse Gold:\t\t%v", MdCombineGold)
	log.Printf("Parameter Func:\t%v", vmaParamFuncName)

	log.Println()
	log.Println("Data")
	log.Printf("Disamb. lattice file:\t%s", tLatDis)
	if !VerifyExists(tLatDis) {
		return
	}
	log.Printf("Ambig.  lattice file:\t%s", tLatAmb)
	if !VerifyExists(tLatAmb) {
		return
	}
}

func ValidateInstance(decoded perceptron.DecodedInstance) string {
	conf := &disambig.MDConfig{
		ETokens: ETokens,
		POP:     POP,
	}

	extractor := &perceptron.EmptyFeatureExtractor{}

	deterministic := &search.Deterministic{
		TransFunc:          transitionSystem,
		FeatExtractor:      extractor,
		ReturnModelValue:   false,
		ReturnSequence:     false,
		ShowConsiderations: false,
		Base:               conf,
		NoRecover:          false,
	}

	deterministic.DecodeGold(decoded, nil)
	return "ok"
}

func ValidateCorpus(goldSequences []perceptron.DecodedInstance) map[string]int {
	retval := make(map[string]int, 4)
	prefix := log.Prefix()
	for i, goldSeq := range goldSequences {
		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		log.Println("At sent", i)
		result := ValidateInstance(goldSeq)
		curval, exists := retval[result]
		if !exists {
			curval = 0
		}
		retval[result] = curval + 1
	}
	log.SetPrefix(prefix)
	return retval
}

func genInstance(goldLat, ambLat nlp.LatticeSentence) *disambig.MDConfig {
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
		mappings[i] = mapping
	}

	m := &disambig.MDConfig{
		Mappings: mappings,
		Lattices: ambLat,
	}
	return m
}

func genInstances(goldLats, ambLats []interface{}) []interface{} {
	configs := make([]interface{}, 0, len(goldLats))
	for i, goldMap := range goldLats {
		ambLat := ambLats[i].(nlp.LatticeSentence)
		result := genInstance(goldMap.(nlp.LatticeSentence), ambLat)
		configs = append(configs, result)
	}
	return configs
}

func ValidMAGold(cmd *commander.Command, args []string) error {
	paramFunc, exists := nlp.MDParams[vmaParamFuncName]
	if !exists {
		log.Fatalln("Param Func", vmaParamFuncName, "does not exist")
	}
	var mdTrans transition.TransitionSystem
	mdTrans = &disambig.MDTrans{
		ParamFunc: paramFunc,
	}

	// arcSystem := &morph.Idle{morphArcSystem, IDLE}
	transitionSystem = transition.TransitionSystem(mdTrans)

	REQUIRED_FLAGS := []string{"d", "l"}

	VerifyFlags(cmd, REQUIRED_FLAGS)
	// RegisterTypes()

	ValidMAGoldConfigOut(transitionSystem)

	if allOut {
		log.Println()
		// start processing - setup enumerations
		log.Println("Setup enumerations")
	}
	SetupMDEnum()
	mdTrans.(*disambig.MDTrans).POP = POP
	mdTrans.(*disambig.MDTrans).Transitions = ETrans
	mdTrans.AddDefaultOracle()

	if allOut {
		log.Println("Amb. Lat:\tReading ambiguous lattices from", tLatAmb)
	}
	lAmb, lAmbE := lattice.ReadFile(tLatAmb, 0)
	if lAmbE != nil {
		log.Println(lAmbE)
		return lAmbE
	}
	if allOut {
		log.Println("Amb. Lat:\tRead", len(lAmb), "ambiguous lattices")
		log.Println("Amb. Lat:\tConverting lattice format to internal structure")
	}
	goldAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
	// goldAmbLat = Limit(goldAmbLat, 1)

	if allOut {
		log.Println("Dis. Lat.:\tReading disambiguated lattices from", tLatDis)
	}
	lDis, lDisE := lattice.ReadFile(tLatDis, 0)
	if lDisE != nil {
		log.Println(lDisE)
		return lDisE
	}
	if allOut {
		log.Println("Dis. Lat.:\tRead", len(lDis), "disambiguated lattices")
		log.Println("Dis. Lat.:\tConverting lattice format to internal structure")
	}
	goldDisLat := lattice.Lattice2SentenceCorpus(lDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
	// goldDisLat = Limit(goldDisLat, 1)

	if allOut {
		log.Println("Combining train files into gold morph graphs with original lattices")
	}
	var combined []interface{}
	if MdCombineGold {
		combined, _, _, _ = CombineLatticesCorpus(goldDisLat, goldAmbLat)
	} else {
		combined = genInstances(goldDisLat, goldAmbLat)
	}
	goldSequences := TrainingSequences(combined, GetMDConfigAsLattices, GetMDConfigAsMappings)
	if allOut {
		log.Println("Validating corpus of", len(goldSequences), "lattices")
	}
	stats := ValidateCorpus(goldSequences)
	log.Println("Results", stats)
	return nil
}

func ValidateMAGoldCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       ValidMAGold,
		UsageLine: "vma <file options> [arguments]",
		Short:     "validates gold paths in given lattices",
		Long: `
validates gold paths in given lattices

	$ ./yap vma -d <disamb. lat> -l <amb. lat> [-p <param func>] [options]

`,
		Flag: *flag.NewFlagSet("vma", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&tLatDis, "d", "", "Disambiguated Lattices File")
	cmd.Flag.StringVar(&tLatAmb, "l", "", "Ambiguous Lattices File")
	cmd.Flag.StringVar(&outMap, "om", "", "Output Mapping File")
	cmd.Flag.BoolVar(&MdCombineGold, "infuse", false, "Infuse gold morphs into lattices")
	cmd.Flag.StringVar(&vmaParamFuncName, "p", "Funcs_Main_POS_Both_Prop_WLemma", "Param Func types: ["+nlp.AllParamFuncNames+"]")
	return cmd
}
