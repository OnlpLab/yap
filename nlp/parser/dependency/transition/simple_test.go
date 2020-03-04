package transition

// import (
// 	AbstractTransition "github.com/OnlpLab/yap/alg/transition"
// 	nlp "github.com/OnlpLab/yap/nlp/types"
// 	"github.com/OnlpLab/yap/util"
// 	"log"
// 	"reflect"
// 	"testing"
// )

// type SimpleConfTest struct {
// 	conf *SimpleConfiguration
// 	t    *testing.T
// }

// func (t *SimpleConfTest) Init() {
// 	c := t.conf
// 	sent := nlp.BasicETaggedSentence{
// 		{nlp.TaggedToken{"a", "NN"}, 1, 1, 1},
// 		{nlp.TaggedToken{"b", "VB"}, 2, 2, 2},
// 	}
// 	c.Init(sent)
// 	if c.Stack() == nil || c.Queue() == nil || c.Arcs() == nil {
// 		t.t.Error("Afte initialization got nil Stack/Queue/Arcs")
// 	}
// 	if len(c.Nodes) != 3 {
// 		t.t.Error("Got wrong size for Nodes slice")
// 	}
// 	if !(&c.Nodes[0]).Equal(NewArcCachedDepNode(&TaggedDepNode{0, 0, 0, 0, nlp.ROOT_TOKEN, nlp.ROOT_TOKEN})) {
// 		t.t.Error("Init did not create root node")
// 	}
// 	if !(&c.Nodes[1]).Equal(NewArcCachedDepNode(&TaggedDepNode{1, 1, 1, 1, sent[0].Token, sent[0].POS})) {
// 		t.t.Error("Init did not create node for tagged token")
// 	}
// 	if !(&c.Nodes[2]).Equal(NewArcCachedDepNode(&TaggedDepNode{2, 2, 2, 2, sent[1].Token, sent[1].POS})) {
// 		t.t.Error("Init did not create node for tagged token")
// 	}
// 	if c.Stack().Size() != 0 {
// 		t.t.Error("Stack does not have correct initial size:")
// 	}
// 	_, sExists := c.Stack().Peek()
// 	if sExists {
// 		t.t.Error("Stack should be empty", sExists)
// 	}
// 	if c.Queue().Size() != 2 {
// 		t.t.Error("Queue has wrong size")
// 	}
// 	qPeekVal, _ := c.Queue().Peek()
// 	if qPeekVal != 1 {
// 		t.t.Error("Queue head has wrong value")
// 	}
// 	qIdx1Val, _ := c.Queue().Index(1)
// 	if qIdx1Val != 2 {
// 		t.t.Error("Queue has wrong value at depth 1")
// 	}
// 	if c.Last != -1 {
// 		t.t.Error("Wrong last action string")
// 	}
// 	if c.InternalPrevious != nil {
// 		t.t.Error("Pointer to previous configuration is not nil")
// 	}
// 	if c.Arcs().Size() != 0 {
// 		t.t.Error("Initialized configuration has non-empty arc set")
// 	}
// }

// func (t *SimpleConfTest) Terminal() {
// 	c := t.conf
// 	c.Init(nlp.BasicETaggedSentence{{nlp.TaggedToken{"a", "NN"}, 1, 1, 1}})
// 	c.Queue().Clear()
// 	if !c.Terminal() {
// 		t.t.Error("Expected terminal configuration after queue cleared")
// 	}
// 	c.Queue().Push(0)
// 	if c.Terminal() {
// 		t.t.Error("Expected non-terminal configuration when queue is not empty")
// 	}
// }

