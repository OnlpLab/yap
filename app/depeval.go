package app

import (
	"github.com/OnlpLab/yap/eval"
	"github.com/OnlpLab/yap/nlp/format/conll"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	dep "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	"github.com/OnlpLab/yap/util"
	"github.com/OnlpLab/yap/util/conf"

	"log"
	"os"
	// "strings"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

func DepEvalConfigOut() {
	log.Println("Configuration")
	log.Printf("Labels File:\t\t%s", DepLabelsFile)
	if !VerifyExists(DepLabelsFile) {
		os.Exit(1)
	}
	log.Println()
	log.Println("Data")
	log.Printf("Parsed result file:\t%s", input)
	if !VerifyExists(input) {
		os.Exit(1)
	}
	log.Printf("Gold file:\t%s", input)
	if !VerifyExists(inputGold) {
		os.Exit(1)
	}
}

func SetupEvalEnum(relations []string) {
	SetupRelationEnum(relations)
	EWord, EPOS, EWPOS = util.NewEnumSet(APPROX_WORDS), util.NewEnumSet(APPROX_POS), util.NewEnumSet(APPROX_WORDS*WORDS_POS_FACTOR)
	EMHost, EMSuffix = util.NewEnumSet(APPROX_MHOSTS), util.NewEnumSet(APPROX_MSUFFIXES)
	EMorphProp = util.NewEnumSet(130) // random guess of number of possible values
	// adding empty string as an element in the morph enum sets so that '0' default values
	// map to empty morphs
	EMHost.Add("")
	EMSuffix.Add("")
}

// Assumes sorted inputs of equal length
func DepEvalConll(test, gold interface{}) *eval.Result {
	// testConf, testOk := test.(*dep.SimpleConfiguration)
	testGraph, _ := test.(*dep.BasicDepGraph)
	goldGraph, _ := gold.(*dep.BasicDepGraph)
	// log.Println(testMorph.GetSequence())
	// log.Println(goldMorph.GetSequence())
	// if !testOk {
	// 	panic("Test argument should be MDConfig")
	// }
	// if !goldOk {
	// 	panic("Gold argument should be nlp.Mappings")
	// }
	// testArcs := testConf.Arcs().(*dep.ArcSetSimple).Arcs
	testArcs := testGraph.Arcs
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
func DepEvalTrainAndParse(cmd *commander.Command, args []string) error {
	REQUIRED_FLAGS := []string{"p", "g"}

	VerifyFlags(cmd, REQUIRED_FLAGS)
	if allOut {
		DepEvalConfigOut()
	}
	relations, err := conf.ReadFile(DepLabelsFile)
	if err != nil {
		log.Println("Failed reading dependency labels configuration file:", DepLabelsFile)
		log.Fatalln(err)
	}
	SetupEvalEnum(relations.Values)

	devi, e2 := conll.ReadFile(input, 0)
	if e2 != nil {
		log.Fatalln(e2)
	}
	// const NUM_SENTS = 20

	// s = s[:NUM_SENTS]
	if allOut {
		log.Println("Read", len(devi), "sentences from", input)
		log.Println("Converting from conll to internal format")
	}
	predGraphs := conll.Conll2GraphCorpus(devi, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)

	devigold, e3 := conll.ReadFile(inputGold, 0)
	if e3 != nil {
		log.Fatalln(e3)
	}
	if allOut {
		log.Println("Read", len(devigold), "sentences from", inputGold)
		log.Println("Converting from conll to internal format")
	}
	goldGraphs := conll.Conll2GraphCorpus(devigold, EWord, EPOS, EWPOS, ERel, EMHost, EMSuffix)

	if len(goldGraphs) != len(predGraphs) {
		panic("Evaluation set sizes are different")
	}
	var total = &eval.Total{
		Results: make([]*eval.Result, 0, len(predGraphs)),
	}
	var utotal = &eval.Total{
		Results: make([]*eval.Result, 0, len(predGraphs)),
	}
	for i, instance := range predGraphs {
		goldInstance := goldGraphs[i]
		if goldInstance != nil {
			result := DepEvalConll(instance, goldInstance)
			// log.Println("Correct: ", result.TP)
			total.Add(result)
			utotal.Add(result.Other.(*eval.Result))
		}
	}
	log.Println("Result (UAS, LAS, UEM #, UEM %): ", utotal.Precision(), total.Precision(), utotal.Exact, float64(utotal.Exact)/float64(total.Population), "TruePos:", total.TP, "in", total.Population)
	if allOut {
		log.Println()
	}
	return nil
}

func DepEvalCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       DepEvalTrainAndParse,
		UsageLine: "depeval <file options> [arguments]",
		Short:     "runs dependency eval",
		Long: `
runs dependency eval

	$ ./yap depeval -p <conll> -g <conll> [options]

`,
		Flag: *flag.NewFlagSet("depeval", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&DepLabelsFile, "l", "", "Dependency Labels Configuration File")
	cmd.Flag.StringVar(&input, "p", "", "Parse Result Conll File")
	cmd.Flag.StringVar(&inputGold, "g", "", "Gold Conll File")
	cmd.Flag.BoolVar(&lattice.IGNORE_LEMMA, "nolemma", false, "Ignore lemmas")
	return cmd
}
