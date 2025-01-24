package flux

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

type TickHandler = func(nodeAlias string, deltaTime time.Duration, timestamp time.Time)

func (n *Service[T]) OnServiceTick(ctx context.Context, r *message.Router, nodes NodesConfig[any], handler TickHandler) {
	var (
		tick          = make(chan struct{})
		hasGlobalTick = false
	)

	for _, node := range nodes {
		if node.Timer == nil {
			continue
		}

		switch node.Timer.Type {
		case TimerTypeGlobal:
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
		case TimerTypeLocal:
			go func() {
				lastTick := time.Now()
				for {
					select {
					case <-ctx.Done():
						return

					default:
						handler(node.Alias, lastTick.Sub(time.Now()), time.Now())
						lastTick = time.Now()

						time.Sleep(time.Duration(node.Timer.Interval) * time.Millisecond)
					}
				}
			}()
		case TimerTypeNone:
			continue
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

func (n *Service[T]) OnServiceRestart(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_restart",
		n.topics.Restart(),
		n.sub,
		func(msg *message.Message) error {
			if err := n.UpdateStatus(ServiceStatusPaused); err != nil {
				return fmt.Errorf("cannot update status before restart: %w", err)
			}

			err := handler(msg)
			if err != nil {
				return err
			}

			if err := n.UpdateStatus(ServiceStatusActive); err != nil {
				return fmt.Errorf("cannot update status after restart: %w", err)
			}

			return nil
		},
	)
}

func (n *Service[T]) OnServiceError(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_error",
		n.topics.Errors(),
		n.sub,
		handler,
	)
}

func (n *Service[T]) OnServiceStart(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_start",
		n.topics.Start(),
		n.sub,
		n.handlerWithStatusUpdate(handler, ServiceStatusActive),
	)
}

func (n *Service[T]) OnServiceStop(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_stop",
		n.topics.Stop(),
		n.sub,
		n.handlerWithStatusUpdate(handler, ServiceStatusPaused),
	)
}

func (n *Service[T]) handlerWithStatusUpdate(
	handler message.NoPublishHandlerFunc,
	status ServiceStatus,
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

func (n *Service[T]) OnServiceConnect(handler func() error) {
	n.onConnect = handler
}

func (n *Service[T]) OnServiceReady(handler func(NodesConfig[T], *message.Router, message.Publisher, message.Subscriber) error) {
	n.onReady = handler
}

func (n *Service[T]) OnServiceShutdown() {
	if err := n.sub.Close(); err != nil {
		n.logger.Error("failed to close subscriber", slog.String("err", err.Error()))
	}
}
