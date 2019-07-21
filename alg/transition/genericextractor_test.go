package transition

// import (
// 	"github.com/OnlpLab/yap/algmodel/perceptron"
// 	"strings"
// 	"testing"
// )

// func TestExtractor(t *testing.T) {
// 	x := new(GenericExtractor)
// 	// verify load
// 	for _, feature := range TEST_FEATURES {
// 		if err := x.LoadFeature(feature[0]); err != nil {
// 			t.Error("Failed to load feature " + feature[0] + ": " + err.Error())
// 			t.FailNow()
// 		}
// 	}

// 	// get test configuration
// 	conf := GetTestConfiguration()

// 	// extract features as map
// 	features := x.Features(perceptron.Instance(conf))
// 	extracted := make(map[string]string, len(features))
// 	for _, feature := range features {
// 		parsed := strings.Split(string(feature), TEMPLATE_PREFIX)
// 		extracted[parsed[0]] = parsed[1]
// 	}

// 	// test features
// 	for _, feature := range TEST_FEATURES {
// 		if result, exists := extracted[feature[0]]; !exists || result != feature[1] {
// 			if !exists {
// 				t.Error(feature[0], "not found")
// 			} else {
// 				t.Error("Failed to extract", feature[0], "got", extracted[feature[0]], "expected", feature[1])
// 			}
// 		}
// 	}
// }

// func TestExtractorLoad(t *testing.T) {
// 	features := make([]string, len(TEST_FEATURES)+1)
// 	features[0] = "# Comment"
// 	for i, feat := range TEST_FEATURES {
// 		features[i+1] = feat[0]
// 	}
// 	allFeats := strings.Join(features, "\n")
// 	extractor := new(GenericExtractor)

// 	// this should pass
// 	extractor.LoadFeatures(strings.NewReader(allFeats))
// }

// // Transition-based Dependency Parsing with Rich Non-local Features
// // http://www.sutd.edu.sg/cmsresource/faculty/yuezhang/acl11j.pdf
// var TEST_FEATURES [][2]string = [][2]string{
// 	// BASELINE
// 	// from single words
// 	{"S0|w|p", "effect|NN"},
// 	{"S0|w", "effect"},
// 	{"N0|w|p", ".|yyDOT"},

// 	// from word pairs
// 	{"S0|w|p+N0|w|p", "effect|NN+.|yyDOT"},

// 	// from three words
// 	{"S0h|p+S0|p+N0|p", "VB+NN+yyDOT"},
// 	{"S0|p+S0l|p+N0|p", "NN+ADJ+yyDOT"},
// 	{"S0|p+S0r|p+N0|p", "NN+NN+yyDOT"},

// 	// RICH
// 	// distance
// 	{"N0|p|d", "yyDOT|4"},
// 	{"S0|w+N0|w|d", "effect+.|4"},

// 	// valency
// 	{"S0|w|vr", "effect|1"},
// 	{"S0|p|vl", "NN|1"},
// 	{"N0|w|vl", ".|0"},

// 	// unigrams
// 	{"S0h|w", "had"},
// 	{"S0l|l", "ATT"},
// 	{"S0r|w", "on"},

// 	// third order
// 	{"S0h2|w", "ROOT"},
// 	{"S0h2|p", "ROOT"},
// 	{"S0|p+S0h|p+S0h2|p", "NN+VB+ROOT"},

// 	// label set
// 	{"S0|w|sr", "effect|ATT"},
// 	{"S0|p|sl", "NN|ATT"},
// 	{"N0|w|sl", ".|"},

// 	// labels
// 	{"S0|l", "OBJ"}}
