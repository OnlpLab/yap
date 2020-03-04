package ma

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/OnlpLab/yap/alg/graph"
	"github.com/OnlpLab/yap/nlp/format/conllu"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"github.com/OnlpLab/yap/nlp/format/raw"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
)

const (
	MSR_SEPARATOR = "|"
	PUNCTUATION   = ",.|?!:;-&»«\"[]()<>"
)

type TrainingFile struct {
	Lattice, Raw, LatMD5, RawMD5 string
}

type TokenDictionary map[string][]BasicMorphemes

type MSRFreq map[string]int

type MADict struct {
	Language  string
	NumTokens int

	// for OOV
	MaxTopPOS, MaxMSRsPerPOS int
	TopPOS                   []string
	OOVMSRs                  []string
	POSMSRs                  map[string]MSRFreq

	// data
	Files []TrainingFile
	Data  TokenDictionary

	Stats *AnalyzeStats

	TopPOSSet map[string]bool
	Dope      bool
}

var _ MorphologicalAnalyzer = &MADict{}

func (m *MADict) LearnFromConllU(conlluFile string, limit int) (int, error) {
	latmd5, err := util.MD5File(conlluFile)
	if err != nil {
		return 0, err
	}
	// whatever a conllu is.. a morpho-syntactic sentence?
	conllus, _, err := conllu.ReadFile(conlluFile, limit)
	if err != nil {
		log.Println("Error reading conllu file")
		return 0, err
	}
	if m.Data == nil {
		m.Data = make(TokenDictionary)
	}
	if m.POSMSRs == nil {
		m.POSMSRs = make(map[string]MSRFreq, 100)
	}
	eWord := util.NewEnumSet(100)
	ePOS := util.NewEnumSet(100)
	eWPOS := util.NewEnumSet(100)
	eMorphFeat := util.NewEnumSet(100)
	eMHost := util.NewEnumSet(100)
	eMSuffix := util.NewEnumSet(100)
	eRel := util.NewEnumSet(100)

	corpus := conllu.ConllU2MorphGraphCorpus(conllus, eWord, ePOS, eWPOS, eRel, eMorphFeat, eMHost, eMSuffix)
	for _, _sentLat := range corpus {
		// log.Println("At sentence", i)
		sentLat := _sentLat.(MorphDependencyGraph)
		for _, mapping := range sentLat.GetMappings() {
			standalone := Morphemes(mapping.Spellout).Standalone()
			// log.Println("\tAt token", curToken)
			m.AddAnalyses(string(mapping.Token), standalone)
			m.AddMSRs(standalone)
		}
	}
	if m.Files == nil {
		m.Files = make([]TrainingFile, 0, 1)
	}
	m.Files = append(m.Files, TrainingFile{conlluFile, "", latmd5, ""})

	tokensRead := len(m.Data) - m.NumTokens
	m.NumTokens = len(m.Data)
	if m.MaxTopPOS == 0 {
		m.MaxTopPOS = 6
	}
	// if m.MaxMSRsPerPOS == 0 {
	// 	m.MaxMSRsPerPOS = 5
	// }

	m.ComputeTopPOS()
	m.ComputeOOVMSRs(m.MaxMSRsPerPOS)

	return tokensRead, nil
}

