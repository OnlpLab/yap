package dependency

import (
	TransitionModel "github.com/OnlpLab/yap/alg/transition/model"
	. "github.com/OnlpLab/yap/nlp/types"
)

type ConstraintModel interface{}

type ParameterModelValue interface {
	// Increment(interface{})

	// Copy() ParameterModelValue
	Clear()
}

type ParameterModel interface {
}

type TransitionParameterModel interface {
	ParameterModel
	TransitionModel.Interface
}

type DependencyParser interface {
	Parse(Sentence, ConstraintModel, ParameterModel) (DependencyGraph, interface{})
}

type Dependency struct {
	Constraints ConstraintModel
	Parameters  ParameterModel
	Parser      DependencyParser
}

func (d *Dependency) Parse(sent Sentence) (DependencyGraph, interface{}) {
	return d.Parser.Parse(sent, d.Constraints, d.Parameters)
}
