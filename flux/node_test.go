package flux

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"testing"
	"time"
)

func TestNode_Lifecycle(t *testing.T) {
	t.Parallel()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	t.Cleanup(cancel)

	service := NewService[string]("keyFireTestService")

	resChan := make(chan struct{})

	service.OnNodeReady(func(cfg NodeConfig[string]) error {
		fmt.Printf("node %s is ready\n", cfg.Alias)
		resChan <- struct{}{}
		return nil
	})

	service.OnTick(func(node NodeConfig[string], deltaTime time.Duration, timestamp time.Time) error {
		fmt.Println("tick")
		return nil
	})

	<-resChan

	if _, err := service.Run(ctx); err != nil {
		t.Errorf("failed to run service: %s", err)
		return
	}

	fmt.Printf("service started\n")
}