func (m *MADict) Init() {
	m.TopPOSSet = make(map[string]bool, len(m.TopPOS))
	for _, pos := range m.TopPOS {
		m.TopPOSSet[pos] = true
	}
}
func (m *MADict) LearnFromLat(latticeFile, rawFile string, limit int) (int, error) {
	latmd5, err := util.MD5File(latticeFile)
	if err != nil {
		return 0, err
	}
	rawmd5, err := util.MD5File(rawFile)
	if err != nil {
		return 0, err
	}
	lattices, err := lattice.ReadFile(latticeFile, limit)
	if err != nil {
		log.Println("Error reading lattice file")
		return 0, err
	}
	tokens, err := raw.ReadFile(rawFile, limit)
	if err != nil {
		log.Println("Error reading raw file")
		return 0, err
	}
	if len(lattices) != len(tokens) {
		log.Println("Read", len(lattices), "lattices and", len(tokens), "raw tokens")
		return 0, errors.New("Number of read sentences differ for lattice and raw files")
	}
	if m.Data == nil {
		m.Data = make(TokenDictionary)
	}
	if m.POSMSRs == nil {
		m.POSMSRs = make(map[string]MSRFreq, 100)
	}
	eWord := util.NewEnumSet(100)
	ePOS := util.NewEnumSet(100)
	eWPOS := util.NewEnumSet(100)
	eMorphFeat := util.NewEnumSet(100)
	eMHost := util.NewEnumSet(100)
	eMSuffix := util.NewEnumSet(100)

	corpus := lattice.Lattice2SentenceCorpus(lattices, eWord, ePOS, eWPOS, eMorphFeat, eMHost, eMSuffix)
	for i, _sentLat := range corpus {
		// log.Println("At sentence", i)
		sentLat := _sentLat.(LatticeSentence)
		curTokens := tokens[i]
		for j, lat := range sentLat {
			curToken := curTokens[j]
			// log.Println("\tAt token", curToken)
			m.AddAnalyses(string(curToken), lat.Morphemes.Standalone())
			m.AddMSRs(lat.Morphemes.Standalone())
		}
	}
	if m.Files == nil {
		m.Files = make([]TrainingFile, 0, 1)
	}
	m.Files = append(m.Files, TrainingFile{latticeFile, rawFile, latmd5, rawmd5})

	tokensRead := len(m.Data) - m.NumTokens
	m.NumTokens = len(m.Data)
	if m.MaxTopPOS == 0 {
		m.MaxTopPOS = 5
	}
	if m.MaxMSRsPerPOS == 0 {
		m.MaxMSRsPerPOS = 10
	}

	m.ComputeTopPOS()
	m.ComputeOOVMSRs(m.MaxMSRsPerPOS)

	return tokensRead, nil
}

func (m *MADict) ComputeTopPOS() {
	// compute frequency of CPOS in data dict
	posCnt := make(map[string]int, 100)
	puncArray := strings.Split(PUNCTUATION, "")
	for _, allmorphs := range m.Data {
		for _, morphs := range allmorphs {
		morphLoop:
			for _, morph := range morphs {
				if len(morph.CPOS) == 1 && strings.Contains(PUNCTUATION, morph.CPOS) {
					// punctuation specified as CPOS is skipped
					continue morphLoop
				}
				for _, punc := range puncArray {
					if punc != "|" && strings.Contains(morph.FeatureStr, punc) {
						continue morphLoop
					}
				}
				if cnt, exists := posCnt[morph.CPOS]; exists {
					posCnt[morph.CPOS] = cnt + 1
				} else {
					posCnt[morph.CPOS] = 1
				}
			}
		}
	}

	topN := util.GetTopNStrInt(posCnt, m.MaxTopPOS)

	m.TopPOS = make([]string, len(topN))
	for i, val := range topN {
		m.TopPOS[i] = val.S
	}
}

func (m *MADict) oldComputeOOVMSRs(maxMSRs int) {
	maxMSRs = util.Min(m.MaxMSRsPerPOS, maxMSRs)
	m.OOVMSRs = make([]string, 0, len(m.TopPOS)*maxMSRs)
	maxMSRs = util.Min(m.MaxMSRsPerPOS, maxMSRs)
	log.Println("Computing OOV MSRs, max MSRs:", maxMSRs)
	for _, pos := range m.TopPOS {
		log.Println(pos + ":")
		msrfreq, exists := m.POSMSRs[pos]
		if !exists {
			fmt.Println("Top POS has no non-empty MSRs")
			continue
			// fmt.Println("Top POSs:")
			// fmt.Println(m.TopPOS)
			// fmt.Println("Top MSRs by POS:")
			// fmt.Println(m.POSMSRs)
			// panic("Top POS does not have an MSR frequency entry")
		}
		topN := util.GetTopNStrInt(msrfreq, maxMSRs)
		for _, msrkv := range topN {
			log.Println("\t", strings.Split(msrkv.S, MSR_SEPARATOR), "# occurences:", msrkv.N)
			m.OOVMSRs = append(m.OOVMSRs, strings.Join([]string{pos, msrkv.S}, MSR_SEPARATOR))
		}
	}
}

