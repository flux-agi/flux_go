package flux_test

import (
	"context"
	"os"
	"os/signal"
	"testing"
)

type NodeConf struct {
	Id  int    `json:"idx"`
	Raw []byte `json:"raw"`
}

func TestNode_GetConfig(t *testing.T) {
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	<-ctx.Done()
}
