package app

import (
	"fmt"
	"log"
	"github.com/OnlpLab/yap/nlp/format/raw"
	"github.com/OnlpLab/yap/nlp/parser/xliter8"
	"github.com/OnlpLab/yap/nlp/types"

	"github.com/gonuts/commander"
	"github.com/gonuts/flag"
)

var direction string

func Xliter8ConfigOut() {
	log.Println("Configuration")
	log.Printf("Direction:\t%s", direction)
	log.Printf("Limit:\t%v", limit)
	log.Println("Data")
	log.Printf("Input File:\t%s", input)
	if !VerifyExists(input) {
		return
	}
	log.Printf("Output File:\t%s", outMap)
}

func Xliter8(cmd *commander.Command, args []string) error {
	REQUIRED_FLAGS := []string{"i", "o"}

	VerifyFlags(cmd, REQUIRED_FLAGS)

	Xliter8ConfigOut()

	xliter8r := &xliter8.Hebrew{}
	var xf func(string) string
	switch direction {
	case "to":
		xf = xliter8r.To
	case "from":
		xf = xliter8r.From
	default:
		panic("Unknown direction use 'from' or 'to'")
	}

	data, err := raw.ReadFile(input, limit)
	if err != nil {
		panic(fmt.Sprintf("Failed reading raw file - %v", err))
	}
	log.Println("Read", len(data), "raw sentences from", input)
	results := make([]interface{}, len(data))
	log.Println("Processing", direction, "transliterated representation")
	for i, sent := range data {
		newSent := make(types.BasicSentence, len(sent))
		for j, token := range sent {
			newSent[j] = types.Token(xf(string(token)))
		}
		results[i] = newSent
	}
	raw.WriteFile(outMap, results)
	log.Println("Wrote", len(results), "sentences to", outMap)
	return nil
}

func Xliter8Cmd() *commander.Command {
	cmd := &commander.Command{
		Run:       Xliter8,
		UsageLine: "xliter8 <file options> [arguments]",
		Short:     "transliterates to<->from a hebrew file",
		Long: `
transliterates a hebrew file based on Sima'an et al.

	$ ./yap xliter8 -i <input file> -o <output file> [-d to|from]

`,
		Flag: *flag.NewFlagSet("vma", flag.ExitOnError),
	}
	cmd.Flag.StringVar(&input, "i", "", "Input File")
	cmd.Flag.StringVar(&outMap, "o", "", "Output File")
	cmd.Flag.StringVar(&direction, "d", "to", "Direction of transliteration [to:heb->xliter8ed, from:xliter8ed->heb]")
	cmd.Flag.IntVar(&limit, "limit", 0, "Limit # of rows read")
	return cmd
}
