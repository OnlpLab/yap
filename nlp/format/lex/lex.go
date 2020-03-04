package lex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/OnlpLab/yap/alg/graph"
	"github.com/OnlpLab/yap/nlp/parser/xliter8"
	"github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
)

const (
	APPROX_LEX_SIZE         = 100000
	SEPARATOR               = " "
	MSR_SEPARATOR           = ":"
	FEATURE_SEPARATOR       = "-"
	PREFIX_SEPARATOR        = "^"
	PREFIX_MSR_SEPARATOR    = "+"
	FEATURE_PAIR_SEPARATOR  = "|"
	FEATURE_VALUE_SEPARATOR = "="
)

var (
	ADD_NNP_NO_FEATS       = false
	STRIP_ALL_NNP_OF_FEATS = false
	HEBREW_XLITER8         = &xliter8.Hebrew{}
	LOG_FAILURES           = false
	SKIP_BINYAN            = true
	SKIP_POLAR             = true
	SUFFIX_ONLY_CPOS       = map[string]bool{
		"NN":       true,
		"DT":       true,
		"EX":       true,
		"PRP":      true,
		"PRP-REF":  true,
		"PRP-PERS": true,
		"QW":       true,
	}
	MSR_TYPE_FROM_VALUE = map[string]string{
		"1":              "per=1",
		"2":              "per=2",
		"3":              "per=3",
		"A":              "per=A",
		"BEINONI":        "tense=BEINONI",
		"D":              "num=D",
		"DP":             "num=D|num=P",
		"F":              "gen=F",
		"FUTURE":         "tense=FUTURE",
		"IMPERATIVE":     "tense=IMPERATIVE",
		"M":              "gen=M",
		"MF":             "gen=F|gen=M",
		"SP":             "num=S|num=P",
		"NEGATIVE":       "polar=neg",
		"P":              "num=P",
		"PAST":           "tense=PAST",
		"POSITIVE":       "polar=pos",
		"S":              "num=S",
		"PERS":           "type=PERS",
		"DEM":            "type=DEM",
		"REF":            "type=REF",
		"IMP":            "type=IMP",
		"INT":            "type=INT", // is not in bgulex
		"HIFIL":          "binyan=HIFIL",
		"PAAL":           "binyan=PAAL",
		"NIFAL":          "binyan=NIFAL",
		"HITPAEL":        "binyan=HITPAEL",
		"PIEL":           "binyan=PIEL",
		"PUAL":           "binyan=PUAL",
		"HUFAL":          "binyan=HUFAL",
		"TOINFINITIVE":   "type=TOINFINITIVE",
		"BAREINFINITIVE": "type=BAREINFINITIVE", // is not in bgulex
		"COORD":          "type=COORD",
		"SUB":            "type=SUB",
		"REL":            "type=REL",     // only prefix
		"SUBCONJ":        "type=SUBCONJ", // only prefix
	}
	SKIP_ALL_TYPE bool = true
	SKIP_TYPES         = map[string]bool{
		"COORD": true,
	}
	PP_FROM_MSR      map[string][]string
	PP_FROM_MSR_DATA = []string{
		// Based on Tsarfaty 2010 Relational-Realizational Parsing, p. 86
		"gen=MF|num=P|per=1:אנחנו",
		"gen=MF|num=S|per=1:אני",
		"gen=F|num=S|per=2:את",
		"gen=M|num=S|per=2:אתה",
		"gen=M|num=P|per=2:אתם",
		"gen=F|num=P|per=2:אתן",
		"gen=M|num=S|per=3:הוא",
		"gen=F|num=S|per=3:היא",
		"gen=M|num=P|per=3:הם",
		"gen=F|num=P|per=3:הן",
	}
	PP_BRIDGE = map[string]string{
		"CD":   "של",
		"NN":   "של",
		"VB":   "את",
		"BN":   "את",
		"IN":   "",
		"INTJ": "",
		"RB":   "",
	}
)

