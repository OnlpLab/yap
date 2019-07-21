package joint

import (
	"log"

	. "github.com/OnlpLab/yap/alg/transition"
	. "github.com/OnlpLab/yap/nlp/types"
	"github.com/OnlpLab/yap/util"

	"strings"
	dep "github.com/OnlpLab/yap/nlp/parser/dependency/transition"
	morph "github.com/OnlpLab/yap/nlp/parser/dependency/transition/morph"
	"github.com/OnlpLab/yap/nlp/parser/disambig"
)

var (
	TSAllOut         bool
	JointStrategies  string
	OracleStrategies string
)

func init() {
	jointStrategies := []string{"MDFirst", "All", "ArcGreedy"}
	oracleStrategies := []string{"MDFirst", "ArcGreedy"}
	JointStrategies = strings.Join(jointStrategies, ", ")
	OracleStrategies = strings.Join(oracleStrategies, ", ")
}

type JointTrans struct {
	MDTrans       TransitionSystem
	ArcSys        TransitionSystem
	Transitions   *util.EnumSet
	oracle        Oracle
	JointStrategy string
	MDTransition  Transition
	Log           bool
}

var _ TransitionSystem = &JointTrans{}

func (t *JointTrans) Transition(from Configuration, transition Transition) Configuration {
	// TODO: inefficient double copying of internal configurations by underlying
	// transition systems
	c := from.Copy().(*JointConfig)
	if transition.Type() == 'M' || transition.Type() == 'P' || transition.Type() == 'L' {
		t.MDTrans.(*disambig.MDTrans).Log = t.Log
		// log.Println("Applying transition", t.Transitions.ValueOf(transition.Value()), "to\n", c.MDConfig)
		c.MDConfig = *t.MDTrans.Transition(&c.MDConfig, transition).(*disambig.MDConfig)
		// log.Println("MD Config is now:\n", c.MDConfig)
		if transition.Type() == 'M' && len(c.MDConfig.Morphemes) > len(from.(*JointConfig).MDConfig.Morphemes) {
			// if a new morpheme was disambiguated
			// enqueue last disambiguated morpheme
			// and add as "node"
			nodeId := len(c.Nodes)
			if nodeId != len(c.MDConfig.Morphemes)-1 {
				// log.Println("With original config", from)
				// log.Println("Currently", c)
				// log.Println("Sequence", c.GetSequence())
				// log.Println("Got nodeId", nodeId, "but len(c.MDConfig.Morphemes)-1 is different:", len(c.MDConfig.Morphemes)-1)
				log.Println("Nodes is", c.Nodes, "with morphemes", c.MDConfig.Morphemes)
				panic("Mismatch between Nodes and Morphemes")
			}
			curMorpheme := c.MDConfig.Morphemes[nodeId]
			c.SimpleConfiguration.Queue().Enqueue(nodeId)
			newNode := &dep.TaggedDepNode{
				nodeId,
				curMorpheme.EForm,
				curMorpheme.EPOS,
				curMorpheme.EFCPOS,
				curMorpheme.EMHost,
				curMorpheme.EMSuffix,
				curMorpheme.Form,
				curMorpheme.Lemma,
				curMorpheme.POS,
			}

			c.SimpleConfiguration.Nodes = append(c.SimpleConfiguration.Nodes,
				dep.NewArcCachedDepNode(DepNode(newNode)))
			c.Assign(c.MDConfig.Assignment())

		}
	} else {
		c.SimpleConfiguration = *t.ArcSys.Transition(&c.SimpleConfiguration, transition).(*dep.SimpleConfiguration)
		c.Assign(c.SimpleConfiguration.Assignment())
	}
	c.SetLastTransition(transition)
	// paramStr := t.Transitions.ValueOf(int(transition))
	return c
}

func (t *JointTrans) TransitionTypes() []string {
	return append(t.MDTrans.TransitionTypes(), t.ArcSys.TransitionTypes()...)
}

