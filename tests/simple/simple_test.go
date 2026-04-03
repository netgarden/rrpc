package simple

import (
	"context"
	"fmt"
	"net/http/httptest"
	"rrpc/tests/simple/rpc"
	"testing"
)

func Test(t *testing.T) {

	rpcServer := rpc.NewServer()
	rpcServer.SetTestService(&TestService{t: t})

	ts := httptest.NewServer(rpcServer)
	defer ts.Close()

	c := rpc.NewClient(ts.URL)
	err := c.Test.Ping(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.Test.Test(context.Background(), &rpc.Request{Data: "blabla"})
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || !resp.Ok {
		t.Fatal("bad response")
	}

	fmt.Printf("response: %+v\n", resp)
	fmt.Println("ok")

}

type TestService struct {
	t *testing.T
}

func (s *TestService) Ping(ctx context.Context) error {
	fmt.Println("received ping")
	return nil
}

func (s *TestService) Test(ctx context.Context, request *rpc.Request) (*rpc.Response, error) {
	fmt.Println("received test")
	fmt.Printf("request: %+v\n", request)
	return &rpc.Response{
		Ok: true,
	}, nil
}
