package canonicalizer

import "github.com/nationallibraryofnorway/whatwg-url/url"

// LaxPathPercentEncodeSet is the path escaping preset used by lax
// canonicalization profiles. It permits '<', '>', and '^' in addition to the
// characters permitted by url.PathPercentEncodeSet.
var LaxPathPercentEncodeSet = url.PathPercentEncodeSet.Clear(0x2e, 0x3c, 0x3e, 0x5e)

// LaxQueryPercentEncodeSet is the query escaping preset used by lax
// canonicalization profiles. It permits '"' in addition to the characters
// permitted by url.QueryPercentEncodeSet.
var LaxQueryPercentEncodeSet = url.QueryPercentEncodeSet.Clear(0x22, 0x25, 0x2f, 0x3b, 0x3f, 0x7b)

// RepeatedQueryPercentEncodeSet escapes query names and values after repeated
// percent decoding. It protects '#', '%', '&', and '=', along with controls,
// spaces, and non-ASCII bytes, while retaining literal '+'.
var RepeatedQueryPercentEncodeSet = url.C0OrSpacePercentEncodeSet.Set('#', '%', '&', '=')
