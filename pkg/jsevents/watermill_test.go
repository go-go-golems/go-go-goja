package jsevents_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dop251/goja"
	gggengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/pkg/jsevents"
	"github.com/stretchr/testify/require"
)

type fakeSubscriber struct {
	mu     sync.Mutex
	chans  map[string]chan *message.Message
	closed bool
}

func newFakeSubscriber() *fakeSubscriber {
	return &fakeSubscriber{chans: map[string]chan *message.Message{}}
}

func (s *fakeSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil, fmt.Errorf("subscriber closed")
	}
	ch, ok := s.chans[topic]
	if !ok {
		ch = make(chan *message.Message, 16)
		s.chans[topic] = ch
	}
	s.mu.Unlock()

	out := make(chan *message.Message)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				select {
				case <-ctx.Done():
					return
				case out <- msg:
				}
			}
		}
	}()
	return out, nil
}

func (s *fakeSubscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	for _, ch := range s.chans {
		close(ch)
	}
	return nil
}

func (s *fakeSubscriber) publish(topic string, msg *message.Message) {
	s.mu.Lock()
	ch, ok := s.chans[topic]
	if !ok {
		ch = make(chan *message.Message, 16)
		s.chans[topic] = ch
	}
	s.mu.Unlock()
	ch <- msg
}

func TestWatermillHelperConnectsJSEmitterAndAckMessage(t *testing.T) {
	sub := newFakeSubscriber()
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.WatermillHelper(jsevents.WatermillOptions{
			Subscriber: sub,
			AllowTopic: func(topic string) bool {
				return topic == "orders"
			},
		}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.watermill.setup", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.seen = [];
			globalThis.orders = new EventEmitter();
			globalThis.conn = watermill.connect("orders", globalThis.orders);
			globalThis.orders.on("message", (msg) => {
				seen.push({ uuid: msg.uuid, payload: msg.payload, source: msg.metadata.source });
				msg.ack();
			});
		`)
		return nil, err
	})
	require.NoError(t, err)

	msg := message.NewMessage("msg-1", []byte(`{"id":1}`))
	msg.Metadata.Set("source", "test")
	sub.publish("orders", msg)

	require.Eventually(t, func() bool {
		select {
		case <-msg.Acked():
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)

	got := runJS(t, rt, `JSON.stringify(globalThis.seen)`)
	require.JSONEq(t, `[{"uuid":"msg-1","payload":"{\"id\":1}","source":"test"}]`, got)
}

func TestWatermillHelperNacksWhenNoListener(t *testing.T) {
	sub := newFakeSubscriber()

	// Create a connection without registering a message listener.
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.WatermillHelper(jsevents.WatermillOptions{Subscriber: sub}),
	)
	_, err := rt.Owner.Call(context.Background(), "jsevents.watermill.no-listener", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			globalThis.orders = new EventEmitter();
			watermill.connect("missing-listener", globalThis.orders);
		`)
		return nil, err
	})
	require.NoError(t, err)

	msg := message.NewMessage("msg-2", []byte(`{"id":2}`))
	sub.publish("missing-listener", msg)

	require.Eventually(t, func() bool {
		select {
		case <-msg.Nacked():
			return true
		default:
			return false
		}
	}, time.Second, 10*time.Millisecond)
}

func TestWatermillHelperRejectsDisallowedTopic(t *testing.T) {
	sub := newFakeSubscriber()
	rt := newRuntime(t,
		jsevents.Install(),
		jsevents.WatermillHelper(jsevents.WatermillOptions{
			Subscriber: sub,
			AllowTopic: func(topic string) bool { return topic == "allowed" },
		}),
	)

	_, err := rt.Owner.Call(context.Background(), "jsevents.watermill.disallowed", func(_ context.Context, vm *goja.Runtime) (any, error) {
		_, err := vm.RunString(`
			const EventEmitter = require("events");
			watermill.connect("blocked", new EventEmitter());
		`)
		return nil, err
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
}

func TestWatermillHelperRequiresManager(t *testing.T) {
	sub := newFakeSubscriber()
	factory, err := gggengine.NewBuilder().
		WithRuntimeInitializers(jsevents.WatermillHelper(jsevents.WatermillOptions{Subscriber: sub})).
		Build()
	require.NoError(t, err)
	_, err = factory.NewRuntime(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "manager is not installed")
}
