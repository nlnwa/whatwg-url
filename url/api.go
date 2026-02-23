package url

import (
	"github.com/nationallibraryofnorway/whatwg-url/internal"
)

type Url = internal.Url
type NameValuePair = internal.NameValuePair
type SearchParams = internal.SearchParams
type ParserOption = internal.ParserOption

type Parser interface {
	Parse(rawUrl string) (*Url, error)
	ParseRef(rawUrl, ref string) (*Url, error)
}

func NewParser(opts ...ParserOption) Parser {
	p := internal.NewParser()
	for _, opt := range opts {
		opt.Apply(p)
	}
	return p
}

var defaultParser = internal.NewParser()

func Parse(rawUrl string) (*Url, error) {
	return defaultParser.Parse(rawUrl)
}

func ParseRef(rawUrl, ref string) (*Url, error) {
	return defaultParser.ParseRef(rawUrl, ref)
}

func PercentEncode(s string, tr *PercentEncodeSet) string {
	return internal.PercentEncode(s, tr)
}

type PercentEncodeSet = internal.PercentEncodeSet

var (
	C0PercentEncodeSet           *PercentEncodeSet = internal.C0PercentEncodeSet
	C0OrSpacePercentEncodeSet    *PercentEncodeSet = internal.C0OrSpacePercentEncodeSet
	FragmentPercentEncodeSet     *PercentEncodeSet = internal.FragmentPercentEncodeSet
	PathPercentEncodeSet         *PercentEncodeSet = internal.PathPercentEncodeSet
	QueryPercentEncodeSet        *PercentEncodeSet = internal.QueryPercentEncodeSet
	SpecialQueryPercentEncodeSet *PercentEncodeSet = internal.SpecialQueryPercentEncodeSet
	UserInfoPercentEncodeSet     *PercentEncodeSet = internal.UserInfoPercentEncodeSet
	HostPercentEncodeSet         *PercentEncodeSet = internal.HostPercentEncodeSet
)