func (m *MADict) ComputeOOVMSRs(maxMSRs int) {
	maxMSRs = util.Min(m.MaxMSRsPerPOS, maxMSRs)
	m.OOVMSRs = make([]string, 0, maxMSRs)
	log.Println("Computing OOV MSRs, max MSRs:", maxMSRs)
	allMSRFreq := make(map[string]int, maxMSRs)
	for _, pos := range m.TopPOS {
		log.Println(pos + ":")
		msrfreq, exists := m.POSMSRs[pos]
		if !exists {
			fmt.Println("Top POS has no non-empty MSRs")
			continue
			// fmt.Println("Top POSs:")
			// fmt.Println(m.TopPOS)
			// fmt.Println("Top MSRs by POS:")
			// fmt.Println(m.POSMSRs)
			// panic("Top POS does not have an MSR frequency entry")
		}
		for k, v := range msrfreq {
			allMSRFreq[strings.Join([]string{pos, k}, MSR_SEPARATOR)] = v
		}
	}
	topN := util.GetTopNStrInt(allMSRFreq, maxMSRs)
	for _, msrkv := range topN {
		log.Println("\t", strings.Split(msrkv.S, MSR_SEPARATOR), "# occurences:", msrkv.N)
		m.OOVMSRs = append(m.OOVMSRs, msrkv.S)
	}
	for _, topPOS := range m.TopPOS {
		if len(m.OOVMSRs) < maxMSRs {
			m.OOVMSRs = append(m.OOVMSRs, topPOS+MSR_SEPARATOR+topPOS+MSR_SEPARATOR)
		} else {
			break
		}
	}
}

// MSR: Morpho-Syntactic Representation
func (m *MADict) AddMSRs(morphs BasicMorphemes) {
	for _, morph := range morphs {
		msr := strings.Join([]string{morph.CPOS, morph.FeatureStr}, MSR_SEPARATOR)
		if len(morph.FeatureStr) == 0 || morph.FeatureStr == "_" {
			continue
		}
		if freq, exists := m.POSMSRs[morph.CPOS]; exists {
			if cnt, msrexists := freq[msr]; msrexists {
				freq[msr] = cnt + 1
				m.POSMSRs[morph.CPOS] = freq
			} else {
				freq[msr] = 1
			}
		} else {
			freq := make(MSRFreq, 1000)
			freq[msr] = 1
			m.POSMSRs[morph.CPOS] = freq
		}
	}
}

func (m *MADict) AddAnalyses(token string, morphs BasicMorphemes) {
	if curAnalysis, exists := m.Data[token]; exists {
		// curAnalysis := _curAnalysis.(BasicMorphemes)
		// log.Println("\t\tFound, unioning")
		for _, existAnalysis := range curAnalysis {
			// check if an analysis already exists
			if existAnalysis.Equal(morphs) {
				return
			}
		}
		curAnalysis = append(curAnalysis, morphs)
		m.Data[token] = curAnalysis
		// log.Println("\t\tPost union")
		// log.Println("\t\t", m.Data[string(token)])
	} else {
		// log.Println("\t\tAdding")
		m.Data[token] = []BasicMorphemes{morphs}
		// log.Println("\t\tPost adding")
		// log.Println("\t\t", m.Data[string(token)])
	}

}

// func (m *MADict) SetOOVs(oovs string) {
// 	m.OOVPOS = strings.Split(oovs, OOV_POS_SET_SEPARATOR)
// }

