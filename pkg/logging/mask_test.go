package logging

import (
	"fmt"
	"testing"
)

func TestMask(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty stays empty", in: "", want: ""},
		{name: "E.164 with plus keeps country code and last four", in: "+5511987654321", want: "+55*******4321"},
		{name: "E.164 without plus", in: "5511987654321", want: "+55*******4321"},
		{name: "shortest phone length", in: "+12345678", want: "+12**5678"},
		{name: "longest phone length", in: "+123456789012345", want: "+12*********2345"},
		{name: "too short to be a phone is text", in: "1234567", want: "[redacted len=7]"},
		{name: "too long to be a phone is text", in: "1234567890123456", want: "[redacted len=16]"},
		{name: "formatted phone is treated as text", in: "+55 11 98765-4321", want: "[redacted len=17]"},
		{name: "message text keeps only rune length", in: "tomei o remédio", want: "[redacted len=15]"},
		{name: "emoji counts as one rune", in: "bebi 💧", want: "[redacted len=6]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Mask(tt.in)
			if got.String() != tt.want {
				t.Errorf("Mask(%q).String() = %q, want %q", tt.in, got.String(), tt.want)
			}
			if v := got.LogValue().String(); v != tt.want {
				t.Errorf("Mask(%q).LogValue() = %q, want %q", tt.in, v, tt.want)
			}
		})
	}
}

func TestMaskIsSafeInFmt(t *testing.T) {
	got := fmt.Sprintf("to=%v", Mask("+5511987654321"))
	if want := "to=+55*******4321"; got != want {
		t.Errorf("fmt output = %q, want %q", got, want)
	}
}
