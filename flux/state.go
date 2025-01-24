package flux

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
)

func (s *Service[T]) RegisterStateHandler(r *message.Router) {
	r.AddHandler(
		"flux.request_common_state",
		s.topics.GetCommonState(),
		s.sub,
		s.topics.SetCommonState(),
		s.pub,
		s.handleStateRequest,
	)
}

func (s *Service[T]) handleStateRequest(_ *message.Message) ([]*message.Message, error) {
	state := s.State()

	return []*message.Message{
		message.NewMessage(watermill.NewUUID(), state),
	}, nil
}

func (s *Service[T]) SetState(value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("could not marshal state: %w", err)
	}

	s.state.Set(raw)
	err = s.pub.Publish(
		s.topics.SetCommonState(),
	)
	if err != nil {
		return fmt.Errorf("could not publish state: %w", err)
	}

	return nil
}

//func (s *Service[T]) handleOnNodeReady() error {
//	for _, node := range s.nodes {
//		node.
//	}
//}
