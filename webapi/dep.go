package webapi

import (
	"log"
	"github.com/OnlpLab/yap/nlp/format/conll"
	"github.com/OnlpLab/yap/alg/search"
	"github.com/OnlpLab/yap/alg/transition"
	"github.com/gonuts/commander"
	"github.com/OnlpLab/yap/util"
	"fmt"
	"github.com/OnlpLab/yap/util/conf"
	"github.com/OnlpLab/yap/nlp/format/lattice"
	"strings"
	"github.com/OnlpLab/yap/app"
	transitionmodel "github.com/OnlpLab/yap/alg/transition/model"
	. "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"bytes"
	"sync"
)

var (
	depBeam *search.Beam
	depLock sync.Mutex
)

func DepParserInitialize(cmd *commander.Command, args []string) {
	var (
		arcSystem transition.TransitionSystem
		terminalStack int
	)
	arcSystem = &ArcEager{}
	terminalStack = 0
	arcSystem.AddDefaultOracle()
	transitionSystem := transition.TransitionSystem(arcSystem)
	featuresLocation, found := util.LocateFile(app.DepFeaturesFile, app.DEFAULT_CONF_DIRS)
	if !found {
		panic(fmt.Sprintf("Dep features not found"))
	}
	app.DepFeaturesFile = featuresLocation
	labelsLocation, found := util.LocateFile(app.DepLabelsFile, app.DEFAULT_CONF_DIRS)
	if !found {
		panic(fmt.Sprintf("Dep labels not found"))
	}
	app.DepLabelsFile = labelsLocation
	var (
		model *transitionmodel.AvgMatrixSparse = &transitionmodel.AvgMatrixSparse{}
	)
	modelLocation, found := util.LocateFile(app.DepModelName, app.DEFAULT_MODEL_DIRS)
	if !found {
		panic(fmt.Sprintf("Dep model not found"))
	}
	app.DepModelName = modelLocation
	app.DepConfigOut(modelLocation, &search.Beam{}, transitionSystem)
	relations, err := conf.ReadFile(labelsLocation)
	if err != nil {
		panic(fmt.Sprintf("Failed reading Dep labels from file: %v", labelsLocation))
	}
	app.SetupDepEnum(relations.Values)
	arcSystem = &ArcEager{
		ArcStandard: ArcStandard{
			SHIFT: app.SH.Value(),
			LEFT: app.LA.Value(),
			RIGHT: app.RA.Value(),
			Relations: app.ERel,
			Transitions: app.ETrans,
		},
		REDUCE: app.RE.Value(),
		POPROOT: app.PR.Value(),
	}
	arcSystem.AddDefaultOracle()
	transitionSystem = transition.TransitionSystem(arcSystem)
	log.Println()
	log.Println("Loading features")

	featureSetup, err := transition.LoadFeatureConfFile(featuresLocation)
	if err != nil {
		panic(fmt.Sprintf("Failed reading Dep features from file: %v", featuresLocation))
	}
	extractor := app.SetupExtractor(featureSetup, []byte("A"))
	group, _ := extractor.TransTypeGroups['A']
	formatters := make([]util.Format, len(group.FeatureTemplates))
	for i, formatter := range group.FeatureTemplates {
		formatters[i] = formatter
	}

	log.Println("Found model file", modelLocation, " ... loading model")
	serialization := app.ReadModel(modelLocation)
	model.Deserialize(serialization.WeightModel)
	app.EWord = serialization.EWord
	app.EPOS = serialization.EPOS
	app.EWPOS = serialization.EWPOS
	app.EMHost = serialization.EMHost
	app.EMSuffix = serialization.EMSuffix
	log.Println("Loaded model")

	conf := &SimpleConfiguration{
		EWord: app.EWord,
		EPOS: app.EPOS,
		EWPOS: app.EWPOS,
		EMHost: app.EMHost,
		EMSuffix: app.EMSuffix,
		ERel: app.ERel,
		ETrans: app.ETrans,
		TerminalStack: terminalStack,
		TerminalQueue: 0,
	}

	depBeam = &search.Beam{
		TransFunc: transitionSystem,
		FeatExtractor: extractor,
		Base: conf,
		Model: model,
		Size: app.BeamSize,
		ConcurrentExec: app.ConcurrentBeam,
		ShortTempAgenda: true,
		EstimatedTransitions: app.EstimatedBeamTransitions(),
		ScoredStoreDense: true,
	}
}

func DepParseDisambiguatedLattice(input string) string {
	depLock.Lock()
	log.Println("Reading disambiguated lattice")
	log.Println("input:\n",input)
	reader := strings.NewReader(input)
	lDisamb, lDisambE := lattice.Read(reader, 0)
	if lDisambE != nil {
		panic(fmt.Sprintf("Failed reading raw input - %v", lDisamb))
	}
	internalSents := lattice.Lattice2SentenceCorpus(lDisamb, app.EWord, app.EPOS, app.EWPOS, app.EMorphProp, app.EMHost, app.EMSuffix)
	sents := make([]interface{}, len(internalSents))
	for i, instance := range internalSents {
		sents[i] = instance.(nlp.LatticeSentence).TaggedSentence()
	}
	parsedGraphs := app.Parse(sents, depBeam)
	graphAsConll := conll.Graph2ConllCorpus(parsedGraphs, app.EMHost, app.EMSuffix)
	buf := new(bytes.Buffer)
	conll.Write(buf, graphAsConll)
	depLock.Unlock()
	return buf.String()
}
