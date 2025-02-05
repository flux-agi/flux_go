package flux

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type (
	NodeEventHandler = func(nodeAlias string) error
)

type NodeHandlers[T any] struct {
	onReadyHandler func(cfg NodeConfig[T]) error
	onStartHandler NodeEventHandler
	onStopHandler  NodeEventHandler
	onSubscribe    map[string]func(node NodeConfig[T], payload []byte) error
	onDestroy      func(node NodeConfig[T]) error
	onTick         func(node NodeConfig[T], deltaTime time.Duration, timestamp time.Time) error
	mu             sync.Mutex
}

func (n *NodeHandlers[T]) OnReady(handler func(cfg NodeConfig[T]) error) {
	n.onReadyHandler = handler
}

func (n *NodeHandlers[T]) OnStart(handler NodeEventHandler) {
	n.onStartHandler = handler
}

func (n *NodeHandlers[T]) OnStop(handler NodeEventHandler) {
	n.onStopHandler = handler
}

func (n *NodeHandlers[T]) OnSubscribe(port string, handler func(node NodeConfig[T], payload []byte) error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.onSubscribe == nil {
		n.onSubscribe = make(map[string]func(node NodeConfig[T], payload []byte) error)
	}
	n.onSubscribe[port] = handler
}

func (n *NodeHandlers[T]) GetSubscribeHandler(port string) (func(node NodeConfig[T], payload []byte) error, bool) {
	n.mu.Lock()
	defer n.mu.Unlock()
	handler, ok := n.onSubscribe[port]
	return handler, ok
}

func (n *NodeHandlers[T]) OnTick(handler func(node NodeConfig[T], deltaTime time.Duration, timestamp time.Time) error) {
	n.onTick = handler
}

func (n *NodeHandlers[T]) OnDestroy(handler func(node NodeConfig[T]) error) {
	n.onDestroy = handler
}

type Node[T any] struct {
	ctx context.Context

	router *message.Router
	pub    message.Publisher
	sub    message.Subscriber
	config NodeConfig[T]
	status *AtomicValue[NodeStatus]
	state  *AtomicValue[[]byte]

	// Handlers
	onDestroyHandler func(node NodeConfig[T]) error

	// Private
	lastTick time.Time
}

func NewNode[T any](
	ctx context.Context,
	router *message.Router,
	sub message.Subscriber,
	pub message.Publisher,
	config NodeConfig[T],
) *Node[T] {
	return &Node[T]{
		ctx:      ctx,
		router:   router,
		sub:      sub,
		pub:      pub,
		config:   config,
		state:    NewAtomicValue[[]byte](nil),
		lastTick: time.Now(),
	}
}

func (n *Node[T]) RegisterHandlers(handlers *NodeHandlers[T]) error {
	if handlers.onReadyHandler != nil {
		if err := handlers.onReadyHandler(n.config); err != nil {
			return fmt.Errorf("could not run ready handler: %w", err)
		}
	}

	if handlers.onStartHandler != nil {
		n.OnStart(handlers.onStartHandler)
	}

	if handlers.onStopHandler != nil {
		n.OnStop(handlers.onStopHandler)
	}

	if handlers.onDestroy != nil {
		n.onDestroyHandler = handlers.onDestroy
	}

	if handlers.onTick != nil {
		n.OnTick(handlers.onTick)
	}

	for port, handler := range handlers.onSubscribe {
		if err := n.OnSubscribe(port, handler); err != nil {
			return fmt.Errorf("could not register subscribe handler: %w", err)
		}
	}

	go func() {
		if err := n.router.Run(n.ctx); err != nil {
			slog.Error("error run router", slog.String("node", n.config.ID), slog.Any("err", err))
		}
	}()

	return nil
}

func (n *Node[T]) OnReady(handler func(node NodeConfig[T]) error) error {
	if err := handler(n.config); err != nil {
		return fmt.Errorf("could not run ready handler: %w", err)
	}
	return nil
}

