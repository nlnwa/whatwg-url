package internal

type ParserOption interface {
	Apply(*Parser)
}

type CanonicalizerOption interface {
	ApplyProfile(*Canonicalizer)
}

func NewParserOption(f func(*Parser)) parserOption {
	return parserOption{fn: f}
}

func NewCanonicalizerOption(f func(*Canonicalizer)) canonicalizerOption {
	return canonicalizerOption{fn: f}
}

type emptyParserOption struct{}

func (emptyParserOption) Apply(*Parser) {}

type parserOption struct {
	fn func(*Parser)
}

func (opt parserOption) Apply(parser *Parser) {
	opt.fn(parser)
}

type canonicalizerOption struct {
	emptyParserOption
	fn func(*Canonicalizer)
}

func (opt canonicalizerOption) ApplyProfile(canonicalizer *Canonicalizer) {
	opt.fn(canonicalizer)
}
