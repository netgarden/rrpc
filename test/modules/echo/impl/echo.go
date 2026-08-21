package impl

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/coder/websocket"
	"github.com/netgarden/rrpc"
	"github.com/netgarden/rrpc/test/modules/echo"
)

func NewEchoService() echo.EchoService {
	return &EchoService{}
}

type EchoService struct{}

func (s *EchoService) Echo(ctx *rrpc.Context, request *echo.EchoRequest) (*echo.EchoResponse, error) {
	result := fmt.Sprintf("message=%q count=%d flag=%v", request.Message, request.Count, request.Flag)
	return &echo.EchoResponse{Result: result}, nil
}

// Chat is a bidirectional test method: it echoes every received
// ChatMessage back as a ChatReply, except a message with text "trigger"
// which produces a SendError instead - end-to-end proof that the client
// sees a distinct RRPCError at exactly that point in the stream.
func (s *EchoService) Chat(ctx *rrpc.Context, stream *rrpc.Stream[echo.ChatMessage, echo.ChatReply]) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				return nil
			}
			return err
		}

		if msg.Text == "trigger" {
			if err := stream.SendError(echo.ErrStreamTriggerError); err != nil {
				return err
			}
			continue
		}

		if err := stream.Send(echo.ChatReply{From: "server", Text: msg.Text}); err != nil {
			return err
		}
	}
}

// Collect is a client->server-only test method: it accumulates every
// received LogLine and logs the total once the client disconnects.
func (s *EchoService) Collect(ctx *rrpc.Context, stream *rrpc.InStream[echo.LogLine]) error {
	var lines []string
	for {
		msg, err := stream.Recv()
		if err != nil {
			if websocket.CloseStatus(err) != -1 {
				return nil
			}
			return err
		}
		lines = append(lines, msg.Line)
	}
}

// Tick is a server->client-only test method: it sends a TickResponse
// every 200ms until the client disconnects (observed via the stream's
// connection-scoped Context being cancelled).
func (s *EchoService) Tick(ctx *rrpc.Context, stream *rrpc.OutStream[echo.TickResponse]) error {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	n := 0
	for {
		select {
		case <-stream.Context().Done():
			return nil
		case <-ticker.C:
			n++
			if err := stream.Send(echo.TickResponse{N: n}); err != nil {
				return nil
			}
		}
	}
}

// UploadBytes is a raw byte-stream test method: it proves *rrpc.RawInStream
// genuinely satisfies io.Reader by handing it directly to io.Copy.
func (s *EchoService) UploadBytes(ctx *rrpc.Context, stream *rrpc.RawInStream) error {
	n, err := io.Copy(io.Discard, stream)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	fmt.Printf("UploadBytes: copied %d bytes\n", n)
	return nil
}