// func (t *SimpleConfTest) Copy() {
// 	c := t.conf
// 	sent := nlp.BasicETaggedSentence{{nlp.TaggedToken{"a", "NN"}, 1, 1, 1}, {nlp.TaggedToken{"a", "NN"}, 2, 2, 2}}
// 	c.Init(sent)
// 	newConf := c.Copy().(*SimpleConfiguration)
// 	if !c.Equal(newConf) {
// 		t.t.Error("Copy is not equal")
// 	}
// 	newConf.Stack().Push(5)
// 	if c.Equal(newConf) {
// 		t.t.Error("Copy is equal after stack push")
// 	}
// 	newConf.Stack().Pop()
// 	if !c.Equal(newConf) {
// 		t.t.Error("Copy is not equal after stack push,pop")
// 	}
// 	newConf.Queue().Push(0)
// 	if c.Equal(newConf) {
// 		t.t.Error("Copy is equal after queue push")
// 	}
// 	newConf.Queue().Pop()
// 	if !c.Equal(newConf) {
// 		t.t.Error("Copy is not equal after queue push,pop")
// 	}
// 	arc1, arc2 := &BasicDepArc{0, 1, 1, "a"}, &BasicDepArc{1, 2, 2, "b"}
// 	c.Arcs().Add(arc1)
// 	newConf.Arcs().Add(arc2)
// 	if c.Equal(newConf) {
// 		t.t.Error("Copy is equal after different arc set additions")
// 	}
// 	c.Arcs().Add(arc2)
// 	newConf.Arcs().Add(arc1)
// 	if !c.Equal(newConf) {
// 		t.t.Error("Copy is not equal after arc set additions in different order")
// 	}
// 	if newConf.InternalPrevious != c {
// 		t.t.Error("Copy reports wrong previous configuration")
// 	}
// }

// func (t *SimpleConfTest) Arcs() {
// 	if t.conf.InternalArcs != t.conf.Arcs() {
// 		t.t.Error("Returned wrong arcset object")
// 	}
// }

// func (t *SimpleConfTest) GetArc() {
// 	for _, arcIndex := range t.conf.GetEdges() {
// 		if !reflect.DeepEqual(
// 			t.conf.GetArc(arcIndex),
// 			t.conf.Arcs().Index(arcIndex)) {
// 			t.t.Error("Got wrong arc")
// 		}
// 	}
// }

// func (t *SimpleConfTest) GetDirectedEdge() {
// 	for _, arcIndex := range t.conf.GetEdges() {
// 		if !reflect.DeepEqual(
// 			t.conf.GetDirectedEdge(arcIndex),
// 			t.conf.Arcs().Index(arcIndex)) {
// 			t.t.Error("Got wrong arc")
// 		}
// 	}
// }

// func (t *SimpleConfTest) GetEdge() {
// 	for _, arcIndex := range t.conf.GetEdges() {
// 		if !reflect.DeepEqual(
// 			t.conf.GetEdge(arcIndex),
// 			t.conf.Arcs().Index(arcIndex)) {
// 			t.t.Error("Got wrong arc")
// 		}
// 	}
// }

// func (t *SimpleConfTest) GetEdges() {
// 	if !reflect.DeepEqual(t.conf.GetEdges(),
// 		util.RangeInt(t.conf.Arcs().Size())) {
// 		t.t.Error("Got wrong edge index slice")
// 	}
// }

// func (t *SimpleConfTest) GetLabeledArc() {
// 	for _, arcIndex := range t.conf.GetEdges() {
// 		if !reflect.DeepEqual(
// 			t.conf.GetLabeledArc(arcIndex),
// 			t.conf.Arcs().Index(arcIndex)) {
// 			t.t.Error("Got wrong arc")
// 		}
// 	}
// }

// func (t *SimpleConfTest) GetNode() {
// 	for i, node := range t.conf.Nodes {
// 		if !reflect.DeepEqual(t.conf.GetNode(i), node.Node) {
// 			t.t.Error("Got wrong node")
// 		}
// 	}
// }

// func (t *SimpleConfTest) GetSequence() {
// 	copied := t.conf.Copy().(*SimpleConfiguration)
// 	seq := []Abstracttransition.Configuration(copied.GetSequence())
// 	if len(seq) != 2 {
// 		t.t.Error("Returned sequence of wrong length")
// 	}
// 	if !seq[1].(*SimpleConfiguration).Equal(t.conf) {
// 		t.t.Error("First configuration not equal in sequence")
// 	}
// 	if !seq[0].(*SimpleConfiguration).Equal(copied) {
// 		t.t.Error("Second configuration not equal in sequence")
// 	}
// }

// func (t *SimpleConfTest) GetVertex() {
// 	for i, node := range t.conf.Nodes {
// 		if !reflect.DeepEqual(t.conf.GetVertex(i), node) {
// 			t.t.Error("Got wrong vertex")
// 		}
// 	}
// }

// func (t *SimpleConfTest) GetVertices() {
// 	if !reflect.DeepEqual(t.conf.GetVertices(),
// 		util.RangeInt(len(t.conf.Nodes))) {
// 		t.t.Error("Got wrong vertex index slice")
// 	}
// }

