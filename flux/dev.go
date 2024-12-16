package flux

import (
	"errors"
	"fmt"
	"os"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/google/uuid"
)

func (n *Node) EnableDevMode() error {
	var (
		filename = "dev_mode.ini"
		id       string
	)

	makeFileFunc := func() error {
		file, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("could not create dev_mode.ini: %w", err)
		}

		id = uuid.NewString()

		if _, err := file.WriteString(id); err != nil {
			return fmt.Errorf("could not write dev_mode.ini: %w", err)
		}

		return nil
	}

	openedFile, err := os.ReadFile(filename)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("could not read dev_mode.ini: %w", err)
		}

		if err := makeFileFunc(); err != nil {
			return err
		}
	} else {
		id = string(openedFile)
	}

	msg := message.NewMessage(watermill.NewUUID(), []byte(n.serviceName))
	publishErr := n.pub.Publish(
		n.topics.PushDevelopmentMode(id),
		msg,
	)
	if publishErr != nil {
		return fmt.Errorf("could not publish request config: %w", err)
	}

	return nil
}
