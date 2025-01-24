package flux

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (n *Service[T]) RegisterStateHandler(r *message.Router) {
	r.AddHandler(
		"flux.request_common_state",
		n.topics.GetCommonState(),
		n.sub,
		n.topics.SetCommonState(),
		n.pub,
		n.handleStateRequest,
	)
}

func (n *Service[T]) handleStateRequest(_ *message.Message) ([]*message.Message, error) {
	state := n.State()

	return []*message.Message{
		message.NewMessage(watermill.NewUUID(), state),
	}, nil
}

func (n *Service[T]) SetState(value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not marshal state: %w", err)
	}

	n.state.Set(raw)
	err = n.pub.Publish(
		n.topics.SetCommonState(),
	)
	if err != nil {
		return fmt.Errorf("could not publish state: %w", err)
	}

	return nil
}
