package url

import (
	"sort"
	"strings"
)

// RawQueryParameter is one serialized query term. Name and Value retain their
// exact spelling. HasEquals distinguishes a bare term such as "flag" from an
// empty value such as "flag=".
type RawQueryParameter struct {
	Name      string
	Value     string
	HasEquals bool
}

// RawQuery is a lossless sequence of serialized query terms. Unlike
// SearchParams, RawQuery does not percent-decode, translate '+', discard empty
// terms, or form-serialize values.
//
// Use SearchParams when the query represents form data. Use RawQuery when its
// serialized spelling matters.
//
// This API is an EXPERIMENTAL extension and is not part of the WHATWG URL API.
type RawQuery []RawQueryParameter

// ParseRawQuery parses a serialized query without a leading '?'. It splits on
// '&' and the first '=' only, retaining empty terms and exact byte spelling.
func ParseRawQuery(raw string) RawQuery {
	query := make(RawQuery, 0, strings.Count(raw, "&")+1)
	for term := range strings.SplitSeq(raw, "&") {
		name, value, hasEquals := strings.Cut(term, "=")
		query = append(query, RawQueryParameter{
			Name:      name,
			Value:     value,
			HasEquals: hasEquals,
		})
	}
	return query
}

// String serializes q without a leading '?'.
func (q RawQuery) String() string {
	var out strings.Builder
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
func (q RawQuery) Sort() {
	sort.SliceStable(q, func(i, j int) bool {
		return q[i].Name < q[j].Name
	})
}

// SortAbsolute stably sorts q by each complete serialized term.
func (q RawQuery) SortAbsolute() {
	sort.SliceStable(q, func(i, j int) bool {
		return q[i].serializedTerm() < q[j].serializedTerm()
	})
}

// Filter removes parameters for which keep returns false. It never decodes or
// re-encodes data. A nil keep function leaves q unchanged.
func (q *RawQuery) Filter(keep func(name, value string, hasEquals bool) bool) {
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

func (p RawQueryParameter) serializedTerm() string {
	if !p.HasEquals {
		return p.Name
	}
	return p.Name + "=" + p.Value
}
