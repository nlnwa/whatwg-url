package whatwg

import "github.com/nationallibraryofnorway/whatwg-url/internal"

type Url = internal.Url
type ParserOption = internal.ParserOption

type Parser interface {
	BasicParser(urlOrRef string, base *Url, url *Url, stateOverride State) (*Url, error)
}

func NewParser(opts ...ParserOption) Parser {
	p := internal.NewParser()
	for _, opt := range opts {
		opt.Apply(p)
	}
	return p
}

type State = internal.State

const (
	NoState State = iota
	StateSchemeStart
	StateScheme
	StateNoScheme
	StateOpaquePath
	StateSpecialRelativeOrAuthority
	StateSpecialAuthoritySlashes
	StateSpecialAuthorityIgnoreSlashes
	StatePathOrAuthority
	StateAuthority
	StateHost
	StateHostname
	StateFile
	StateFileHost
	StateFileSlash
	StatePort
	StatePath
	StatePathStart
	StateQuery
	StateFragment
	StateRelative
	StateRelativeSlash
)
