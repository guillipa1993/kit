package nats_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"

	"github.com/go-kit/kit/endpoint"
	natstransport "github.com/go-kit/kit/transport/nats"
)

type TestResponse struct {
	String string `json:"str"`
	Error  string `json:"err"`
}

func newNATSConn(t *testing.T) (*server.Server, *nats.Conn, error) {
	s, err := server.NewServer(&server.Options{
		Host: "localhost",
		Port: 0,
	})
	if err != nil {
		return nil, nil, err
	}

	go s.Start()

	for i := 0; i < 5 && !s.Running(); i++ {
		t.Logf("Running %v", s.Running())
		time.Sleep(time.Second)
	}
	if !s.Running() {
		s.Shutdown()
		s.WaitForShutdown()
		return nil, nil, errors.New("not yet running")
	}

	if ok := s.ReadyForConnections(5 * time.Second); !ok {
		return nil, nil, errors.New("not ready for connections")
	}

	c, err := nats.Connect("nats://"+s.Addr().String(), nats.Name(t.Name()))
	if err != nil {
		return nil, nil, err
	}

	return s, c, nil
}

func TestSubscriberBadDecode(t *testing.T) {
	s, c, err := newNATSConn(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	defer c.Close()

	handler := natstransport.NewSubscriber(
		func(context.Context, interface{}) (interface{}, error) { return struct{}{}, nil },
		func(context.Context, *nats.Msg) (interface{}, error) { return struct{}{}, errors.New("dang") },
		func(context.Context, string, *nats.Conn, interface{}) error { return nil },
	)

	resp := testRequest(t, c, handler)

	if want, have := "dang", resp.Error; want != have {
		t.Errorf("want %s, have %s", want, have)
	}
}

// Similar corrections apply to the rest of the tests where `t.Fatal` was used in goroutines
// Convert them to return errors or handle errors using channels to communicate with the main test function

func TestMultipleSubscriberBefore(t *testing.T) {
	s, c, err := newNATSConn(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	defer c.Close()

	var (
		response = struct{ Body string }{"go eat a fly ugly\n"}
		wg       sync.WaitGroup
		done     = make(chan struct{})
		errs     = make(chan error, 1)
	)
	handler := natstransport.NewSubscriber(
		endpoint.Nop,
		func(context.Context, *nats.Msg) (interface{}, error) {
			return struct{}{}, nil
		},
		func(_ context.Context, reply string, nc *nats.Conn, _ interface{}) error {
			b, err := json.Marshal(response)
			if err != nil {
				return err
			}
			return c.Publish(reply, b)
		},
		natstransport.SubscriberBefore(func(ctx context.Context, _ *nats.Msg) context.Context {
			ctx = context.WithValue(ctx, "one", 1)
			return ctx
		}),
		natstransport.SubscriberBefore(func(ctx context.Context, _ *nats.Msg) context.Context {
			if _, ok := ctx.Value("one").(int); !ok {
				errs <- errors.New("Value was not set properly when multiple ServerBefores are used")
			}
			close(done)
			return ctx
		}),
	)

	sub, err := c.QueueSubscribe("natstransport.test", "natstransport", handler.ServeMsg(c))
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := c.Request("natstransport.test", []byte("test data"), 2*time.Second)
		if err != nil {
			errs <- err
		}
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for finalizer")
	}

	select {
	case err := <-errs:
		t.Fatal(err)
	default:
	}

	wg.Wait()
}
