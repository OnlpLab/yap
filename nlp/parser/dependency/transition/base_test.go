package transition

import (
	AbstractTransition "github.com/OnlpLab/yap/alg/transition"
	nlp "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"
)

var rawTestSent nlp.BasicETaggedSentence = nlp.BasicETaggedSentence{
	{TaggedToken: nlp.TaggedToken{"Economic", "NN"}},
	{TaggedToken: nlp.TaggedToken{"news", "NN"}},
	{TaggedToken: nlp.TaggedToken{"had", "VB"}},
	{TaggedToken: nlp.TaggedToken{"little", "ADJ"}},
	{TaggedToken: nlp.TaggedToken{"effect", "NN"}},
	{TaggedToken: nlp.TaggedToken{"on", "NN"}},
	{TaggedToken: nlp.TaggedToken{"financial", "NN"}},
	{TaggedToken: nlp.TaggedToken{"markets", "NN"}},
	{TaggedToken: nlp.TaggedToken{".", "yyDOT"}}}

var TEST_SENT nlp.TaggedSentence

var rawNodes []TaggedDepNode = []TaggedDepNode{
	{Id: 0, RawToken: "Economic", RawPOS: "NN"},
	{Id: 1, RawToken: "news", RawPOS: "NN"},
	{Id: 2, RawToken: "had", RawPOS: "VB"},
	{Id: 3, RawToken: "little", RawPOS: "ADJ"},
	{Id: 4, RawToken: "effect", RawPOS: "NN"},
	{Id: 5, RawToken: "on", RawPOS: "NN"},
	{Id: 6, RawToken: "financial", RawPOS: "NN"},
	{Id: 7, RawToken: "markets", RawPOS: "NN"},
	{Id: 8, RawToken: ".", RawPOS: "yyDOT"}}

var rawArcs []BasicDepArc = []BasicDepArc{
	{Head: 1, RawRelation: nlp.DepRel("ATT"), Modifier: 0},
	{Head: 2, RawRelation: nlp.DepRel("SBJ"), Modifier: 1},
	{Head: -1, RawRelation: nlp.DepRel(nlp.ROOT_LABEL), Modifier: 2},
	{Head: 4, RawRelation: nlp.DepRel("ATT"), Modifier: 3},
	{Head: 2, RawRelation: nlp.DepRel("OBJ"), Modifier: 4},
	{Head: 4, RawRelation: nlp.DepRel("ATT"), Modifier: 5},
	{Head: 7, RawRelation: nlp.DepRel("ATT"), Modifier: 6},
	{Head: 5, RawRelation: nlp.DepRel("PC"), Modifier: 7},
	{Head: 2, RawRelation: nlp.DepRel("PU"), Modifier: 8}}

var (
	TEST_RELATIONS      []nlp.DepRel = []nlp.DepRel{"ATT", "SBJ", "PC", "OBJ", "PU", "PRED", nlp.ROOT_LABEL}
	TRANSITIONS_ENUM    *util.EnumSet
	TEST_ENUM_RELATIONS *util.EnumSet
	EWord, EPOS, EWPOS  *util.EnumSet
	SH, RE, PR, LA, RA  AbstractTransition.Transition
)

//ALL RICH FEATURES
// var TEST_RICH_FEATURES [][2]string = [][2]string{
// 	{"S0|w", "S0|w"},
// 	{"S0|p", "S0|w"},
// 	{"S0|w|p", "S0|w"},

// 	{"N0|w", "N0|w"},
// 	{"N0|p", "N0|w"},
// 	{"N0|w|p", "N0|w"},

// 	{"N1|w", "N1|w"},
// 	{"N1|p", "N1|w"},
// 	{"N1|w|p", "N1|w"},

// 	{"N2|w", "N2|w"},
// 	{"N2|p", "N2|w"},
// 	{"N2|w|p", "N2|w"},

// 	{"S0h|w", "S0h|w"},
// 	{"S0h|p", "S0h|w"},
// 	{"S0|l", "S0h|w"},

// 	{"S0h2|w", "S0h2|w"},
// 	{"S0h2|p", "S0h2|w"},
// 	{"S0h|l", "S0h2|w"},

// 	{"S0l|w", "S0l|w"},
// 	{"S0l|p", "S0l|w"},
// 	{"S0l|l", "S0l|w"},

