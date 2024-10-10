package nats_test

import (
	"context"
	"strings"
	"testing"
	"time"

	natstransport "github.com/go-kit/kit/transport/nats"
	"github.com/nats-io/nats.go"
)

func TestPublisher(t *testing.T) {
	var (
		testdata = "testdata"
		encode   = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode   = func(_ context.Context, msg *nats.Msg) (interface{}, error) {
			return TestResponse{string(msg.Data), ""}, nil
		}
	)

	s, c, err := newNATSConn(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	defer c.Close()

	sub, err := c.QueueSubscribe("natstransport.test", "natstransport", func(msg *nats.Msg) {
		if err := c.Publish(msg.Reply, []byte(testdata)); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	publisher := natstransport.NewPublisher(
		c,
		"natstransport.test",
		encode,
		decode,
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	response, ok := res.(TestResponse)
	if !ok {
		t.Fatal("response should be TestResponse")
	}
	if want, have := testdata, response.String; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

}

func TestPublisherBefore(t *testing.T) {
	var (
		testdata = "testdata"
		encode   = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode   = func(_ context.Context, msg *nats.Msg) (interface{}, error) {
			return TestResponse{string(msg.Data), ""}, nil
		}
	)

	s, c, err := newNATSConn(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	defer c.Close()

	sub, err := c.QueueSubscribe("natstransport.test", "natstransport", func(msg *nats.Msg) {
		if err := c.Publish(msg.Reply, msg.Data); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	publisher := natstransport.NewPublisher(
		c,
		"natstransport.test",
		encode,
		decode,
		natstransport.PublisherBefore(func(ctx context.Context, msg *nats.Msg) context.Context {
			msg.Data = []byte(strings.ToUpper(string(testdata)))
			return ctx
		}),
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	response, ok := res.(TestResponse)
	if !ok {
		t.Fatal("response should be TestResponse")
	}
	if want, have := strings.ToUpper(testdata), response.String; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

}

func TestPublisherAfter(t *testing.T) {
	var (
		testdata = "testdata"
		encode   = func(context.Context, *nats.Msg, interface{}) error { return nil }
		decode   = func(_ context.Context, msg *nats.Msg) (interface{}, error) {
			return TestResponse{string(msg.Data), ""}, nil
		}
	)

	s, c, err := newNATSConn(t)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Shutdown()
	defer c.Close()

	sub, err := c.QueueSubscribe("natstransport.test", "natstransport", func(msg *nats.Msg) {
		if err := c.Publish(msg.Reply, []byte(testdata)); err != nil {
			t.Fatal(err)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sub.Unsubscribe()

	publisher := natstransport.NewPublisher(
		c,
		"natstransport.test",
		encode,
		decode,
		natstransport.PublisherAfter(func(ctx context.Context, msg *nats.Msg) context.Context {
			msg.Data = []byte(strings.ToUpper(string(msg.Data)))
			return ctx
		}),
	)

	res, err := publisher.Endpoint()(context.Background(), struct{}{})
	if err != nil {
		t.Fatal(err)
	}

	response, ok := res.(TestResponse)
	if !ok {
		t.Fatal("response should be TestResponse")
	}
	if want, have := strings.ToUpper(testdata), response.String; want != have {
		t.Errorf("want %q, have %q", want, have)
	}

}