func (m *MADict) Write(writer io.Writer) error {
	enc := json.NewEncoder(writer)
	err := enc.Encode(m)
	return err
}

func (m *MADict) Read(r io.Reader) error {
	dec := json.NewDecoder(r)
	err := dec.Decode(m)
	return err
}

func (m *MADict) WriteFile(filename string) error {
	file, err := os.Create(filename)
	defer file.Close()

	if err != nil {
		return err
	}

	return m.Write(file)
}

func (m *MADict) ReadFile(filename string) error {
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		return err
	}

	return m.Read(file)
}

type AnalyzeStats struct {
	TotalTokens, OOVTokens    int
	UniqTokens, UniqOOVTokens map[string]int
}

func (a *AnalyzeStats) Init() {
	a.UniqTokens = make(map[string]int, 10000)
	a.UniqOOVTokens = make(map[string]int, 1000)
}

func (a *AnalyzeStats) AddToken(token string) {
	if cnt, exists := a.UniqTokens[token]; exists {
		a.UniqTokens[token] = cnt + 1
	} else {
		a.UniqTokens[token] = 1
	}
}

func (a *AnalyzeStats) AddOOVToken(token string) {
	if cnt, exists := a.UniqOOVTokens[token]; exists {
		a.UniqOOVTokens[token] = cnt + 1
	} else {
		a.UniqOOVTokens[token] = 1
	}
}

func (m *MADict) oldApplyOOV(token string, lat *Lattice, curID *int, curNode, i int) {
	// add morphemes for Out-Of-Vocabulary
	lat.Morphemes = make([]*EMorpheme, 0, len(m.OOVMSRs)+len(m.TopPOS))
	for _, pos := range m.TopPOS {
		lat.AddAnalysis(nil, []BasicMorphemes{BasicMorphemes{&Morpheme{
			graph.BasicDirectedEdge{*curID, curNode, curNode + 1},
			token,
			"_",
			pos,
			pos,
			nil,
			i,
			"_",
		},
		}}, i+1)
		*curID++
	}
	for _, msr := range m.OOVMSRs {
		split := strings.Split(msr, MSR_SEPARATOR)
		lat.AddAnalysis(nil, []BasicMorphemes{BasicMorphemes{&Morpheme{
			graph.BasicDirectedEdge{*curID, curNode, curNode + 1},
			token,
			"_",
			split[0],
			split[1],
			nil,
			i,
			split[2],
		},
		}}, i+1)
		*curID++
	}
}

func (m *MADict) ApplyOOV(token string, lat *Lattice, curID *int, curNode, i int) {
	// add morphemes for Out-Of-Vocabulary
	lat.Morphemes = make([]*EMorpheme, 0, len(m.OOVMSRs))
	for _, msr := range m.OOVMSRs {
		split := strings.Split(msr, MSR_SEPARATOR)
		lat.AddAnalysis(nil, []BasicMorphemes{BasicMorphemes{&Morpheme{
			graph.BasicDirectedEdge{*curID, curNode, curNode + 1},
			token,
			"_",
			split[0],
			split[1],
			nil,
			i,
			strings.Join(split[2:], MSR_SEPARATOR),
		},
		}}, i+1)
		*curID++
	}
}

