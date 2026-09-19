package rrpc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/netgarden/rrpc/test/modules/echo"
)

// TestGetRequestStructTravelsInTheQueryString is the end-to-end regression
// for two bugs that hid each other:
//
//   - the generated Go client sent a GET's request struct as a JSON body,
//     which the server never reads, so every field silently arrived as its
//     zero value (browsers use the TS client, which was correct);
//   - the server guessed each query value's type from its text, so a string
//     field could not receive "2024" or "true" (a 400).
//
// Both are exercised together through the real generated client and server.
func TestGetRequestStructTravelsInTheQueryString(t *testing.T) {
	client := newEchoTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cases := []struct {
		name string
		req  echo.EchoRequest
	}{
		{"plain", echo.EchoRequest{Message: "hello", Count: 3, Flag: true}},
		{"zero values", echo.EchoRequest{}},
		{"numeric-looking string", echo.EchoRequest{Message: "2024", Count: 1}},
		{"bool-looking string", echo.EchoRequest{Message: "true"}},
		{"leading zeros", echo.EchoRequest{Message: "007", Count: 7}},
		{"query-string metacharacters", echo.EchoRequest{Message: "a&b=c?d#e f+g%h"}},
		{"unicode", echo.EchoRequest{Message: "žluťoučký kůň 🐴", Count: -5}},
		{"large number", echo.EchoRequest{Count: 1000000}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := client.Echo.Echo(ctx, &tc.req)
			if err != nil {
				t.Fatalf("Echo: %v", err)
			}
			want := fmt.Sprintf("message=%q count=%d flag=%v", tc.req.Message, tc.req.Count, tc.req.Flag)
			if resp.Result != want {
				t.Errorf("server received %q, want %q", resp.Result, want)
			}
		})
	}
}
