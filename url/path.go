package url

import "strings"

// path stores the serialized URL path. Hierarchical paths include their slash
// separators; opaque paths are stored exactly as serialized.
//
// During parsing, buf holds a mutable copy of value. finalize converts that
// buffer to an immutable string once parsing is complete.
type path struct {
	value  string
	buf    []byte
	opaque bool
	dirty  bool
}

func (p *path) isOpaque() bool {
	return p.opaque
}

func (p *path) isEmpty() bool {
	// An opaque path is represented by one path value, even when that value is
	// empty. This preserves the old []string representation's distinction.
	return !p.opaque && p.serializedLen() == 0
}

func (p *path) setOpaque(opaquePath string) {
	p.value = opaquePath
	p.buf = nil
	p.opaque = true
	p.dirty = false
}

// setSerialized replaces the serialized path without changing whether it is
// opaque or hierarchical.
func (p *path) setSerialized(serialized string) {
	p.value = serialized
	p.buf = nil
	p.dirty = false
}

func (p *path) addSegment(segment string) {
	if segment == "" && p.isEmpty() {
		p.value = "/"
		p.buf = nil
		p.opaque = false
		p.dirty = false
		return
	}
	p.ensureMutable()
	p.buf = append(p.buf, '/')
	p.buf = append(p.buf, segment...)
	p.opaque = false
}

func (p *path) setSingleSegmentBytes(segment []byte) {
	if len(segment) == 0 {
		p.value = "/"
	} else {
		var serialized strings.Builder
		serialized.Grow(1 + len(segment))
		serialized.WriteByte('/')
		_, _ = serialized.Write(segment)
		p.value = serialized.String()
	}
	p.buf = nil
	p.opaque = false
	p.dirty = false
}

// addSegmentBytes appends a hierarchical path segment. If collapseEmpty is
// true, a final empty segment is replaced, matching CollapseConsecutiveSlashes.
func (p *path) addSegmentBytes(segment []byte, collapseEmpty bool) {
	if !collapseEmpty && len(segment) == 0 && p.isEmpty() {
		p.value = "/"
		p.buf = nil
		p.opaque = false
		p.dirty = false
		return
	}

	p.ensureMutable()
	if !collapseEmpty || len(p.buf) == 0 || p.buf[len(p.buf)-1] != '/' {
		p.buf = append(p.buf, '/')
	}
	p.buf = append(p.buf, segment...)
	p.opaque = false
}

func (p *path) init() {
	p.value = ""
	p.buf = nil
	p.opaque = false
	p.dirty = false
}

func (p *path) shortenPath(scheme string) {
	if p.isEmpty() || p.opaque || (scheme == "file" && p.hasSingleNormalizedWindowsDriveLetter()) {
		return
	}

	if !p.dirty {
		p.value = p.value[:strings.LastIndexByte(p.value, '/')]
		return
	}

	for i := len(p.buf) - 1; i >= 0; i-- {
		if p.buf[i] == '/' {
			p.buf = p.buf[:i]
			return
		}
	}
}

func (p *path) hasSingleNormalizedWindowsDriveLetter() bool {
	return p.serializedLen() == 3 && p.byteAt(0) == '/' &&
		isASCIIAlpha(p.byteAt(1)) && p.byteAt(2) == ':'
}

func (p *path) firstSegmentIsNormalizedWindowsDriveLetter() bool {
	if p.opaque || p.serializedLen() < 3 || p.byteAt(0) != '/' ||
		!isASCIIAlpha(p.byteAt(1)) || p.byteAt(2) != ':' {
		return false
	}
	return p.serializedLen() == 3 || p.byteAt(3) == '/'
}

func (p *path) addFirstSegmentFrom(src *path) {
	end := src.firstSegmentEnd()
	if end <= 1 {
		p.addSegment("")
		return
	}

	p.ensureMutable()
	p.buf = append(p.buf, '/')
	if src.dirty {
		p.buf = append(p.buf, src.buf[1:end]...)
	} else {
		p.buf = append(p.buf, src.value[1:end]...)
	}
	p.opaque = false
}

func (p *path) firstSegmentEnd() int {
	n := p.serializedLen()
	for i := 1; i < n; i++ {
		if p.byteAt(i) == '/' {
			return i
		}
	}
	return n
}

func (p *path) needsDisambiguation() bool {
	return !p.opaque && p.serializedLen() >= 2 && p.byteAt(0) == '/' && p.byteAt(1) == '/'
}

func (p *path) stripTrailingSpacesIfOpaque() {
	n := p.serializedLen()
	if !p.opaque || n == 0 || p.byteAt(n-1) != ' ' {
		return
	}

	if p.dirty {
		for len(p.buf) > 0 && p.buf[len(p.buf)-1] == ' ' {
			p.buf = p.buf[:len(p.buf)-1]
		}
		return
	}

	i := len(p.value)
	for i > 0 && p.value[i-1] == ' ' {
		i--
	}
	p.value = p.value[:i]
}

func (p *path) clone() *path {
	if p == nil {
		return nil
	}

	clone := &path{
		value:  p.value,
		opaque: p.opaque,
		dirty:  p.dirty,
	}
	if p.dirty {
		clone.buf = append([]byte(nil), p.buf...)
	}
	return clone
}

func (p *path) ensureMutable() {
	if p.dirty {
		return
	}
	p.buf = append(p.buf[:0], p.value...)
	p.value = ""
	p.dirty = true
}

func (p *path) finalize() {
	if !p.dirty {
		return
	}
	p.value = string(p.buf)
	p.buf = nil
	p.dirty = false
}

func (p *path) serializedLen() int {
	if p.dirty {
		return len(p.buf)
	}
	return len(p.value)
}

func (p *path) byteAt(i int) byte {
	if p.dirty {
		return p.buf[i]
	}
	return p.value[i]
}

func (p *path) String() string {
	p.finalize()
	return p.value
}
