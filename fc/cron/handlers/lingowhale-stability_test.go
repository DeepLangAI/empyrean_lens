package handlers

import (
	"context"
	"empyrean_lens/conf"
	"testing"
)

func TestLingowhaleStability_Handle(t *testing.T) {
	conf.InitConfig()
	ctx := context.Background()

	h := NewLingowhaleStability()
	h.Handle(ctx, "")
}
