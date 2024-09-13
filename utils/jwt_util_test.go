package utils

import (
	"testing"
	"time"
)

func TestJWT(t *testing.T) {
	token, err := GenerateJWT(map[string]any{
		"username": "test",
	}, time.Hour, "123")
	if err != nil {
		t.Error(err)
	}
	claim, err := ParseJWT(token, "123")
	if err != nil {
		t.Error(err)
	}
	t.Log(claim["username"])
	t.Log(claim["exp"])
}