func (m *MADict) ReadUDLex(reader io.Reader) error {
	// empty current dictionary
	m.Data = make(TokenDictionary)

	// read ud lex file
	var (
		i                int
		line             int
		token            string
		segments         BasicMorphemes = make(BasicMorphemes, 0, 1)
		tokenEnd         int64
		curStart, curEnd int64
	)
	bufReader := bufio.NewReaderSize(reader, 16384)
	// log.Println("At record", i)
	for curLine, isPrefix, err := bufReader.ReadLine(); err == nil; curLine, isPrefix, err = bufReader.ReadLine() {
		// log.Println("\tLine", line)
		if isPrefix {
			panic("Buffer not large enough, fix me :(")
		}
		buf := bytes.NewBuffer(curLine)
		record := strings.Split(buf.String(), "\t")
		// '#' is a start of comment
		if record[0][0] == '#' {
			line++
			continue
		}

		if strings.Contains(record[0], "-") {
			if len(segments) > 0 {
				// make sure a previously started multi-segment has not completed
				panic(fmt.Sprintf("Previous multi-segment not completed at line %d", line))
			}
			// start of multi segment token
			token = record[1]
			rangeSplit := strings.Split(record[0], "-")
			if tokenEnd, err = strconv.ParseInt(rangeSplit[1], 0, 0); err != nil {
				panic(fmt.Sprintf("Error reading UD Lex at line %d: %s", line, err))
			}
			segments = make(BasicMorphemes, 0, 2)
			line++
			continue
		}
		if curStart, err = strconv.ParseInt(record[0], 0, 0); err != nil {
			panic(fmt.Sprintf("Error reading UD Lex at line %d: %s", line, err))
		}
		if curEnd, err = strconv.ParseInt(record[1], 0, 0); err != nil {
			panic(fmt.Sprintf("Error reading UD Lex at line %d: %s", line, err))
		}
		token = record[2]
		morpheme := &Morpheme{
			BasicDirectedEdge: graph.BasicDirectedEdge{0, int(curStart), int(curEnd)},
			Form:              record[2],
			Lemma:             record[3],
			CPOS:              record[4],
			POS:               record[5],
			FeatureStr:        record[6],
		}
		if curEnd == tokenEnd {
			// if edge's end == end of multirange
			//   add segment and previous segments to new entry
			segments = append(segments, morpheme)
			m.AddAnalyses(token, segments)

			// restart
			token = ""
			segments = nil
			tokenEnd = 0
			i++
			line++
			continue
		}
		if curEnd < tokenEnd {
			// if part of multirange
			//   append to segments
			segments = append(segments, morpheme)
			line++
			continue
		}
		if curStart != 0 {
			// if start != 0 error
			panic(fmt.Sprintf("Unexpected mid-multi segment morpheme at line %d", line))
		}
		m.AddAnalyses(token, BasicMorphemes{morpheme})
		segments = nil
		i++
		line++
	}
	return nil
}

func (m *MADict) ReadUDLexFile(filename string) error {
	file, err := os.Open(filename)
	defer file.Close()

	if err != nil {
		return err
	}

	return m.ReadUDLex(file)
}
func (m *MADict) Analyze(input []string) (LatticeSentence, interface{}) {
	retval := make(LatticeSentence, len(input))
	oovVector := make(BasicSentence, len(input))
	var (
		curNode, curID, lastTop int
		hasOOVPOS               bool
	)
	for i, token := range input {
		if m.Stats != nil {
			m.Stats.TotalTokens++
			m.Stats.AddToken(token)
		}
		lat := &retval[i]
		lat.Token = Token(token)
		lat.Next = make(map[int][]int)
		lat.BottomId = lastTop
		lat.TopId = lastTop
		hasOOVPOS = false
		// TODO: Add regexes for NUM (& times, dates, etc)
		if allmorphs, exists := m.Data[token]; exists {
			oovVector[i] = "0"
		outer:
			for _, morphs := range allmorphs {
				for _, morph := range morphs {
					if _, oovexists := m.TopPOSSet[morph.CPOS]; oovexists {
						hasOOVPOS = true
						break outer
					}
				}
			}
			if m.Dope && hasOOVPOS {
				m.ApplyOOV(token, lat, &curID, curNode, i)
			}
			lat.AddAnalysis(nil, allmorphs, i+1)
		} else {
			oovVector[i] = "1"
			if m.Stats != nil {
				m.Stats.OOVTokens++
				m.Stats.AddOOVToken(token)
			}
			m.ApplyOOV(token, lat, &curID, curNode, i)
		}
		lastTop = lat.Top()
		curNode++
	}
	return retval, oovVector
}
