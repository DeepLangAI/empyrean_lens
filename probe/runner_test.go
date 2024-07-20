package probe

import (
	"context"
	"testing"
)

func TestProbeRunner_Run(t *testing.T) {
	ctx := context.Background()
	runner := ProbeRunner{}
	runner.Run(ctx)
}
