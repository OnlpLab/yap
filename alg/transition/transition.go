package transition

import (
	"bytes"
	"fmt"
	// "log"
	"strings"
	"text/tabwriter"
	. "github.com/OnlpLab/yap/alg/featurevector"
	"github.com/OnlpLab/yap/util"
)

var IDLE = &TypedTransition{'I', 0}

type FeaturesList struct {
	Features   []Feature
	Transition Transition
	Previous   *FeaturesList
}

func (l *FeaturesList) String() string {
	var (
		retval []string      = make([]string, 0, 100)
		cur    *FeaturesList = l
	)
	for cur != nil {
		retval = append(retval, fmt.Sprintf("%v", cur.Transition))
		for _, val := range cur.Features {
			retval = append(retval, fmt.Sprintf("\t%v", val))
		}
		cur = cur.Previous
	}
	return strings.Join(retval, "\n")
}

type Transition interface {
	Type() byte
	Value() int
	Equal(other Transition) bool
}

type ConstTransition int

func (b ConstTransition) Type() byte {
	return '-'
}

func (b ConstTransition) Value() int {
	return int(b)
}

func (b ConstTransition) Equal(other Transition) bool {
	return b.Type() == other.Type() && b.Value() == other.Value()
}

type TypedTransition struct {
	T byte
	V int
}

func (t *TypedTransition) Type() byte {
	return t.T
}

func (t *TypedTransition) Value() int {
	return t.V
}

func (t *TypedTransition) Equal(other Transition) bool {
	// log.Println("Comparing transitions", t, "=", other, "? is ", t.Type() == other.Type() && t.Value() == other.Value())
	return t.Type() == other.Type() && t.Value() == other.Value()
}

var _ Transition = ConstTransition(0)
var _ Transition = &TypedTransition{}

type Configuration interface {
	Init(interface{})
	Terminal() bool

	Copy() Configuration
	CopyTo(Configuration)
	Clear()

	Len() int
	Previous() Configuration
	SetPrevious(Configuration)
	GetSequence() ConfigurationSequence
	SetLastTransition(Transition)
	GetLastTransition() Transition
	String() string
	Equal(otherEq util.Equaler) bool

	Address(location []byte, offset int) (nodeID int, exists bool, isGenerator bool)
	GenerateAddresses(nodeID int, location []byte) (nodeIDs []int)
	Attribute(source byte, nodeID int, attribute []byte, transitions []int) (attributeValue interface{}, exists bool, isGenerator bool)

	Assignment() uint16

	State() byte
}

type ConfigurationSequence []Configuration

type TransitionSystem interface {
	Transition(from Configuration, transition Transition) Configuration

	TransitionTypes() []string

	YieldTransitions(conf Configuration) (transType byte, transitions chan int)
	GetTransitions(conf Configuration) (transType byte, transitions []int)

	Oracle() Oracle
	AddDefaultOracle()

	Name() string
}

type Decision interface {
	Transition(Configuration) Transition
}

type Oracle interface {
	Decision
	SetGold(interface{})
	Name() string
}

func (seq ConfigurationSequence) String() string {
	var buf bytes.Buffer
	w := new(tabwriter.Writer)
	w.Init(&buf, 0, 8, 0, '\t', 0)
	seqLength := len(seq)
	for i, _ := range seq {
		conf := seq[seqLength-i-1]
		asString := conf.String()
		asBytes := []byte(asString)
		w.Write(asBytes)
		if i < seqLength-1 {
			w.Write([]byte{'\n', '\t', '\t', '\t'})
		}
	}
	w.Flush()
	return buf.String()
}

func (seq ConfigurationSequence) SharedTransitions(other ConfigurationSequence) int {
	lenOther := len(other)
	lenSeq := len(seq)
	sharedSeq := 0
	for i, _ := range seq {
		if len(other) <= i {
			break
		}
		if !other[lenOther-i-1].GetLastTransition().Equal(seq[lenSeq-i-1].GetLastTransition()) {
			break
		}
		sharedSeq++
	}
	return sharedSeq
}

func (seq ConfigurationSequence) Equal(otherEq util.Equaler) bool {
	other, ok := otherEq.(ConfigurationSequence)
	if !ok {
		panic("Can't equate sequence to unknown type")
	}
	for i, val := range seq {
		if !other[i].Equal(val) {
			return false
		}
	}
	return true
}
