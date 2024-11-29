package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGenerateErrLogDao_CountGenerateErrLogsByTimeRange(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	t.Run("", func(t *testing.T) {
		self := &GenerateErrLogDao{}
		got, err := self.CountGenerateErrLogsByTimeRange(ctx, "2024-11-20", "2024-11-24")
		assert.Nil(t, err)
		fmt.Println(got)
	})

	t.Run("全部", func(t *testing.T) {
		self := &GenerateErrLogDao{}
		got, err := self.CountGenerateErrLogsByTimeRange(ctx, "", "")
		assert.Nil(t, err)
		fmt.Println(got)
	})
}