// func (t *SimpleConfTest) ID() {
// 	if t.conf.ID() != 0 {
// 		t.t.Error("ID should be constant 0")
// 	}
// }

// func (t *SimpleConfTest) NumberOfArcs() {
// 	if t.conf.InternalArcs.Size() != t.conf.NumberOfArcs() {
// 		t.t.Error("Reported wrong number of arcs")
// 	}
// }

// func (t *SimpleConfTest) NumberOfEdges() {
// 	if t.conf.InternalArcs.Size() != t.conf.NumberOfEdges() {
// 		t.t.Error("Reported wrong number of edges")
// 	}
// }

// func (t *SimpleConfTest) NumberOfNodes() {
// 	if len(t.conf.Nodes) != t.conf.NumberOfNodes() {
// 		t.t.Error("Reported wrong number of nodes")
// 	}
// }

// func (t *SimpleConfTest) NumberOfVertices() {
// 	if len(t.conf.Nodes) != t.conf.NumberOfVertices() {
// 		t.t.Error("Reported wrong number of nodes")
// 	}
// }

// func (t *SimpleConfTest) Previous() {
// 	if t.conf.InternalPrevious != t.conf.Previous() {
// 		t.t.Error("Reported wrong previous pointer")
// 	}
// }

// func (t *SimpleConfTest) Queue() {
// 	if t.conf.InternalQueue != t.conf.Queue() {
// 		t.t.Error("Returned wrong queue object")
// 	}
// }

// func (t *SimpleConfTest) SetLastTransition() {
// 	t.conf.SetLastTransition(LA)
// 	if t.conf.Last != LA {
// 		t.t.Error("Setting last transition failed")
// 	}
// }

// func (t *SimpleConfTest) Stack() {
// 	if t.conf.InternalStack != t.conf.Stack() {
// 		t.t.Error("Returned wrong stack object")
// 	}
// }

// func (t *SimpleConfTest) String() {
// 	str := t.conf.String()
// 	if len(str) == 0 {
// 		t.t.Error("Non empty configuration returns empty String")
// 	}
// }

// func (t *SimpleConfTest) StringArcs() {
// 	t.conf.SetLastTransition(LA)
// 	str := t.conf.StringArcs()
// 	if len(str) == 0 {
// 		t.t.Error("Non empty configuration returns empty StringArcs")
// 	}
// 	copied := t.conf.Copy().(*SimpleConfiguration)
// 	copied.SetLastTransition(SH)
// 	str = copied.StringArcs()
// 	if len(str) == 0 {
// 		t.t.Error("Non reduce configuration returns non empty StringArcs")
// 	}
// }

// func (t *SimpleConfTest) StringQueue() {
// 	str := t.conf.StringQueue()
// 	if len(str) == 0 {
// 		t.t.Error("Non empty configuration returns empty StringQueue")
// 	}
// 	t.conf.Queue().Clear()
// 	str = t.conf.StringQueue()
// 	if len(str) != 0 {
// 		t.t.Error("Empty queue in configuration returns non empty StringQueue")
// 	}
// 	t.conf.Queue().Push(0)
// 	t.conf.Queue().Push(0)
// 	t.conf.Queue().Push(0)
// 	t.conf.Queue().Push(0)
// 	str = t.conf.StringQueue()
// 	if len(str) == 0 {
// 		t.t.Error("Non-empty queue in configuration returns non empty StringQueue")
// 	}
// }

// func (t *SimpleConfTest) StringStack() {
// 	t.conf.Stack().Clear()
// 	t.conf.Stack().Push(0)
// 	str := t.conf.StringStack()
// 	if len(str) == 0 {
// 		t.t.Error("Non empty configuration returns empty StringStack")
// 	}
// 	t.conf.Stack().Clear()
// 	str = t.conf.StringStack()
// 	if len(str) != 0 {
// 		t.t.Error("Empty stack in configuration returns non empty StringStack")
// 	}
// 	t.conf.Stack().Push(1)
// 	t.conf.Stack().Push(1)
// 	t.conf.Stack().Push(1)
// 	t.conf.Stack().Push(1)
// 	str = t.conf.StringStack()
// 	if len(str) == 0 {
// 		t.t.Error("Non-empty stack in configuration returns non empty StringStack")
// 	}
// }