func init() {
	PP_FROM_MSR = make(map[string][]string, len(PP_FROM_MSR_DATA))
	for _, mapping := range PP_FROM_MSR_DATA {
		splitMap := strings.Split(mapping, ":")
		splitFeats := strings.Split(splitMap[0], FEATURE_PAIR_SEPARATOR)
		valuesStr := strings.Join(FeatureValues(splitFeats, true), FEATURE_SEPARATOR)
		valuesNoTypeStr := strings.Join(FeatureValues(splitFeats, false), FEATURE_SEPARATOR)
		if val, exists := PP_FROM_MSR[valuesStr]; exists {
			val = append(val, splitMap[1])
			PP_FROM_MSR[valuesStr] = val
		} else {
			PP_FROM_MSR[valuesStr] = []string{splitMap[1]}
		}
		if val, exists := PP_FROM_MSR[valuesNoTypeStr]; exists {
			val = append(val, splitMap[1])
			PP_FROM_MSR[valuesNoTypeStr] = val
		} else {
			PP_FROM_MSR[valuesNoTypeStr] = []string{splitMap[1]}
		}
	}
}

func FeatureValues(pairs []string, withType bool) []string {
	retval := make([]string, 0, len(pairs))
	var split []string
	for _, val := range pairs {
		split = strings.Split(val, FEATURE_VALUE_SEPARATOR)
		if withType || split[0] != "type" {
			retval = append(retval, split[1])
		}
	}
	return retval
}

type AnalyzedToken struct {
	Token     string
	Morphemes []types.BasicMorphemes
}

func (a *AnalyzedToken) NumMorphemes() (num int) {
	for _, m := range a.Morphemes {
		num += len(m)
	}
	return
}

func ParseMSR(msr string, add_suf bool) (string, string, map[string]string, string, error) {
	hostMSR := strings.Split(msr, FEATURE_SEPARATOR)
	sort.Strings(hostMSR[1:])
	featureMap := make(map[string]string, len(hostMSR)-1)
	resultMSR := make([]string, 0, len(hostMSR)-1)
	for _, msrFeatValue := range hostMSR[1:] {
		if lkpStr, exists := MSR_TYPE_FROM_VALUE[msrFeatValue]; exists {
			split := strings.Split(lkpStr, "=")
			if SKIP_BINYAN && len(split) > 0 && split[0] == "binyan" {
				continue
			}
			if SKIP_ALL_TYPE && split[0] == "type" {
				continue
			}
			if SKIP_POLAR && split[0] == "polar" {
				continue
			}
			if _, skipType := SKIP_TYPES[split[1]]; skipType {
				continue
			}
			if add_suf {
				featureSplit := strings.Split(lkpStr, FEATURE_PAIR_SEPARATOR)
				for j, val := range featureSplit {
					featureSplit[j] = "suf_" + val
				}
				lkpStr = strings.Join(featureSplit, FEATURE_PAIR_SEPARATOR)
			}
			resultMSR = append(resultMSR, lkpStr)
			if len(split) == 2 {
				featureMap[split[0]] = split[1]
			} else {
				featureMap[split[0]] = msrFeatValue
			}
		} else {
			if LOG_FAILURES {
				log.Println("Encountered unknown morph feature value", msrFeatValue, "- skipping")
			}
		}
	}
	sort.Strings(resultMSR)
	return hostMSR[0], hostMSR[0], featureMap, strings.Join(resultMSR, FEATURE_PAIR_SEPARATOR), nil
}

