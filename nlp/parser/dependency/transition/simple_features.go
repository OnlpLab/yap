package transition

import (
	. "github.com/OnlpLab/yap/alg"
	// "log"
	// nlp "github.com/OnlpLab/yap/nlp/types"
	// "github.com/OnlpLab/yap/util"
	// "math"
	// "regexp"
	// "sort"
	// "strconv"
	// "strings"
)

const (
	SET_SEPARATOR = "-"
)

var _Zpar_Bug_N1N2 bool = false

func (c *SimpleConfiguration) Address(location []byte, sourceOffset int) (int, bool, bool) {
	source := c.GetSource(location[0])
	if source == nil {
		return 0, false, false
	}
	atAddress, exists := source.Index(int(sourceOffset))
	if !exists {
		// zpar bug parity
		if _Zpar_Bug_N1N2 && location[0] == 'N' && c.Queue().Size() == 0 && sourceOffset > 0 {
			return sourceOffset - 1, true, false
		}
		// end zpar bug parity
		return 0, false, false
	}
	// test if feature address is a generator of feature (e.g. for each child..)
	locationLen := len(location)
	if locationLen >= 4 {
		if string(location[2:4]) == "Ci" {
			return atAddress, true, true
		}
	}

	location = location[2:]
	if len(location) == 0 {
		return atAddress, true, false
	}
	switch location[0] {
	case 'l', 'r':
		leftMods, rightMods := c.GetModifiers(atAddress)
		if location[0] == 'l' {
			if len(leftMods) == 0 {
				return 0, false, false
			}
			if len(location) > 1 && location[1] == '2' {
				if len(leftMods) > 1 {
					return leftMods[1], true, false
				}
			} else {
				return leftMods[0], true, false
			}
		} else {
			if len(rightMods) == 0 {
				return 0, false, false
			}
			if len(location) > 1 && location[1] == '2' {
				if len(rightMods) > 1 {
					return rightMods[len(rightMods)-2], true, false
				}
			} else {
				return rightMods[len(rightMods)-1], true, false
			}
		}
	case 'h':
		head, headExists := c.GetHead(atAddress)
		if headExists {
			if len(location) > 1 && location[1] == '2' {
				headOfHead, headOfHeadExists := c.GetHead(head.ID())
				if headOfHeadExists {
					return headOfHead.ID(), true, false
				}
			} else {
				return head.ID(), true, false
			}
		}
	}
	return 0, false, false
}

func (c *SimpleConfiguration) GenerateAddresses(nodeID int, location []byte) (nodeIDs []int) {
	if nodeID < 0 || nodeID >= len(c.Nodes) {
		return
	}
	if string(location[2:4]) == "Ci" {
		leftChildren, rightChildren := c.GetModifiers(nodeID)
		numLeft := len(leftChildren)
		nodeIDs = make([]int, numLeft+len(rightChildren))
		for i, leftChild := range leftChildren {
			nodeIDs[i] = leftChild
		}
		for j, rightChild := range rightChildren {
			nodeIDs[j+numLeft] = rightChild
		}
		return
	}
	return
}

func (c *SimpleConfiguration) GetModifierLabel(modifierID int) (int, bool, bool) {
	arcs := c.Arcs().Get(&BasicDepArc{-1, -1, modifierID, ""})
	if len(arcs) > 0 {
		index, _ := c.ERel.IndexOf(arcs[0].GetRelation())
		return index, true, false
	}
	return 0, false, false
}

func (c *SimpleConfiguration) Attribute(source byte, nodeID int, attribute []byte, transitions []int) (att interface{}, exists bool, isGen bool) {
	if nodeID < 0 || nodeID >= len(c.Nodes) {
		return 0, false, false
	}
	exists = true
	switch attribute[0] {
	case 'o':
		att = c.NumHeadStack
		return
	case 'd':
		return c.GetConfDistance()
	case 'w':
		node := c.GetRawNode(nodeID)
		if len(attribute) > 1 && attribute[1] == 'p' {
			att = node.TokenPOS
			return
		} else {
			att = node.Token
			return
		}
	case 'p':
		node := c.GetRawNode(nodeID)
		// TODO: CPOS
		att = node.POS
		return
	case 'l':
		//		relation, relExists :=
		return c.GetModifierLabel(nodeID)
	case 'v':
		if len(attribute) != 2 {
			return 0, false, false
		}
		leftMods, rightMods := c.GetNumModifiers(nodeID)
		switch attribute[1] {
		case 'l':
			att = leftMods
		case 'r':
			att = rightMods
		case 'f':
			att = leftMods + rightMods
		}
		return
	case 's':
		if len(attribute) != 2 {
			return 0, false, false
		}
		leftLabelSet, rightLabelSet, allLabels := c.GetModifierLabelSets(nodeID)
		switch attribute[1] {
		case 'l':
			att = leftLabelSet
		case 'r':
			att = rightLabelSet
		case 'f':
			att = allLabels
		}
		return
	case 'f':
		if len(attribute) == 2 && attribute[1] == 'p' {
			allModPOS := c.GetModifiersPOS(nodeID)
			att = allModPOS
			return
		}
	case 'h':
		node := c.GetRawNode(nodeID)
		att = node.MHost
		return
	case 'x':
		node := c.GetRawNode(nodeID)
		att = node.MSuffix
		return
	}
	return 0, false, false
}

func (c *SimpleConfiguration) GetConfDistance() (int, bool, bool) {
	stackTop, stackExists := c.Stack().Peek()
	queueTop, queueExists := c.Queue().Peek()
	if stackExists && queueExists {
		dist := queueTop - stackTop
		// "normalize" to
		// 0 1 2 3 4 5 ... 10 ...
		// 0 1 2 3 4 ---5--  --- 6 ---
		if dist < 0 {
			dist = -dist
		}
		switch {
		case dist > 10:
			return 6, true, false
		case dist > 5:
			return 5, true, false
		default:
			return dist, true, false
		}
	}
	return 0, false, false
}

func (c *SimpleConfiguration) GetSource(location byte) Index {
	switch location {
	case 'N':
		return c.Queue()
	case 'S':
		return c.Stack()
	}
	return nil
}

func (c *SimpleConfiguration) GetHead(nodeID int) (*ArcCachedDepNode, bool) {
	head := c.Nodes[nodeID].Head
	if head == -1 {
		return nil, false
	}
	return c.Nodes[head], true
}

func (c *SimpleConfiguration) GetModifiers(nodeID int) ([]int, []int) {
	node := c.Nodes[nodeID]
	return node.LeftMods(), node.RightMods()
}

func (c *SimpleConfiguration) GetNumModifiers(nodeID int) (int, int) {
	node := c.Nodes[nodeID]
	return len(node.LeftMods()), len(node.RightMods())
}

func (c *SimpleConfiguration) GetModifierLabelSets(nodeID int) (interface{}, interface{}, interface{}) {
	node := c.Nodes[nodeID]
	return node.LeftLabelSet(), node.RightLabelSet(), node.AllLabelSet()
}

func (c *SimpleConfiguration) GetModifiersPOS(nodeID int) interface{} {
	node := c.Nodes[nodeID]
	return node.AllModPOS()
}
