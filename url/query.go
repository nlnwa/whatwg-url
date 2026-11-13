package url

import (
	"slices"
	"strings"
)

// QueryParameter is one serialized query term. Name and Value retain their
// exact spelling. HasEquals distinguishes a bare term such as "flag" from an
// empty value such as "flag=".
type QueryParameter struct {
	Name      string
	Value     string
	HasEquals bool
}

// Query is a lossless sequence of serialized query terms. Unlike
// SearchParams, Query does not percent-decode, translate '+', discard empty
// terms, or form-serialize values.
//
// Use SearchParams when the query represents form data. Use Query when its
// serialized spelling matters. Use Url.RewriteQuery to edit terms while
// preserving query presence and synchronizing an existing SearchParams object.
//
// This API is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
type Query []QueryParameter

// ParseQuery parses a serialized query without a leading '?'. It splits on
// '&' and the first '=' only, retaining empty terms and exact byte spelling.
func ParseQuery(raw string) Query {
	query := make(Query, 0, strings.Count(raw, "&")+1)
	for term := range strings.SplitSeq(raw, "&") {
		name, value, hasEquals := strings.Cut(term, "=")
		query = append(query, QueryParameter{
			Name:      name,
			Value:     value,
			HasEquals: hasEquals,
		})
	}
	return query
}

// String serializes q without a leading '?'.
func (q Query) String() string {
	var out strings.Builder
	length := max(0, len(q)-1)
	for _, parameter := range q {
		length += parameter.serializedLen()
	}
	out.Grow(length)
	for i, parameter := range q {
		if i > 0 {
			out.WriteByte('&')
		}
		out.WriteString(parameter.Name)
		if parameter.HasEquals {
			out.WriteByte('=')
			out.WriteString(parameter.Value)
		}
	}
	return out.String()
}

// Sort stably sorts q by the raw parameter name.
func (q Query) Sort() {
	slices.SortStableFunc(q, func(a, b QueryParameter) int {
		return strings.Compare(a.Name, b.Name)
	})
}

// SortAbsolute stably sorts q by each complete serialized term.
func (q Query) SortAbsolute() {
	slices.SortStableFunc(q, compareQueryTerms)
}

// Filter removes parameters for which keep returns false. It never decodes or
// re-encodes data. A nil keep function leaves q unchanged.
func (q *Query) Filter(keep func(name, value string, hasEquals bool) bool) {
	if q == nil || keep == nil {
		return
	}
	filtered := (*q)[:0]
	for _, parameter := range *q {
		if keep(parameter.Name, parameter.Value, parameter.HasEquals) {
			filtered = append(filtered, parameter)
		}
	}
	*q = filtered
}

func (p QueryParameter) serializedLen() int {
	if p.HasEquals {
		return len(p.Name) + 1 + len(p.Value)
	}
	return len(p.Name)
}

func (p QueryParameter) serializedByte(i int) byte {
	if i < len(p.Name) {
		return p.Name[i]
	}
	if i == len(p.Name) {
		return '='
	}
	return p.Value[i-len(p.Name)-1]
}

func compareQueryTerms(a, b QueryParameter) int {
	// Compare the common name prefix in bulk, then the virtual serialized
	// suffix. Comparing whole names first would misorder "a=z" and "a!".
	n := min(len(a.Name), len(b.Name))
	if order := strings.Compare(a.Name[:n], b.Name[:n]); order != 0 {
		return order
	}
	aLen, bLen := a.serializedLen(), b.serializedLen()
	for i := n; i < min(aLen, bLen); i++ {
		if ac, bc := a.serializedByte(i), b.serializedByte(i); ac != bc {
			return int(ac) - int(bc)
		}
	}
	return aLen - bLen
}