func ParseMSRSuffix(hostPOS, msr string) (string, string, map[string]string, string, error) {
	hostMSR := strings.Split(msr, FEATURE_SEPARATOR)
	feats := strings.Join(hostMSR[1:], FEATURE_SEPARATOR)
	var resultMorph string
	// log.Println("Looking for", feats)
	// log.Println("In:")
	// for k, v := range PP_FROM_MSR {
	// 	log.Println("\t", k, ":", v)
	// }
	if suffixes, exists := PP_FROM_MSR[feats]; exists {
		resultMorph = suffixes[0]
	} else {
		resultMorph = "הם"
	}
	sort.Strings(hostMSR[1:])
	featureMap := make(map[string]string, len(hostMSR)-1)
	resultMSR := make([]string, 0, len(hostMSR)-1)
	for _, msrFeatValue := range hostMSR[1:] {
		if lkpStr, exists := MSR_TYPE_FROM_VALUE[msrFeatValue]; exists {
			split := strings.Split(lkpStr, "=")
			if SKIP_BINYAN && len(split) > 0 && split[0] == "binyan" {
				continue
			}
			if SKIP_ALL_TYPE && split[0] == "type" {
				continue
			}
			resultMSR = append(resultMSR, lkpStr)
			if len(split) == 2 {
				featureMap[split[0]] = split[1]
			} else {
				featureMap[split[0]] = msrFeatValue
			}
		} else {
			if LOG_FAILURES {
				log.Println("Encountered unknown morph feature value", msrFeatValue, "- skipping")
			}
		}
	}
	sort.Strings(resultMSR)
	resultMSRStr := strings.Join(resultMSR, FEATURE_PAIR_SEPARATOR)
	var bridge string = ""
	if bridgeVal, exists := PP_BRIDGE[hostPOS]; exists {
		bridge = bridgeVal
	} else {
		if LOG_FAILURES {
			log.Println("Encountered unknown POS for bridge", hostPOS)
		}
	}
	return bridge, resultMorph, featureMap, resultMSRStr, nil
}

