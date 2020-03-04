package app

import (
	"github.com/OnlpLab/yap/nlp/format/lattice"

	nlp "github.com/OnlpLab/yap/nlp/types"

	"fmt"
	"log"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func GoldSegConfigOut() {
	log.Println("Configuration")
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

func GenSegSequence(iAmbLat, iGoldLat interface{}) nlp.LatticeSentence {
	ambLat, _ := iAmbLat.(nlp.LatticeSentence)
	goldLat, _ := iGoldLat.(nlp.LatticeSentence)
	retval := make(nlp.LatticeSentence, len(ambLat))
	lastTop := 0
	for i, aLat := range ambLat {
		gLat := goldLat[i]
		aLat.GenSpellouts()
		gLat.GenSpellouts()
		sharedSpellouts := aLat.Spellouts.Intersect(gLat.Spellouts, "Form", lastTop)
		// log.Println("Got shared", sharedSpellouts)
		newLat := &nlp.Lattice{
			aLat.Token,
			sharedSpellouts.UniqueMorphemes(),
			nil,
			nil,
			sharedSpellouts[0][0].From(),
			sharedSpellouts[0][len(sharedSpellouts[0])-1].To(),
		}

		newLat.GenNexts(false)
		// newLat.Compact(lastTop)
		// lastTop = newLat.Top()
		retval[i] = *newLat
		// log.Println("Got lat", newLat)
		lastTop = newLat.Top()
	}
	return retval
}

func GenSegSequences(ambLats, goldLats []interface{}) []nlp.LatticeSentence {
	retval := make([]nlp.LatticeSentence, len(ambLats))
	prefix := log.Prefix()
	for i, ambLat := range ambLats {
		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		log.Println("At sent", i)
		retval[i] = GenSegSequence(ambLat, goldLats[i])
	}
	log.SetPrefix(prefix)
	return retval
}

func GoldSeg(cmd *commander.Command, args []string) error {
	REQUIRED_FLAGS := []string{"d", "l"}

	VerifyFlags(cmd, REQUIRED_FLAGS)
	// RegisterTypes()

	GoldSegConfigOut()

	if allOut {
		log.Println()
		// start processing - setup enumerations
		log.Println("Setup enumerations")
	}
	SetupMDEnum()

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
	combined, _, _, _ = CombineLatticesCorpus(goldDisLat, goldAmbLat)
	if allOut {
		log.Println("Generate segmentation-gold corpus of", len(combined), "lattices")
	}
	segSequences := GenSegSequences(goldAmbLat, goldDisLat)
	if allOut {
		log.Println("Writing", len(segSequences), "to", outLatticeFile)
	}
	output := lattice.Sentence2LatticeCorpus(segSequences, nil)
	if allOut {
		log.Println("Got", len(output), "lattices")
	}
	lattice.WriteFile(outLatticeFile, output)
	log.Println("Done")
	return nil
}

func GoldSegCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       GoldSeg,
		UsageLine: "gseg <file options> [arguments]",
		Short:     "gets ma without segmentation ambiguity",
		Long: `
gets ma without segmentation ambiguity

	$ ./yap gseg -d <disamb. lat> -l <amb. lat> -o <out file> [-p <param func>] [options]

`,
		Flag: *flag.NewFlagSet("gseg", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&tLatDis, "d", "", "Disambiguated Lattices File")
	cmd.Flag.StringVar(&tLatAmb, "l", "", "Ambiguous Lattices File")
	cmd.Flag.StringVar(&outLatticeFile, "o", "", "Output Lattice File")
	cmd.Flag.BoolVar(&MdCombineGold, "infuse", false, "Infuse gold morphs into lattices")
	cmd.Flag.StringVar(&vmaParamFuncName, "p", "Funcs_Main_POS_Both_Prop_WLemma", "Param Func types: ["+nlp.AllParamFuncNames+"]")
	return cmd
}
