package flux

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

type (
	TickHandler = func(nodeAlias string, deltaTime time.Duration, timestamp time.Time)
	PortHandler = func(nodeAlias string, timestamp time.Time, payload []byte)
)

func (n *Service) OnTick(ctx context.Context, r *message.Router, nodes NodesConfig[any], handler TickHandler) {
	var (
		tick          = make(chan struct{})
		hasGlobalTick = false
	)

	for _, node := range nodes {
		if node.Timer == nil {
			continue
		}

		if node.Timer.IsGlobal {
			hasGlobalTick = true
			go func() {
				defer close(tick)

				lastTick := time.Now()
				for {
					select {
					case <-ctx.Done():
						return

					case <-tick:
						handler(node.Alias, lastTick.Sub(time.Now()), time.Now())
						lastTick = time.Now()
					}
				}
			}()
		} else {
			go func() {
				lastTick := time.Now()
				for {
					select {
					case <-ctx.Done():
						return

					default:
						handler(node.Alias, lastTick.Sub(time.Now()), time.Now())
						lastTick = time.Now()

						time.Sleep(node.Timer.Delay)
					}
				}
			}()
		}
	}

	if hasGlobalTick {
		r.AddNoPublisherHandler(
			"flux.global_tick",
			n.topics.GlobalTick(),
			n.sub,
			func(msg *message.Message) error {
				tick <- struct{}{}
				return nil
			},
		)
	}
}
func (n *Service) OnRestart(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_restart",
		n.topics.Restart(),
		n.sub,
		func(msg *message.Message) error {
			if err := n.UpdateStatus(StatusPaused); err != nil {
				return fmt.Errorf("cannot update status before restart: %w", err)
			}

			err := handler(msg)
			if err != nil {
				return err
			}

			if err := n.UpdateStatus(StatusActive); err != nil {
				return fmt.Errorf("cannot update status after restart: %w", err)
			}

			return nil
		},
	)
}

func (n *Service) OnError(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_error",
		n.topics.Errors(),
		n.sub,
		handler,
	)
}

func (n *Service) OnStart(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_start",
		n.topics.Start(),
		n.sub,
		n.handlerWithStatusUpdate(handler, StatusActive),
	)
}

func (n *Service) OnStop(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_stop",
		n.topics.Stop(),
		n.sub,
		n.handlerWithStatusUpdate(handler, StatusPaused),
	)
}

func (n *Service) handlerWithStatusUpdate(
	handler message.NoPublishHandlerFunc,
	status Status,
) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		err := handler(msg)
		if err != nil {
			return fmt.Errorf("cannot handle message with status update: %w", err)
		}

		if err := n.UpdateStatus(status); err != nil {
			return fmt.Errorf("cannot update status after handler call: %w", err)
		}

		return nil
	}
}

func (n *Service) OnConnect(handler func() error) {
	n.onConnect = handler
}

func (n *Service) OnReady(handler func([]byte, *message.Router, message.Publisher, message.Subscriber) error) {
	n.onReady = handler
}

func (n *Service) OnPort(ctx context.Context, ports NodesPort, r *message.Router, handler PortHandler) error {
	type StreamItem struct {
		Timestamp time.Time
		NodeAlias string
		Payload   []byte
	}

	stream := make(chan StreamItem, len(ports)*3)

	for nodeAlias, port := range ports {
		r.AddNoPublisherHandler(
			fmt.Sprintf("flux.on_port.%s", nodeAlias),
			port.Topic,
			n.sub,
			func(msg *message.Message) error {
				stream <- StreamItem{
					Timestamp: time.Now(),
					NodeAlias: nodeAlias,
					Payload:   msg.Payload,
				}
				msg.Ack()
				return nil
			},
		)
	}

	go func() {
		defer close(stream)

		for {
			select {
			case <-ctx.Done():
				return

			case msg := <-stream:
				handler(msg.NodeAlias, msg.Timestamp, msg.Payload)
			}
		}
	}()

	return nil
}

func (n *Service) OnShutdown() {
	n.sub.Close()
}
