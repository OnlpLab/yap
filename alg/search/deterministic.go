package search

import (
	"fmt"
	"log"
	"sort"
	"github.com/OnlpLab/yap/alg/featurevector"
	"github.com/OnlpLab/yap/alg/perceptron"
	"github.com/OnlpLab/yap/alg/transition"
	TransitionModel "github.com/OnlpLab/yap/alg/transition/model"
	"github.com/OnlpLab/yap/nlp/parser/dependency"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
)

var SHOW_ORACLE = false

type Deterministic struct {
	Model              TransitionModel.Interface
	TransFunc          transition.TransitionSystem
	FeatExtractor      perceptron.FeatureExtractor
	ReturnModelValue   bool
	ReturnSequence     bool
	ShowConsiderations bool
	Base               transition.Configuration
	NoRecover          bool
	TransEnum          *util.EnumSet
	DefaultTransType  byte
}

var _ perceptron.InstanceDecoder = &Deterministic{}

// Parser functions
func (d *Deterministic) Parse(problem Problem) (transition.Configuration, interface{}) {
	if d.TransFunc == nil {
		panic("Can't parse without a transition system")
	}
	transitionClassifier := &TransitionClassifier{Model: d.Model, TransFunc: d.TransFunc, FeatExtractor: d.FeatExtractor}
	transitionClassifier.Init()
	transitionClassifier.ShowConsiderations = d.ShowConsiderations

	c := d.Base.Copy()
	c.Clear()
	c.Init(problem)
	var prevConf transition.Configuration
	// deterministic parsing algorithm
	for !c.Terminal() {
		prevConf = c
		c, _ = transitionClassifier.TransitionWithConf(c)
		if c == nil {
			// log.Println("Got nil configuration!")
			c = prevConf
			break
		}
	}

	// build result parameters
	var resultParams *ParseResultParameters
	if d.ReturnModelValue || d.ReturnSequence {
		resultParams = new(ParseResultParameters)
		if d.ReturnModelValue {
			resultParams.ModelValue = transitionClassifier.FeaturesList
		}
		if d.ReturnSequence {
			resultParams.Sequence = c.GetSequence()
		}
	}

	return c, resultParams
}

func (d *Deterministic) ParseOracle(gold perceptron.DecodedInstance) (configuration transition.Configuration, result interface{}) {
	if !d.NoRecover {
		defer func() {
			if r := recover(); r != nil {
				configuration = nil
				result = nil
				log.Println("Recovering parse error: ", r)
			}
		}()
	}
	if d.TransFunc == nil {
		panic("Can't parse without a transition system")
	}
	c := d.Base.Copy()
	c.Clear()
	c.Init(gold.Instance())

	oracle := d.TransFunc.Oracle()
	oracle.SetGold(gold.Decoded())
	transitionNum := 0
	if SHOW_ORACLE {
		log.Println(c.String())
	}
	for !c.Terminal() {
		transition := oracle.Transition(c)
		c = d.TransFunc.Transition(c, transition)
		if SHOW_ORACLE {
			log.Println(c.String())
		}
		transitionNum++
	}

	// build result parameters
	var resultParams *ParseResultParameters
	if d.ReturnModelValue || d.ReturnSequence {
		resultParams = new(ParseResultParameters)
		if d.ReturnModelValue {
			// resultParams.ModelValue = classifier.FeaturesList
		}
		if d.ReturnSequence {
			resultParams.Sequence = c.GetSequence()
		}
	}
	configuration = c
	result = resultParams
	return
}

func (d *Deterministic) ParseOracleEarlyUpdate(sent nlp.Sentence, gold transition.ConfigurationSequence, constraints interface{}, model dependency.ParameterModel) (transition.Configuration, transition.Configuration, interface{}, interface{}, int) {
	if constraints != nil {
		panic("Got non-nil constraints; deterministic dependency parsing does not consider constraints")
	}
	if d.TransFunc == nil {
		panic("Can't parse without a transition system")
	}

	// Initializations
	c := d.Base.Copy()
	c.Clear()
	c.Init(sent)

	classifier := TransitionClassifier{Model: model.(dependency.TransitionParameterModel), FeatExtractor: d.FeatExtractor, TransFunc: d.TransFunc}
	classifier.ShowConsiderations = d.ShowConsiderations

	classifier.Init()

	var (
		predTrans                          transition.Transition
		prevConf, goldConf                 transition.Configuration
		predFeatures                       []featurevector.Feature
		goldFeaturesList, predFeaturesList *transition.FeaturesList
		i                                  int = 1
	)
	prefix := log.Prefix()
	for !c.Terminal() {
		log.SetPrefix(fmt.Sprintf("%s %d ", prefix, i))
		goldConf = gold[i] // Oracle's gold sequence
		// log.Printf("Gold Transition: %s\n", goldConf)
		prevConf = c
		c, predTrans = classifier.TransitionWithConf(c)
		// log.Printf("Pred Transition: %s\n", c)

		// verify the right transition was chosen
		if c == nil || !predTrans.Equal(goldConf.GetLastTransition()) {
			c = prevConf
			// d.FeatExtractor.(*GenericExtractor).Log = true
			predFeatures = d.FeatExtractor.Features(c, false, predTrans.Type(), nil)
			goldFeatures := d.FeatExtractor.Features(gold[i-1], false, goldConf.GetLastTransition().Type(), nil)
			// d.FeatExtractor.(*GenericExtractor).Log = false
			goldFeaturesList = &transition.FeaturesList{goldFeatures, goldConf.GetLastTransition(),
				&transition.FeaturesList{goldFeatures, transition.ConstTransition(0), nil}}
			predFeaturesList = &transition.FeaturesList{predFeatures, predTrans,
				&transition.FeaturesList{predFeatures, transition.ConstTransition(0), nil}}
			break
		}
		i++
	}

	return c, goldConf, predFeaturesList, goldFeaturesList, i
}

