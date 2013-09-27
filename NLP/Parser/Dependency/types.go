package Dependency

import (
	TransitionModel "chukuparser/Algorithm/Transition/Model"
	. "chukuparser/NLP/Types"
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
	TransitionModel() TransitionModel.Interface
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