func (n *Node[T]) OnStart(handler NodeEventHandler) {
	n.router.AddNoPublisherHandler(
		"flux.node.on_start."+n.config.ID,
		buildTopicNodeEvent(n.config.ID, "start"),
		n.sub,
		func(msg *message.Message) error {
			if err := handler(n.config.ID); err != nil {
				return err
			}
			msg.Ack()
			return nil
		},
	)
}

func (n *Node[T]) OnStop(handler NodeEventHandler) {
	n.router.AddNoPublisherHandler(
		"flux.node.on_stop."+n.config.ID,
		buildTopicNodeEvent(n.config.ID, "stop"),
		n.sub,
		func(msg *message.Message) error {
			if err := handler(n.config.ID); err != nil {
				return err
			}
			msg.Ack()
			return nil
		},
	)
}

func (n *Node[T]) Push(port string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	return n.pub.Publish(
		buildTopicNodePort(n.config.ID, port),
		message.NewMessage(watermill.NewUUID(), payload),
	)
}

func (n *Node[T]) OnSubscribe(port string, handler func(node NodeConfig[T], payload []byte) error) error {
	for _, p := range n.config.Inputs {
		if p.Alias != port {
			continue
		}

		for _, topic := range p.Topics {
			n.router.AddNoPublisherHandler(
				fmt.Sprintf("flux.node.on_subscribe.%s", topic),
				topic,
				n.sub,
				func(msg *message.Message) error {
					if err := handler(n.config, msg.Payload); err != nil {
						return err
					}
					msg.Ack()
					return nil
				},
			)
		}
	}
	return nil
}

func (n *Node[T]) OnTick(handler func(node NodeConfig[T], deltaTime time.Duration, timestamp time.Time) error) {
	switch n.config.Timer.Type {
	case TimerTypeNone:
		return

	case TimerTypeLocal:
		go func() {
			for {
				select {
				case <-n.ctx.Done():
					return
				default:
					if err := handler(n.config, time.Since(n.lastTick), time.Now()); err != nil {
						slog.Error("could not handle tick", slog.Any("err", err))
					}
					time.Sleep(time.Duration(n.config.Timer.Interval) * time.Millisecond)
				}
			}
		}()

	case TimerTypeGlobal:
		n.router.AddNoPublisherHandler(
			"flux.node.on_tick."+n.config.ID,
			"service/tick",
			n.sub,
			func(msg *message.Message) error {
				if err := handler(n.config, time.Since(n.lastTick), time.Now()); err != nil {
					return err
				}
				return nil
			},
		)
	}

}

func (n *Node[T]) OnDestroy(handler func(node NodeConfig[T]) error) {
	n.onDestroyHandler = handler
}

func (n *Node[T]) State() []byte {
	value, ok := n.state.Get()
	if !ok {
		return nil
	}
	return value
}

func (n *Node[T]) SetState(value []byte) error {
	n.state.Set(value)
	return nil
}

func (n *Node[T]) Status() NodeStatus {
	value, ok := n.status.Get()
	if !ok {
		return NodeStatusError
	}
	return value
}

func (n *Node[T]) SetStatus(status NodeStatus) error {
	n.status.Set(status)
	return nil
}

func (n *Node[T]) Close() error {
	var err error
	if routerErr := n.router.Close(); routerErr != nil {
		errors.Join(err, fmt.Errorf("could not close router: %w", routerErr))
	}

	if subErr := n.sub.Close(); subErr != nil {
		errors.Join(err, fmt.Errorf("could not close subscriber: %w", subErr))
	}

	if n.onDestroyHandler != nil {
		if destroyErr := n.onDestroyHandler(n.config); destroyErr != nil {
			errors.Join(destroyErr, fmt.Errorf("could not run destroy handler: %w", destroyErr))
		}
	}

	return err
}

func buildTopicNodePort(alias, port string) string { return fmt.Sprintf("node/%s/%s", alias, port) }
func buildTopicNodeEvent(alias, event string) string {
	return fmt.Sprintf("node/%s/event/%s", alias, event)
}
