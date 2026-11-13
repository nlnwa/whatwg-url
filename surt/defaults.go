package surt

import (
	"github.com/nationallibraryofnorway/whatwg-url/canonicalizer"
	"github.com/nationallibraryofnorway/whatwg-url/url"
)

// Named profiles are selected only here. Formatting consumes URL components;
// String consumes the parser interface, including caller-supplied profiles.
var defaultSURTParser url.Parser = canonicalizer.IA()

// SSURT parses without applying IA canonicalization.
var defaultSSURTParser url.Parser = canonicalizer.IA().Parser()
