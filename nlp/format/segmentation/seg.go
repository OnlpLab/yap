package segmentation

import (
	nlp "github.com/OnlpLab/yap/nlp/types"
	"io"
	"os"
	"strings"
)

func Write(writer io.Writer, graphs []interface{}) {
	for _, graph := range graphs {
		for _, mapping := range graph.(nlp.MorphDependencyGraph).GetMappings() {
			if mapping.Token == nlp.ROOT_TOKEN {
				continue
			}
			writer.Write([]byte(mapping.Token))
			writer.Write([]byte{'\t'})
			morphForms := make([]string, len(mapping.Spellout))
			for i, morph := range mapping.Spellout {
				morphForms[i] = morph.Form
			}
			writer.Write([]byte(strings.Join(morphForms, ":")))
			writer.Write([]byte{'\n'})
		}
		writer.Write([]byte{'\n'})
	}
}

func WriteFile(filename string, graphs []interface{}) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	Write(file, graphs)
	return nil
}
