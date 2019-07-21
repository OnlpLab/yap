package app

import (
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/format/raw"
	// "github.com/OnlpLab/yap/nlp/format/mapping"
	"github.com/OnlpLab/yap/nlp/parser/disambig"

	nlp "github.com/OnlpLab/yap/nlp/types"

	"fmt"
	"log"
	"os"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func GenUnAmbLemmasConfigOut() {
	log.Println("Configuration")
	log.Printf("Parameter Func:\t%v", MdParamFuncName)

	log.Println()
	log.Println("Data")
	log.Printf("Raw Input:\t\t\t%s", inRawFile)
	if !VerifyExists(tLatDis) {
		return
	}
	log.Printf("Disamb. lattice file:\t%s", tLatDis)
	if !VerifyExists(tLatDis) {
		return
	}
	log.Printf("Ambig.  lattice file:\t%s", tLatAmb)
	if !VerifyExists(tLatAmb) {
		return
	}
}

func GetUnAmbLemmas(conf *disambig.MDConfig, pf nlp.MDParam) nlp.DisAmbMorphs {
	return conf.Lattices.FindGoldDisAmbMorphs(conf.Mappings, pf)
}

func GetUnAmbLemmasCorpus(goldSequences []*disambig.MDConfig, rawSents []nlp.BasicSentence, pf nlp.MDParam) {
	f, _ := os.Create(outMap)
	defer f.Close()
	prefix := log.Prefix()
	for i, goldSeq := range goldSequences {
		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		result := GetUnAmbLemmas(goldSeq, pf)
		for _, ambLemma := range result {
			fmt.Fprintf(f, "%v,%v,%v,%v,%v\n", i, ambLemma.Token, ambLemma.GoldMorph, ambLemma.Lemma, ambLemma.CPOS)
		}
	}
	log.SetPrefix(prefix)
}

func GenUnAmbLemmas(cmd *commander.Command, args []string) error {
	paramFunc, exists := nlp.MDParams[MdParamFuncName]
	if !exists {
		log.Fatalln("Param Func", MdParamFuncName, "does not exist")
	}

	REQUIRED_FLAGS := []string{"d", "l"}

	VerifyFlags(cmd, REQUIRED_FLAGS)
	// RegisterTypes()

	GenLemmasConfigOut()

	if allOut {
		log.Println()
		// start processing - setup enumerations
		log.Println("Setup enumerations")
	}
	SetupMDEnum()

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
	goldAmbLat := lattice.Lattice2SentenceCorpus(lAmb, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
	// goldAmbLat = Limit(goldAmbLat, 1)

	if allOut {
		log.Println("Dis. Lat.:\tReading disambiguated lattices from", tLatDis)
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
	goldDisLat := lattice.Lattice2SentenceCorpus(lDis, EWord, EPOS, EWPOS, EMorphProp, EMHost, EMSuffix)
	// goldDisLat = Limit(goldDisLat, 1)

	if allOut {
		log.Println("Combining train files into gold morph graphs with original lattices")
	}
	combined := genLemmasInstances(goldDisLat, goldAmbLat)
	rawSents, err := raw.ReadFile(inRawFile, limit)
	if err != nil {
		panic(fmt.Sprintf("Failed reading raw file - %v", err))
	}
	if allOut {
		log.Println("Read", len(rawSents), "raw sentences")
		log.Println("Getting lemmas for", len(combined), "sentences")
	}
	GetUnAmbLemmasCorpus(combined, rawSents, paramFunc)
	return nil
}

func GenUnAmbLemmasCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       GenUnAmbLemmas,
		UsageLine: "unamblemmas <file options> [arguments]",
		Short:     "gets unambiguous lemmas in the hebrew tb for gold paths",
		Long: `
gets unambiguous lemmas in the hebrew tb for gold paths

	$ ./yap unamblemmas -d <disamb. lat> -l <amb. lat> [-p <param func>] [options]

`,
		Flag: *flag.NewFlagSet("unamblemmas", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&tLatDis, "d", "", "Disambiguated Lattices File")
	cmd.Flag.StringVar(&tLatAmb, "l", "", "Ambiguous Lattices File")
	cmd.Flag.StringVar(&inRawFile, "r", "", "Input raw (tokenized) file")
	cmd.Flag.StringVar(&outMap, "om", "", "Output Mapping File")
	cmd.Flag.StringVar(&MdParamFuncName, "p", "Funcs_Main_POS_Both_Prop", "Param Func types: ["+nlp.AllParamFuncNames+"]")
	cmd.Flag.IntVar(&limit, "limit", 0, "limit training set")
	return cmd
}
