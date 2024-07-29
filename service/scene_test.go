package service

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
	"time"
)

func TestSceneResult(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	timeBegin := time.Date(2024, 7, 20, 0, 0, 0, 0, time.UTC)
	timeEnd := time.Date(2024, 7, 26, 0, 0, 0, 0, time.UTC)

	models, err := SceneResult(ctx, timeBegin, timeEnd)
	if err != nil {

		t.Errorf("SceneResult failed: %v", err)
		return
	}

	for _, model := range models {
		t.Logf("Model: %v", model)
	}
}
