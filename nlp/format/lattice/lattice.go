package lattice

// Package Lattice reads lattice format files

import (
	"encoding/json"
	"github.com/OnlpLab/yap/alg/graph"
	"github.com/OnlpLab/yap/nlp/parser/xliter8"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"

	"log"
)

const (
	PRONOMINAL_CLITIC_POS = "S_PRN"
)

var (
	_FIX_FUSIONAL_H        = true
	_FIX_PRONOMINAL_CLITIC = true
	_FIX_ECMx              = false

	_FUSIONAL_PREFIXES      = map[string]bool{"B": true, "K": true, "L": true}
	ECMx_INSTANCES          = map[string]bool{"ECMW": true, "ECMI": true, "ECMH": true, "ECMM": true}
	IGNORE_LEMMA            = true
	IGNORE_DUP              = true
	WORD_TYPE               = "form"
	IGNORE_NNP_FEATS        = false
	OVERRIDE_XPOS_WITH_UPOS bool
)

type Features map[string]string

type JSONEdge struct {
	Next    string `json:"next"`
	Form    string `json:"form"`
	Lemma   string `json:"lemma,omitempty"`
	UPOSTag string `json:"upostag"`
	XPOSTag string `json:"xpostag,omitempty"`
	Feats   string `json:"feats,omitempty"`
	Misc    string `json:"misc,omitempty"`
	TokenID int    `json:"tokenid,omitempty"`
}

type JSONLattice map[string][]JSONEdge

func (f Features) String() string {
	if f != nil || len(f) == 0 {
		return "_"
	}
	return fmt.Sprintf("%v", map[string]string(f))
}

func (f Features) Union(other Features) {
	for k, v := range other {
		if curVal, exists := f[k]; exists {
			f[k] = curVal + "," + v
		} else {
			f[k] = v
		}
	}
}

func (f Features) Copy() Features {
	newF := make(Features, len(f))
	for k, v := range f {
		newF[k] = v
	}
	return newF
}

func (f Features) MorphHost() string {
	hostStrs := make([]string, 0, len(f))
	for name, value := range f {
		if len(name) > 2 && name[0:3] != "suf" {
			hostStrs = append(hostStrs, fmt.Sprintf("%v=%v", name, value))
		}
	}
	sort.Strings(hostStrs)
	return strings.Join(hostStrs, "|")
}

func (f Features) MorphSuffix() string {
	hostStrs := make([]string, 0, len(f))
	for name, value := range f {
		if len(name) > 2 && name[0:3] == "suf" {
			hostStrs = append(hostStrs, fmt.Sprintf("%v=%v", name, value))
		}
	}
	sort.Strings(hostStrs)
	return strings.Join(hostStrs, "|")
}

type Edge struct {
	Start    int // can be negative, if so skip over
	End      int
	Word     string
	Lemma    string
	CPosTag  string
	PosTag   string
	Feats    Features
	FeatStr  string
	Token    int
	Id       int
	TokenStr string
}

type EdgeSlice []Edge

func (e EdgeSlice) Len() int {
	return len(e)
}

func (e EdgeSlice) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (e EdgeSlice) Less(i, j int) bool {
	return e[i].Start <= e[j].Start && e[i].End <= e[j].End && e[i].Word <= e[j].Word
}

func (e Edge) Equal(other Edge) bool {
	return e.Start == other.Start &&
		e.End == other.End &&
		e.Word == other.Word &&
		e.Lemma == other.Lemma &&
		e.CPosTag == other.CPosTag &&
		e.PosTag == other.PosTag &&
		e.FeatStr == other.FeatStr &&
		e.Token == other.Token
}

func (e Edge) String() string {
	fields := []string{
		fmt.Sprintf("%d", e.Start),
		fmt.Sprintf("%d", e.End),
		e.Word,
		e.Lemma,
		e.CPosTag,
		e.PosTag,
		e.FeatStr,
		fmt.Sprintf("%d", e.Token),
	}
	if len(e.Lemma) == 0 {
		fields[3] = "_"
	}
	return strings.Join(fields, "\t")
}

func (e Edge) UDString() string {
	var xpostag string
	if OVERRIDE_XPOS_WITH_UPOS {
		xpostag = e.CPosTag
	} else {
		xpostag = e.PosTag
	}
	fields := []string{
		fmt.Sprintf("%d", e.Start),
		fmt.Sprintf("%d", e.End),
		e.Word,
		e.Lemma,
		e.CPosTag,
		xpostag,
		e.FeatStr,
		"_",
		"_",
	}
	if len(e.Lemma) == 0 {
		fields[3] = "_"
	}
	return strings.Join(fields, "\t")
}

func (e *Edge) Copy() *Edge {
	newEdge := new(Edge)
	*newEdge = *e
	newEdge.Feats = e.Feats.Copy()
	return newEdge
}

type Lattice map[int][]Edge

func (l Lattice) MaxKey() (retval int) {
	for k, _ := range l {
		if k > retval {
			retval = k
		}
	}
	return
}

type Lattices []Lattice

