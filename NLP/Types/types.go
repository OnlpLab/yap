package Types

import (
	"chukuparser/Util"
	"reflect"
)

const ROOT_TOKEN = "ROOT"

type Token string

type EnumToken struct {
	Token Token
	Enum  int
}

type TaggedToken struct {
	Token, POS string
}

type EnumTaggedToken struct {
	TaggedToken
	EToken, EPOS, ETPOS int
}

type Sentence interface {
	Util.Equaler
	Tokens() []string
}

type EnumSentence interface {
	Util.Equaler
	Tokens() []EnumToken
}

type TaggedSentence interface {
	Sentence
	TaggedTokens() []TaggedToken
}

type EnumTaggedSentence interface {
	TaggedSentence
	EnumTaggedTokens() []EnumTaggedToken
}

type BasicTaggedSentence []TaggedToken

var _ TaggedSentence = BasicTaggedSentence{}

func (b BasicTaggedSentence) Tokens() []string {
	tokens := make([]string, len(b))
	for i, token := range b {
		tokens[i] = token.Token
	}
	return tokens
}

func (b BasicTaggedSentence) TaggedTokens() []TaggedToken {
	return []TaggedToken(b)
}

func (b BasicTaggedSentence) Equal(otherEq Util.Equaler) bool {
	asTagged := otherEq.(BasicTaggedSentence)
	return reflect.DeepEqual(b, asTagged)
}

type BasicETaggedSentence []EnumTaggedToken

var _ EnumTaggedSentence = BasicETaggedSentence{}

func (b BasicETaggedSentence) Tokens() []string {
	tokens := make([]string, len(b))
	for i, token := range b {
		tokens[i] = token.Token
	}
	return tokens
}

func (b BasicETaggedSentence) TaggedTokens() []TaggedToken {
	tokens := make([]TaggedToken, len(b))
	for i, token := range b {
		tokens[i] = token.TaggedToken
	}
	return tokens
}

func (b BasicETaggedSentence) EnumTaggedTokens() []EnumTaggedToken {
	return []EnumTaggedToken(b)
}

func (b BasicETaggedSentence) Equal(otherEq Util.Equaler) bool {
	asTagged := otherEq.(BasicETaggedSentence)
	return reflect.DeepEqual(b, asTagged)
}