// Perceptron functions
func (d *Deterministic) Decode(instance perceptron.Instance, m perceptron.Model) (perceptron.DecodedInstance, interface{}) {
	d.ReturnModelValue = true
	d.Model = m.(TransitionModel.Interface)
	graph, parseParamsInterface := d.Parse(instance)
	parseParams := parseParamsInterface.(*ParseResultParameters)
	return &perceptron.Decoded{instance, graph}, parseParams.ModelValue
}

func (d *Deterministic) DecodeGold(goldInstance perceptron.DecodedInstance, m perceptron.Model) (perceptron.DecodedInstance, interface{}) {
	d.ReturnModelValue = true
	_, goldParams := d.ParseOracle(goldInstance)
	// if !graph.Equal(parsedGraph) {
	// if !parsedGraph.Equal(graph) {
	// 	log.Println("Oracle failed for", graph)
	// 	log.Println("Got graph", parsedGraph)
	// 	panic("Oracle parse result does not equal gold")
	// }
	// 	_, goldParams := deterministic.ParseOracle(graph, nil, tempModel)
	if goldParams != nil {
		seq := goldParams.(*ParseResultParameters).Sequence

		goldSequence := make(ScoredConfigurations, len(seq))
		var (
			lastFeatures *transition.FeaturesList
			curFeats     []featurevector.Feature
		)
		for i := len(seq) - 1; i >= 0; i-- {
			val := seq[i]
			// log.Println("Gold seq val", i, val)
			// log.Println("Pre extract")
			nextTransition := make([]int, 0, 1)
			nextTransitionType := d.DefaultTransType // default to MD
			if i > 0 {
				// if i < len(seq)-1 {
				// log.Println("Configuration for transition is:", seq[i-1])
				// log.Println("Configuration Type for transition is:", seq[i-1].GetLastTransition().Type())
				// log.Println("Configuration is:", val)
				nextTransition = append(nextTransition, int(seq[i-1].GetLastTransition().Value()))
				nextTransitionType = seq[i-1].GetLastTransition().Type()
			}
			// nextTransitionType = seq[i].State()
			// nextTransition = append(nextTransition, int(val.GetLastTransition()))
			// d.FeatExtractor.SetLog(true)
			// log.Println("Features")
			curFeats = d.FeatExtractor.Features(val, false, nextTransitionType, nextTransition)
			// d.FeatExtractor.SetLog(false)
			// log.Println("Features")
			// log.Println(curFeats)
			// log.Println("Post extract")
			lastFeatures = &transition.FeaturesList{curFeats, val.GetLastTransition(), lastFeatures}
			goldSequence[len(seq)-i-1] = &ScoredConfiguration{val, val.GetLastTransition(), NewScoreState(), lastFeatures, 0, 0, true, false}
		}

		// log.Println("Gold seq:\n", seq)
		decoded := &perceptron.Decoded{goldInstance.Instance(), goldSequence}
		return decoded, nil
	} else {
		return nil, nil
	}
}

func (d *Deterministic) DecodeEarlyUpdate(goldInstance perceptron.DecodedInstance, m perceptron.Model) (perceptron.DecodedInstance, interface{}, interface{}, int, int, int64) {
	sent := goldInstance.Instance().(nlp.Sentence)

	// abstract casting >:-[
	rawGoldSequence := goldInstance.Decoded().(ScoredConfigurations)
	goldSequence := make(transition.ConfigurationSequence, len(rawGoldSequence))
	for i, val := range rawGoldSequence {
		goldSequence[i] = val.C
	}

	transitionModel := m.(TransitionModel.Interface)
	model := transitionModel
	d.ReturnModelValue = true
	parsedConf, _, parsedWeights, goldWeights, earlyUpdatedAt := d.ParseOracleEarlyUpdate(sent, goldSequence, nil, model)
	// log.Println("Parsed Features:")
	// log.Println(parsedWeights)
	// log.Println("Gold Features:")
	// log.Println(goldWeights)
	return &perceptron.Decoded{goldInstance.Instance(), parsedConf}, parsedWeights, goldWeights, earlyUpdatedAt, len(rawGoldSequence), 0
}