const (
	FIELD_SEPARATOR      = '\t'
	NUM_FIELDS           = 8
	FEATURES_SEPARATOR   = "|"
	FEATURE_SEPARATOR    = "="
	FEATURE_CONCAT_DELIM = ","
)

func ParseInt(value string) (int, error) {
	if value == "_" {
		return 0, nil
	}
	i, err := strconv.ParseInt(value, 10, 0)
	return int(i), err
}

func ParseString(value string) string {
	val := value
	if val == "_" {
		val = ""
	}
	return val
}

func ParseFeatures(featuresStr string) (Features, error) {
	var featureMap Features = make(Features)
	if featuresStr == "_" {
		return featureMap, nil
	}

	featureList := strings.Split(featuresStr, FEATURES_SEPARATOR)
	if len(featureList) == 0 {
		return nil, errors.New("No features found, field should be '_'")
	}
	featureMap = make(Features, len(featureList))
	for _, featureStr := range featureList {
		featureKV := strings.Split(featureStr, FEATURE_SEPARATOR)
		switch len(featureKV) {
		case 1:
			featureMap[featureKV[0]] = featureKV[0]
		case 2:
			featName := featureKV[0]
			featValue := featureKV[1]
			existingFeatValue, featExist := featureMap[featName]
			if featExist {
				featureMap[featName] = existingFeatValue + FEATURE_CONCAT_DELIM + featValue
			} else {
				featureMap[featName] = featValue
			}
		case 3:
			// special hack for case where feature value is the feature
			// separator, like "SubPOS==", which (of course) exists in the SPMRL
			// corpus
			featName := featureKV[0]
			featValue := FEATURE_SEPARATOR
			existingFeatValue, featExist := featureMap[featName]
			if featExist {
				featureMap[featName] = existingFeatValue + FEATURE_CONCAT_DELIM + featValue
			} else {
				featureMap[featName] = featValue
			}

		default:
			return nil, errors.New("Wrong number of fields for split of feature" + featureStr)
		}
	}
	return featureMap, nil
}

func ParseULEdge(record []string) (*Edge, error) {
	row := &Edge{}
	start, err := ParseInt(record[0])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing START field (%s): %s", record[0], err.Error()))
	}
	row.Start = start

	end, err := ParseInt(record[1])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing END field (%s): %s", record[1], err.Error()))
	}
	row.End = end

	word := ParseString(record[2])
	// if word == "" {
	// 	return row, errors.New("Empty WORD field")
	// }
	row.Word = word

	//if !IGNORE_LEMMA {
		lemma := ParseString(record[3])
		row.Lemma = lemma
	//}
	// if lemma == "" {
	// 	return row, errors.New("Empty LEMMA field")
	// }

	upostag := ParseString(record[4])
	if upostag == "" {
		return row, errors.New("Empty UPOSTAG field")
	}
	row.CPosTag = upostag

	xpostag := ParseString(record[5])
	// Note, xpostag may be empty in conllu
	row.PosTag = xpostag

	if IGNORE_NNP_FEATS && upostag == "NNP" {
		record[6] = "_"
	}
	features, err := ParseFeatures(record[6])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing FEATS field (%s): %s", record[6], err.Error()))
	}
	row.Feats = features
	row.FeatStr = ParseString(record[6])
	return row, nil
}

func ParseUDEdge(record []string) (*Edge, error) {
	row := &Edge{}
	start, err := ParseInt(record[0])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing START field (%s): %s", record[0], err.Error()))
	}
	row.Start = start

	end, err := ParseInt(record[1])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing END field (%s): %s", record[1], err.Error()))
	}
	row.End = end

	word := ParseString(record[2])
	// if word == "" {
	// 	return row, errors.New("Empty WORD field")
	// }
	row.Word = word

	//if !IGNORE_LEMMA {
		lemma := ParseString(record[3])
		row.Lemma = lemma
	//}
	// if lemma == "" {
	// 	return row, errors.New("Empty LEMMA field")
	// }

	upostag := ParseString(record[4])
	if upostag == "" {
		return row, errors.New("Empty UPOSTAG field")
	}
	row.CPosTag = upostag

	xpostag := ParseString(record[5])
	// Note, xpostag may be empty in conllu
	row.PosTag = xpostag

	token, err := ParseInt(record[8])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing TOKEN field (%s): %s", record[8], err.Error()))
	}
	row.Token = token

	if IGNORE_NNP_FEATS && upostag == "NNP" {
		record[6] = "_"
	}
	features, err := ParseFeatures(record[6])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing FEATS field (%s): %s", record[6], err.Error()))
	}
	row.Feats = features
	row.FeatStr = ParseString(record[6])
	return row, nil
}