// func (t *SimpleConfTest) Address() {
// 	t.conf = GetTestConfiguration()
// 	// [ROOT Economic news had little effect on financial markets .]
// 	//   0      1      2    3    4      5    6      7       8     9
// 	// Set up configuration:
// 	// C=(	[ROOT,had,effect], [.], A)
// 	// A={	(ROOT,	PRED,	had)
// 	// 		(had,	OBJ,	effect)
// 	// 		(effect,ATT,	little)
// 	//		(effect,ATT,	on)}

// 	// verify S0,1,2; 3 should fail
// 	if s0, s0Exists := t.conf.Address([]byte("S0"), 0); !s0Exists || s0 != 5 {
// 		if !s0Exists {
// 			t.t.Error("Expected S0")
// 		} else {
// 			t.t.Error("S0 should be 5, got", s0)
// 		}
// 	}
// 	if s1, s1Exists := t.conf.Address([]byte("S1"), 1); !s1Exists || s1 != 3 {
// 		if !s1Exists {
// 			t.t.Error("Expected S1")
// 		} else {
// 			t.t.Error("S1 should be 3, got", s1)
// 		}
// 	}
// 	if _, s2Exists := t.conf.Address([]byte("S2"), 3); s2Exists {
// 		t.t.Error("S2 should not exist")
// 	}

// 	// verify N0; N1 should fail
// 	if n0, n0Exists := t.conf.Address([]byte("N0"), 0); !n0Exists || n0 != 9 {
// 		if !n0Exists {
// 			t.t.Error("Expected N1")
// 		} else {
// 			t.t.Error("N0 should be 9, got", n0)
// 		}
// 	}
// 	if _, n1Exists := t.conf.Address([]byte("N1"), 1); n1Exists {
// 		t.t.Error("N1 should not exist")
// 	}

// 	// verify S0h, S0h2
// 	if s0h, s0hExists := t.conf.Address([]byte("S0h"), 0); !s0hExists || s0h != 3 {
// 		if !s0hExists {
// 			t.t.Error("Expected S0h")
// 		} else {
// 			t.t.Error("S0h should be 3, got", s0h)
// 		}
// 	}
// 	if s0h2, s0h2Exists := t.conf.Address([]byte("S0h2"), 0); !s0h2Exists || s0h2 != 0 {
// 		if !s0h2Exists {
// 			t.t.Error("Expected S0h2")
// 		} else {
// 			t.t.Error("S0h2 should be 0, got", s0h2)
// 		}
// 	}
// 	// verify S0l, S0r
// 	if s0l, s0lExists := t.conf.Address([]byte("S0l"), 0); !s0lExists || s0l != 4 {
// 		if !s0lExists {
// 			s0, _ := t.conf.Address([]byte("S0"), 0)
// 			log.Println(t.conf.Nodes[s0].AsString())
// 			t.t.Error("Expected S0l")
// 		} else {
// 			t.t.Error("S0l should be 4, got", s0l)
// 		}
// 	}
// 	if s0r, s0rExists := t.conf.Address([]byte("S0r"), 0); !s0rExists || s0r != 6 {
// 		if !s0rExists {
// 			t.t.Error("Expected S0r")
// 		} else {
// 			t.t.Error("S0r should be 6, got", s0r)
// 		}
// 	}
// 	// verify S0l2, s0r2 don't exist
// 	if _, s0l2Exists := t.conf.Address([]byte("S0l2"), 0); s0l2Exists {
// 		t.t.Error("S0l2 should not exist")
// 	}
// 	if _, s0r2Exists := t.conf.Address([]byte("S0r2"), 0); s0r2Exists {
// 		t.t.Error("S0r2 should not exist")
// 	}

// 	// verify Q is not addressable
// 	if _, q0Exists := t.conf.Address([]byte("Q0"), 0); q0Exists {
// 		t.t.Error("Q0 should not be addressable")
// 	}

// 	// verify N0h doesn't exist
// 	if _, n0hExists := t.conf.Address([]byte("N0h"), 0); n0hExists {
// 		t.t.Error("N0h shouldn't exist")
// 	}
// }

// func (t *SimpleConfTest) Attribute() {
// 	// assumes t.conf has state after SimpleConfTest.Address:
// 	// [ROOT Economic news had little effect on financial market .]
// 	// POS:					VB			NN
// 	//   0      1      2    3    4      5    6      7       8    9
// 	// Set up configuration:
// 	// C=(	[ROOT,had,effect], [.], A)
// 	// A={	(ROOT,	PRED,	had)
// 	// 		(had,	OBJ,	effect)
// 	// 		(effect,ATT,	little)
// 	//		(effect,ATT,	on)}

