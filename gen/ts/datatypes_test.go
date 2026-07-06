package rrpc_gen_ts

import "testing"

func TestRrpcDataTypeToTSType(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"string", "string"},
		{"bool", "boolean"},
		{"int", "number"},
		{"int64", "number"},
		{"bytes", "Uint8Array"},
		{"[]string", "string[]"},
		{"[]int", "number[]"},
		{"map[string]string", "Record<string, string>"},
		{"map[string]int", "Record<string, number>"},
		{"[]map[string]string", "Record<string, string>[]"},
		{"CustomStruct", "CustomStruct"},
	}

	for _, c := range cases {
		if got := rrpcDataTypeToTSType(c.in); got != c.want {
			t.Errorf("rrpcDataTypeToTSType(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