// 	{"S0r|w", "S0r|w"},
// 	{"S0r|p", "S0r|w"},
// 	{"S0r|l", "S0r|w"},

// 	{"S0l2|w", "S0l2|w"},
// 	{"S0l2|p", "S0l2|w"},
// 	{"S0l2|l", "S0l2|w"},

// 	{"S0r2|w", "S0r2|w"},
// 	{"S0r2|p", "S0r2|w"},
// 	{"S0r2|l", "S0r2|w"},

// 	{"N0l|w", "N0l|w"},
// 	{"N0l|p", "N0l|w"},
// 	{"N0l|l", "N0l|w"},

// 	{"N0l2|w", "N0l2|w"},
// 	{"N0l2|p", "N0l2|w"},
// 	{"N0l2|l", "N0l2|w"},

// 	{"S0|w|p+N0|w|p", "S0|w"},
// 	{"S0|w|p+N0|w", "S0|w"},
// 	{"S0|w+N0|w|p", "S0|w"},
// 	{"S0|w|p+N0|p", "S0|w"},
// 	{"S0|p+N0|w|p", "S0|w"},
// 	{"S0|w+N0|w", "S0|w"},
// 	{"S0|p+N0|p", "S0|w"},

// 	{"N0|p+N1|p", "S0|w;N0|w"},
// 	{"N0|p+N1|p+N2|p", "S0|w;N0|w"},
// 	{"S0|p+N0|p+N1|p", "S0|w;N0|w"},
// 	{"S0|p+N0|p+N0l|p", "S0|w;N0|w"},
// 	{"N0|p+N0l|p+N0l2|p", "S0|w;N0|w"},

// 	{"S0h|p+S0|p+N0|p", "S0|w"},
// 	{"S0h2|p+S0h|p+S0|p", "S0|w"},
// 	{"S0|p+S0l|p+N0|p", "S0|w"},
// 	{"S0|p+S0l|p+S0l2|p", "S0|w"},
// 	{"S0|p+S0r|p+N0|p", "S0|w"},
// 	{"S0|p+S0r|p+S0r2|p", "S0|w"},

// 	{"S0|w|d", "S0|w;N0|w"},
// 	{"S0|p|d", "S0|w;N0|w"},
// 	{"N0|w|d", "S0|w;N0|w"},
// 	{"N0|p|d", "S0|w;N0|w"},
// 	{"S0|w+N0|w|d", "S0|w;N0|w"},
// 	{"S0|p+N0|p|d", "S0|w;N0|w"},

// 	{"S0|w|vr", "S0|w"},
// 	{"S0|p|vr", "S0|w"},
// 	{"S0|w|vl", "S0|w"},
// 	{"S0|p|vl", "S0|w"},
// 	{"N0|w|vl", "N0|w"},
// 	{"N0|p|vl", "N0|w"},

