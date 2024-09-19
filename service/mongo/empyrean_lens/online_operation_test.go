package empyrean_lens

import (
	"context"
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"fmt"
	"testing"
)

func TestFindOnlineOperations(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	req := empyrean_lens.OnlineOperationReq{
		TimeBegin: "2024-09-19 12:37:06",
		TimeEnd:   "2024-09-19 14:37:06",
		AppName:   "",
	}
	operations, err := FindOnlineOperations(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	for _, operation := range operations {
		fmt.Println(operation)
	}
}
