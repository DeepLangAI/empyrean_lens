package probe

import (
	"context"
	"testing"
)

func TestAgent_Login(t *testing.T) {
	ctx := context.Background()
	agent := NewAgent()
	agent.Login(ctx)
	agent.Logout(ctx)
}
