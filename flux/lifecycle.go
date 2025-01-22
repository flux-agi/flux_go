package flux

import (
	"context"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

type GlobalTickHandler = func(deltaTime time.Duration, timestamp time.Time)

func (n *Service) OnGlobalTick(ctx context.Context, r *message.Router, handler GlobalTickHandler) {
	tick := make(chan struct{})
	go func() {
		lastTick := time.Now()
		for {
			select {
			case <-ctx.Done():
				return

			case <-tick:
				handler(lastTick.Sub(time.Now()), time.Now())
				lastTick = time.Now()
			}
		}
	}()

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
