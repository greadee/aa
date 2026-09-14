package transport

import (
	"context"
	"errors"
	"testing"

	aasync "github.com/greadee/aa/sync"
)

func TestWireRoundTrip(t *testing.T) {
	ctx := context.Background()
	w := NewWire()
	a := aasync.NodeID("node-a")
	b := aasync.NodeID("node-b")

	aConn, err := w.Dial(ctx, a, b)
	if err != nil {
		t.Fatalf("dial a: %v", err)
	}
	bConn, err := w.Dial(ctx, b, a)
	if err != nil {
		t.Fatalf("dial b: %v", err)
	}

	msg, err := NewMessage(KindHello, a, b, 1, map[string]string{"name": "a"})
	if err != nil {
		t.Fatalf("new message: %v", err)
	}
	if err := aConn.Send(ctx, msg); err != nil {
		t.Fatalf("send: %v", err)
	}
	got, err := bConn.Receive(ctx)
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if got.Kind != KindHello || got.From != a || got.To != b {
		t.Fatalf("unexpected message: %+v", got)
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := DecodeBody(got, &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Name != "a" {
		t.Fatalf("body name = %q, want a", body.Name)
	}
	if w.Delivered(b) != 1 {
		t.Fatalf("delivered = %d, want 1", w.Delivered(b))
	}
}

func TestWireRejectsMismatchedEnvelope(t *testing.T) {
	ctx := context.Background()
	w := NewWire()
	a := aasync.NodeID("a")
	b := aasync.NodeID("b")
	conn, _ := w.Dial(ctx, a, b)
	msg := Message{Kind: KindHello, From: a, To: aasync.NodeID("c")}
	if err := conn.Send(ctx, msg); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("send err = %v, want ErrInvalid", err)
	}
}

func TestWireInjectedDrop(t *testing.T) {
	ctx := context.Background()
	w := NewWire()
	a := aasync.NodeID("a2")
	b := aasync.NodeID("b2")
	conn, _ := w.Dial(ctx, a, b)
	w.FailNext(b, 1)
	msg, _ := NewMessage(KindAck, a, b, 1, nil)
	if err := conn.Send(ctx, msg); !errors.Is(err, ErrDropped) {
		t.Fatalf("send err = %v, want ErrDropped", err)
	}
	if err := conn.Send(ctx, msg); err != nil {
		t.Fatalf("second send: %v", err)
	}
	if w.Delivered(b) != 0 {
		t.Fatalf("delivered = %d, want 0 (no receive yet)", w.Delivered(b))
	}
}

func TestConnClose(t *testing.T) {
	ctx := context.Background()
	w := NewWire()
	a := aasync.NodeID("a3")
	b := aasync.NodeID("b3")
	conn, _ := w.Dial(ctx, a, b)
	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	msg, _ := NewMessage(KindClose, a, b, 1, nil)
	if err := conn.Send(ctx, msg); !errors.Is(err, ErrClosed) {
		t.Fatalf("send after close err = %v, want ErrClosed", err)
	}
	if _, err := conn.Receive(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("receive after close err = %v, want ErrClosed", err)
	}
}

func TestDecodeBodyRejectsEmpty(t *testing.T) {
	if err := DecodeBody(Message{}, &struct{}{}); !errors.Is(err, aasync.ErrInvalid) {
		t.Fatalf("decode empty err = %v, want ErrInvalid", err)
	}
}
