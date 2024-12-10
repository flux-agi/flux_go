package fluxtest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

type FakeManager struct {
	pub message.Publisher
	sub message.Subscriber

	configs map[string]any
}

func NewFakeManager(
	pub message.Publisher,
	sub message.Subscriber,
	configs map[string]any,
) *FakeManager {
	return &FakeManager{
		pub:     pub,
		sub:     sub,
		configs: configs,
	}
}

// Run will be handle get_config requests and send config into it's output topics.
//
// It returns stop function. Call it before closing subscriber, to shut down gracefully.
// Stop should be called before closing pub/sub (and router).
func (f *FakeManager) Run(ctx context.Context) func() {
	ctx, cancel := context.WithCancel(ctx)
	for service, config := range f.configs {
		f.run(ctx, service, config)
	}

	return cancel
}

func (f *FakeManager) run(ctx context.Context, service string, config any) {
	messages, err := f.sub.Subscribe(
		ctx,
		fmt.Sprintf("service/%s/get_config", service),
	)
	if err != nil {
		panic(fmt.Errorf("could not subscribe to config: %w", err))
	}

	go func() {
		for {
			select {
			case msg := <-messages:
				data, err := json.Marshal(config)
				if err != nil {
					panic(fmt.Errorf("failed to marshal config: %w", err))
				}

				err = f.pub.Publish(
					fmt.Sprintf("service/%s/set_config", service),
					message.NewMessage(watermill.NewUUID(), data),
				)
				if err != nil {
					panic(fmt.Errorf("could not publish requested config: %w", err))
				}

				msg.Ack()
			case <-ctx.Done():
				return
			}
		}
	}()
}