func ParseEdge(record []string) (*Edge, error) {
	row := &Edge{}
	start, err := ParseInt(record[0])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing START field (%s): %s", record[0], err.Error()))
	}
	row.Start = start

	end, err := ParseInt(record[1])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing END field (%s): %s", record[1], err.Error()))
	}
	row.End = end

	word := ParseString(record[2])
	// if word == "" {
	// 	return row, errors.New("Empty WORD field")
	// }
	row.Word = word

	//if !IGNORE_LEMMA {
		lemma := ParseString(record[3])
		row.Lemma = lemma
	//}
	// if lemma == "" {
	// 	return row, errors.New("Empty LEMMA field")
	// }

	cpostag := ParseString(record[4])
	if cpostag == "" {
		return row, errors.New("Empty CPOSTAG field")
	}
	row.CPosTag = cpostag

	postag := ParseString(record[5])
	if postag == "" {
		return row, errors.New("Empty POSTAG field")
	}
	row.PosTag = postag

	token, err := ParseInt(record[7])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing TOKEN field (%s): %s", record[7], err.Error()))
	}
	row.Token = token

	if IGNORE_NNP_FEATS && cpostag == "NNP" {
		record[6] = "_"
	}
	features, err := ParseFeatures(record[6])
	if err != nil {
		return row, errors.New(fmt.Sprintf("Error parsing FEATS field (%s): %s", record[6], err.Error()))
	}
	row.Feats = features
	row.FeatStr = ParseString(record[6])
	return row, nil
}

func ReadStream(in *os.File, limit int) chan Lattice {
	s := make(chan Lattice, 2)
	go func(sentences chan Lattice, r *os.File) {
		defer r.Close()
		log.Println("Starting to read stream")
		bufReader := bufio.NewReader(r)

		var (
			currentLatt  Lattice = make(Lattice)
			currentEdge  int
			i            int
			dup          bool
			numSentences int
		)
		curLine, isPrefix, err := bufReader.ReadLine()
		if err != nil {
			log.Println("Error at record", i, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, numSentences, err.Error())))
			panic("Error reading input")
		}
		for ; err == nil; curLine, isPrefix, err = bufReader.ReadLine() {
			if isPrefix {
				panic("Buffer not large enough, fix me :(")
			}
			buf := bytes.NewBuffer(curLine)
			// a record with id '1' indicates a new sentence
			// since csv reader ignores empty lines
			// TODO: fix to work with empty lines as new sentence indicator
			if len(curLine) == 0 {
				// store current sentence
				sentences <- currentLatt
				numSentences++
				if limit > 0 && numSentences >= limit {
					close(sentences)
					return
				}
				currentLatt = make(Lattice)
				currentEdge = 0
				i++
				continue
			} else {
				currentEdge += 1
			}
			record := strings.Split(buf.String(), "\t")

			edge, err := ParseEdge(record)
			if edge.Start == edge.End {
				log.Println("At sent:", len(sentences), "Warning: found circular edge", edge, ", optimistically incrementing end")
				edge.End += 1
			}
			if err != nil {
				log.Println("Error at record", i, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, numSentences, err.Error())))
			}
			edge.Id = currentEdge
			edges, exists := currentLatt[edge.Start]
			if exists {
				dup = false
				for _, otherEdge := range edges {
					if edge.Equal(otherEdge) {
						dup = true
					}
				}
				if IGNORE_DUP && !dup {
					currentLatt[edge.Start] = append(edges, *edge)
				}
			} else {
				currentLatt[edge.Start] = []Edge{*edge}
			}
			i++
		}
		close(sentences)
	}(s, in)
	return s
}

func Read(r io.Reader, limit int) ([]Lattice, error) {
	var sentences []Lattice
	bufReader := bufio.NewReader(r)

	var (
		currentLatt Lattice = make(Lattice)
		currentEdge int
		i           int
		dup         bool
	)
	for curLine, isPrefix, err := bufReader.ReadLine(); err == nil; curLine, isPrefix, err = bufReader.ReadLine() {
		if isPrefix {
			panic("Buffer not large enough, fix me :(")
		}
		buf := bytes.NewBuffer(curLine)
		// a record with id '1' indicates a new sentence
		// since csv reader ignores empty lines
		// TODO: fix to work with empty lines as new sentence indicator
		if len(curLine) == 0 {
			// store current sentence
			sentences = append(sentences, currentLatt)
			if limit > 0 && len(sentences) >= limit {
				break
			}
			currentLatt = make(Lattice)
			currentEdge = 0
			i++
			continue
		} else {
			currentEdge += 1
		}
		record := strings.Split(buf.String(), "\t")

		edge, err := ParseEdge(record)
		if edge.Start == edge.End {
			log.Println("At sent:", len(sentences), "Warning: found circular edge", edge, ", optimistically incrementing end")
			edge.End += 1
		}
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, len(sentences), err.Error()))
		}
		edge.Id = currentEdge
		edges, exists := currentLatt[edge.Start]
		if exists {
			dup = false
			for _, otherEdge := range edges {
				if edge.Equal(otherEdge) {
					dup = true
				}
			}
			if IGNORE_DUP && !dup {
				currentLatt[edge.Start] = append(edges, *edge)
			}
		} else {
			currentLatt[edge.Start] = []Edge{*edge}
		}
		i++
	}
	return sentences, nil
}

