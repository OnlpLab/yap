package model

// import (
// 	. "github.com/OnlpLab/yap/alg/featurevector"
// 	"github.com/OnlpLab/yap/alg/perceptron"
// 	"github.com/OnlpLab/yap/alg/transition"
// )

// type Trivial struct {
// 	Vec Sparse
// }

// var _ perceptron.Model = &Trivial{}
// var _ Interface = &Trivial{}

// func (t *Trivial) Score(features interface{}) int64 {
// 	var (
// 		retval int64
// 		feat   interface{}
// 	)
// 	featuresList := features.(*transition.FeaturesList)
// 	for featuresList != nil {
// 		for i, feature := range featuresList.Features {
// 			feat = MakeFeature(int(featuresList.Transition), i, feature)
// 			retval += t.Vec[feat]
// 		}
// 		featuresList = featuresList.Previous
// 	}
// 	return retval
// }

// func (t *Trivial) Add(features interface{}) perceptron.Model {
// 	var (
// 		curval int64
// 		feat   interface{}
// 	)
// 	featuresList := features.(*transition.FeaturesList)
// 	for featuresList != nil {
// 		for i, feature := range featuresList.Features {
// 			feat = MakeFeature(int(featuresList.Transition), i, feature)
// 			curval, _ = t.Vec[feat]
// 			t.Vec[feat] = curval + 1
// 		}
// 		featuresList = featuresList.Previous
// 	}
// 	return t
// }

// func (t *Trivial) Subtract(features interface{}) perceptron.Model {
// 	var (
// 		curval int64
// 		feat   interface{}
// 	)
// 	featuresList := features.(*transition.FeaturesList)
// 	for featuresList != nil {
// 		for i, feature := range featuresList.Features {
// 			feat = MakeFeature(int(featuresList.Transition), i, feature)
// 			curval, _ = t.Vec[feat]
// 			t.Vec[feat] = curval - 1
// 		}
// 		featuresList = featuresList.Previous
// 	}
// 	return t
// }

// func (t *Trivial) ScalarDivide(val int64) {
// 	t.Vec.UpdateScalarDivide(val)
// }

// func (t *Trivial) Copy() perceptron.Model {
// 	return &Trivial{t.Vec.Copy()}
// }

// func (t *Trivial) New() perceptron.Model {
// 	return NewTrivial()
// }

// func (t *Trivial) AddModel(m perceptron.Model) {
// 	other, ok := m.(*Trivial)
// 	if !ok {
// 		panic("Can't add perceptron model not of the same type")
// 	}
// 	t.Vec.Add(other.Vec)
// }

// func (t *Trivial) TransitionScore(transition transition.Transition, features []Feature) int64 {
// 	var (
// 		retval int64
// 	)

// 	for i, feat := range features {
// 		retval += t.Vec[MakeFeature(int(transition), i, feat)]
// 	}
// 	return retval
// }

// func NewTrivial() *Trivial {
// 	return &Trivial{NewSparse()}
// }