func ProcessUDAnalyzedToken(analysis string) (*AnalyzedToken, error) {
	var (
		split, msrs           []string
		curToken              *AnalyzedToken
		i                     int
		curNode, curID        int
		lemma                 string
		def, noMerge          bool
		UDMSR, UDPOS, UDFeats string
		udPOSExists           bool
	)
	split = strings.Split(analysis, SEPARATOR)
	splitLen := len(split)
	if splitLen < 3 || splitLen%2 != 1 {
		return nil, errors.New("Wrong number of fields (" + analysis + ")")
	}
	curToken = &AnalyzedToken{
		Token:     split[0],
		Morphemes: make([]types.BasicMorphemes, 0, (splitLen-1)/2),
	}
	if ADD_NNP_NO_FEATS {
		// manually add NNP stripped of feats
		for i = 1; i < splitLen; i += 2 {
			msrs = strings.Split(split[i], MSR_SEPARATOR)
			if len(msrs[0]) == 0 && len(msrs[2]) == 0 {
				CPOS, _, _, FeatureStr, err := ParseMSR(msrs[1], false)
				if err != nil {
					continue
				}
				if CPOS == "NNP" && len(FeatureStr) > 0 {
					split = append(split, []string{":NNP:", split[i+1]}...)
				}
			}
		}
		splitLen = len(split)
	}
	prefix := log.Prefix()
	log.SetPrefix(prefix + "Token " + curToken.Token + " ")
	for i = 1; i < splitLen; i += 2 {
		curNode, curID = 0, 0
		morphs := make(types.BasicMorphemes, 0, 4)
		msrs = strings.Split(split[i], MSR_SEPARATOR)
		lemma = split[i+1]
		def = false
		// Prefix morpheme (if exists)
		if len(msrs[0]) > 0 {
			if msrs[0] == "DEF" {
				hebanalysis := HEBREW_XLITER8.To(analysis)
				if hebanalysis[0] == 'H' {
					continue
				}
				def = true
			} else {
				return nil, errors.New("Unknown prefix MSR(" + msrs[0] + ")")
			}
		}
		if len(msrs[1]) == 0 {
			return nil, errors.New("Empty host MSR (" + analysis + ")")
		}
		// Host morpheme
		CPOS, _, Features, FeatureStr, err := ParseMSR(msrs[1], false)
		if err != nil {
			return nil, err
		}
		if CPOS == "UNK" {
			continue
		}
		if def {
			Features["def"] = "D"
			FeatureStr = util.AddToFeatureStr(FeatureStr, "def=D")
		}

		// convert POS to UDv2
		if UDMSR, udPOSExists = util.HEB2UDPOS[CPOS]; !udPOSExists {
			panic(fmt.Sprintf("Unknown POS for UD conversion lookup %s", CPOS))
		}
		UDSplit := strings.Split(UDMSR, "-")
		if len(UDSplit) == 0 {
			panic("Got empty UDMSR")
		} else if len(UDSplit) == 1 {
			UDPOS, UDFeats = UDSplit[0], ""
		} else if len(UDSplit) == 2 {
			UDPOS, UDFeats = UDSplit[0], UDSplit[1]
		} else if len(UDSplit) > 2 {
			panic("Error - UD MSR should have POS and features split by dash (-)")
		}

		if CPOS == "CC" {
			hasCoord := strings.Contains(FeatureStr, "type=COORD")
			hasREL := strings.Contains(FeatureStr, "type=REL")
			hasSUB := strings.Contains(FeatureStr, "type=SUB")
			if hasCoord {
				UDPOS = "CCONJ"
			}
			if hasREL || hasSUB {
				UDPOS = "SCONJ"
			}
			if hasCoord || hasSUB || hasREL {
				noMerge = true
				FeatureStr = ""
				Features = nil
			}
		}
		// convert lexicon feature string to UD
		FeatureStr = util.Heb2UDFeaturesString(FeatureStr)

		// special handling of tense=BEINONI
		if tenseValue, tenseExists := Features["tense"]; tenseExists {
			if tenseValue == "BEINONI" {
				switch CPOS {
				case "VB":
					FeatureStr = util.AddToFeatureStr(FeatureStr, "VerbForm=Part")
				case "MD":
					FeatureStr = util.AddToFeatureStr(FeatureStr, "VerbForm=Part")
				case "EX":
					FeatureStr = util.AddToFeatureStr(FeatureStr, "VerbForm=Part")
				}
			}
		}

		// special case for PRP-REF, add Reflex=Yes
		if _, refExists := Features["type=REF"]; CPOS == "PRP" && refExists {
			UDFeats = util.AddToFeatureStr(UDFeats, "Reflex=Yes")
		}
		if STRIP_ALL_NNP_OF_FEATS && UDPOS == "PROPN" {
			FeatureStr, Features = "", nil
		} else {
			if !noMerge {
				FeatureStr, Features = util.MergeFeatureStrs(FeatureStr, UDFeats)
			}
		}
		// special handling of CC [COORD|SUB|REL]
		hostMorph := &types.Morpheme{
			BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
			Form:              split[0],
			Lemma:             lemma,
			CPOS:              UDPOS,
			POS:               "_",
			Features:          Features,
			TokenID:           0,
			FeatureStr:        FeatureStr,
		}
		morphs = append(morphs, hostMorph)
		curID++
		curNode++
		// Suffix morphemes
		if len(msrs[2]) > 0 {
			if _, exists := SUFFIX_ONLY_CPOS[CPOS]; CPOS != "NN" && exists && hostMorph.Features != nil /* && msrs[1] != "PRP-REF"  */ {
				// add prepositional pronoun features
				_, _, sufFeatures, sufFeatureStr, _ := ParseMSR(msrs[2], msrs[1] != "PRP-REF")
				featList := make([]string, 0, 2)
				if len(hostMorph.FeatureStr) > 0 {
					featList = append(featList, hostMorph.FeatureStr)
				}
				if len(sufFeatureStr) > 0 {
					featList = append(featList, sufFeatureStr)
				}
				hostMorph.FeatureStr = strings.Join(featList, FEATURE_PAIR_SEPARATOR)
				for k, v := range sufFeatures {
					hostMorph.Features[k] = v
				}
			} else if msrs[2][0] == '-' || (msrs[2][0] == 'S' && msrs[2][:5] != "S_ANP") {
				// fix host of previous add morphemes
				lastM := morphs[len(morphs)-1]
				lastM.Form = lastM.Lemma + "_"

				// add prepositional pronoun morphemes
				bridge, sufForm, sufFeatures, sufFeatureStr, err := ParseMSRSuffix(CPOS, msrs[2])
				if err != nil {
					return nil, err
				}
				if len(bridge) > 0 {
					morphs = append(morphs, &types.Morpheme{
						BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
						Form:              "_" + bridge + "_",
						Lemma:             bridge,
						CPOS:              "ADP",
						POS:               "_",
						Features:          nil,
						TokenID:           0,
						FeatureStr:        "",
					})
					curID++
					curNode++
				}
				sufFeatureStr = util.Heb2UDFeaturesString(sufFeatureStr)
				sufFeatureStr, sufFeatures = util.MergeFeatureStrs(sufFeatureStr, "")
				morphs = append(morphs, &types.Morpheme{
					BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
					Form:              "_" + sufForm,
					Lemma:             "הוא",
					CPOS:              "PRON",
					POS:               "_",
					Features:          sufFeatures,
					TokenID:           0,
					FeatureStr:        sufFeatureStr,
				})
				curID++
				curNode++
			}
		}
		curToken.Morphemes = append(curToken.Morphemes, morphs)
	}
	log.SetPrefix(prefix)
	if len(curToken.Morphemes) == 0 {
		return nil, nil
	}
	return curToken, nil
}

