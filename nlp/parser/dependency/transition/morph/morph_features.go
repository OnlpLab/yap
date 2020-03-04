package morph

// import (
// 	. "github.com/OnlpLab/yap/alg"
// 	nlp "github.com/OnlpLab/yap/nlp/types"
// 	// "github.com/OnlpLab/yap/util"
// 	"strings"
// )
//
// func (m *MorphConfiguration) Attribute(source byte, nodeID int, attribute []byte) (interface{}, bool) {
// 	switch source {
// 	// case 'A':
// 	// 	arc := m.Arcs().Last()
// 	// 	if arc == nil {
// 	// 		return 0, false
// 	// 	}
// 	// 	head, mod := m.GetMorpheme(arc.GetHead()), m.GetMorpheme(arc.GetModifier())
// 	// 	switch attribute[0] {
// 	// 	case 'g': // gen
// 	// 		val, exists := mod.Features["gen"]
// 	// 		other, otherExists := head.Features["gen"]
// 	// 		if exists && otherExists && len(val) == len(other) {
// 	// 			if val == other {
// 	// 				return 1, true
// 	// 			} else {
// 	// 				return 0, true
// 	// 			}
// 	// 		}
// 	// 		return 0, false
// 	// 	case 'n': // num
// 	// 		val, exists := mod.Features["num"]
// 	// 		other, otherExists := head.Features["num"]
// 	// 		if exists && otherExists {
// 	// 			if val == "D" || other == "D" {
// 	// 				return 1, true
// 	// 			}
// 	// 			if len(val) == len(other) {
// 	// 				if val == other {
// 	// 					return 1, true
// 	// 				} else {
// 	// 					return 0, true
// 	// 				}
// 	// 			}
// 	// 		}
// 	// 		return 0, false
// 	// 	case 'p': // per
// 	// 		val, exists := mod.Features["per"]
// 	// 		other, otherExists := head.Features["per"]
// 	// 		if exists && otherExists {
// 	// 			if val == "A" || other == "A" {
// 	// 				return 1, true
// 	// 			}
// 	// 			if val == other {
// 	// 				return 1, true
// 	// 			} else {
// 	// 				return 0, true
// 	// 			}
// 	// 		}
// 	// 		return 0, false
// 	// 	case 'o': // polar
// 	// 		val, exists := mod.Features["polar"]
// 	// 		other, otherExists := head.Features["polar"]
// 	// 		if exists && otherExists {
// 	// 			if val == other {
// 	// 				return 1, true
// 	// 			} else {
// 	// 				return 0, true
// 	// 			}
// 	// 		}
// 	// 		return 0, false
// 	// 	case 't': // tense
// 	// 		val, exists := mod.Features["tense"]
// 	// 		other, otherExists := head.Features["tense"]
// 	// 		if exists && otherExists {
// 	// 			if val == other {
// 	// 				return 1, true
// 	// 			} else {
// 	// 				return 0, true
// 	// 			}
// 	// 		}
// 	// 		return 0, false
// 	// 	default:
// 	// 		panic("Unknown attribute " + string(attribute))
// 	// 	}
// 	case 'M':
// 		latId, exists := m.LatticeQueue.Index(nodeID)
// 		if !exists {
// 			return 0, false
// 		}
// 		switch attribute[0] {
// 		case 'w':
// 			lat := m.Lattices[latId]
// 			return string(lat.Token), true
// 		default:
// 			panic("Unknown attribute " + string(attribute))
// 		}
// 	case 'N', 'S':
// 		if nodeID < 0 || nodeID >= len(m.Nodes) {
// 			return 0, false
// 		}
// 		switch attribute[0] {
// 		case 't':
// 			return m.GetQueueMorphs()
// 		case 'm':
// 			return m.GetMorphFeatures(nodeID), true
// 		case 'd':
// 			return m.GetConfDistance()
// 		case 'w':
// 			node := m.GetMorpheme(nodeID)
// 			if len(attribute) == 2 && attribute[1] == 'p' {
// 				return node.EFCPOS, true
// 			}
// 			return node.EForm, true
// 		case 'p':
// 			node := m.GetMorpheme(nodeID)
// 			return node.EPOS, true
// 		case 'l':
// 			//		relation, relExists :=
// 			return m.GetModifierLabel(nodeID)
// 		case 'v':
// 			if len(attribute) != 2 {
// 				return 0, false
// 			}
// 			leftMods, rightMods := m.GetNumModifiers(nodeID)
// 			switch attribute[1] {
// 			case 'l':
// 				return leftMods, true
// 			case 'r':
// 				return rightMods, true
// 			}
// 		case 's':
// 			if len(attribute) != 2 {
// 				return 0, false
// 			}
// 			leftLabelSet, rightLabelSet := m.GetModifierLabelSets(nodeID)
// 			switch attribute[1] {
// 			case 'l':
// 				return leftLabelSet, true
// 			case 'r':
// 				return rightLabelSet, true
// 			}
// 		default:
// 			panic("Unknown attribute " + string(attribute))
// 		}
// 	default:
// 		panic("Unknown attribute " + string(attribute))
// 	}
// 	return 0, false
// }
//
// func (m *MorphConfiguration) GetHead(nodeID int) (*nlp.EMorpheme, bool) {
// 	head := m.Nodes[nodeID].Head
// 	if head == -1 {
// 		return nil, false
// 	}
// 	return m.GetMorpheme(head), true
// }
//
// func (m *MorphConfiguration) GetModifierLabelSets(nodeID int) (interface{}, interface{}) {
// 	node := m.Nodes[nodeID]
// 	return node.LeftLabelSet(), node.RightLabelSet()
// }
//
// func (m *MorphConfiguration) GetModifiers(nodeID int) ([]int, []int) {
// 	node := m.Nodes[nodeID]
// 	return node.LeftMods(), node.RightMods()
// }
//
// func (m *MorphConfiguration) GetNumModifiers(nodeID int) (int, int) {
// 	node := m.Nodes[nodeID]
// 	return len(node.LeftMods()), len(node.RightMods())
// }
//
// func (m *MorphConfiguration) GetSource(location byte) Index {
// 	switch location {
// 	case 'N':
// 		return m.Queue()
// 	case 'S':
// 		return m.Stack()
// 	case 'M':
// 		return m.LatticeQueue
// 	default:
// 		panic("Unknown location " + string(location))
// 	}
// 	return nil
// }
//
// func (m *MorphConfiguration) GetMorphFeatures(nodeID int) int {
// 	node := m.Nodes[nodeID]
// 	return node.Node.(*nlp.EMorpheme).EFeatures
// }
//
// func (m *MorphConfiguration) GetQueueMorphs() (string, bool) {
// 	if m.Queue().Size() == 0 {
// 		return "", false
// 	}
// 	strs := make([]string, m.Queue().Size())
// 	for i := 0; i < m.Queue().Size(); i++ {
// 		atI := m.GetMorpheme(i)
// 		strs[i] = atI.CPOS
// 	}
// 	return strings.Join(strs, "-"), true
// }
//
// func (m *MorphConfiguration) GetConfDistance() (int, bool) {
// 	stackTop, stackExists := m.Stack().Peek()
// 	queueTop, queueExists := m.Queue().Peek()
// 	if stackExists && queueExists {
// 		dist := queueTop - stackTop
// 		// "normalize" to
// 		// 0 1 2 3 4 5 ... 10 ...
// 		// 0 1 2 3 4 ---5--  --- 6 ---
// 		if dist < 0 {
// 			dist = -dist
// 		}
// 		switch {
// 		case dist > 10:
// 			return 6, true
// 		case dist > 5:
// 			return 5, true
// 		default:
// 			return dist, true
// 		}
// 	}
// 	return 0, false
// }
//
// func (m *MorphConfiguration) GetModifierLabel(modifierID int) (int, bool) {
// 	label := m.Nodes[modifierID].ELabel
// 	if label >= 0 {
// 		return label, true
// 	} else {
// 		return label, false
// 	}
// }
//
// func (m *MorphConfiguration) Address(location []byte, sourceOffset int) (int, bool, bool) {
// 	s := m.GetSource(location[0])
// 	if s == nil {
// 		return 0, false, false
// 	}
// 	atAddress, exists := s.Index(int(sourceOffset))
// 	if !exists {
// 		return 0, false, false
// 	}
// 	location = location[2:]
// 	if len(location) == 0 {
// 		return atAddress, true, false
// 	}
// 	switch location[0] {
// 	case 'l', 'r':
// 		leftMods, rightMods := m.GetModifiers(atAddress)
// 		if location[0] == 'l' {
// 			if len(leftMods) == 0 {
// 				return 0, false, false
// 			}
// 			if len(location) > 1 && location[1] == '2' {
// 				if len(leftMods) > 1 {
// 					return leftMods[1], true, false
// 				}
// 			} else {
// 				return leftMods[0], true, false
// 			}
// 		} else {
// 			if len(rightMods) == 0 {
// 				return 0, false, false
// 			}
// 			if len(location) > 1 && location[1] == '2' {
// 				if len(rightMods) > 1 {
// 					return rightMods[len(rightMods)-2], true, false
// 				}
// 			} else {
// 				return rightMods[len(rightMods)-1], true, false
// 			}
// 		}
// 	case 'h':
// 		head, headExists := m.GetHead(atAddress)
// 		if headExists {
// 			if len(location) > 1 && location[1] == '2' {
// 				headOfHead, headOfHeadExists := m.GetHead(head.ID())
// 				if headOfHeadExists {
// 					return headOfHead.ID(), true, false
// 				}
// 			} else {
// 				return head.ID(), true, false
// 			}
// 		}
// 	}
// 	return 0, false, false
// }
