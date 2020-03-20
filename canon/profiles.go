/*
 * Copyright 2020 National Library of Norway.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package canon

import (
	"github.com/nlnwa/whatwg-url/url"
	"golang.org/x/text/encoding/charmap"
	"regexp"
	"strings"
)

var LaxQueryPercentEncodeSet = url.QueryPercentEncodeSet.Clear(0x22, 0x25, 0x2F, 0x3B, 0x3F, 0x7B)

var GoogleSafeBrowsing = New(
	url.WithLaxHostParsing(),
	url.WithQueryPercentEncodeSet(LaxQueryPercentEncodeSet),
	url.WithCollapseConsecutiveSlashes(),
	url.WithAcceptInvalidCodepoints(),
	url.WithPercentEncodeSinglePercentSign(),
	url.WithEncodingOverride(charmap.ISO8859_1),
	url.WithPreParseHostFunc(func(u *url.Url, host string) string {
		host = strings.Trim(host, ".")
		var re = regexp.MustCompile("\\.\\.+")
		host = re.ReplaceAllString(host, ".")
		return host
	}),
	WithRemovePort(),
	WithRemoveFragment(),
	WithRepeatedPercentDecoding(),
	WithDefaultScheme("http"),
)