func ULReadStream(in *os.File, limit int) chan Lattice {
	s := make(chan Lattice, 2)
	go func(sentences chan Lattice, r *os.File) {
		defer r.Close()
		log.Println("Starting to read stream")
		bufReader := bufio.NewReader(r)

		var (
			currentLatt       Lattice = make(Lattice)
			currentEdge       int
			i                 int
			dup               bool
			tokens            []string
			curToken          int = -1
			tokTop, tokBottom int
			parseErr          error
			numSentences      int
		)
		tokens = make([]string, 0, 10)
		curLine, isPrefix, err := bufReader.ReadLine()
		if err != nil {
			log.Println("Error at record", i, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, numSentences, err.Error())))
			panic("Error reading input")
		}
		for ; err == nil; curLine, isPrefix, err = bufReader.ReadLine() {
			if isPrefix {
				panic("Buffer not large enough, fix me :(")
			}
			buf := bytes.NewBuffer(curLine)
			if strings.HasPrefix(string(curLine), "#") {
				continue
			}
			if len(curLine) == 0 {
				// store current sentence
				sentences <- currentLatt
				numSentences++
				if limit > 0 && numSentences >= limit {
					close(sentences)
					break
				}
				currentLatt = make(Lattice)
				currentEdge = 0
				i++
				tokens = make([]string, 0, 10)
				curToken = -1
				tokTop, tokBottom = 0, 0
				continue
			} else {
				currentEdge++
			}
			record := strings.Split(buf.String(), "\t")

			if strings.Contains(record[0], "-") {
				tokens = append(tokens, record[1])
				curToken++
				tokSpan := strings.Split(record[0], "-")
				tokTop, parseErr = ParseInt(tokSpan[0])
				if parseErr != nil {
					log.Printf("Error processing record %d at statement %d: %s\n", i, len(sentences), parseErr.Error())
					close(sentences)
					return
				}
				tokBottom, parseErr = ParseInt(tokSpan[1])
				if parseErr != nil {
					log.Printf("Error processing record %d at statement %d: %s\n", i, len(sentences), parseErr.Error())
					close(sentences)
					return
				}
				continue
			}

			edge, err := ParseULEdge(record)
			// for non-multi-segment tokens, detect when a single-segment edge is
			// a new token
			if edge.Start >= tokTop && edge.End > tokBottom {
				// log.Println("Starting a new token for edge", currentEdge, buf.String())
				tokens = append(tokens, edge.Word)
				tokTop = edge.Start
				tokBottom = edge.End
				curToken++
			}

			edge.Token = curToken + 1
			edge.TokenStr = tokens[edge.Token-1]
			if edge.Start == edge.End {
				log.Println("At sent:", len(sentences), "Warning: found circular edge", edge, ", optimistically incrementing end")
				edge.End += 1
			}
			if err != nil {
				log.Printf("Error processing record %d at statement %d: %s\n", i, len(sentences), parseErr.Error())
				close(sentences)
				return
			}
			edge.Id = currentEdge
			edges, exists := currentLatt[edge.Start]
			if exists {
				dup = false
				for _, otherEdge := range edges {
					if edge.Equal(otherEdge) {
						dup = true
					}
				}
				if IGNORE_DUP && !dup {
					currentLatt[edge.Start] = append(edges, *edge)
				}
			} else {
				currentLatt[edge.Start] = []Edge{*edge}
			}
			i++
		}
		close(sentences)
	}(s, in)
	return s
}

func ULRead(r io.Reader, limit int) ([]Lattice, error) {
	var sentences []Lattice
	bufReader := bufio.NewReader(r)

	var (
		currentLatt       Lattice = make(Lattice)
		currentEdge       int
		i                 int
		dup               bool
		tokens            []string
		curToken          int = -1
		tokTop, tokBottom int
		parseErr          error
	)
	tokens = make([]string, 0, 10)
	for curLine, isPrefix, err := bufReader.ReadLine(); err == nil; curLine, isPrefix, err = bufReader.ReadLine() {
		if isPrefix {
			panic("Buffer not large enough, fix me :(")
		}
		buf := bytes.NewBuffer(curLine)
		if strings.HasPrefix(string(curLine), "#") {
			continue
		}
		if len(curLine) == 0 {
			// store current sentence
			sentences = append(sentences, currentLatt)
			if limit > 0 && len(sentences) >= limit {
				break
			}
			currentLatt = make(Lattice)
			currentEdge = 0
			i++
			tokens = make([]string, 0, 10)
			curToken = -1
			tokTop, tokBottom = 0, 0
			continue
		} else {
			currentEdge += 1
		}
		record := strings.Split(buf.String(), "\t")

		if strings.Contains(record[0], "-") {
			tokens = append(tokens, record[1])
			curToken++
			tokSpan := strings.Split(record[0], "-")
			tokTop, parseErr = ParseInt(tokSpan[0])
			if parseErr != nil {
				return nil, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, len(sentences), parseErr.Error()))
			}
			tokBottom, parseErr = ParseInt(tokSpan[1])
			if parseErr != nil {
				return nil, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, len(sentences), parseErr.Error()))
			}
			continue
		}

		edge, err := ParseULEdge(record)
		// for non-multi-segment tokens, detect when a single-segment edge is
		// a new token
		if edge.Start >= tokTop && edge.End > tokBottom {
			// log.Println("Starting a new token for edge", currentEdge, buf.String())
			tokens = append(tokens, edge.Word)
			tokTop = edge.Start
			tokBottom = edge.End
			curToken++
		}

		edge.Token = curToken + 1
		edge.TokenStr = tokens[edge.Token-1]
		if edge.Start == edge.End {
			log.Println("At sent:", len(sentences), "Warning: found circular edge", edge, ", optimistically incrementing end")
			edge.End += 1
		}
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, len(sentences), err.Error()))
		}
		edge.Id = currentEdge
		edges, exists := currentLatt[edge.Start]
		if exists {
			dup = false
			for _, otherEdge := range edges {
				if edge.Equal(otherEdge) {
					dup = true
				}
			}
			if IGNORE_DUP && !dup {
				currentLatt[edge.Start] = append(edges, *edge)
			}
		} else {
			currentLatt[edge.Start] = []Edge{*edge}
		}
		i++
	}
	return sentences, nil
}