// 	{"S0|w|sr", "S0|w"},
// 	{"S0|p|sr", "S0|w"},
// 	{"S0|w|sl", "S0|w"},
// 	{"S0|p|sl", "S0|w"},
// 	{"N0|w|sl", "N0|w"},
// 	{"N0|p|sl", "N0|w"}}
var TEST_RICH_FEATURES [][2]string = [][2]string{
	{"S0|w", "S0|w"},
	// {"S0|p", "S0|w"},
	{"S0|w|p", "S0|w"},

	{"N0|w", "N0|w"},
	// {"N0|p", "N0|w"},
	{"N0|w|p", "N0|w"},

	{"N1|w", "N1|w"},
	// {"N1|p", "N1|w"},
	{"N1|w|p", "N1|w"},

	{"N2|w", "N2|w"},
	// {"N2|p", "N2|w"},
	{"N2|w|p", "N2|w"},

	// {"S0h|w", "S0h|w"},
	// {"S0h|p", "S0h|w"},
	// {"S0|l", "S0h|w"},

	// {"S0h2|w", "S0h2|w"},
	// {"S0h2|p", "S0h2|w"},
	// {"S0h|l", "S0h2|w"},

	// {"S0l|w", "S0l|w"},
	// {"S0l|p", "S0l|w"},
	// {"S0l|l", "S0l|w"},

	// {"S0r|w", "S0r|w"},
	// {"S0r|p", "S0r|w"},
	// {"S0r|l", "S0r|w"},

	// {"S0l2|w", "S0l2|w"},
	// {"S0l2|p", "S0l2|w"},
	// {"S0l2|l", "S0l2|w"},

	// {"S0r2|w", "S0r2|w"},
	// {"S0r2|p", "S0r2|w"},
	// {"S0r2|l", "S0r2|w"},

	// {"N0l|w", "N0l|w"},
	// {"N0l|p", "N0l|w"},
	// {"N0l|l", "N0l|w"},

	// {"N0l2|w", "N0l2|w"},
	// {"N0l2|p", "N0l2|w"},
	// {"N0l2|l", "N0l2|w"},

	// {"S0|w|p+N0|w|p", "S0|w"},
	// {"S0|w|p+N0|w", "S0|w"},
	// {"S0|w+N0|w|p", "S0|w"},
	// {"S0|w|p+N0|p", "S0|w"},
	// {"S0|p+N0|w|p", "S0|w"},
	// {"S0|w+N0|w", "S0|w"},
	// {"S0|p+N0|p", "S0|w"},

	// {"N0|p+N1|p", "S0|w;N0|w"},
	// {"N0|p+N1|p+N2|p", "S0|w;N0|w"},
	// {"S0|p+N0|p+N1|p", "S0|w;N0|w"},
	// {"S0|p+N0|p+N0l|p", "S0|w;N0|w"},
	// {"S0|p+S0|fp", "S0|w"},
	{"S0Ci|w+S0|w", "S0|w"},
	{"N0Ci|w+N0|w", "N0|w"},

	// {"N0|p+N0l|p+N0l2|p", "S0|w;N0|w"},

	// {"S0h|p+S0|p+N0|p", "S0|w"},
	// {"S0h2|p+S0h|p+S0|p", "S0|w"},
	// {"S0|p+S0l|p+N0|p", "S0|w"},
	// {"S0|p+S0l|p+S0l2|p", "S0|w"},
	// {"S0|p+S0r|p+N0|p", "S0|w"},
	// {"S0|p+S0r|p+S0r2|p", "S0|w"},

	// {"S0|w|d", "S0|w;N0|w"},
	// {"S0|p|d", "S0|w;N0|w"},
	// {"N0|w|d", "S0|w;N0|w"},
	// {"N0|p|d", "S0|w;N0|w"},
	// {"S0|w+N0|w|d", "S0|w;N0|w"},
	// {"S0|p+N0|p|d", "S0|w;N0|w"},
	{"S0|p|vf", "S0|w"},
	{"N0|w|vf", "N0|w"},
	{"N0|p|vf", "N0|w"},

	{"S0|w|sf", "S0|w"},
	{"S0|p|sf", "S0|w"},

	// {"S0|w|vr", "S0|w"},
	// {"S0|p|vr", "S0|w"},
	// {"S0|w|vl", "S0|w"},
	// {"S0|p|vl", "S0|w"},
	// {"N0|w|vl", "N0|w"},
	// {"N0|p|vl", "N0|w"},

	// {"S0|w|sr", "S0|w"},
	// {"S0|p|sr", "S0|w"},
	// {"S0|w|sl", "S0|w"},
	// {"S0|p|sl", "S0|w"},
	// {"N0|w|sl", "N0|w"},
	// {"N0|p|sl", "N0|w"}
	{"S0|w|o", "S0|w;N0|w"},
	{"S0|p|o", "S0|w;N0|w"},
	{"N0|w|o", "S0|w;N0|w"},
	{"N0|p|o", "S0|w;N0|w"},
}

func SetupRelationEnum() {
	if TEST_ENUM_RELATIONS != nil {
		return
	}
	TEST_ENUM_RELATIONS = util.NewEnumSet(len(TEST_RELATIONS))
	for _, label := range TEST_RELATIONS {
		TEST_ENUM_RELATIONS.Add(label)
	}
}

