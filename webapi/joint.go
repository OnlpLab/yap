package webapi

import (
	"log"
	"yap/nlp/format/conll"
	"yap/nlp/parser/disambig"
	"yap/nlp/parser/joint"
	"yap/alg/search"
	"yap/util/conf"
	"yap/nlp/format/lattice"
	nlp "yap/nlp/types"
	transitionmodel "yap/alg/transition/model"
	. "yap/nlp/parser/dependency/transition"
	"yap/app"
	"yap/alg/transition"
	"fmt"
	"bytes"
	"yap/nlp/format/mapping"
	"yap/nlp/format/segmentation"
	"yap/util"
	"strings"
	"sync"
)

var (
	extractor *transition.GenericExtractor
	arcSystem transition.TransitionSystem
	transitionSystem transition.TransitionSystem
	model *transitionmodel.AvgMatrixSparse
	terminalStack int
	paramFunc nlp.MDParam
	jointLock sync.Mutex
	beam *search.Beam
)

func JointParserInitialize() {
	paramFunc, exists := nlp.MDParams[app.MdParamFuncName]
	if !exists {
		log.Fatalln("Param Func", app.MdParamFuncName, "does not exist")
	}
	mdTrans := &disambig.MDTrans{
		ParamFunc: paramFunc,
		UsePOP:    app.UsePOP,
	}
	arcSystem = &ArcEager{
		ArcStandard: ArcStandard{},
	}
	terminalStack = 0
	arcSystem.AddDefaultOracle()
	jointTrans := &joint.JointTrans{
		MDTrans:       mdTrans,
		ArcSys:        arcSystem,
		JointStrategy: app.JointStrategy,
	}
	jointTrans.AddDefaultOracle()
	jointTrans.Oracle().(*joint.JointOracle).OracleStrategy = app.OracleStrategy
	transitionSystem = transition.TransitionSystem(jointTrans)
	if !app.VerifyExists(app.JointFeaturesFile) {
		featuresLocation, found := util.LocateFile(app.JointFeaturesFile, app.DEFAULT_CONF_DIRS)
		if !found {
			panic(fmt.Sprintf("Joint features not found"))
		}
		app.JointFeaturesFile = featuresLocation
	}
	if !app.VerifyExists(app.DepLabelsFile) {
		labelsLocation, found := util.LocateFile(app.DepLabelsFile, app.DEFAULT_CONF_DIRS)
		if !found {
			panic(fmt.Sprintf("Dep labels not found"))
		}
		app.DepLabelsFile = labelsLocation
	}
	if !app.VerifyExists(app.JointModelFile) {
		modelLocation, found := util.LocateFile(app.JointModelFile, app.DEFAULT_MODEL_DIRS)
		if !found {
			panic(fmt.Sprintf("Joint model not found"))
		}
		app.JointModelFile = modelLocation
	}
	confBeam := &search.Beam{}
	confBeam.Align = app.AlignBeam
	confBeam.Averaged = app.AverageScores
	app.JointConfigOut(app.JointModelFile, confBeam, transitionSystem)
	relations, err := conf.ReadFile(app.DepLabelsFile)
	if err != nil {
		panic(fmt.Sprintf("Joint labels not found"))
	}
	app.SetupEnum(relations.Values)
	arcSystem = &ArcEager{
		ArcStandard: ArcStandard{
			SHIFT:       app.SH.Value(),
			LEFT:        app.LA.Value(),
			RIGHT:       app.RA.Value(),
			Relations:   app.ERel,
			Transitions: app.ETrans,
		},
		REDUCE:  app.RE.Value(),
		POPROOT: app.PR.Value(),
	}
	arcSystem.AddDefaultOracle()
	jointTrans.ArcSys = arcSystem
	jointTrans.Transitions = app.ETrans
	mdTrans.Transitions = app.ETrans
	mdTrans.UsePOP = app.UsePOP
	mdTrans.POP = app.POP
	disambig.UsePOP = app.UsePOP
	disambig.SwitchFormLemma = !lattice.IGNORE_LEMMA
	disambig.LEMMAS = !lattice.IGNORE_LEMMA
	mdTrans.AddDefaultOracle()
	jointTrans.MDTransition = app.MD
	jointTrans.JointStrategy = app.JointStrategy
	jointTrans.AddDefaultOracle()
	jointTrans.Oracle().(*joint.JointOracle).OracleStrategy = app.OracleStrategy
	transitionSystem = transition.TransitionSystem(jointTrans)
	featureSetup, err := transition.LoadFeatureConfFile(app.JointFeaturesFile)
	if err != nil {
		panic(fmt.Sprintf("Joint features not found"))
	}
	groups := []byte("MPLA")
	extractor = app.SetupExtractor(featureSetup, groups)
	log.Println()
	nlp.InitOpenParamFamily("HEBTB")
	log.Println()

	log.Println("Found model file", app.JointModelFile, " ... loading model")
	serialization := app.ReadModel(app.JointModelFile)
	model = &transitionmodel.AvgMatrixSparse{}
	model.Deserialize(serialization.WeightModel)
	app.EWord = serialization.EWord
	app.EPOS = serialization.EPOS
	app.EWPOS = serialization.EWPOS
	app.EMHost = serialization.EMHost
	app.EMSuffix = serialization.EMSuffix
	app.EMorphProp = serialization.EMorphProp
	app.ETrans = serialization.ETrans
	app.ETokens = serialization.ETokens
	log.Println("Loaded model")
	arcSystem = &ArcEager{
		ArcStandard: ArcStandard{
			SHIFT:       app.SH.Value(),
			LEFT:        app.LA.Value(),
			RIGHT:       app.RA.Value(),
			Relations:   app.ERel,
			Transitions: app.ETrans,
		},
		REDUCE:  app.RE.Value(),
		POPROOT: app.PR.Value(),
	}
	arcSystem.AddDefaultOracle()
	jointTrans.ArcSys = arcSystem
	jointTrans.Transitions = app.ETrans
	mdTrans.Transitions = app.ETrans
	mdTrans.UsePOP = app.UsePOP
	mdTrans.POP = app.POP
	disambig.UsePOP = app.UsePOP
	disambig.SwitchFormLemma = !lattice.IGNORE_LEMMA
	disambig.LEMMAS = !lattice.IGNORE_LEMMA
	mdTrans.AddDefaultOracle()
	jointTrans.MDTransition = app.MD
	jointTrans.JointStrategy = app.JointStrategy
	transitionSystem = transition.TransitionSystem(jointTrans)
	joinConf := &joint.JointConfig{
		SimpleConfiguration: SimpleConfiguration{
			EWord: app.EWord,
			EPOS: app.EPOS,
			EWPOS: app.EWPOS,
			EMHost: app.EMHost,
			EMSuffix: app.EMSuffix,
			ERel: app.ERel,
			ETrans: app.ETrans,
			TerminalStack: terminalStack,
			TerminalQueue: 0,
		},
		MDConfig: disambig.MDConfig{
			ETokens: app.ETokens,
			POP: app.POP,
			Transitions: app.ETrans,
			ParamFunc: paramFunc,
		},
		MDTrans: app.MD,
	}
	beam = &search.Beam{
		TransFunc: transitionSystem,
		FeatExtractor: extractor,
		Base: joinConf,
		Size: app.BeamSize,
		ConcurrentExec: app.ConcurrentBeam,
		Transitions: app.ETrans,
		EstimatedTransitions: 1000, // chosen by random dice roll
	}
	beam.Model = model
	beam.ShortTempAgenda = true
}

