package app

import (
	"os"
	"github.com/OnlpLab/yap/nlp/format/conllu"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/format/raw"

	"github.com/OnlpLab/yap/nlp/parser/ma"
	nlp "github.com/OnlpLab/yap/nlp/types"
	// "github.com/OnlpLab/yap/util"

	"fmt"
	"log"
	// "os"
	"strings"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var (
	dictFile, inRawFile, outLatticeFile string
	maxOOVMSRPerPOS                     int = 10
	oovFile                             string
	dopeOOV                             bool
	outFormat                           string
	udLex                               string
)

func MAConfigOut() {
	log.Println("Configuration")
	log.Printf("MA Dict:\t\t%s", dictFile)
	log.Printf("MA UD Lexicon:\t%s", udLex)
	log.Printf("Limit:\t\t%v", limit)
	log.Printf("Max OOV Msrs/POS:\t%v", maxOOVMSRPerPOS)
	log.Printf("Dope:\t\t%v", dopeOOV)
	log.Println()
	if useConllU {
		log.Printf("CoNLL-U Input:\t%s", conlluFile)
	} else {
		log.Printf("Raw Input:\t\t%s", inRawFile)
	}
	log.Printf("Output:\t\t%s", outLatticeFile)
	log.Printf("Output Format:\t%v", outFormat)
	log.Println()
}

func MA(cmd *commander.Command, args []string) error {
	useConllU = len(conlluFile) > 0
	var REQUIRED_FLAGS []string
	if useConllU {
		REQUIRED_FLAGS = []string{"dict", "conllu", "out"}
	} else {
		REQUIRED_FLAGS = []string{"dict", "raw", "out"}
	}

	VerifyFlags(cmd, REQUIRED_FLAGS)

	MAConfigOut()

	log.Println("Reading Morphological Analyzer Dictionary")
	maData := new(ma.MADict)
	if err := maData.ReadFile(dictFile); err != nil {
		panic(fmt.Sprintf("Failed reading MA dict file - %v", err))
	}
	log.Println("OOV POSs:", strings.Join(maData.TopPOS, ", "))
	maData.ComputeOOVMSRs(maxOOVMSRPerPOS)
	log.Println()
	if udLex != "" {
		// Reading a UD lexicon will override the data-driven lexicon
		// but the OOV MSRs will remain
		log.Println("Reading UD Lex file", udLex)
		if err := maData.ReadUDLexFile(udLex); err != nil {
			panic(fmt.Sprintf("Failed reading UD lex file - %v", err))
		}
	}
	log.Println()
	var (
		sents        []nlp.BasicSentence
		sentComments [][]string
		oovVectors   []interface{}
		rawOOV       interface{}
		err          error
	)
	if useConllU {
		conllSents, _, err := conllu.ReadFile(conlluFile, limit)
		if err != nil {
			panic(fmt.Sprintf("Failed reading CoNLL-U file - %v", err))
		}
		sents = make([]nlp.BasicSentence, len(conllSents))
		sentComments = make([][]string, len(conllSents))
		for i, sent := range conllSents {
			newSent := make([]nlp.Token, len(sent.Tokens))
			for j, token := range sent.Tokens {
				newSent[j] = nlp.Token(token)
			}
			sentComments[i] = sent.Comments
			sents[i] = newSent
		}
	} else {
		sents, err = raw.ReadFile(inRawFile, limit)
		sentComments = make([][]string, len(sents))
		for i, sent := range sents {
			sentComments[i] = []string{fmt.Sprintf("# text %s", strings.Join(sent.Tokens(), " ")) }
		}
		if err != nil {
			panic(fmt.Sprintf("Failed reading raw file - %v", err))
		}
	}
	log.Println("Running Morphological Analysis")
	lattices := make([]nlp.LatticeSentence, len(sents))
	stats := new(ma.AnalyzeStats)
	stats.Init()
	maData.Init()
	maData.Stats = stats
	maData.Dope = dopeOOV
	if len(oovFile) > 0 {
		oovVectors = make([]interface{}, len(sents))
	}
	var (
		outFile         *os.File
		streamOut       bool
		latticesWritten int
		outFileError    error
	)

	if outFormat == "ud" && !outJSON {
		log.Println("Using streaming analysis and output")
		// horrible hack for now :(
		streamOut = true

		lattices = make([]nlp.LatticeSentence, 1)
		outFile, outFileError = os.Create(outLatticeFile)
		if outFileError != nil {
			return outFileError
		}
		defer outFile.Close()
	}
	for i, sent := range sents {
		if streamOut {
			lattices[0], rawOOV = maData.Analyze(sent.Tokens())
			output := lattice.Sentence2LatticeCorpus(lattices, nil)
			lattice.UDWrite(outFile, output, sentComments[i:i+1], []nlp.BasicSentence{rawOOV.(nlp.BasicSentence)})
			latticesWritten += 1
		} else {
			lattices[i], rawOOV = maData.Analyze(sent.Tokens())
			if oovVectors != nil {
				oovVectors[i] = rawOOV
			}
		}
	}
	log.Println("Analyzed", stats.TotalTokens, "occurences of", len(stats.UniqTokens), "unique tokens")
	log.Println("Encountered", stats.OOVTokens, "occurences of", len(stats.UniqOOVTokens), "unknown tokens")
	if !streamOut {
		output := lattice.Sentence2LatticeCorpus(lattices, nil)
		if outFormat == "ud" {
			if !outJSON {
				// lattice.WriteUDJSONFile(outLatticeFile, output)
				// } else {
				lattice.WriteUDFile(outLatticeFile, output, sentComments, nil)
			}
		} else if outFormat == "spmrl" {
			lattice.WriteFile(outLatticeFile, output)
		} else {
			panic(fmt.Sprintf("Unknown lattice output format - %v", outFormat))
		}
		if oovVectors != nil {
			raw.WriteFile(oovFile, oovVectors)
		}
		log.Println("Wrote", len(output), "lattices")
	} else {

		log.Println("Wrote", latticesWritten, "lattices")
	}
	return nil
}

func MACmd() *commander.Command {
	cmd := &commander.Command{
		Run:       MA,
		UsageLine: "ma <file options> [arguments]",
		Short:     "run data-driven morphological analyzer on raw input",
		Long: `
run data-driven morphological analyzer on raw input

	$ ./yap ma -dict <dict file> [-udlex <udlex file>] -raw <raw file> [-format <sprml|ud>] -out <output file> [options]

`,
		Flag: *flag.NewFlagSet("ma", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&dictFile, "dict", "", "Dictionary for morphological analyzer")
	cmd.Flag.StringVar(&udLex, "udlex", "", "UD Lexicon for morphological analyzer")
	cmd.Flag.StringVar(&inRawFile, "raw", "", "Input raw (tokenized) file")
	cmd.Flag.StringVar(&conlluFile, "conllu", "", "CoNLL-U-format input file")
	cmd.Flag.StringVar(&outLatticeFile, "out", "", "Output lattice file")
	cmd.Flag.StringVar(&outFormat, "format", "spmrl", "Output lattice format [spmrl|ud]")
	cmd.Flag.StringVar(&oovFile, "oov", "", "OOV File")
	cmd.Flag.IntVar(&maxOOVMSRPerPOS, "maxmsrperpos", 10, "For OOV tokens, max MSRs per POS to add")
	cmd.Flag.BoolVar(&dopeOOV, "dope", false, "Dope potential OOV tokens")
	cmd.Flag.IntVar(&limit, "limit", 0, "limit training set")
	cmd.Flag.BoolVar(&outJSON, "json", false, "Output using JSON")
	return cmd
}
