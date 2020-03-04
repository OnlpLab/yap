package ma

import (
	"github.com/OnlpLab/yap/alg/graph"
	"github.com/OnlpLab/yap/nlp/format/lex"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	"fmt"
	"log"
	"regexp"
	"strings"
)

const ESTIMATED_MORPHS_PER_TOKEN = 5

type BGULex struct {
	MaxPrefixLen int
	Prefixes     map[string][]BasicMorphemes

	Lex map[string][]BasicMorphemes

	Files []string
	Stats *AnalyzeStats

	AlwaysNNP bool
	LogOOV    bool
	MAType    string
}

var (
	PUNCT = map[string]string{
		":":   "yyCLN",
		",":   "yyCM",
		"-":   "yyDASH",
		".":   "yyDOT",
		"...": "yyELPS",
		"!":   "yyEXCL",
		"(":   "yyLRB",
		"?":   "yyQM",
		")":   "yyRRB",
		";":   "yySCLN",
		"\"":  "yyQUOT",
	}
	OOVMSRS = []string{
		"NNP-",
		"NNP-gen=F|gen=M|num=S",
		"NNP-gen=M|num=S",
		"NNP-gen=F|num=S",
		"NN-gen=M|num=P|num=S",
		"NN-gen=M|num=S",
		"NN-gen=F|num=S",
		"NN-gen=M|num=P",
		"NN-gen=F|num=P",
	}
	REGEX = []struct {
		RE  *regexp.Regexp
		POS string
	}{
		{regexp.MustCompile("^\\d+(\\.\\d+)?$|^\\d{1,3}(,\\d{3})*(\\.\\d+)?$"), "CD"},
		{regexp.MustCompile("\\d"), "NCD"},
	}
	_ MorphologicalAnalyzer = &BGULex{}
)

func (l *BGULex) loadTokens(file, format string) {
	tokens, err := lex.ReadFile(file, format, l.MAType)
	if err != nil {
		panic(fmt.Sprintf("Failed to load %v: %v", file, err))
	}
	var m map[string][]BasicMorphemes
	if format == "prefix" {
		l.Prefixes = make(map[string][]BasicMorphemes, len(tokens))
		m = l.Prefixes
	} else if format == "lexicon" {
		l.Lex = make(map[string][]BasicMorphemes, len(tokens))
		m = l.Lex
	}
	log.Println("Found", len(tokens), "tokens in lexicon file:", file)
	for _, token := range tokens {
		if cur, exists := m[token.Token]; exists {
			m[token.Token] = append(cur, token.Morphemes...)
		} else {
			m[token.Token] = token.Morphemes
		}
	}
}

func (l *BGULex) LoadPrefixes(file string) {
	l.loadTokens(file, "prefix")
	l.MaxPrefixLen = 0
	for _, morphs := range l.Prefixes {
		if l.MaxPrefixLen < len(morphs) {
			l.MaxPrefixLen = len(morphs)
		}
	}
	log.Println("Loaded", len(l.Prefixes), "prefixes from lexicon")
}

func (l *BGULex) LoadLex(file string, nnpnofeats bool) {
	lex.ADD_NNP_NO_FEATS = nnpnofeats
	l.loadTokens(file, "lexicon")
	log.Println("Loaded", len(l.Lex), "tokens from lexicon")
}

func makeMorphWithPOS(input, lemma, POS string) []BasicMorphemes {
	return []BasicMorphemes{BasicMorphemes([]*Morpheme{
		&Morpheme{
			BasicDirectedEdge: graph.BasicDirectedEdge{0, 0, 1},
			Form:              input,
			Lemma:             lemma,
			CPOS:              POS,
			POS:               POS,
			FeatureStr:        "",
		},
	})}
}

func (l *BGULex) AddOOVAnalysis(lat *Lattice, prefix BasicMorphemes, hostStr string, numToken int) {
	var OOVPOS, featuresStr string
	for _, msr := range OOVMSRS {
		// if logAnalyze {
		// 	log.Println("Adding msr", msr)
		// }
		msrsplit := strings.Split(msr, "-")
		OOVPOS, featuresStr = msrsplit[0], msrsplit[1]
		if l.MAType == "ud" {
			OOVPOS = util.HEB2UDPOS[OOVPOS]
			if len(featuresStr) > 0 {
				featuresStr = util.Heb2UDFeaturesString(featuresStr)
			}
		}
		newMorph := []BasicMorphemes{BasicMorphemes([]*Morpheme{
			&Morpheme{
				BasicDirectedEdge: graph.BasicDirectedEdge{0, 0, 1},
				Form:              hostStr,
				Lemma:             hostStr,
				CPOS:              OOVPOS,
				POS:               OOVPOS,
				FeatureStr:        featuresStr,
			},
		})}
		lat.AddAnalysis(prefix, newMorph, numToken)
	}
}

func checkRegexes(input string) ([]BasicMorphemes, bool) {
	for _, curRegex := range REGEX {
		if curRegex.RE.MatchString(input) {
			return makeMorphWithPOS(input, "", curRegex.POS), true
		}
	}
	return nil, false
}

var logAnalyze bool = false

