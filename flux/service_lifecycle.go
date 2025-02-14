package flux

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

type TickHandler = func(nodeAlias string, deltaTime time.Duration, timestamp time.Time)

func (s *Service[T]) OnServiceTick(ctx context.Context, r *message.Router, nodes NodesConfig[any], handler TickHandler) {
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
						now := time.Now()
						handler(node.ID, lastTick.Sub(now), now)
						lastTick = now
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
						now := time.Now()
						handler(node.ID, lastTick.Sub(now), now)
						lastTick = now

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
			s.topics.GlobalTick(),
			s.sub,
			func(msg *message.Message) error {
				tick <- struct{}{}
				return nil
			},
		)
	}
}

func (s *Service[T]) OnServiceRestart(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_restart",
		s.topics.Restart(),
		s.sub,
		func(msg *message.Message) error {
			if err := s.UpdateStatus(ServiceStatusPaused); err != nil {
				return fmt.Errorf("cannot update status before restart: %w", err)
			}

			err := handler(msg)
			if err != nil {
				return err
			}

			if err := s.UpdateStatus(ServiceStatusReady); err != nil {
				return fmt.Errorf("cannot update status after restart: %w", err)
			}

			return nil
		},
	)
}

func (s *Service[T]) OnServiceError(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_error",
		s.topics.Errors(),
		s.sub,
		handler,
	)
}

func (s *Service[T]) OnServiceStart(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_start",
		s.topics.Start(),
		s.sub,
		s.handlerWithStatusUpdate(handler, ServiceStatusReady),
	)
}

func (s *Service[T]) OnServiceStop(r *message.Router, handler message.NoPublishHandlerFunc) {
	r.AddNoPublisherHandler(
		"flux.on_stop",
		s.topics.Stop(),
		s.sub,
		s.handlerWithStatusUpdate(handler, ServiceStatusPaused),
	)
}

func (s *Service[T]) handlerWithStatusUpdate(
	handler message.NoPublishHandlerFunc,
	status ServiceStatus,
) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		err := handler(msg)
		if err != nil {
			return fmt.Errorf("cannot handle message with status update: %w", err)
		}

		if err := s.UpdateStatus(status); err != nil {
			return fmt.Errorf("cannot update status after handler call: %w", err)
		}

		return nil
	}
}

func (s *Service[T]) OnServiceConnect(handler func() error) {
	s.onConnect = handler
}

func (s *Service[T]) OnServiceReady(handler func(*message.Router, NodesConfig[T]) error) {
	s.onReady = handler
}

func (s *Service[T]) OnServiceShutdown() {
	if err := s.sub.Close(); err != nil {
		s.logger.Error("failed to close subscriber", slog.String("err", err.Error()))
	}
}
