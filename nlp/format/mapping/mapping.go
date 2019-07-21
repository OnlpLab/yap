package mapping

import (
	"github.com/OnlpLab/yap/nlp/parser/disambig"
	nlp "github.com/OnlpLab/yap/nlp/types"

	"fmt"
	"io"
	"os"
	// "log"
	"github.com/OnlpLab/yap/nlp/format/conllul"
)

func UDWriteMorph(writer io.Writer, morph *nlp.EMorpheme, curMorph int) {
	writer.Write([]byte(fmt.Sprintf("%d\t", curMorph)))
	//writer.Write([]byte(morph.Lemma))
	writer.Write([]byte(morph.Form))
	writer.Write([]byte{'\t'})
	writer.Write([]byte(morph.Form))
	writer.Write([]byte{'\t'})
	writer.Write([]byte(morph.CPOS))
	writer.Write([]byte{'\t'})
	//writer.Write([]byte(morph.POS))
	writer.Write([]byte(morph.CPOS))
	writer.Write([]byte{'\t'})
	if len(morph.FeatureStr) == 0 {
		writer.Write([]byte{'_'})
	} else {
		writer.Write([]byte(morph.FeatureStr))
	}
	for j := 0; j < 4; j++ {
		writer.Write([]byte("\t_"))
	}
	writer.Write([]byte{'\n'})
}

func WriteMorph(writer io.Writer, morph *nlp.EMorpheme, curMorph, curToken int) {
	writer.Write([]byte(fmt.Sprintf("%d\t%d\t", curMorph, curMorph+1)))
	writer.Write([]byte(morph.Form))
	writer.Write([]byte{'\t'})
	if len(morph.Lemma) > 0 {
		writer.Write([]byte(morph.Lemma))
	} else {
		writer.Write([]byte{'_'})
	}
	writer.Write([]byte{'\t'})
	writer.Write([]byte(morph.CPOS))
	writer.Write([]byte{'\t'})
	writer.Write([]byte(morph.POS))
	writer.Write([]byte{'\t'})
	if len(morph.FeatureStr) == 0 {
		writer.Write([]byte{'_'})
	} else {
		writer.Write([]byte(morph.FeatureStr))
	}
	writer.Write([]byte{'\t'})
	writer.Write([]byte(fmt.Sprintf("%d\n", curToken+1)))
}

func UDWrite(writer io.Writer, mappedSents []interface{}, conllul []conllul.ConlluLattice) {
	var curMorph int
	for i, mappedSent := range mappedSents {
		curMorph = 1
		lattice := conllul[i]
		for _, comment := range lattice.Comments {
			writer.Write([]byte(comment))
			writer.Write([]byte("\n"))
		}
		for _, mapping := range mappedSent.(*disambig.MDConfig).Mappings {
			if len(mapping.Spellout) > 1 {
				writer.Write([]byte(fmt.Sprintf("%d-%d\t%s", curMorph, curMorph+len(mapping.Spellout)-1, mapping.Token)))
				for j := 0; j < 8; j++ {
					writer.Write([]byte("\t_"))
				}
				writer.Write([]byte("\n"))
			}
			for _, morph := range mapping.Spellout {
				if morph == nil {
					// log.Println("\t", "Morph is nil, continuing")
					continue
				}
				UDWriteMorph(writer, morph, curMorph)
				curMorph++
			}
		}
		writer.Write([]byte{'\n'})
	}
}

func Write(writer io.Writer, mappedSents []interface{}) {
	var curMorph int
	for _, mappedSent := range mappedSents {
		curMorph = 0
		for i, mapping := range mappedSent.(*disambig.MDConfig).Mappings {
			// log.Println("At token", i, mapping.Token)
			if mapping.Token == nlp.ROOT_TOKEN {
				continue
			}
			// if mapping.Spellout != nil {
			// 	log.Println("\t", mapping.Spellout.AsString())
			// } else {
			// 	log.Println("\t", "*No spellout")
			// }
			for _, morph := range mapping.Spellout {
				if morph == nil {
					// log.Println("\t", "Morph is nil, continuing")
					continue
				}
				WriteMorph(writer, morph, curMorph, i)
				// log.Println("\t", "At morph", j, morph.Form)
				curMorph++
			}
		}
		writer.Write([]byte{'\n'})
	}
}

func WriteStream(writer *os.File, mappedSents chan interface{}) {
	var curMorph int
	var i int
	for mappedSent := range mappedSents {
		curMorph = 0
		for i, mapping := range mappedSent.(*disambig.MDConfig).Mappings {
			// log.Println("At token", i, mapping.Token)
			if mapping.Token == nlp.ROOT_TOKEN {
				continue
			}
			// if mapping.Spellout != nil {
			// 	log.Println("\t", mapping.Spellout.AsString())
			// } else {
			// 	log.Println("\t", "*No spellout")
			// }
			for _, morph := range mapping.Spellout {
				if morph == nil {
					// log.Println("\t", "Morph is nil, continuing")
					continue
				}
				WriteMorph(writer, morph, curMorph, i)
				// log.Println("\t", "At morph", j, morph.Form)
				curMorph++
			}
		}
		writer.Write([]byte{'\n'})
		i++
	}
	writer.Close()
}

func UDWriteFile(filename string, mappedSents []interface{}, conllul []conllul.ConlluLattice) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	UDWrite(file, mappedSents, conllul)
	return nil
}

func WriteFile(filename string, mappedSents []interface{}) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	Write(file, mappedSents)
	return nil
}

func WriteStreamToFile(filename string, mappedSents chan interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	WriteStream(file, mappedSents)
	return nil
}