func SetupSentEnum() {
	EWord, EPOS, EWPOS =
		util.NewEnumSet(len(rawNodes)),
		util.NewEnumSet(5), // 4 POS + ROOT
		util.NewEnumSet(len(rawNodes))
	var (
		// val   int
		node  *TaggedDepNode
		arc   *BasicDepArc
		token *nlp.EnumTaggedToken
	)
	EWord.Add(nlp.ROOT_TOKEN)
	EPOS.Add(nlp.ROOT_TOKEN)
	EWPOS.Add([2]string{nlp.ROOT_TOKEN, nlp.ROOT_TOKEN})
	for i, _ := range rawNodes {
		node = &rawNodes[i]
		node.Token, _ = EWord.Add(node.RawToken)
		node.POS, _ = EPOS.Add(node.RawPOS)
		node.TokenPOS, _ = EWPOS.Add([2]string{node.RawToken, node.RawPOS})
	}
	for i, _ := range rawArcs {
		arc = &rawArcs[i]
		arc.Relation, _ = TEST_ENUM_RELATIONS.Add(arc.RawRelation)
	}
	for i, _ := range rawTestSent {
		token = &rawTestSent[i]
		token.EToken, _ = EWord.Add(token.Token)
		token.EPOS, _ = EPOS.Add(token.POS)
		token.ETPOS, _ = EWPOS.Add([2]string{token.Token, token.POS})
	}
	TEST_SENT = nlp.TaggedSentence(rawTestSent)
}

func SetupTestEnum() {
	SetupRelationEnum()
	SetupSentEnum()
}

func GetTestDepGraph() nlp.LabeledDependencyGraph {
	var (
		nodes []nlp.DepNode  = make([]nlp.DepNode, len(rawNodes))
		arcs  []*BasicDepArc = make([]*BasicDepArc, len(rawArcs))
	)
	for i, rawNode := range rawNodes {
		node := new(TaggedDepNode)
		*node = rawNode
		nodes[i] = nlp.DepNode(node)
	}
	for i, rawArc := range rawArcs {
		// make sure to get a heap pointer with it's own copy
		// otherwise &rawArc will be constant
		newArcPtr := new(BasicDepArc)
		*newArcPtr = rawArc
		arcs[i] = newArcPtr
	}
	return nlp.LabeledDependencyGraph(&BasicDepGraph{nodes, arcs})
}

// func GetTestConfiguration() *SimpleConfiguration {
// 	SetupTestEnum()
// 	SetupEagerTransEnum() // default trans is eager
// 	conf := &SimpleConfiguration{
// 		EWord:  EWord,
// 		EPOS:   EPOS,
// 		EWPOS:  EWPOS,
// 		ERel:   TEST_ENUM_RELATIONS,
// 		ETrans: TRANSITIONS_ENUM,
// 	}
// 	conf.Init(TEST_SENT)
// 	// [Economic news had little effect on financial markets .]
// 	//     1      2    3    4      5    6      7       8     9
// 	// Setup configuration:
// 	// C=(	[had,effect], [.], A)
// 	// A={	(had,	OBJ,	effect)
// 	// 		(effect,ATT,	little)
// 	//		(effect,ATT,	on)}
// 	predInd, _ := TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("PRED"))
// 	objInd, _ := TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("OBJ"))
// 	attInd, _ := TEST_ENUM_RELATIONS.IndexOf(nlp.DepRel("ATT"))
// 	conf.Nodes[3].Head = 0
// 	conf.Nodes[3].ELabel = predInd
// 	conf.Nodes[5].Head = 3
// 	conf.Nodes[4].Head = 5
// 	conf.Nodes[6].Head = 5
// 	conf.Nodes[3].AddModifier(5, objInd)
// 	conf.Nodes[5].ELabel = objInd
// 	conf.Nodes[5].AddModifier(4, attInd)
// 	conf.Nodes[4].ELabel = attInd
// 	conf.Nodes[5].AddModifier(6, attInd)
// 	conf.Nodes[6].ELabel = attInd

// 	// S=[had,effect]
// 	// stack should be empty
// 	if _, sExists := conf.Stack().Peek(); sExists {
// 		panic("Initialized configuration should have empty stack")
// 	}
// 	conf.Stack().Push(3)
// 	conf.Stack().Push(5)

// 	// B=[.]
// 	conf.Queue().Clear()
// 	conf.Queue().Push(9)

// 	// A = {...}
// 	conf.Arcs().Add(&BasicDepArc{Head: 0, RawRelation: nlp.DepRel("PRED"), Modifier: 3})
// 	conf.Arcs().Add(&BasicDepArc{Head: 3, RawRelation: nlp.DepRel("OBJ"), Modifier: 5})
// 	conf.Arcs().Add(&BasicDepArc{Head: 5, RawRelation: nlp.DepRel("ATT"), Modifier: 4})
// 	conf.Arcs().Add(&BasicDepArc{Head: 5, RawRelation: nlp.DepRel("ATT"), Modifier: 6})

// 	return conf
// }
