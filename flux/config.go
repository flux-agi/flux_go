package flux

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

// GetConfig returns raw config from manager.
//
// It subscribes on /set_config topic and sends /get_config message to request config from manager.
// It's a blocking function. Use context.WithTimeout to set waiting timeout. When the context will be canceled,
// GetConfig will return context error.
func (s *Service[T]) GetConfig(ctx context.Context) (*NodesConfig[T], error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messages, err := s.sub.Subscribe(
		ctx,
		s.topics.ResponseConfig(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to config: %w", err)
	}

	err = s.sendConfigRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to register request config: %w", err)
	}

	select {
	case msg := <-messages:
		msg.Ack()

		var config NodesConfig[T]
		err := json.Unmarshal(msg.Payload, &config)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		return &config, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context is canceled before the configuration message is received: %w", ctx.Err())
	}
}

func (s *Service[T]) sendConfigRequest() error {
	msg := message.NewMessage(watermill.NewUUID(), []byte(s.serviceID))
	err := s.pub.Publish(
		s.topics.RequestConfig(),
		msg,
	)
	if err != nil {
		return fmt.Errorf("could not publish request config: %w", err)
	}

	return nil
}
