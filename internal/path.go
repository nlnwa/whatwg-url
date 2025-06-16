package internal

import "strings"

type path struct {
	p      []string
	opaque bool
}

func (p *path) isOpaque() bool {
	return p.opaque
}

func (p *path) isEmpty() bool {
	return len(p.p) == 0
}

func (p *path) setOpaque(opaquePath string) {
	p.opaque = true
	p.p = p.p[:0]
	p.p = append(p.p, opaquePath)
}

func (p *path) addSegment(segment string) {
	p.p = append(p.p, segment)
	p.opaque = false
}

func (p *path) init() {
	p.p = p.p[:0]
	p.opaque = false
}

func (p *path) shortenPath(scheme string) {
	if scheme == "file" && len(p.p) == 1 && isNormalizedWindowsDriveLetter(p.p[0]) {
		return
	}
	if len(p.p) == 0 {
		return
	}
	p.p = p.p[0 : len(p.p)-1]
}

func (p *path) stripTrailingSpacesIfOpaque() {
	if !p.opaque || len(p.p) == 0 {
		return
	}
	s := p.p[0]
	// fast check: no trailing space => no alloc
	if len(s) == 0 || s[len(s)-1] != ' ' {
		return
	}
	i := len(s)
	for i > 0 && s[i-1] == ' ' {
		i--
	}
	p.p[0] = s[:i] // slice header only; no alloc
}

func (p *path) clone() *path {
	if p == nil {
		return nil
	}

	newSegments := make([]string, len(p.p))
	copy(newSegments, p.p)

	return &path{
		opaque: p.opaque,
		p:      newSegments,
	}
}

func (p *path) String() string {
	if p.opaque && len(p.p) > 0 {
		return p.p[0]
	}
	size := 0
	for _, pp := range p.p {
		size += 1 + len(pp)
	}

	var sb strings.Builder
	sb.Grow(size)
	for _, pp := range p.p {
		sb.WriteByte('/')
		sb.WriteString(pp)
	}

	return sb.String()
}