func (l *BGULex) OOVForLen(lat *Lattice, input string, startingNode, numToken, prefixLen int) bool {
	var (
		found   bool
		hostStr string
	)
	if len(input) < prefixLen*2 {
		return found
	}
	prefixLat, prefixExists := l.Prefixes[input[0:prefixLen*2]]
	// log.Println("\tPrefixes", input[0:prefixLen*2], prefixExists)
	if prefixExists {
		hostStr = input[2*prefixLen:]
		if len(hostStr) > 2 {
			// Always add NNP hosts for len(hosts)>1 (unicode = 2 runes)
			for _, prefix := range prefixLat {
				l.AddOOVAnalysis(lat, prefix, hostStr, numToken)
				// lat.AddAnalysis(prefix, l.OOVAnalysis(hostStr), numToken)
			}
		}
	}
	return found
}

func (l *BGULex) analyzeTokenForLen(lat *Lattice, input string, startingNode, numToken, prefixLen int) bool {
	var (
		found, hostExists bool
		hostLat           []BasicMorphemes
		hostStr           string
	)
	if len(input) < prefixLen*2 {
		return found
	}
	prefixLat, prefixExists := l.Prefixes[input[0:prefixLen*2]]
	// log.Println("\tPrefixes", input[0:prefixLen*2], prefixExists)
	if prefixExists {
		hostStr = input[2*prefixLen:]
		if l.AlwaysNNP {
			if len(hostStr) > 2 {
				// Always add NNP hosts for len(hosts)>1 (unicode = 2 runes)
				for _, prefix := range prefixLat {
					l.AddOOVAnalysis(lat, prefix, hostStr, numToken)
					// lat.AddAnalysis(prefix, l.OOVAnalysis(hostStr), numToken)
				}
			}
		}
		hostLat, hostExists = l.Lex[hostStr]
		if !hostExists {
			hostLat, hostExists = checkRegexes(hostStr)
		}
		// log.Println("\tHosts", input[2*prefixLen:], hostExists)
		if hostExists {
			for _, prefix := range prefixLat {
				// log.Println("\t\tAdding", prefix, hostLat)
				lat.AddAnalysis(prefix, hostLat, numToken)
			}
			found = true
		}
	}
	return found
}

func (l *BGULex) AnalyzeToken(input string, startingNode, indexToken int) (*Lattice, interface{}) {
	numToken := indexToken + 1
	if logAnalyze {
		log.Println("Analyzing token", numToken, "starting at", startingNode)
	}
	lat := &Lattice{
		Token:     Token(input),
		Morphemes: make(Morphemes, 0, ESTIMATED_MORPHS_PER_TOKEN),
		Next:      make(map[int][]int, ESTIMATED_MORPHS_PER_TOKEN),
		BottomId:  startingNode,
		TopId:     startingNode,
	}
	lat.Next[0] = make([]int, 0, 1)
	var (
		hostLat               []BasicMorphemes
		hostExists, anyExists bool
		punctPOS              string
	)
	if punctVal, exists := PUNCT[input]; exists {
		punctPOS = punctVal
		if l.MAType == "ud" {
			punctPOS = "PUNCT"
		}
		m := &Morpheme{
			BasicDirectedEdge: graph.BasicDirectedEdge{0, 0, 0},
			Form:              input,
			CPOS:              punctPOS,
			POS:               punctPOS,
		}
		basics := []BasicMorphemes{BasicMorphemes{m}}
		lat.AddAnalysis(nil, basics, numToken)
		return lat, false
	}
	if l.AlwaysNNP {
		l.AddOOVAnalysis(lat, nil, input, numToken)
		// oovLat := l.OOVAnalysis(input)
		// lat.AddAnalysis(nil, oovLat, numToken)
	}
	hostLat, hostExists = l.Lex[input]
	if !hostExists {
		hostLat, hostExists = checkRegexes(input)
	}
	if hostExists {
		if logAnalyze {
			log.Println("\tPrefix 0")
		}
		lat.AddAnalysis(nil, hostLat, numToken)
		anyExists = true
	} else {
		if !l.AlwaysNNP {
			l.AddOOVAnalysis(lat, nil, input, numToken)
			// oovLat := l.OOVAnalysis(input)
			// lat.AddAnalysis(nil, oovLat, numToken)
		}
	}
	for i := 1; i <= util.Min(l.MaxPrefixLen, len(input)); i++ {
		if logAnalyze {
			log.Println("\ti is", i)
		}
		found := l.analyzeTokenForLen(lat, input, startingNode, numToken, i)
		anyExists = anyExists || found
	}
	if !anyExists {
		// if logAnalyze {
		if l.LogOOV {
			log.Println("Token", numToken, "is OOV:", input)
		}
		for i := 1; i < util.Min(l.MaxPrefixLen, len(input)); i++ {
			if logAnalyze {
				log.Println("\ti is", i)
			}
			_ = l.OOVForLen(lat, input, startingNode, numToken, i)
		}
		// }
		if l.Stats != nil {
			l.Stats.OOVTokens++
			l.Stats.AddOOVToken(input)
		}
	}
	lat.Optimize()
	return lat, !anyExists
}

func (l *BGULex) Analyze(input []string) (LatticeSentence, interface{}) {
	retval := make(LatticeSentence, len(input))
	var (
		lat     *Lattice
		curNode int
		oovInd  BasicSentence
		oovFlag interface{}
	)
	oovInd = make(BasicSentence, len(input))
	for i, token := range input {
		if l.Stats != nil {
			l.Stats.TotalTokens++
			l.Stats.AddToken(token)
		}
		lat, oovFlag = l.AnalyzeToken(token, curNode, i)
		if oovFlag.(bool) {
			oovInd[i] = Token("1")
		} else {
			oovInd[i] = Token("0")
		}
		curNode = lat.Top()
		// log.Println("New top is", curNode)
		retval[i] = *lat
	}
	return retval, oovInd
}
