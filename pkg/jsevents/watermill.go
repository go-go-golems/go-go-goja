package jsevents

import (
	"context"
	"fmt"
	"sync"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/dop251/goja"
	"github.com/go-go-golems/go-go-goja/engine"
)

// WatermillOptions configures the opt-in JavaScript helper installed by
// WatermillHelper.
type WatermillOptions struct {
	// GlobalName is the JavaScript global object name. Defaults to "watermill".
	GlobalName string
	Subscriber message.Subscriber
	AllowTopic func(topic string) bool
}

type watermillHelper struct {
	opts WatermillOptions
}

// WatermillHelper installs a JS-callable helper object with connect(topic,
// emitter). It does not subscribe to anything until JavaScript calls connect.
func WatermillHelper(opts WatermillOptions) engine.RuntimeInitializer {
	return &watermillHelper{opts: opts}
}

func (h *watermillHelper) ID() string { return "jsevents.watermill-helper" }

func (h *watermillHelper) InitRuntime(ctx *engine.RuntimeContext) error {
	if ctx == nil || ctx.VM == nil {
		return fmt.Errorf("jsevents watermill: incomplete runtime context")
	}
	if h.opts.Subscriber == nil {
		return fmt.Errorf("jsevents watermill: subscriber is nil")
	}
	managerValue, ok := ctx.Value(RuntimeValueKey)
	if !ok {
		return fmt.Errorf("jsevents watermill: manager is not installed; add jsevents.Install() before WatermillHelper")
	}
	manager, ok := managerValue.(*Manager)
	if !ok || manager == nil {
		return fmt.Errorf("jsevents watermill: invalid manager value")
	}

	globalName := h.opts.GlobalName
	if globalName == "" {
		globalName = "watermill"
	}

	obj := ctx.VM.NewObject()
	if err := obj.Set("connect", func(call goja.FunctionCall) goja.Value {
		topic := call.Argument(0).String()
		if h.opts.AllowTopic != nil && !h.opts.AllowTopic(topic) {
			panic(ctx.VM.NewGoError(fmt.Errorf("watermill topic %q is not allowed", topic)))
		}
		ref, err := manager.AdoptEmitterOnOwner(call.Argument(1))
		if err != nil {
			panic(ctx.VM.NewGoError(err))
		}
		subCtx, cancel := context.WithCancel(ctx.Context)
		ref.SetCancel(cancel)
		go runWatermillSubscription(subCtx, ref, h.opts.Subscriber, topic)
		return connectionObject(ctx.VM, ref)
	}); err != nil {
		return err
	}

	return ctx.VM.Set(globalName, obj)
}

func connectionObject(vm *goja.Runtime, ref *EmitterRef) *goja.Object {
	obj := vm.NewObject()
	_ = obj.Set("id", ref.ID())
	_ = obj.Set("close", func() bool {
		return ref.Close(context.Background()) == nil
	})
	return obj
}

func runWatermillSubscription(ctx context.Context, ref *EmitterRef, sub message.Subscriber, topic string) {
	messages, err := sub.Subscribe(ctx, topic)
	if err != nil {
		_ = ref.Emit(ctx, "error", map[string]any{
			"source":  "watermill",
			"topic":   topic,
			"message": err.Error(),
		})
		_ = ref.Close(ctx)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				_ = ref.Emit(ctx, "close")
				_ = ref.Close(ctx)
				return
			}
			dispatchWatermillMessage(ctx, ref, msg)
		}
	}
}

func dispatchWatermillMessage(ctx context.Context, ref *EmitterRef, msg *message.Message) {
	delivered, err := ref.EmitWithBuilderSync(ctx, "message", func(vm *goja.Runtime) ([]goja.Value, error) {
		jsMsg := vm.NewObject()
		metadata := map[string]string{}
		for key, value := range msg.Metadata {
			metadata[key] = value
		}

		var settleOnce sync.Once
		_ = jsMsg.Set("uuid", msg.UUID)
		_ = jsMsg.Set("payload", string(msg.Payload))
		_ = jsMsg.Set("metadata", metadata)
		_ = jsMsg.Set("ack", func() bool {
			called := false
			settleOnce.Do(func() { called = msg.Ack() })
			return called
		})
		_ = jsMsg.Set("nack", func() bool {
			called := false
			settleOnce.Do(func() { called = msg.Nack() })
			return called
		})

		return []goja.Value{jsMsg}, nil
	})
	if err != nil || !delivered {
		_ = msg.Nack()
	}
}