type TransitionClassifier struct {
	Model              dependency.TransitionParameterModel
	TransFunc          transition.TransitionSystem
	FeatExtractor      perceptron.FeatureExtractor
	Score              int64
	FeaturesList       *transition.FeaturesList
	ShowConsiderations bool
}

func (tc *TransitionClassifier) Init() {
	tc.Score = 0.0
}

// func (tc *TransitionClassifier) Increment(c transition.Configuration) *TransitionClassifier {
// 	features := tc.FeatExtractor.Features(perceptron.Instance(c), false, nil)
// 	tc.FeaturesList = &transition.FeaturesList{features, c.GetLastTransition(), tc.FeaturesList}
// 	tc.Score += tc.Model.TransitionScore(c.GetLastTransition(), features)
// 	return tc
// }
//
func (tc *TransitionClassifier) ScoreWithConf(c transition.Configuration) int64 {
	features := tc.FeatExtractor.Features(perceptron.Instance(c), false, c.GetLastTransition().Type(), nil)
	return tc.Score + tc.Model.TransitionScore(c.GetLastTransition(), features)
}

func (tc *TransitionClassifier) Transition(c transition.Configuration) transition.Transition {
	_, transition := tc.TransitionWithConf(c)
	return transition
}

func (tc *TransitionClassifier) TransitionWithConf(c transition.Configuration) (transition.Configuration, transition.Transition) {
	var (
		bestScore, prevScore int64
		bestTransition       transition.Transition
		notFirst             bool
	)
	prevScore = -1
	tType, tChan := tc.TransFunc.YieldTransitions(c)
	feats := tc.FeatExtractor.Features(c, false, tType, nil)
	if tc.ShowConsiderations {
		log.Println(" Showing Considerations For", c)
	}
	for t := range tChan {
		currentScore := tc.Model.TransitionScore(transition.ConstTransition(t), feats)
		if tc.ShowConsiderations && currentScore != prevScore {
			log.Println(" Considering transition", t, "  ", currentScore)
		}
		if !notFirst || currentScore > bestScore {
			bestScore, bestTransition = currentScore, &transition.TypedTransition{tType, t}
			notFirst = true
		}
		prevScore = currentScore
	}
	if tc.ShowConsiderations {
		if notFirst {
			log.Println("Chose transition", bestTransition)
		} else {
			log.Println("No transitions possible")
		}
	}
	tc.Score += bestScore
	return tc.TransFunc.Transition(c, bestTransition), bestTransition
}

type PerceptronModel struct {
	PerceptronModel TransitionModel.Interface
}

var _ dependency.ParameterModel = &PerceptronModel{}

func (p *PerceptronModel) TransitionModel() TransitionModel.Interface {
	return p.PerceptronModel
}

func (p *PerceptronModel) Model() interface{} {
	return p.PerceptronModel
}

type PerceptronModelValue struct {
	vector []featurevector.Feature
}

var _ dependency.ParameterModelValue = &PerceptronModelValue{}

func (pmv *PerceptronModelValue) Clear() {
	pmv.vector = nil
}

func ArrayDiff(left []featurevector.Feature, right []featurevector.Feature) ([]string, []string) {
	var (
		leftStr, rightStr   []string = make([]string, len(left)), make([]string, len(right))
		onlyLeft, onlyRight []string = make([]string, 0, len(left)), make([]string, 0, len(right))
	)
	for i, val := range left {
		leftStr[i] = val.(string)
	}
	for i, val := range right {
		rightStr[i] = val.(string)
	}
	sort.Strings(leftStr)
	sort.Strings(rightStr)
	i, j := 0, 0
	for i < len(leftStr) || j < len(rightStr) {
		switch {
		case i < len(leftStr) && j < len(rightStr):
			comp := util.Strcmp(leftStr[i], rightStr[j])
			switch {
			case comp == 0:
				i++
				j++
			case comp < 0:
				onlyLeft = append(onlyLeft, leftStr[i])
				i++
			case comp > 0:
				onlyRight = append(onlyRight, rightStr[j])
				j++
			}
		case i < len(leftStr):
			onlyLeft = append(onlyLeft, leftStr[i])
			i++
		case j < len(rightStr):
			onlyRight = append(onlyRight, rightStr[j])
			j++
		}
	}
	return onlyLeft, onlyRight
}
