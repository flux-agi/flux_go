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
func (n *Service) GetConfig(ctx context.Context) ([]byte, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	messages, err := n.sub.Subscribe(
		ctx,
		n.topics.SetConfig(),
	)
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to config: %w", err)
	}

	err = n.sendConfigRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to register request config: %w", err)
	}

	select {
	case msg := <-messages:
		msg.Ack()
		return msg.Payload, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("context is canceled before the configuration message is received: %w", ctx.Err())
	}
}

func (n *Service) sendConfigRequest() error {
	msg := message.NewMessage(watermill.NewUUID(), []byte(n.serviceName))
	err := n.pub.Publish(
		n.topics.GetConfig(),
		msg,
	)
	if err != nil {
		return fmt.Errorf("could not publish request config: %w", err)
	}

	return nil
}

// GetConfig returns unmarshalled config.
//
// It uses Service.GetConfig and unmarshal it into generic type.
func GetConfig[T any](ctx context.Context, service *Service) (NodesConfig[T], error) { //nolint:ireturn
	var zero NodesConfig[T]

	raw, err := service.GetConfig(ctx)
	if err != nil {
		return zero, fmt.Errorf("could not get config: %w", err)
	}

	var cfg NodesConfig[T]
	if err = json.Unmarshal(raw, &cfg); err != nil {
		return zero, fmt.Errorf("could not unmarshal config: %w", err)
	}

	return cfg, nil
}