func UDRead(r io.Reader, limit int) ([]Lattice, error) {
	var sentences []Lattice
	bufReader := bufio.NewReader(r)

	var (
		currentLatt Lattice = make(Lattice)
		currentEdge int
		i           int
		dup         bool
		tokens      []string
	)
	for curLine, isPrefix, err := bufReader.ReadLine(); err == nil; curLine, isPrefix, err = bufReader.ReadLine() {
		if isPrefix {
			panic("Buffer not large enough, fix me :(")
		}
		buf := bytes.NewBuffer(curLine)
		if strings.HasPrefix(string(curLine), "# ") {
			tokens = strings.Split(string(curLine[2:]), " ")
			continue
		}
		// a record with id '1' indicates a new sentence
		// since csv reader ignores empty lines
		// TODO: fix to work with empty lines as new sentence indicator
		if len(curLine) == 0 {
			// store current sentence
			sentences = append(sentences, currentLatt)
			if limit > 0 && len(sentences) >= limit {
				break
			}
			currentLatt = make(Lattice)
			currentEdge = 0
			i++
			tokens = []string{}
			continue
		} else {
			currentEdge += 1
		}
		record := strings.Split(buf.String(), "\t")

		edge, err := ParseUDEdge(record)
		edge.TokenStr = tokens[edge.Token-1]
		if edge.Start == edge.End {
			log.Println("At sent:", len(sentences), "Warning: found circular edge", edge, ", optimistically incrementing end")
			edge.End += 1
		}
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Error processing record %d at statement %d: %s", i, len(sentences), err.Error()))
		}
		edge.Id = currentEdge
		edges, exists := currentLatt[edge.Start]
		if exists {
			dup = false
			for _, otherEdge := range edges {
				if edge.Equal(otherEdge) {
					dup = true
				}
			}
			if IGNORE_DUP && !dup {
				currentLatt[edge.Start] = append(edges, *edge)
			}
		} else {
			currentLatt[edge.Start] = []Edge{*edge}
		}
		i++
	}
	return sentences, nil
}

func UDWrite(writer io.Writer, lattices []Lattice, comments [][]string, oovVectors []nlp.BasicSentence) error {
	var (
		lastToken    int
		tokenComment string
	)
	for latIdx, lattice := range lattices {
		if comments != nil {
			for _, comment := range comments[latIdx] {
				fmt.Fprintln(writer, comment)
			}
		}
		var max int
		for k, _ := range lattice {
			if k > max {
				max = k
			}
		}
		lastToken = 0
		for i := 0; i <= max; i++ {
			if row, exists := lattice[i]; exists {
				for _, edge := range row {
					if edge.Token > lastToken {
						// run forward and find the bottom and top of the current token
						bottom, top := edge.Start, edge.End
					outerLoop:
						for j := i; j <= max; j++ {
							if otherRow, otherExists := lattice[j]; otherExists {
								for _, otherEdge := range otherRow {
									if otherEdge.Token > edge.Token {
										break outerLoop
									}
									if otherEdge.Start < bottom {
										bottom = otherEdge.Start
									}
									if otherEdge.End > top {
										top = otherEdge.End
									}
								}
							}
						}
						if oovVectors != nil && oovVectors[latIdx][edge.Token-1] == "1" {
							tokenComment = "oov=1"
						} else {
							tokenComment = "_"
						}
						fmt.Fprintf(writer, "%d-%d\t%s\t%s\n", bottom, top, edge.TokenStr, tokenComment)
						lastToken = edge.Token
					}
					writer.Write(append([]byte(edge.UDString()), '\n'))
				}
			}
		}
		writer.Write([]byte{'\n'})
	}
	return nil
}