func JointParseAmbiguousLattices(input string) (string, string, string) {
	jointLock.Lock()
	log.Println("Reading ambiguous lattices")
	log.Println("input:\n",input)
	reader := strings.NewReader(input)
	lAmb, lAmbE := lattice.Read(reader, 0)
	if lAmbE != nil {
		panic(fmt.Sprintf("Failed reading raw input - %v", lAmbE))
	}
	predAmbLat := lattice.Lattice2SentenceCorpus(lAmb, app.EWord, app.EPOS, app.EWPOS, app.EMorphProp, app.EMHost, app.EMSuffix)
	parsedGraphs := app.Parse(predAmbLat, beam)
	graphAsConll := conll.MorphGraph2ConllCorpus(parsedGraphs)
	buf1 := new(bytes.Buffer)
	conll.Write(buf1, graphAsConll)
	conllDepOut := buf1.String()
	buf2 := new(bytes.Buffer)
	mapping.Write(buf2, app.GetInstances(parsedGraphs, app.GetJointMDConfig))
	mappingMdOut := buf2.String()
	buf3 := new(bytes.Buffer)
	segmentation.Write(buf3, parsedGraphs)
	segmentationMdOut := buf3.String()
	jointLock.Unlock()
	return conllDepOut, mappingMdOut, segmentationMdOut
}
