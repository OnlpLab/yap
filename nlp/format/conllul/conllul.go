package conllul

import (
	"errors"
	"fmt"
	"strings"
	"log"
	"bufio"
	"bytes"
	"io"
	"os"
	"github.com/OnlpLab/yap/nlp/format/lattice"
)

type ConlluEdge struct {
	Id int
	Start    int
	End      int
	Lemma    string
	Word     string
	UPosTag  string
	XPosTag   string
	Feats    lattice.Features
	FeatStr string
	TokenId int
}

type ConlluLattice struct {
	Edges map[int][]ConlluEdge
	Tokens   []string
	Comments []string
}

func NewConlluLattice() ConlluLattice {
	return ConlluLattice{
		Edges:     make(map[int][]ConlluEdge),
		Tokens:   []string{},
		Comments: make([]string, 0, 2),
	}
}

func ParseTokenRow(record []string) (string, int, int, error) {
	token := lattice.ParseString(record[1])
	if token == "" {
		return token, -1, -1, errors.New("Empty TOKEN field for token row")
	}
	ids := strings.Split(record[0], "-")
	if len(ids) != 2 {
		return token, -1, -1, errors.New(fmt.Sprintf("Error parsing ID span field (%s): wrong format for ID span for token row - needs <num>-<num>", record[0]))
	}
	id1, err := lattice.ParseInt(ids[0])
	if err != nil {
		return token, -1, -1, errors.New(fmt.Sprintf("Error parsing ID span field (%s): %s for token row", record[0], err.Error()))
	}
	id2, err := lattice.ParseInt(ids[1])
	if err != nil {
		return token, id1, -1, errors.New(fmt.Sprintf("Error parsing ID span field (%s): %s for token row", record[0], err.Error()))
	}
	if !(id2-id1 > 0) {
		return token, id1, id2, errors.New(fmt.Sprintf("Error parsing ID span field (%s): wrong format for ID span for token row - needs second num (%d) - first num (%d) > 0", record[0], id2, id1))
	}

	return token, id1,  id2, nil
}

func ParseEdge(record []string) (*ConlluEdge, error) {
	edge := &ConlluEdge{}
	var err error
	edge.Start, err = lattice.ParseInt(record[0])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing START field (%s): %s", record[0], err.Error()))
	}
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing END field (%s): %s", record[1], err.Error()))
	}
	edge.End, err = lattice.ParseInt(record[1])
	edge.Word = lattice.ParseString(record[2])
	edge.Lemma = lattice.ParseString(record[3])
	edge.UPosTag = lattice.ParseString(record[4])
	if edge.UPosTag == "" {
		return nil, errors.New("Empty UPOSTAG field")
	}
	edge.XPosTag = lattice.ParseString(record[5])
	edge.Feats, err = lattice.ParseFeatures(record[6])
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error parsing FEATS field (%s): %s", record[6], err.Error()))
	}
	edge.FeatStr = lattice.ParseString(record[6])
	return edge, nil
}

func Read(r io.Reader, limit int) ([]ConlluLattice, error) {
	var lattices []ConlluLattice
	bufReader := bufio.NewReader(r)

	var (
		currentLatt ConlluLattice = NewConlluLattice()
		currentEdge int
		lineCount int
		skip bool
	)
	for curLineBuf, isPrefix, err := bufReader.ReadLine(); err == nil; curLineBuf, isPrefix, err = bufReader.ReadLine() {
		if isPrefix {
			panic("Buffer not large enough, fix me :(")
		}
		lineCount++
		if len(curLineBuf) == 0 {
			lattices = append(lattices, currentLatt)
			if limit > 0 && len(lattices) >= limit {
				break
			}
			currentLatt = NewConlluLattice()
			currentEdge = 0
			skip = false
			continue
		} else if skip {
			continue;
		}
		buf := bytes.NewBuffer(curLineBuf)
		line := buf.String()
		record := strings.Split(buf.String(), "\t")
		if record[0][0] == '#' {
			currentLatt.Comments = append(currentLatt.Comments, line)
			continue
		}
		if strings.Contains(record[0], "-") {
			token, _, _, err := ParseTokenRow(record)
			if err != nil {
				log.Println(errors.New(fmt.Sprintf("Error processing lattice #%d edge %d at line %d: %s", len(lattices), currentEdge, line, err.Error())))
				currentLatt = NewConlluLattice()
				currentEdge = 0
				skip = true
				continue
			}
			currentLatt.Tokens = append(currentLatt.Tokens, token)
		} else {
			currentEdge++
			edge, err := ParseEdge(record)
			if err != nil {
				log.Println(errors.New(fmt.Sprintf("Error processing lattice #%d edge %d at line %d: %s", len(lattices), currentEdge, line, err.Error())))
				currentLatt = NewConlluLattice()
				currentEdge = 0
				skip = true
				continue
			}
			edge.Id = currentEdge
			edge.TokenId = len(currentLatt.Tokens)
			edges, exists := currentLatt.Edges[edge.Start]
			if exists {
				currentLatt.Edges[edge.Start] = append(edges, *edge)
			} else {
				currentLatt.Edges[edge.Start] = []ConlluEdge{*edge}
			}
		}
	}
	return lattices, nil
}

func ReadFile(filename string, limit int) ([]ConlluLattice, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return Read(file, limit)
}
