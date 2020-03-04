package app

import (
	// "github.com/OnlpLab/yap/nlp/format/lattice"

	// nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/nlp/parser/ma"
	// "github.com/OnlpLab/yap/util"

	// "fmt"
	"log"
	// "os"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var (
	latFile, rawFile, conlluFile, dataFile string
	useConllU                              bool // TODO: whatever i don't care anymore
	maxPOS, maxMSRPerPOS                   int
)

func MALearnConfigOut() {
	log.Println("Configuration")
	if useConllU {
		log.Printf("CoNLL-U:\t%s", conlluFile)
	} else {
		log.Printf("Lattice:\t%s", latFile)
		log.Printf("Raw:\t\t%s", rawFile)
	}
	log.Printf("Limit:\t%v", limit)
	log.Println()
	log.Printf("Output:\t%s", dataFile)
	log.Println()
}

func MALearn(cmd *commander.Command, args []string) error {
	var REQUIRED_FLAGS []string
	useConllU = len(conlluFile) > 0
	if useConllU {
		useConllU = true
		REQUIRED_FLAGS = []string{"conllu", "out"}
	} else {
		REQUIRED_FLAGS = []string{"lattice", "raw", "out"}
	}

	VerifyFlags(cmd, REQUIRED_FLAGS)

	MALearnConfigOut()
	log.Println("Starting learning for data-driven morphological analyzer")
	maData := new(ma.MADict)
	maData.Language = "Test"
	maData.MaxTopPOS = maxPOS
	maData.MaxMSRsPerPOS = maxMSRPerPOS
	var (
		numLearned int
		err        error
	)
	if useConllU {
		numLearned, err = maData.LearnFromConllU(conlluFile, limit)
	} else {
		numLearned, err = maData.LearnFromLat(latFile, rawFile, limit)
	}
	if err != nil {
		log.Println("Got error learning", err)
		return err
	}
	log.Println("Learned", numLearned, "new tokens")
	maData.WriteFile(dataFile)
	return nil
}

func MALearnCmd() *commander.Command {
	cmd := &commander.Command{
		Run:       MALearn,
		UsageLine: "malearn <file options> [arguments]",
		Short:     "generate a data-driven morphological analysis dictionary for a set of files",
		Long: `
generate a data-driven morphological analysis dictionary for a set of files

	$ ./yap malearn -lattice <lattice file> -raw <raw file> [options]

`,
		Flag: *flag.NewFlagSet("malearn", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&latFile, "lattice", "", "Lattice-format input file")
	cmd.Flag.StringVar(&rawFile, "raw", "", "raw sentences input file")
	cmd.Flag.StringVar(&conlluFile, "conllu", "", "CoNLL-U-format input file")
	cmd.Flag.StringVar(&dataFile, "out", "", "output file")
	cmd.Flag.IntVar(&maxMSRPerPOS, "maxmsrperpos", 5, "For OOV tokens, max MSRs per POS to add")
	cmd.Flag.IntVar(&maxPOS, "maxpos", 5, "For OOV tokens, max POS to add")
	cmd.Flag.IntVar(&limit, "limit", 0, "limit training set")
	return cmd
}
