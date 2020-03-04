package disambig

import (
	. "github.com/OnlpLab/yap/nlp/types"
)

type MorphologicalDisambiguator interface {
	Parse(LatticeSentence) (Mappings, interface{})
}