func UDWriteJSON(writer io.Writer, lattices []Lattice) error {
	for _, lattice := range lattices {
		var (
			max, lastToken int
			jsonEdge       *JSONEdge
			jsonLat        JSONLattice
		)

		for k, _ := range lattice {
			if k > max {
				max = k
			}
		}
		for i := 0; i <= max; i++ {
			if row, exists := lattice[i]; exists {
				if len(row) > 0 && row[0].Token > lastToken {
					lastToken = row[0].Token
					if jsonLat != nil {
						marshalled, err := json.Marshal(jsonLat)
						if err != nil {
							panic(fmt.Sprintf("Failure marshalling %v", err))
						}
						writer.Write(marshalled)
						writer.Write([]byte{'\n'})
					}
					jsonLat = make(JSONLattice)
				}
				for _, edge := range row {
					jsonEdge = &JSONEdge{
						Next:    fmt.Sprint(edge.End),
						Form:    edge.Word,
						UPOSTag: edge.CPosTag,
					}
					if edge.Lemma != "_" {
						jsonEdge.Lemma = edge.Lemma
					}
					if edge.FeatStr != "_" {
						jsonEdge.Feats = edge.FeatStr
					}
					if edge.PosTag != "_" {
						jsonEdge.XPOSTag = edge.PosTag
					}
					startStr := fmt.Sprint(edge.Start)
					if outEdges, edgesExist := jsonLat[startStr]; edgesExist {
						outEdges = append(outEdges, *jsonEdge)
						jsonLat[startStr] = outEdges
					} else {
						newList := make([]JSONEdge, 1, 2)
						newList[0] = *jsonEdge
						jsonLat[startStr] = newList
					}
				}
			}
		}
		if jsonLat != nil {
			marshalled, err := json.Marshal(jsonLat)
			if err != nil {
				panic(fmt.Sprintf("Failure marshalling %v", err))
			}
			writer.Write(marshalled)
			writer.Write([]byte{'\n'})
		}
		writer.Write([]byte{'\n'})
	}
	return nil
}

func WriteStream(writer io.Writer, lattices chan Lattice) error {
	for lattice := range lattices {
		var max int
		for k, _ := range lattice {
			if k > max {
				max = k
			}
		}
		for i := 0; i <= max; i++ {
			if row, exists := lattice[i]; exists {
				for _, edge := range row {
					writer.Write(append([]byte(edge.String()), '\n'))
				}
			}
		}
		writer.Write([]byte{'\n'})
	}
	return nil
}

func Write(writer io.Writer, lattices []Lattice) error {
	for _, lattice := range lattices {
		var max int
		for k, _ := range lattice {
			if k > max {
				max = k
			}
		}
		for i := 0; i <= max; i++ {
			if row, exists := lattice[i]; exists {
				for _, edge := range row {
					writer.Write(append([]byte(edge.String()), '\n'))
				}
			}
		}
		writer.Write([]byte{'\n'})
	}
	return nil
}

func ReadFile(filename string, limit int) ([]Lattice, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return Read(file, limit)
}

func StreamFile(filename string, limit int) (chan Lattice, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return ReadStream(file, limit), nil
}

func ReadULFile(filename string, limit int) ([]Lattice, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return ULRead(file, limit)
}
func StreamULFile(filename string, limit int) (chan Lattice, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}

	return ULReadStream(file, limit), nil
}

func ReadUDFile(filename string, limit int) ([]Lattice, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return UDRead(file, limit)
}

func WriteStreamToFile(filename string, sents chan Lattice) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	WriteStream(file, sents)
	return nil
}

func WriteFile(filename string, sents []Lattice) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	Write(file, sents)
	return nil
}

func WriteUDFile(filename string, sents []Lattice, comments [][]string, oov interface{}) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	UDWrite(file, sents, comments, oov.([]nlp.BasicSentence))
	return nil
}

func WriteUDJSONFile(filename string, sents []Lattice) error {
	file, err := os.Create(filename)
	defer file.Close()
	if err != nil {
		return err
	}
	UDWriteJSON(file, sents)
	return nil
}

