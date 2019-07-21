package model

// import (
// 	. "github.com/OnlpLab/yap/alg/featurevector"
// 	"github.com/OnlpLab/yap/alg/perceptron"
// 	"github.com/OnlpLab/yap/alg/transition"
// )

// type MatrixSparse struct {
// 	Mat                   [][]Sparse
// 	Transitions, Features int
// }

// var _ perceptron.Model = &MatrixSparse{}
// var _ Interface = &MatrixSparse{}

// func (t *MatrixSparse) Score(features interface{}) int64 {
// 	var (
// 		retval   int64
// 		intTrans int
// 	)
// 	featuresList := features.(*transition.FeaturesList)
// 	for featuresList != nil {
// 		intTrans = int(featuresList.Transition)
// 		if intTrans < t.Transitions {
// 			for i, feature := range featuresList.Features {
// 				retval += t.Mat[int(featuresList.Transition)][i][feature]
// 			}
// 		}
// 		featuresList = featuresList.Previous
// 	}
// 	return retval
// }

// func (t *MatrixSparse) Add(features interface{}) perceptron.Model {
// 	var (
// 		curval   int64
// 		intTrans int
// 	)
// 	featuresList := features.(*transition.FeaturesList)
// 	for featuresList != nil {
// 		intTrans = int(featuresList.Transition)
// 		if intTrans >= t.Transitions {
// 			t.ExtendTransitions(intTrans)
// 		}
// 		for i, feature := range featuresList.Features {
// 			curval, _ = t.Mat[int(featuresList.Transition)][i][feature]
// 			t.Mat[int(featuresList.Transition)][i][feature] = curval + 1
// 		}
// 		featuresList = featuresList.Previous
// 	}
// 	return t
// }

// func (t *MatrixSparse) Subtract(features interface{}) perceptron.Model {
// 	var (
// 		curval   int64
// 		intTrans int
// 	)
// 	featuresList := features.(*transition.FeaturesList)
// 	for featuresList != nil {
// 		intTrans = int(featuresList.Transition)
// 		if intTrans >= t.Transitions {
// 			t.ExtendTransitions(intTrans)
// 		}
// 		for i, feature := range featuresList.Features {
// 			curval, _ = t.Mat[int(featuresList.Transition)][i][feature]
// 			t.Mat[int(featuresList.Transition)][i][feature] = curval - 1
// 		}
// 		featuresList = featuresList.Previous
// 	}
// 	return t
// }

// func (t *MatrixSparse) ScalarDivide(val int64) {
// 	for i, _ := range t.Mat {
// 		for j, _ := range t.Mat[i] {
// 			t.Mat[i][j].UpdateScalarDivide(val)
// 		}
// 	}
// }

// func (t *MatrixSparse) Copy() perceptron.Model {
// 	newMS := NewMatrixSparse(t.Transitions, t.Features)
// 	for i, sparseArray := range t.Mat {
// 		for j, sparse := range sparseArray {
// 			newMS.Mat[i][j] = sparse.Copy()
// 		}
// 	}
// 	return newMS
// }

// func (t *MatrixSparse) New() perceptron.Model {
// 	return NewMatrixSparse(t.Transitions, t.Features)
// }

// func (t *MatrixSparse) AddModel(m perceptron.Model) {
// 	other, ok := m.(*MatrixSparse)
// 	if !ok {
// 		panic("Can't add perceptron model not of the same type")
// 	}
// 	for i, _ := range t.Mat {
// 		for j, _ := range t.Mat[i] {
// 			t.Mat[i][j].Add(other.Mat[i][j])
// 		}
// 	}
// }

// func (t *MatrixSparse) TransitionScore(transition transition.Transition, features []Feature) int64 {
// 	var (
// 		retval   int64
// 		intTrans int = int(transition)
// 	)
// 	if intTrans >= t.Transitions {
// 		return 0.0
// 	}
// 	if intTrans < 0 {
// 		panic("Got negative transition index")
// 	}
// 	featuresArray := t.Mat[intTrans]

// 	for i, feat := range features {
// 		retval += featuresArray[i][feat]
// 	}
// 	return retval
// }

// func (t *MatrixSparse) ExtendTransitions(extendTo int) {
// 	newTransitions := extendTo - t.Transitions + 1
// 	ExtraMat1D := make([]Sparse, newTransitions*t.Features)
// 	for i := range ExtraMat1D {
// 		ExtraMat1D[i] = make(Sparse)
// 	}
// 	for i := 0; i < newTransitions; i++ {
// 		t.Mat, ExtraMat1D = append(t.Mat, ExtraMat1D[:t.Features]), ExtraMat1D[t.Features:]
// 	}
// 	t.Transitions = extendTo + 1
// }

// // // Allocate the top-level slice, the same as before.
// // picture := make([][]uint8, YSize) // One row per unit of y.
// // // Allocate one large slice to hold all the pixels.
// // pixels := make([]uint8, XSize*YSize) // Has type []uint8 even though picture is [][]uint8.
// // // Loop over the rows, slicing each row from the front of the remaining pixels slice.
// // for i := range picture {
// // 	picture[i], pixels = pixels[:XSize], pixels[XSize:]
// // }
// func NewMatrixSparse(transitions, features int) *MatrixSparse {
// 	var (
// 		Mat1D []Sparse   = make([]Sparse, transitions*features)
// 		Mat2D [][]Sparse = make([][]Sparse, 0, transitions)
// 	)
// 	for i, _ := range Mat1D {
// 		Mat1D[i] = make(Sparse)
// 	}
// 	for i := 0; i < transitions; i++ {
// 		Mat2D, Mat1D = append(Mat2D, Mat1D[:features]), Mat1D[features:]
// 	}
// 	return &MatrixSparse{Mat2D, transitions, features}
// }