func ProcessUDAnalyzedPrefix(analysis string) (*AnalyzedToken, error) {
	var (
		split, forms, prefix_msrs, msrs []string
		curToken                        *AnalyzedToken
		i                               int
		curNode, curID                  int
		HEBPOS, UDPOS, featureStr       string
		featureMap                      map[string]string
	)
	split = strings.Split(analysis, SEPARATOR)
	splitLen := len(split)
	if splitLen < 3 || splitLen%2 != 1 {
		return nil, errors.New("Wrong number of fields (" + analysis + ")")
	}
	curToken = &AnalyzedToken{
		Token:     split[0],
		Morphemes: make([]types.BasicMorphemes, 0, (splitLen-1)/2),
	}
	prefix := log.Prefix()
	log.SetPrefix(prefix + " Token " + curToken.Token)
	for i = 1; i < splitLen; i += 2 {
		curNode, curID = 0, 0
		morphs := make(types.BasicMorphemes, 0, 4)
		forms = strings.Split(split[i], PREFIX_SEPARATOR)
		prefix_msrs = strings.Split(split[i+1], PREFIX_MSR_SEPARATOR)
		if len(forms) != len(prefix_msrs) {
			return nil, errors.New("Mismatch between # of forms and # of MSRs (" + analysis + ")")
		}
		for j := 0; j < len(forms); j++ {
			msrs = strings.Split(prefix_msrs[j], MSR_SEPARATOR)
			featureMap = nil
			featureStr = ""
			// Add prefix morpheme
			if len(msrs[0]) > 0 {
				// replace -SUBCONJ for TEMP-SUBCONJ/REL-SUBCONJ
				HEBPOS = strings.Replace(msrs[0], "-SUBCONJ", "", -1)
				UDPOS = util.HEB2UDPrefixPOS[HEBPOS]
				switch HEBPOS {
				case "TEMP":
					featureMap = make(map[string]string, 1)
					featureMap["Case"] = "Tem"
					featureStr = "Case=Tem"
				case "DEF":
					featureMap = make(map[string]string, 1)
					featureMap["PronType"] = "Art"
					featureStr = "PronType=Art"
				}
				morphs = append(morphs, &types.Morpheme{
					BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
					Form:              forms[j],
					Lemma:             forms[j],
					CPOS:              UDPOS,
					POS:               "_",
					Features:          featureMap,
					TokenID:           0,
					FeatureStr:        featureStr,
				})
				curID++
				curNode++
			}
		}
		curToken.Morphemes = append(curToken.Morphemes, morphs)
	}
	log.SetPrefix(prefix)
	return curToken, nil
}
func ProcessAnalyzedToken(analysis string) (*AnalyzedToken, error) {
	var (
		split, msrs    []string
		curToken       *AnalyzedToken
		i              int
		curNode, curID int
		lemma          string
		def            bool
	)
	split = strings.Split(analysis, SEPARATOR)
	splitLen := len(split)
	if splitLen < 3 || splitLen%2 != 1 {
		return nil, errors.New("Wrong number of fields (" + analysis + ")")
	}
	curToken = &AnalyzedToken{
		Token:     split[0],
		Morphemes: make([]types.BasicMorphemes, 0, (splitLen-1)/2),
	}
	if ADD_NNP_NO_FEATS {
		// manually add NNP stripped of feats
		for i = 1; i < splitLen; i += 2 {
			msrs = strings.Split(split[i], MSR_SEPARATOR)
			if len(msrs[0]) == 0 && len(msrs[2]) == 0 {
				CPOS, _, _, FeatureStr, err := ParseMSR(msrs[1], false)
				if err != nil {
					continue
				}
				if CPOS == "NNP" && len(FeatureStr) > 0 {
					split = append(split, []string{":NNP:", split[i+1]}...)
				}
			}
		}
		splitLen = len(split)
	}
	prefix := log.Prefix()
	log.SetPrefix(prefix + "Token " + curToken.Token + " ")
	for i = 1; i < splitLen; i += 2 {
		curNode, curID = 0, 0
		morphs := make(types.BasicMorphemes, 0, 4)
		msrs = strings.Split(split[i], MSR_SEPARATOR)
		lemma = split[i+1]
		def = false
		// Prefix morpheme (if exists)
		if len(msrs[0]) > 0 {
			if msrs[0] == "DEF" {
				hebanalysis := HEBREW_XLITER8.To(analysis)
				if hebanalysis[0] == 'H' {
					continue
				}
				def = true
			} else {
				return nil, errors.New("Unknown prefix MSR(" + msrs[0] + ")")
			}
		}
		if len(msrs[1]) == 0 {
			return nil, errors.New("Empty host MSR (" + analysis + ")")
		}
		// Host morpheme
		CPOS, POS, Features, FeatureStr, err := ParseMSR(msrs[1], false)
		if err != nil {
			return nil, err
		}
		if CPOS == "UNK" {
			continue
		}
		if def {
			Features["def"] = "D"
			FeatureStr = util.AddToFeatureStr(FeatureStr, "def=D")
		}
		// for _, otherMs := range curToken.Morphemes {
		// 	otherM := otherMs[0]
		// 	if otherM.CPOS == CPOS && otherM.Lemma == lemma && otherM.FeatureStr == FeatureStr && len(msrs[2]) > 0 {
		// 		clitics := "cliticized=true"
		// 		if len(Features) > 0 {
		// 			FeatureStr = FeatureStr + "|" + clitics
		// 		} else {
		// 			FeatureStr = clitics
		// 		}
		// 		Features["cliticized"] = "true"
		// 		break
		// 	}
		// }
		hostMorph := &types.Morpheme{
			BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
			Form:              split[0],
			Lemma:             lemma,
			CPOS:              CPOS,
			POS:               POS,
			Features:          Features,
			TokenID:           0,
			FeatureStr:        FeatureStr,
		}
		morphs = append(morphs, hostMorph)
		curID++
		curNode++
		// Suffix morphemes
		if len(msrs[2]) > 0 {
			if _, exists := SUFFIX_ONLY_CPOS[CPOS]; exists /* && msrs[1] != "PRP-REF"  */ {
				// add prepositional pronoun features
				_, _, sufFeatures, sufFeatureStr, _ := ParseMSR(msrs[2], msrs[1] != "PRP-REF")
				featList := make([]string, 0, 2)
				if len(hostMorph.FeatureStr) > 0 {
					featList = append(featList, hostMorph.FeatureStr)
				}
				if len(sufFeatureStr) > 0 {
					featList = append(featList, sufFeatureStr)
				}
				hostMorph.FeatureStr = strings.Join(featList, FEATURE_PAIR_SEPARATOR)
				for k, v := range sufFeatures {
					hostMorph.Features[k] = v
				}
			} else if msrs[2][0] == '-' || (msrs[2][0] == 'S' && msrs[2][:5] != "S_ANP") {
				// fix host of previous add morphemes
				lastM := morphs[len(morphs)-1]
				lastM.Form = lastM.Lemma

				// add prepositional pronoun morphemes
				bridge, sufForm, sufFeatures, sufFeatureStr, err := ParseMSRSuffix(hostMorph.CPOS, msrs[2])
				if err != nil {
					return nil, err
				}
				if len(bridge) > 0 {
					morphs = append(morphs, &types.Morpheme{
						BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
						Form:              bridge,
						Lemma:             bridge,
						CPOS:              "POS",
						POS:               "POS",
						Features:          nil,
						TokenID:           0,
						FeatureStr:        "",
					})
					curID++
					curNode++
				}
				morphs = append(morphs, &types.Morpheme{
					BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
					Form:              sufForm,
					Lemma:             sufForm,
					CPOS:              "S_PRN",
					POS:               "S_PRN",
					Features:          sufFeatures,
					TokenID:           0,
					FeatureStr:        sufFeatureStr,
				})
				curID++
				curNode++
			}
		}
		curToken.Morphemes = append(curToken.Morphemes, morphs)
	}
	log.SetPrefix(prefix)
	if len(curToken.Morphemes) == 0 {
		return nil, nil
	}
	return curToken, nil
}