func Lattice2Sentence(lattice Lattice, eWord, ePOS, eWPOS, eMorphFeat, eMHost, eMSuffix *util.EnumSet) nlp.LatticeSentence {
	tokenSizes := make(map[int]int)
	var (
		maxToken int = 0
		skipEdge bool
		tokens   []string
	)
	for _, edges := range lattice {
		for _, edge := range edges {
			curval, _ := tokenSizes[edge.Token]
			tokenSizes[edge.Token] = curval + 1
			if edge.Token > maxToken {
				maxToken = edge.Token
			}
		}
	}
	sent := make(nlp.LatticeSentence, maxToken)
	tokens = make([]string, maxToken)
	// sent[0] = nlp.NewRootLattice()
	for sourceId := 0; sourceId <= lattice.MaxKey(); sourceId++ {
		edges, exists := lattice[sourceId]

		// a future sourceid may have been removed during processing
		// skip over these
		if !exists {
			// log.Println("Skipping sourceId", sourceId)
			continue
		}
		// log.Println("At sourceId", sourceId)
		for i, edge := range edges {
			// a negative start indicates a "deleted" edge, it should skipped over
			if edge.Start < 0 {
				continue
			}
			tokens[edge.Token-1] = edge.TokenStr
			skipEdge = false

			lat := &sent[edge.Token-1]

			// FIX Fusional 'H' in Modern Hebrew Corpus
			if _FIX_FUSIONAL_H {
				if _, prefixExists := _FUSIONAL_PREFIXES[edge.Word]; prefixExists {
					// log.Println("Fusional H: Found prefix", edge.Word)
					for _, otherEdge := range edges {
						if otherEdge.Id == edge.Id {
							continue
						}
						_, otherPrefixExists := _FUSIONAL_PREFIXES[otherEdge.Word]
						if otherPrefixExists && edge.Word == otherEdge.Word && edge.PosTag == otherEdge.PosTag {
							if fusionalEdges, nextExists := lattice[otherEdge.End]; nextExists && len(fusionalEdges) == 1 &&
								fusionalEdges[0].Word == "H" && fusionalEdges[0].End == edge.End {
								// log.Println("Fusional H: Found fusionalEdges", fusionalEdges)
								skipEdge = true
								outgoingEdges, outExists := lattice[edge.End]
								if !outExists {
									continue
								}
								for _, outEdge := range outgoingEdges {
									// log.Println("Fusional H: At outgoing edge", outEdge)
									newEdge := outEdge.Copy()
									newEdge.Start = otherEdge.End
									lattice[otherEdge.End] = append(lattice[otherEdge.End], *newEdge)
								}
							}
						}
					}
					if skipEdge {
						// log.Println("skipedge: Setting start to 0")
						edge.Start = 0
						edges[i] = edge
						continue
					}
				}
			}

			// FIX ECM* PRP cases in Modern Hebrew SPMRL lattice corpus to match
			// gold lattice
			if _FIX_ECMx {
				if _, exists := ECMx_INSTANCES[edge.Word]; edge.PosTag == "PRP" && exists {
					nextMorphs := lattice[edge.End]
					if len(nextMorphs) != 1 {
						log.Println("Warning: ", fmt.Sprintf("ECMx has %v outgoing edges, expected 1", len(nextMorphs)))
						// panic(fmt.Sprintf("ECMx has %v outgoing edges, expected 1", len(nextMorphs)))
					}
					nextMorph := nextMorphs[0]
					if nextMorph.PosTag == PRONOMINAL_CLITIC_POS {
						// log.Println("Fixing ECMx", edge.Word)
						edge.End = nextMorph.End
						edge.FeatStr = nextMorph.FeatStr
						edge.Feats = nextMorph.Feats
						nextMorph.Start = -1 // -1 deletes edge, will be skipped
					}
				}
			}

			// FIX Pronominal Suffix Clitic in Modern Hebrew Corpus
			if _FIX_PRONOMINAL_CLITIC {
				for _, testEdge := range lattice[edge.End] {
					if edge.Word != "H" && testEdge.PosTag == PRONOMINAL_CLITIC_POS {
						// edge is a morpheme in the lattice that leads to a
						// prononimal clitic as a suffix
						// we edit the (c)postag of this preposition to
						// differentiate it from a preposition without a
						// pronominal suffix
						// log.Println("Editing preposition of pronominal clitic", edge, "for", testEdge)
						// edge.PosTag = edge.PosTag + "_S"
						// edge.CPosTag = edge.CPosTag + "_S"
						// log.Println("New edge", edge)
					}
				}
			}

			// log.Println("\t", "At morpheme (s,e) of token", edge.Word, edge.Start, edge.End, edge.Token)
			if lat.Morphemes == nil {
				// log.Println("\t", "Initialize new lattice")
				// initialize new lattice
				lat.Morphemes = make(nlp.Morphemes, 0, tokenSizes[edge.Token])
				// lat.Next = make(map[int][]int) // make next inside Lattice
				lat.BottomId = edge.Start
				lat.TopId = edge.End
			} else {
				// log.Println("\t", "Update existing lattice")
				if edge.Start < lat.BottomId {
					lat.BottomId = edge.Start
				}
				if edge.End > lat.TopId {
					lat.TopId = edge.End
				}
			}
			// if nextList, exists := lat.Next[sourceId]; exists {
			// 	// log.Println("\t", "Append to next sourceId", sourceId)
			// 	lat.Next[sourceId] = append(nextList, len(lat.Morphemes))
			// 	// recheck, _ := lat.Next[sourceId]
			// 	// log.Println("\t", "Post append:", recheck)
			// } else {
			// 	// log.Println("\t", "Create new next for sourceId", sourceId)
			// 	lat.Next[sourceId] = make([]int, 1)
			// 	lat.Next[sourceId][0] = len(lat.Morphemes)
			// }
			newMorpheme := &nlp.EMorpheme{
				Morpheme: nlp.Morpheme{
					graph.BasicDirectedEdge{len(lat.Morphemes), edge.Start, edge.End},
					edge.Word,
					edge.Lemma,
					edge.CPosTag,
					edge.CPosTag,
					edge.Feats,
					edge.Token,
					edge.FeatStr,
				},
			}
			switch WORD_TYPE {
			case "form":
				newMorpheme.EForm, _ = eWord.Add(edge.Word)
				newMorpheme.EFCPOS, _ = eWPOS.Add([2]string{edge.Word, edge.CPosTag})
			case "lemma":
				newMorpheme.EForm, _ = eWord.Add(edge.Lemma)
				newMorpheme.EFCPOS, _ = eWPOS.Add([2]string{edge.Lemma, edge.CPosTag})
			case "lemma+f":
				if edge.Lemma != "" {
					newMorpheme.EForm, _ = eWord.Add(edge.Lemma)
					newMorpheme.EFCPOS, _ = eWPOS.Add([2]string{edge.Lemma, edge.CPosTag})
				} else {
					newMorpheme.EForm, _ = eWord.Add(edge.Word)
					newMorpheme.EFCPOS, _ = eWPOS.Add([2]string{edge.Word, edge.CPosTag})
				}
			default:
				panic(fmt.Sprintf("Unknown WORD_TYPE %s", WORD_TYPE))
			}
			newMorpheme.EPOS, _ = ePOS.Add(edge.CPosTag)
			newMorpheme.EFeatures, _ = eMorphFeat.Add(edge.FeatStr)
			newMorpheme.EMHost, _ = eMHost.Add(edge.Feats.MorphHost())
			newMorpheme.EMSuffix, _ = eMSuffix.Add(edge.Feats.MorphSuffix())
			// log.Println("\t", "Adding morpheme", newMorpheme, newMorpheme.ID(), newMorpheme.From(), newMorpheme.To())
			if newMorpheme.From() == newMorpheme.To() {
				panic("crap adding " + fmt.Sprintf("%v", newMorpheme))
			}
			lat.Morphemes = append(lat.Morphemes, newMorpheme)
		}
	}
	for i, lat := range sent {
		// log.Println("At lat", i)
		lat.SortMorphemes()
		lat.GenSpellouts()
		lat.Token = nlp.Token(tokens[i])
		sent[i] = lat
	}
	return sent
}