// 	s0, _ := t.conf.Address([]byte("S0"), 0)
// 	s1, _ := t.conf.Address([]byte("S1"), 1)
// 	n0, _ := t.conf.Address([]byte("N0"), 0)

// 	// unknown address fails
// 	if _, unkExists := t.conf.Attribute('S', -1, nil); unkExists {
// 		t.t.Error("Out of range nodeid -1 exists")
// 	}
// 	if _, unkExists := t.conf.Attribute('S', len(t.conf.Nodes), nil); unkExists {
// 		t.t.Error("Out of range nodeid>NumberOfNodes exists")
// 	}
// 	// unknown attribute fails
// 	if _, zExists := t.conf.Attribute('S', s0, []byte("z")); zExists {
// 		t.t.Error("Unknown attribute z exists")
// 	}
// 	// d: distance between S0 and N0
// 	if d, dExists := t.conf.Attribute('S', s0, []byte("d")); !dExists || d != 4 {
// 		if !dExists {
// 			t.t.Error("Expected d")
// 		} else {
// 			t.t.Error("Expected d = 4, got", d)
// 		}
// 	}
// 	// w: word
// 	if w, wExists := t.conf.Attribute('S', s0, []byte("w")); !wExists || EWord.ValueOf(w.(int)) != "effect" {
// 		if !wExists {
// 			t.t.Error("Expected w")
// 		} else {
// 			t.t.Error("Expected S0w = effect, got", EWord.ValueOf(w.(int)))
// 		}
// 	}
// 	// p: part-of-speech
// 	if p, pExists := t.conf.Attribute('S', s0, []byte("p")); !pExists || EPOS.ValueOf(p.(int)) != "NN" {
// 		if !pExists {
// 			t.t.Error("Expected p")
// 		} else {
// 			t.t.Error("Expected S0p = NN, got", EPOS.ValueOf(p.(int)))
// 		}
// 	}
// 	// p: part-of-speech
// 	if p, pExists := t.conf.Attribute('S', s1, []byte("p")); !pExists || EPOS.ValueOf(p.(int)) != "VB" {
// 		if !pExists {
// 			t.t.Error("Expected p")
// 		} else {
// 			t.t.Error("Expected S1p = VB, got", EPOS.ValueOf(p.(int)))
// 		}
// 	}
// 	// l: arc label/relation
// 	if l, lExists := t.conf.Attribute('S', s0, []byte("l")); !lExists || string(TEST_ENUM_RELATIONS.ValueOf(l.(int)).(nlp.DepRel)) != "OBJ" {
// 		if !lExists {
// 			t.t.Error("Expected l")
// 		} else {
// 			t.t.Error("Expected S0l = OBJ, got", TEST_ENUM_RELATIONS.ValueOf(l.(int)))
// 		}
// 	}
// 	// l: arc label/relation
// 	if l, lExists := t.conf.Attribute('S', s1, []byte("l")); !lExists || string(TEST_ENUM_RELATIONS.ValueOf(l.(int)).(nlp.DepRel)) != "PRED" {
// 		if !lExists {
// 			t.t.Error("Expected l")
// 		} else {
// 			t.t.Error("Expected S1l = PRED, got", TEST_ENUM_RELATIONS.ValueOf(l.(int)))
// 		}
// 	}
// 	// v[l|r]: valence left/right; number of left/right modifiers
// 	if vl, vlExists := t.conf.Attribute('S', s0, []byte("vl")); !vlExists || vl != 1 {
// 		if !vlExists {
// 			t.t.Error("Expected vl")
// 		} else {
// 			t.t.Error("Expected S0vl = 1, got", vl)
// 		}
// 	}
// 	if vr, vrExists := t.conf.Attribute('S', s0, []byte("vr")); !vrExists || vr != 1 {
// 		if !vrExists {
// 			t.t.Error("Expected vr")
// 		} else {
// 			t.t.Error("Expected S0vr = 1, got", vr)
// 		}
// 	}
// 	if vl, vlExists := t.conf.Attribute('S', s1, []byte("vl")); !vlExists || vl != 0 {
// 		if !vlExists {
// 			t.t.Error("Expected vl")
// 		} else {
// 			t.t.Error("Expected S1vl = 1, got", vl)
// 		}
// 	}
// 	if vr, vrExists := t.conf.Attribute('S', s1, []byte("vr")); !vrExists || vr != 1 {
// 		if !vrExists {
// 			t.t.Error("Expected vr")
// 		} else {
// 			t.t.Error("Expected S1vr = 1, got", vr)
// 		}
// 	}
// 	// s[l|r]: left right modifier sets
// 	if sl, slExists := t.conf.Attribute('S', s0, []byte("sl")); !slExists || t.conf.ERel.ValueOf(sl.(int)).(nlp.DepRel) != nlp.DepRel("ATT") {
// 		if !slExists {
// 			t.t.Error("Expected sl")
// 		} else {
// 			t.t.Error("Expected S0sl = ATT, got", sl)
// 		}
// 	}
// 	if sr, srExists := t.conf.Attribute('S', s0, []byte("sr")); !srExists || t.conf.ERel.ValueOf(sr.(int)).(nlp.DepRel) != nlp.DepRel("ATT") {
// 		if !srExists {
// 			t.t.Error("Expected sr")
// 		} else {
// 			t.t.Error("Expected S0sr = ATT, got", sr)
// 		}
// 	}
// 	if sl, slExists := t.conf.Attribute('S', s1, []byte("sl")); !slExists || sl != nil {
// 		if !slExists {
// 			t.t.Error("Expected sl")
// 		} else {
// 			t.t.Error("Expected S1sl = <nil>, got", sl)
// 		}
// 	}
// 	if sr, srExists := t.conf.Attribute('S', s1, []byte("sr")); !srExists || t.conf.ERel.ValueOf(sr.(int)).(nlp.DepRel) != nlp.DepRel("OBJ") {
// 		if !srExists {
// 			t.t.Error("Expected sr")
// 		} else {
// 			t.t.Error("Expected S1sr = OBJ, got", sr)
// 		}
// 	}

