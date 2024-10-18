package plugin

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"testing"
	"time"
)

func TestFindFileByUserIdAndCreateTime(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2024-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-01")
	uid := "52dde590491d4a1f898f9d1761a2c11e"
	fmt.Println(d.FindFileByUserIdAndCreateTime(ctx, uid, t1, t2))
}

func TestFindFileByIdAndCreateTime(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2021-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-30")
	id := "669a49a18fb2e717570435d0"
	file, err := d.FindFileByIdAndCreateTime(ctx, id, t1, t2)
	if err != nil {
		t.Error(err)
	}
	t.Log(file)
}

func TestFindFileByFileURLAndCreateTime(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2021-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-30")
	url := "https://wcd-file-bucket-test.oss-cn-zhangjiakou.aliyuncs.com/raw/dcb9d920d1b04df7ab90d3eea06a8a5d/20241018101408_1847098811435323392_202297.pdf"
	fmt.Println(d.FindFileByFileURLAndCreateTime(ctx, url, t1, t2))
}

func TestFindFileByNameAndCreateTime(t *testing.T) {
	conf.TestInit()
	ctx := context.Background()
	Init(ctx)
	d := NewFileDao()
	t1, _ := time.Parse("2006-01-02", "2024-07-01")
	t2, _ := time.Parse("2006-01-02", "2024-10-30")
	fmt.Println(d.FindFileByNameAndCreateTime(ctx, "半导体设备", t1, t2))
	// fmt.Println(d.FindFileByNameAndCreateTime(ctx, "半导体", t1, t2))
	// fmt.Println(d.FindFileByNameAndCreateTime(ctx, "设备", t1, t2))
	// fmt.Println(d.FindFileByNameAndCreateTime(ctx, "-29页.pdf", t1, t2))
}
