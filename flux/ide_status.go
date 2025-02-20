package flux

import (
	"encoding/json"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
)

func (s *Service[T]) RegisterIDEStatusHandler(router *message.Router) {
	router.AddNoPublisherHandler(
		"flux.ide_status",
		s.topics.IDEStatus(),
		s.Sub(),
		s.handleIDEStatus,
	)
}

type IDEStatus string

const (
	IDEStatusConnected    IDEStatus = "CONNECTED"
	IDEStatusDisconnected IDEStatus = "DISCONNECTED"
)

type IDEStatusMessage struct {
	Status IDEStatus `json:"status"`
}

func (s *Service[T]) handleIDEStatus(msg *message.Message) error {
	var payload IDEStatusMessage
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		return fmt.Errorf("flux: failed to unmarshal payload: %w", err)
	}

	if s.onIDEStatus != nil {
		err := s.onIDEStatus(payload)
		if err != nil {
			return fmt.Errorf("flux: failed to handle ide status: %w", err)
		}
	}

	return nil
}
