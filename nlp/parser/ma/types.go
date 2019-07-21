package ma

import . "github.com/OnlpLab/yap/nlp/types"

type MorphologicalAnalyzer interface {
	Analyze(input []string) (LatticeSentence, interface{})
}
