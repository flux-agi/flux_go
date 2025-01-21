package flux

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

type (
	TickHandler  = func(deltaTime time.Duration, timestamp time.Time)
	TickSettings struct {
		IsInfinity bool          `json:"is_infinity"`
		Delay      time.Duration `json:"delay"`
	}
)

func (n *Service) OnTick(ctx context.Context, r *message.Router, handler TickHandler) {
	settings := make(chan TickSettings)

	go func() {
		var (
			data  TickSettings
			timer = time.NewTimer(0)
		)
		defer timer.Stop()

		resetTimer := func() {
			if data.IsInfinity {
				timer.Reset(0)
				return
			}
			timer.Reset(data.Delay)
		}

		for {
			select {
			case <-ctx.Done():
				return

			case d := <-settings:
				data = d
				resetTimer()

			case <-timer.C:
				handler(data.Delay, time.Now())
				resetTimer()
			}
		}
	}()

	r.AddNoPublisherHandler(
		"flux.tick",
		n.topics.Tick(),
		n.sub,
		func(msg *message.Message) error {
			var payload TickSettings
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				return fmt.Errorf("flux: failed to unmarshal tick payload: %w", err)
			}

			settings <- payload

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
