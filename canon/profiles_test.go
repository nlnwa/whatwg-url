package canon

import "testing"

func TestGoogleSafeBrowsing(t *testing.T) {
	tests := []struct {
		name    string
		args    string
		want    string
		wantErr bool
	}{
		{"1", "http://host/%25%32%35", "http://host/%25", false},
		{"2", "http://host/%25%32%35%25%32%35", "http://host/%25%25", false},
		{"3", "http://host/%2525252525252525", "http://host/%25", false},
		{"4", "http://host/asdf%25%32%35asd", "http://host/asdf%25asd", false},
		{"5", "http://host/%%%25%32%35asd%%", "http://host/%25%25%25asd%25%25", false},
		{"6", "http://www.google.com/", "http://www.google.com/", false},
		{"7", "http://%31%36%38%2e%31%38%38%2e%39%39%2e%32%36/%2E%73%65%63%75%72%65/%77%77%77%2E%65%62%61%79%2E%63%6F%6D/", "http://168.188.99.26/.secure/www.ebay.com/", false},
		{"8", "http://195.127.0.11/uploads/%20%20%20%20/.verify/.eBaysecure=updateuserdataxplimnbqmn-xplmvalidateinfoswqpcmlx=hgplmcx/", "http://195.127.0.11/uploads/%20%20%20%20/.verify/.eBaysecure=updateuserdataxplimnbqmn-xplmvalidateinfoswqpcmlx=hgplmcx/", false},
		{"9", "http://host%23.com/%257Ea%2521b%2540c%2523d%2524e%25f%255E00%252611%252A22%252833%252944_55%252B", "http://host%23.com/~a!b@c%23d$e%25f^00&11*22(33)44_55+", false},
		{"10", "http://3279880203/blah", "http://195.127.0.11/blah", false},
		{"11", "http://www.google.com/blah/..", "http://www.google.com/", false},
		{"12", "www.google.com/", "http://www.google.com/", false},
		{"13", "www.google.com", "http://www.google.com/", false},
		{"14", "http://www.evil.com/blah#frag", "http://www.evil.com/blah", false},
		{"15", "http://www.GOOgle.com/", "http://www.google.com/", false},
		{"16", "http://www.google.com.../", "http://www.google.com/", false},
		{"17", "http://www.google.com/foo\tbar\rbaz\n2", "http://www.google.com/foobarbaz2", false},
		{"18", "http://www.google.com/q?", "http://www.google.com/q?", false},
		{"19", "http://www.google.com/q?r?", "http://www.google.com/q?r?", false},
		{"20", "http://www.google.com/q?r?s", "http://www.google.com/q?r?s", false},
		{"21", "http://evil.com/foo#bar#baz", "http://evil.com/foo", false},
		{"22", "http://evil.com/foo;", "http://evil.com/foo;", false},
		{"23", "http://evil.com/foo?bar;", "http://evil.com/foo?bar;", false},
		{"24", "http://\x01\x80.com/", "http://%01%80.com/", false},
		{"25", "http://notrailingslash.com", "http://notrailingslash.com/", false},
		{"26", "http://www.gotaport.com:1234/", "http://www.gotaport.com/", false},
		{"27", "  http://www.google.com/  ", "http://www.google.com/", false},
		{"28", "http:// leadingspace.com/", "http://%20leadingspace.com/", false},
		{"29", "http://%20leadingspace.com/", "http://%20leadingspace.com/", false},
		{"30", "%20leadingspace.com/", "http://%20leadingspace.com/", false},
		{"31", "https://www.securesite.com/", "https://www.securesite.com/", false},
		{"32", "http://host.com/ab%23cd", "http://host.com/ab%23cd", false},
		{"33", "http://host.com//twoslashes?more//slashes", "http://host.com/twoslashes?more//slashes", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GoogleSafeBrowsing.Canonicalize(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("Canonicalize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got.String() != tt.want {
				t.Errorf("Canonicalize() = %v, want %v", got, tt.want)
			}
		})
	}
}
