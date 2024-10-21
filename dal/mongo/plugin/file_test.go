package plugin

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestFindFileByTimeRange(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2024-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-01")
	fmt.Println(d.FindFileByTimeRange(ctx, []int32{}, t1, t2, 0, 10))
}

func TestFindFileByQueryAndTimeRange(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2021-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-30")
	query := "669a49a18fb2e717570435d0"
	file, err := d.FindFileByQueryAndTimeRange(ctx, query, []int32{}, t1, t2, 0, 10)
	if err != nil {
		t.Error(err)
	}
	t.Log(file)
}
