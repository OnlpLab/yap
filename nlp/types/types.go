package types

import (
	"reflect"
	"strings"
	"github.com/OnlpLab/yap/util"
)

const (
	ROOT_TOKEN = "ROOT"
	ROOT_LABEL = "ROOT"
)

type Token string

func (t Token) Signature() string {
	return util.Signature(string(t))
}

func (t Token) Prefixes(n int) []interface{} {
	prefixes := make([]interface{}, 0, n)
	for i := 0; i < util.Min(n, len(t)); i++ {
		prefixes = append(prefixes, util.Prefix(string(t), i+1))
	}
	return prefixes
}

func (t Token) Suffixes(n int) []interface{} {
	suffixes := make([]interface{}, 0, n)
	for i := 0; i < util.Min(n, len(t)); i++ {
		suffixes = append(suffixes, util.Suffix(string(t), i+1))
	}
	return suffixes
}

type EnumToken struct {
	Token Token
	Enum  int
}

type TaggedToken struct {
	Token, Lemma, POS string
}

type EnumTaggedToken struct {
	TaggedToken
	EToken, ELemma, EPOS, ETPOS, EMHost, EMSuffix int
}

type Sentence interface {
	util.Equaler
	Tokens() []string
}

type BasicSentence []Token

func (b BasicSentence) Tokens() []string {
	retval := make([]string, len(b))
	for i, val := range b {
		retval[i] = string(val)
	}
	return retval
}

func (b BasicSentence) Joined(sep string) string {
	temp := make([]string, len(b))
	for i, v := range b {
		temp[i] = string(v)
	}
	return strings.Join(temp, sep)
}

func (b BasicSentence) Equal(other interface{}) bool {
	asBasic := other.(BasicSentence)
	return reflect.DeepEqual(b, asBasic)
}

type EnumSentence interface {
	util.Equaler
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

func (b BasicTaggedSentence) Equal(otherEq util.Equaler) bool {
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

func (b BasicETaggedSentence) Equal(otherEq util.Equaler) bool {
	asTagged := otherEq.(BasicETaggedSentence)
	return reflect.DeepEqual(b, asTagged)
}