func ProcessAnalyzedPrefix(analysis string) (*AnalyzedToken, error) {
	var (
		split, forms, prefix_msrs, msrs []string
		curToken                        *AnalyzedToken
		i                               int
		curNode, curID                  int
	)
	split = strings.Split(analysis, SEPARATOR)
	splitLen := len(split)
	if splitLen < 3 || splitLen%2 != 1 {
		return nil, errors.New("Wrong number of fields (" + analysis + ")")
	}
	curToken = &AnalyzedToken{
		Token:     split[0],
		Morphemes: make([]types.BasicMorphemes, 0, (splitLen-1)/2),
	}
	prefix := log.Prefix()
	log.SetPrefix(prefix + " Token " + curToken.Token)
	for i = 1; i < splitLen; i += 2 {
		curNode, curID = 0, 0
		morphs := make(types.BasicMorphemes, 0, 4)
		forms = strings.Split(split[i], PREFIX_SEPARATOR)
		prefix_msrs = strings.Split(split[i+1], PREFIX_MSR_SEPARATOR)
		if len(forms) != len(prefix_msrs) {
			return nil, errors.New("Mismatch between # of forms and # of MSRs (" + analysis + ")")
		}
		for j := 0; j < len(forms); j++ {
			msrs = strings.Split(prefix_msrs[j], MSR_SEPARATOR)
			// Add prefix morpheme
			if len(msrs[0]) > 0 {
				// replace -SUBCONJ for TEMP-SUBCONJ/REL-SUBCONJ
				morphs = append(morphs, &types.Morpheme{
					BasicDirectedEdge: graph.BasicDirectedEdge{curID, curNode, curNode + 1},
					Form:              forms[j],
					Lemma:             forms[j],
					CPOS:              strings.Replace(msrs[0], "-SUBCONJ", "", -1),
					POS:               strings.Replace(msrs[0], "-SUBCONJ", "", -1),
					Features:          nil,
					TokenID:           0,
					FeatureStr:        "",
				})
				curID++
				curNode++
			}
		}
		curToken.Morphemes = append(curToken.Morphemes, morphs)
	}
	log.SetPrefix(prefix)
	return curToken, nil
}

type LexReader func(string) (*AnalyzedToken, error)

func Read(input io.Reader, format string, maType string) ([]*AnalyzedToken, error) {
	tokens := make([]*AnalyzedToken, 0, APPROX_LEX_SIZE)
	scan := bufio.NewScanner(input)
	var reader LexReader
	switch maType {
	case "spmrl":
		switch format {
		case "lexicon":
			reader = ProcessAnalyzedToken
		case "prefix":
			reader = ProcessAnalyzedPrefix
		default:
		}
	case "ud":
		switch format {
		case "lexicon":
			reader = ProcessUDAnalyzedToken
		case "prefix":
			reader = ProcessUDAnalyzedPrefix
		default:
		}
	}
	for scan.Scan() {
		line := scan.Text()
		token, err := reader(line)
		if err != nil {
			return nil, err
		}
		if token != nil {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}
func ReadFile(filename string, format string, maType string) ([]*AnalyzedToken, error) {
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return nil, err
	}

	return Read(file, format, maType)
}