// 	// test empty cases

// 	// l: arc label/relation
// 	if _, lExists := t.conf.Attribute('N', n0, []byte("l")); lExists {
// 		t.t.Error("N0l should not exist")
// 	}
// 	t.conf.Queue().Clear()

// 	// d: distance between S0 and N0
// 	if _, dExists := t.conf.Attribute('S', s0, []byte("d")); dExists {
// 		t.t.Error("distance should not exist")
// 	}

// 	// try badly formatted existing attributes
// 	if _, vExists := t.conf.Attribute('S', s1, []byte("v")); vExists {
// 		t.t.Error("Missing direction v attribute should not exist")
// 	}
// 	if _, sExists := t.conf.Attribute('S', s0, []byte("s")); sExists {
// 		t.t.Error("Missing direction s attribute should not exist")
// 	}
// }

// func (test *SimpleConfTest) All() {
// 	// grouped by dependent changes to t.conf

// 	// test basic configuration functionality
// 	test.Init()
// 	test.Terminal()
// 	test.Copy()

// 	test.SetLastTransition()

// 	// test getters
// 	test.Arcs()
// 	test.GetArc()
// 	test.GetDirectedEdge()
// 	test.GetEdge()
// 	test.GetEdges()
// 	test.GetLabeledArc()
// 	test.GetNode()
// 	test.GetSequence()
// 	test.GetVertex()
// 	test.GetVertices()
// 	test.ID()
// 	test.NumberOfArcs()
// 	test.NumberOfEdges()
// 	test.NumberOfNodes()
// 	test.NumberOfVertices()
// 	test.Previous()
// 	test.Queue()
// 	test.Stack()

// 	// test output strings
// 	test.String()
// 	test.StringArcs()
// 	test.StringQueue()
// 	test.StringStack()

// 	// test feature functions
// 	test.Address()
// 	test.Attribute()
// }

// func TestSimpleConfiguration(t *testing.T) {
// 	SetupEagerTransEnum()
// 	SetupTestEnum()
// 	conf := &SimpleConfiguration{
// 		EWord:  EWord,
// 		EPOS:   EPOS,
// 		EWPOS:  EWPOS,
// 		ERel:   TEST_ENUM_RELATIONS,
// 		ETrans: TRANSITIONS_ENUM,
// 	}
// 	test := SimpleConfTest{conf, t}
// 	test.All()
// }
