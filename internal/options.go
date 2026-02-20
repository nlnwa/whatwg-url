package internal

type ParserOption interface {
	Apply(*Parser)
}

type CanonicalizerOption interface {
	ParserOption
	ApplyProfile(*Canonicalizer)
}

func NewParserOption(f func(*Parser)) parserOption {
	return parserOption{fn: f}
}

func NewCanonicalizerOption(f func(*Canonicalizer)) canonicalizerOption {
	return canonicalizerOption{fn: f}
}

type parserOption struct {
	fn func(*Parser)
}

func (opt parserOption) Apply(parser *Parser) {
	opt.fn(parser)
}

type canonicalizerOption struct {
	fn func(*Canonicalizer)
}

func (canonicalizerOption) Apply(*Parser) {}

func (opt canonicalizerOption) ApplyProfile(canonicalizer *Canonicalizer) {
	opt.fn(canonicalizer)
}