func Lattice2SentenceCorpus(corpus Lattices, eWord, ePOS, eWPOS, eMorphFeat, eMHost, eMSuffix *util.EnumSet) []interface{} {
	graphCorpus := make([]interface{}, len(corpus))
	prefix := log.Prefix()
	for i, sent := range corpus {
		// log.Println("At sent", i)
		log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
		graphCorpus[i] = Lattice2Sentence(sent, eWord, ePOS, eWPOS, eMorphFeat, eMHost, eMSuffix)
	}
	log.SetPrefix(prefix)
	return graphCorpus
}

func Lattice2SentenceStream(corpus chan Lattice, eWord, ePOS, eWPOS, eMorphFeat, eMHost, eMSuffix *util.EnumSet) chan interface{} {
	out := make(chan interface{}, 2)
	go func(graphCorpus chan interface{}) {
		// prefix := log.Prefix()
		var i int
		for sent := range corpus {
			// log.Println("Conversion at sent", i)
			// log.SetPrefix(fmt.Sprintf("%v graph# %v ", prefix, i))
			graphCorpus <- Lattice2Sentence(sent, eWord, ePOS, eWPOS, eMorphFeat, eMHost, eMSuffix)
			// log.SetPrefix(prefix)
			i++
		}
		close(graphCorpus)
	}(out)
	return out
}

func Sentence2Lattice(lattice nlp.LatticeSentence, xliter8or xliter8.Interface) Lattice {
	retLat := make(Lattice)
	for _, sentlat := range lattice {
		for _, m := range sentlat.Morphemes {
			outForm := m.Form
			outLemma := m.Lemma
			//if IGNORE_LEMMA {
			//	outLemma = ""
			//}
			if xliter8or != nil {
				outForm = xliter8or.To(outForm)
				outLemma = xliter8or.To(outLemma)
			}
			e := Edge{
				m.From(),
				m.To(),
				outForm,
				outLemma,
				m.CPOS,
				m.POS,
				nil,
				m.FeatureStr,
				// should be m.TokenID+1
				m.TokenID,
				m.ID(),
				string(sentlat.Token),
			}
			if len(m.FeatureStr) == 0 {
				e.FeatStr = "_"
			}
			if curOut, exists := retLat[m.From()]; exists {
				curOut = append(curOut, e)
				sort.Sort(EdgeSlice(curOut))
				retLat[m.From()] = curOut
			} else {
				retLat[m.From()] = []Edge{e}
			}
		}
	}
	return retLat
}

func Sentence2LatticeStream(corpus chan nlp.LatticeSentence, xliter8or xliter8.Interface) chan Lattice {
	latticeCorpus := make(chan Lattice, 2)
	go func() {
		for sent := range corpus {
			latticeCorpus <- Sentence2Lattice(sent, xliter8or)
		}
		close(latticeCorpus)
	}()
	return latticeCorpus
}

func Sentence2LatticeCorpus(corpus []nlp.LatticeSentence, xliter8or xliter8.Interface) []Lattice {
	latticeCorpus := make([]Lattice, len(corpus))
	for i, sent := range corpus {
		latticeCorpus[i] = Sentence2Lattice(sent, xliter8or)
	}
	return latticeCorpus
}
