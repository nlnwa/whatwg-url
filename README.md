# whatwg-url

whatwg-url is a spec-compliant URL parser written in Go. See [WHATWG](https://url.spec.whatwg.org/#url-parsing) website for the specification.

## Status

Whatwg-url parser is up to date as of [16 January 2026](https://github.com/web-platform-tests/wpt/tree/b200561bc63dc0951a6df7d11fc333c71cb68e36) and passes all relevant tests from [web-platform-tests](https://github.com/web-platform-tests/wpt/tree/master/url).

## Usage

API is similar to [Chapter 6 in WHATWG URL Standard](https://url.spec.whatwg.org/#api). See [documentation](https://pkg.go.dev/github.com/nlnwa/whatwg-url) for details.

```go
import "github.com/nationallibraryofnorway/whatwg-url/url"

u, _ := url.Parse("http://example.com:80/a?b#c")
fmt.Println(u.Scheme())    // http
fmt.Println(u.Host())      // example.com
fmt.Println(u.Port())      // ""
fmt.Println(u.Pathname())  // "/a"
fmt.Println(u.Href(false)) // http://example.com/a?b#c
fmt.Println(u.Href(true))  // http://example.com/a?b
fmt.Println(u.Hash())      // "#c"
fmt.Println(u.Fragment())  // "c"
fmt.Println(u.Search())    // "?b"
fmt.Println(u.Query())     // "b"
fmt.Println(u)             // http://example.com/a?b#c
```

### Options

The default parser instance follows the WHATWG URL Standard. To adapt parsing to other needs, create a new parser instance and configure it with [options](https://pkg.go.dev/github.com/nationallibraryofnorway/whatwg-url/url#ParserOption).

Example:

```go
p := url.NewParser(
    url.WithAcceptInvalidCodepoints(),
    url.WithCollapseConsecutiveSlashes(),
)
```

### Error handling

Use standard Go error handling with `errors.Is`/`errors.As`.

```go
import (
    "errors"

    urlerrors "github.com/nationallibraryofnorway/whatwg-url/errors"
    "github.com/nationallibraryofnorway/whatwg-url/url"
)

u, err := url.Parse("https://example.org:7z")
if err != nil {
    if errors.Is(err, urlerrors.ErrPortInvalid) {
        // handle invalid port
    }
}

u, err = url.Parse(" https://example.org ")
if err == nil {
    for _, verr := range u.ValidationErrors() {
        if errors.Is(verr, urlerrors.ErrInvalidURLUnit) {
            // optional: report warning
        }
    }
}
```

### Canonicalization

If you want canonicalization beyond what's described in the standard, you can use the
[Canonicalizer API](https://pkg.go.dev/github.com/nationallibraryofnorway/whatwg-url/canonicalizer). You can define your own canonicalization profile:

```go
import "github.com/nationallibraryofnorway/whatwg-url/canonicalizer"

c := canonicalizer.New(
    canonicalizer.WithRemoveUserInfo(),
    canonicalizer.WithRemoveFragment(),
)
u, err := c.Parse("http://user@example.com/a?b#c")
```

Or use one of the predefined profiles:

```go
u, err := canonicalizer.GoogleSafeBrowsing.Parse("http://user@example.com/a?b#c")
```

Parser behavior remains UTF-8 based and does not guess alternate encodings.
If you need compatibility handling for legacy percent-encoded Latin-1 octets during canonicalization,
use canonicalizer options:

```go
c := canonicalizer.New(
    canonicalizer.WithRepeatedPercentDecoding(),
    canonicalizer.WithRepeatedPercentDecodingEncoding(charmap.ISO8859_1),
)
```

For advanced WHATWG state-machine usage (`BasicParser` and parser states), use
`github.com/nationallibraryofnorway/whatwg-url/x/whatwg`.

### Percent-encode set customization

`PercentEncodeSet` can be customized directly with `Set` and `Clear`:

```go
set := url.QueryPercentEncodeSet.Set('#', '&').Clear(';')
p := url.NewParser(url.WithQueryPercentEncodeSet(set))
```

## Web Platform Tests Compliance

Some of web platform tests fail because they require newer Unicode tables than [`golang.org/x/net/idna`](https://pkg.go.dev/golang.org/x/net/idna) is currently providing. The failing tests are skipped by default. To run them:

```shell
$ go test ./url -runSkipped
--- FAIL: TestToASCII (0.00s)
    --- FAIL: TestToASCII/059:_ (0.00s)
        idna_test.go:79: ToASCII(lookout.net) = 'xn--lookout-on3b.net', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+180E
    --- FAIL: TestToASCII/067:_ (0.00s)
        idna_test.go:79: ToASCII(lookout.net) = 'xn--lookout-1m7c.net', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+206B
    --- FAIL: TestToASCII/074:_ (0.00s)
        idna_test.go:79: ToASCII(Ӏ.com) = 'xn--d5a.com', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+04C0
    --- FAIL: TestToASCII/075:_ (0.00s)
        idna_test.go:79: ToASCII(㛼.com) = 'xn--snl.com', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2F868
    --- FAIL: TestToASCII/076:_ (0.00s)
        idna_test.go:79: ToASCII(Ↄ.com) = 'xn--q5g.com', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2183
    --- FAIL: TestToASCII/080:_ (0.00s)
        idna_test.go:87: ToASCII(ẞ.com) = 'ss.com', want 'xn--zca.com'
    --- FAIL: TestToASCII/081:_ (0.00s)
        idna_test.go:87: ToASCII(ẞ.foo.com) = 'ss.foo.com', want 'xn--zca.foo.com'
--- FAIL: TestIdnaV2 (0.03s)
    --- FAIL: TestIdnaV2/064:_ (0.00s)
        idna_test.go:142: ToASCII(FAẞ.de) = 'fass.de', want 'xn--fa-hia.de'
    --- FAIL: TestIdnaV2/065:_ (0.00s)
        idna_test.go:142: ToASCII(FAẞ.DE) = 'fass.de', want 'xn--fa-hia.de'
    --- FAIL: TestIdnaV2/147:_V4;_V2_(ignored) (0.00s)
        idna_test.go:137: ToASCII(xn--xn--a--gua.pt) = 'xn--xn--a--gua.pt', expected error but got none
    --- FAIL: TestIdnaV2/250:_V4;_V2_(ignored) (0.00s)
        idna_test.go:137: ToASCII(xn--xn---epa) = 'xn--xn---epa', expected error but got none
    --- FAIL: TestIdnaV2/351:_P4 (0.00s)
        idna_test.go:137: ToASCII(xn--ASCII-) = 'ascii', expected error but got none
    --- FAIL: TestIdnaV2/353:_P4 (0.00s)
        idna_test.go:137: ToASCII(xn--unicode-.org) = 'unicode.org', expected error but got none
    --- FAIL: TestIdnaV2/355:_ (0.00s)
        idna_test.go:134: ToASCII(陋㛼当𤎫竮䗗) = 'xn--snl253bgitxhzwu2arn60c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2F868
    --- FAIL: TestIdnaV2/396:_ (0.00s)
        idna_test.go:134: ToASCII(≠ᢙ≯.솣-ᡴႠ) = 'xn--jbf911clb.xn----6zg521d196p', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A0
    --- FAIL: TestIdnaV2/397:_ (0.00s)
        idna_test.go:134: ToASCII(≠ᢙ≯.솣-ᡴႠ) = 'xn--jbf911clb.xn----6zg521d196p', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A0
    --- FAIL: TestIdnaV2/523:_ (0.00s)
        idna_test.go:134: ToASCII(𞤥󠅮．ᡄႮ) = 'xn--de6h.xn--mnd799a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AE
    --- FAIL: TestIdnaV2/524:_ (0.00s)
        idna_test.go:134: ToASCII(𞤥󠅮.ᡄႮ) = 'xn--de6h.xn--mnd799a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AE
    --- FAIL: TestIdnaV2/526:_ (0.00s)
        idna_test.go:134: ToASCII(𞤃󠅮.ᡄႮ) = 'xn--de6h.xn--mnd799a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AE
    --- FAIL: TestIdnaV2/530:_ (0.00s)
        idna_test.go:134: ToASCII(𞤃.ᡄႮ) = 'xn--de6h.xn--mnd799a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AE
    --- FAIL: TestIdnaV2/533:_ (0.00s)
        idna_test.go:134: ToASCII(𞤃󠅮．ᡄႮ) = 'xn--de6h.xn--mnd799a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AE
    --- FAIL: TestIdnaV2/536:_ (0.00s)
        idna_test.go:134: ToASCII(𞤥.ᡄႮ) = 'xn--de6h.xn--mnd799a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AE
    --- FAIL: TestIdnaV2/659:_ (0.00s)
        idna_test.go:134: ToASCII(ß｡𐋳Ⴌྸ) = 'xn--zca.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/660:_ (0.00s)
        idna_test.go:134: ToASCII(ß。𐋳Ⴌྸ) = 'xn--zca.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/662:_ (0.00s)
        idna_test.go:134: ToASCII(SS。𐋳Ⴌྸ) = 'ss.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/664:_ (0.00s)
        idna_test.go:134: ToASCII(Ss。𐋳Ⴌྸ) = 'ss.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/667:_ (0.00s)
        idna_test.go:134: ToASCII(SS.𐋳Ⴌྸ) = 'ss.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/668:_ (0.00s)
        idna_test.go:134: ToASCII(Ss.𐋳Ⴌྸ) = 'ss.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/672:_ (0.00s)
        idna_test.go:134: ToASCII(SS｡𐋳Ⴌྸ) = 'ss.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/674:_ (0.00s)
        idna_test.go:134: ToASCII(Ss｡𐋳Ⴌྸ) = 'ss.xn--lgd10cu829c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AC
    --- FAIL: TestIdnaV2/949:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴚ𐋸󠄄。𝟝်) = 'xn--ynd2415j.xn--5-dug9054m', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10BA
    --- FAIL: TestIdnaV2/950:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴚ𐋸󠄄。5်) = 'xn--ynd2415j.xn--5-dug9054m', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10BA
    --- FAIL: TestIdnaV2/954:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴚ𐋸.5်) = 'xn--ynd2415j.xn--5-dug9054m', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10BA
    --- FAIL: TestIdnaV2/1070:_V3_(ignored) (0.00s)
        idna_test.go:134: ToASCII(ꡆ。Ↄྵ놮-) = 'xn--fc9a.xn----qmg787k869k', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2183
    --- FAIL: TestIdnaV2/1071:_V3_(ignored) (0.00s)
        idna_test.go:134: ToASCII(ꡆ。Ↄྵ놮-) = 'xn--fc9a.xn----qmg787k869k', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2183
    --- FAIL: TestIdnaV2/1170:_ (0.00s)
        idna_test.go:134: ToASCII(Ⅎ្.≠) = 'xn--u4e319b.xn--1ch', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2132
    --- FAIL: TestIdnaV2/1171:_ (0.00s)
        idna_test.go:134: ToASCII(Ⅎ្.≠) = 'xn--u4e319b.xn--1ch', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+2132
    --- FAIL: TestIdnaV2/1278:_ (0.00s)
        idna_test.go:134: ToASCII(꧐Ӏ᮪ࣶ．눵) = 'xn--d5a07sn4u297k.xn--2e1b', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+04C0
    --- FAIL: TestIdnaV2/1279:_ (0.00s)
        idna_test.go:134: ToASCII(꧐Ӏ᮪ࣶ．눵) = 'xn--d5a07sn4u297k.xn--2e1b', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+04C0
    --- FAIL: TestIdnaV2/1280:_ (0.00s)
        idna_test.go:134: ToASCII(꧐Ӏ᮪ࣶ.눵) = 'xn--d5a07sn4u297k.xn--2e1b', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+04C0
    --- FAIL: TestIdnaV2/1281:_ (0.00s)
        idna_test.go:134: ToASCII(꧐Ӏ᮪ࣶ.눵) = 'xn--d5a07sn4u297k.xn--2e1b', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+04C0
    --- FAIL: TestIdnaV2/1336:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(。。Ⴃ≯) = '..xn--bnd622g', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A3
    --- FAIL: TestIdnaV2/1337:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(。。Ⴃ≯) = '..xn--bnd622g', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A3
    --- FAIL: TestIdnaV2/1378:_ (0.00s)
        idna_test.go:134: ToASCII(ڹ．ᡳ) = 'xn--skb.xn--osd737a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+115F
    --- FAIL: TestIdnaV2/1379:_ (0.00s)
        idna_test.go:134: ToASCII(ڹ.ᡳ) = 'xn--skb.xn--osd737a', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+115F
    --- FAIL: TestIdnaV2/1395:_ (0.00s)
        idna_test.go:134: ToASCII(SS꫶ᢥ.⊶ჁႶ) = 'xn--ss-4epx629f.xn--undv409k', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
    --- FAIL: TestIdnaV2/1396:_ (0.00s)
        idna_test.go:134: ToASCII(Ss꫶ᢥ.⊶Ⴡⴖ) = 'xn--ss-4epx629f.xn--5nd703gyrh', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
    --- FAIL: TestIdnaV2/1557:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴚ.Σ) = 'xn--ynd.xn--4xa', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10BA
    --- FAIL: TestIdnaV2/1559:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴚ.ς) = 'xn--ynd.xn--3xa', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10BA
    --- FAIL: TestIdnaV2/1561:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴚ.σ) = 'xn--ynd.xn--4xa', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10BA
    --- FAIL: TestIdnaV2/1712:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴁ畝.≮) = 'xn--8md0962c.xn--gdh', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A1
    --- FAIL: TestIdnaV2/1713:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴁ畝.≮) = 'xn--8md0962c.xn--gdh', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A1
    --- FAIL: TestIdnaV2/1758:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(🕼．) = 'xn--my8h.xn--cl7c', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+FFA0
    --- FAIL: TestIdnaV2/1759:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(🕼.) = 'xn--my8h.xn--psd', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+1160
    --- FAIL: TestIdnaV2/1771:_ (0.00s)
        idna_test.go:134: ToASCII(ςჅ。ݚ) = 'xn--3xa677d.xn--epb', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C5
    --- FAIL: TestIdnaV2/1773:_ (0.00s)
        idna_test.go:134: ToASCII(ΣჅ。ݚ) = 'xn--4xa477d.xn--epb', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C5
    --- FAIL: TestIdnaV2/1778:_ (0.00s)
        idna_test.go:134: ToASCII(ΣჅ.ݚ) = 'xn--4xa477d.xn--epb', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C5
    --- FAIL: TestIdnaV2/1791:_ (0.00s)
        idna_test.go:134: ToASCII(𐩗.ႩႵ) = 'xn--pt9c.xn--hndy', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A9
    --- FAIL: TestIdnaV2/1792:_ (0.00s)
        idna_test.go:134: ToASCII(𐩗.Ⴉⴕ) = 'xn--pt9c.xn--hnd666l', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A9
    --- FAIL: TestIdnaV2/1823:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(𲤱20.音.ꡦ1.) = 'xn--20-9802c.xn--0w5a.xn--1-eg4e.', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+32931
    --- FAIL: TestIdnaV2/1824:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(xn--20-9802c.xn--0w5a.xn--1-eg4e.) = 'xn--20-9802c.xn--0w5a.xn--1-eg4e.', unexpected error: unicode ToASCII records an error or returns the empty string: idna: invalid label "\U0003293120"
    --- FAIL: TestIdnaV2/1826:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ。۰≮ß݅) = 'xn--tnd.xn--zca912alh227g', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1827:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ。۰≮ß݅) = 'xn--tnd.xn--zca912alh227g', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1830:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ。۰≮SS݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1831:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ。۰≮SS݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1834:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ。۰≮Ss݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1835:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ。۰≮Ss݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1839:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ.۰≮SS݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1840:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ.۰≮SS݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1841:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ.۰≮Ss݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1842:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴕ.۰≮Ss݅) = 'xn--tnd.xn--ss-jbe65aw27i', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B5
    --- FAIL: TestIdnaV2/1911:_ (0.00s)
        idna_test.go:134: ToASCII(xn--9-i0j5967eg3qz.ss) = 'xn--9-i0j5967eg3qz.ss', unexpected error: unicode ToASCII records an error or returns the empty string: idna: invalid label "\U00032b9a9ꍩ៓"
    --- FAIL: TestIdnaV2/1912:_ (0.00s)
        idna_test.go:134: ToASCII(𲮚9ꍩ៓.ss) = 'xn--9-i0j5967eg3qz.ss', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+32B9A
    --- FAIL: TestIdnaV2/1913:_ (0.00s)
        idna_test.go:134: ToASCII(𲮚9ꍩ៓.SS) = 'xn--9-i0j5967eg3qz.ss', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+32B9A
    --- FAIL: TestIdnaV2/1987:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(Ⴖͦ.) = 'xn--hva929d.', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B6
    --- FAIL: TestIdnaV2/1991:_ (0.00s)
        idna_test.go:134: ToASCII(ࢻ.Ⴃ𞀒) = 'xn--hzb.xn--bnd2938u', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10A3
    --- FAIL: TestIdnaV2/2029:_ (0.00s)
        idna_test.go:134: ToASCII(攌꯭.ᢖ-Ⴘ) = 'xn--p9ut19m.xn----k1g451d', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10B8
    --- FAIL: TestIdnaV2/2335:_ (0.00s)
        idna_test.go:134: ToASCII(🂴Ⴋ.≮) = 'xn--jnd1986v.xn--gdh', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AB
    --- FAIL: TestIdnaV2/2336:_ (0.00s)
        idna_test.go:134: ToASCII(🂴Ⴋ.≮) = 'xn--jnd1986v.xn--gdh', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10AB
    --- FAIL: TestIdnaV2/2433:_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(。9𞀈䬺1.) = 'xn--3f1h.xn--91-030c1650n.', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+1D175
    --- FAIL: TestIdnaV2/2526:_V3_(ignored);_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(឴.쮇-) = 'xn--z3e.xn----938f', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+17B4
    --- FAIL: TestIdnaV2/2527:_V3_(ignored);_A4_2_(ignored) (0.00s)
        idna_test.go:134: ToASCII(឴.쮇-) = 'xn--z3e.xn----938f', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+17B4
    --- FAIL: TestIdnaV2/2575:_ (0.00s)
        idna_test.go:134: ToASCII(ჁႱ6̘。ßᬃ) = 'xn--6-8cb555h2b.xn--zca894k', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
    --- FAIL: TestIdnaV2/2577:_ (0.00s)
        idna_test.go:134: ToASCII(ჁႱ6̘。SSᬃ) = 'xn--6-8cb555h2b.xn--ss-2vq', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
    --- FAIL: TestIdnaV2/2579:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴡⴑ6̘。Ssᬃ) = 'xn--6-8cb306hms1a.xn--ss-2vq', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
    --- FAIL: TestIdnaV2/2582:_ (0.00s)
        idna_test.go:134: ToASCII(ჁႱ6̘.SSᬃ) = 'xn--6-8cb555h2b.xn--ss-2vq', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
    --- FAIL: TestIdnaV2/2583:_ (0.00s)
        idna_test.go:134: ToASCII(Ⴡⴑ6̘.Ssᬃ) = 'xn--6-8cb306hms1a.xn--ss-2vq', unexpected error: unicode ToASCII records an error or returns the empty string: idna: disallowed rune U+10C1
FAIL
FAIL    github.com/nationallibraryofnorway/whatwg-url/url  0.062s
FAIL
```
