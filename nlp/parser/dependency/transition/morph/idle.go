package morph

// import (
// 	. "github.com/OnlpLab/yap/alg/transition"
// )
//
// type Idle struct {
// 	TransitionSystem TransitionSystem
// 	IDLE             Transition
// }
//
// var _ TransitionSystem = &Idle{}
//
// func (i *Idle) Transition(from Configuration, transition Transition) Configuration {
// 	// if transition[:2] == "ID" {
// 	if transition == i.IDLE {
// 		conf := from.Copy()
// 		conf.SetLastTransition(transition)
// 		return conf
// 	} else {
// 		return i.TransitionSystem.Transition(from, transition)
// 	}
// }
//
// func (i *Idle) TransitionTypes() []string {
// 	baseTypes := i.TransitionSystem.TransitionTypes()
// 	baseTypes = append(baseTypes, "IDLE")
// 	return baseTypes
// }
//
// func (a *Idle) GetTransitions(from Configuration) []int {
// 	retval := make([]int, 0, 10)
// 	transitions := a.YieldTransitions(from)
// 	for transition := range transitions {
// 		retval = append(retval, int(transition))
// 	}
// 	return retval
// }
//
// func (i *Idle) YieldTransitions(from Configuration) chan Transition {
// 	idleChan := make(chan Transition)
// 	go func() {
// 		// false is the zero value, setting explicitly for documentation
// 		var embeddedHasTransitions bool = false
// 		for path := range i.TransitionSystem.YieldTransitions(from) {
// 			embeddedHasTransitions = true
// 			idleChan <- Transition(path)
// 		}
// 		if !embeddedHasTransitions {
// 			idleChan <- i.IDLE
// 		}
// 		close(idleChan)
// 	}()
// 	return idleChan
// }
//
// func (i *Idle) AddDefaultOracle() {
// 	i.TransitionSystem.AddDefaultOracle()
// }
//
// func (i *Idle) Oracle() Oracle {
// 	return i.TransitionSystem.Oracle()
// }
//
// func (i *Idle) Name() string {
// 	return "Idle embedded with: " + i.TransitionSystem.Name()
// }
