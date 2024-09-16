package url

import "testing"

func Test_parser_parseHost(t *testing.T) {
	type args struct {
		input        string
		isNotSpecial bool
	}
	tests := []struct {
		name                  string
		args                  args
		failOnValidationError bool
		want                  string
		wantErr               bool
	}{
		{"1-1", args{input: "EXAMPLE.COM", isNotSpecial: false}, false, "example.com", false},
		{"1-2", args{input: "EXAMPLE.COM", isNotSpecial: true}, false, "EXAMPLE.COM", false},
		{"2-1", args{input: "example%2Ecom", isNotSpecial: false}, false, "example.com", false},
		{"2-2", args{input: "example%2Ecom", isNotSpecial: true}, false, "example%2Ecom", false},
		{"3-1", args{input: "faß.example", isNotSpecial: false}, false, "xn--fa-hia.example", false},
		{"3-2", args{input: "faß.example", isNotSpecial: true}, false, "fa%C3%9F.example", false},
		{"4-1", args{input: "0", isNotSpecial: false}, false, "0.0.0.0", false},
		{"4-2", args{input: "0", isNotSpecial: true}, false, "0", false},
		{"5-1", args{input: "%30", isNotSpecial: false}, false, "0.0.0.0", false},
		{"5-2", args{input: "%30", isNotSpecial: true}, false, "%30", false},
		{"6-1", args{input: "0x", isNotSpecial: false}, false, "0.0.0.0", false},
		{"6-2", args{input: "0x", isNotSpecial: true}, false, "0x", false},
		{"7-1", args{input: "0xffffffff", isNotSpecial: false}, false, "255.255.255.255", false},
		{"7-2", args{input: "0xffffffff", isNotSpecial: true}, false, "0xffffffff", false},
		{"8-1", args{input: "[0:0::1]", isNotSpecial: false}, false, "[::1]", false},
		{"8-2", args{input: "[0:0::1]", isNotSpecial: true}, false, "[::1]", false},
		{"9-1", args{input: "[0:0::1%5d]", isNotSpecial: false}, false, "", true},
		{"9-2", args{input: "[0:0::1%5d]", isNotSpecial: true}, false, "", true},
		{"10-1", args{input: "[0:0::1%31]", isNotSpecial: false}, false, "", true},
		{"10-2", args{input: "[0:0::1%31]", isNotSpecial: true}, false, "", true},
		{"11-1", args{input: "09", isNotSpecial: false}, false, "09", true},
		{"11-2", args{input: "09", isNotSpecial: true}, false, "09", false},
		{"12-1", args{input: "example.255", isNotSpecial: false}, false, "example.255", true},
		{"12-2", args{input: "example.255", isNotSpecial: true}, false, "example.255", false},
		{"13-1", args{input: "example^example", isNotSpecial: false}, false, "", true},
		{"13-2", args{input: "example^example", isNotSpecial: true}, false, "", true},
		{"14-1", args{input: "bad%hostname", isNotSpecial: false}, true, "", true},
		{"14-2", args{input: "bad%hostname", isNotSpecial: true}, true, "", true},
		{"15-1", args{input: "bad\\:hostname", isNotSpecial: false}, true, "", true},
		{"15-2", args{input: "bad\\:hostname", isNotSpecial: true}, true, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &parser{opts: parserOptions{failOnValidationError: tt.failOnValidationError}}

			got, err := p.parseHost(&Url{}, p, tt.args.input, tt.args.isNotSpecial)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseHost() got = %v, want %v", got, tt.want)
			}
		})
	}
}
