package perceptron

import (
	. "github.com/OnlpLab/yap/alg/featurevector"
	"github.com/OnlpLab/yap/util"
)

type Model interface {
	// util.Persist
	Score(features interface{}) int64
	Add(features interface{}) Model
	Subtract(features interface{}) Model
	AddSubtract(goldFeatures, decodedFeatures interface{}, amount int64)
	ScalarDivide(int64)
	Copy() Model
	AddModel(Model)
	New() Model
}

type Instance interface {
	util.Equaler
}

type DecodedInstance interface {
	Instance
	Instance() Instance
	Decoded() interface{}
}

type Decoded struct {
	InstanceVal Instance
	DecodedVal  util.Equaler
}

var _ DecodedInstance = &Decoded{}

func (d *Decoded) Decoded() interface{} {
	return d.DecodedVal
}

func (d *Decoded) Instance() Instance {
	return d.InstanceVal
}

func (d *Decoded) Equal(otherEq util.Equaler) bool {
	if otherEq == nil {
		return false
	}
	other := otherEq.(*Decoded)
	instanceEq := d.InstanceVal.Equal(other.InstanceVal)
	decodedEq := d.DecodedVal.Equal(other.DecodedVal)
	return instanceEq && decodedEq
}

type FeatureExtractor interface {
	Features(instance Instance, flag bool, transType byte, trans_values []int) []Feature
	EstimatedNumberOfFeatures() int
	SetLog(bool)
}

type EmptyFeatureExtractor struct {
}

func (e *EmptyFeatureExtractor) Features(i Instance, flag bool, ttype byte, arr []int) []Feature {
	return []Feature{}
}

func (e *EmptyFeatureExtractor) EstimatedNumberOfFeatures() int {
	return 0
}

var _ FeatureExtractor = &EmptyFeatureExtractor{}

func (e *EmptyFeatureExtractor) SetLog(flag bool) {
}

type InstanceDecoder interface {
	Decode(i Instance, m Model) (DecodedInstance, interface{})
	DecodeGold(i DecodedInstance, m Model) (DecodedInstance, interface{})
}

type EarlyUpdateInstanceDecoder interface {
	DecodeEarlyUpdate(i DecodedInstance, m Model) (decoded DecodedInstance, decodedFeatures, goldFeatures interface{}, earlyUpdatedAt, goldSize int, decodeScore float64)
}

type SupervisedTrainer interface {
	Train(instances []DecodedInstance)
}

// unused, here for completeness
type UnsupervisedTrainer interface {
	Train(instances []Instance)
}