func (t *JointTrans) TransitionStrategy(c *JointConfig) (shouldMD bool, shouldDep bool) {
	shouldMD = false
	shouldDep = false
	switch t.JointStrategy {
	case "All":
		shouldMD = true
		shouldDep = true
	case "MDFirst":
		if !c.MDConfig.Terminal() {
			shouldMD = true
		} else {
			shouldDep = true
		}
	case "ArcGreedy":
		if c.SimpleConfiguration.Queue().Size() < 3 && !c.MDConfig.Terminal() {
			shouldMD = true
		} else {
			shouldDep = true
		}
	default:
		panic("Unknown transition strategy: " + t.JointStrategy)
	}
	if !(shouldMD || shouldDep) && !(c.MDConfig.Terminal() && c.SimpleConfiguration.Terminal()) {
		panic("One of the underlying configurations is not terminal but no transition type specified")
	}
	return
}

func (t *JointTrans) GetTransitions(from Configuration) (byte, []int) {
	retval := make([]int, 0, 10)
	tType, transitions := t.YieldTransitions(from)
	for transition := range transitions {
		retval = append(retval, int(transition))
	}
	return tType, retval
}

func (t *JointTrans) YieldTransitions(conf Configuration) (byte, chan int) {
	// Note: Even though we could send transitions of more than one type,
	// the system is limited to only *one* type of transition per candidate
	c := conf.(*JointConfig)
	shouldMD, shouldDep := t.TransitionStrategy(c)
	if shouldMD && shouldDep {
		panic("System does not currenlty support a mixed strategy, choose a single transition type")
	}
	if shouldMD {
		return t.MDTrans.YieldTransitions(&c.MDConfig)
	}
	if shouldDep {
		return t.ArcSys.YieldTransitions(&c.SimpleConfiguration)
	}
	transitions := make(chan int)
	close(transitions)
	return '?', transitions
}

func (t *JointTrans) Oracle() Oracle {
	return t.oracle
}

func (t *JointTrans) AddDefaultOracle() {
	t.oracle = &JointOracle{
		JointStrategy: t.JointStrategy,
		MDOracle:      t.MDTrans.Oracle(),
		ArcSysOracle:  t.ArcSys.Oracle(),
	}
}

func (t *JointTrans) Name() string {
	return "Joint Morpho-Syntactic [MD:" +
		t.MDTrans.Name() +
		", ArcSys:" +
		t.ArcSys.Name() +
		"] - Strategy: " + t.JointStrategy
}

type JointOracle struct {
	gold           *morph.BasicMorphGraph
	MDOracle       Oracle
	ArcSysOracle   Oracle
	JointStrategy  string
	OracleStrategy string
}

var _ Decision = &JointOracle{}

func (o *JointOracle) SetGold(g interface{}) {
	graph, ok := g.(*morph.BasicMorphGraph)
	if !ok {
		panic("Gold is not a morph.BasicMorphGraph")
	}
	o.gold = graph

	o.MDOracle.SetGold(graph.Mappings)
	o.ArcSysOracle.SetGold(&graph.BasicDepGraph)
}

func (o *JointOracle) MDFirst(conf Configuration) Transition {
	c, ok := conf.(*JointConfig)
	if !ok {
		panic("Conf must be *JointConfig")
	}
	if !c.MDConfig.Terminal() {
		return o.MDOracle.Transition(&c.MDConfig)
	} else {
		return o.ArcSysOracle.Transition(&c.SimpleConfiguration)
	}
}

func (o *JointOracle) ArcGreedy(conf Configuration) Transition {
	c, ok := conf.(*JointConfig)
	if !ok {
		panic("Conf must be *JointConfig")
	}
	if c.SimpleConfiguration.Queue().Size() < 3 && !c.MDConfig.Terminal() {
		return o.MDOracle.Transition(&c.MDConfig)
	} else {
		return o.ArcSysOracle.Transition(&c.SimpleConfiguration)
	}
}

func (o *JointOracle) Transition(conf Configuration) Transition {
	if o.gold == nil {
		panic("Oracle needs gold reference, use SetGold")
	}

	switch o.OracleStrategy {
	case "MDFirst":
		return o.MDFirst(conf)
	case "ArcGreedy":
		return o.ArcGreedy(conf)
	default:
		panic("Unknown oracle strategy: " + o.OracleStrategy)
	}

	return ConstTransition(0)
}

func (o *JointOracle) Name() string {
	return "Joint Morpho-Syntactic - Strategy: " + o.OracleStrategy
}
