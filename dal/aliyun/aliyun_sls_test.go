package aliyun

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"fmt"
	sls "github.com/aliyun/aliyun-log-go-sdk"
	"strings"
	"testing"
	"time"
)

func TestQueryLog(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//QueryLog(ctx, 0)
}

func TestQueryLog2(t *testing.T) {
	//client := sls.CreateNormalInterface(consts.ENDPOINT, accessKeyID, accessKeySecret, "")
	client := sls.CreateNormalInterface(consts.ENDPOINT, consts.ACCESS_KEY_ID, consts.ACCESS_KEY_SECRET, "")

	//logstore, err := client.GetLogStore(consts.proj, logStoreName)
	logstore, err := client.GetLogStore(consts.PROJECT_NAME, consts.LOG_STORE_NAME)
	if err != nil {
		panic(err)
	}
	fmt.Println("Get logstore successfully:", logstore.Name)

	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	from := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, yesterday.Location()).Unix()
	to := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 23, 59, 59, 999999999, yesterday.Location()).Unix()

	// 查询日志
	resp, err := logstore.GetLogs("", from, to, "(__tag__:_container_name_ : lingowhale-python-prod and message : \"summary core core_name:大纲\"  )|select message,asctime ", 100000, 0, false)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 打印查询结果
	for _, log := range resp.Logs {
		fmt.Println(log)
	}
}

func TestNginxIngressLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	logs, err := NginxIngressLogQuery(ctx, 0)
	if err != nil {
		t.Error(err)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}

}

func TestOutlineLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	//logs, e := CoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
	logs, e := CoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
	if e != nil {
		t.Error(e)
	} else {
		for _, log := range logs {
			fmt.Println(log)
		}
	}
}

func TestNginxReportThisMonth(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	NginxReportThisMonth(ctx)
}

func TestCoreLogQuery(t *testing.T) {
	ctx := context.Background()
	Init(ctx)

	coreLogs, err := CoreLogQuery(ctx, 0, consts.CORE_NAME_OUTLINE)
	if err != nil {
		t.Error(err)
	}
	nodes := []string{
		consts.ALIYUN_LOG_NODE_OUTLINE_AI_COST,
		consts.ALIYUN_LOG_NODE_OUTLINE_ETOE_COST,
	}
	for _, log := range coreLogs {
		if utils.Contains(nodes, log.Node) {
			fmt.Println(log)
		}
	}

}

func TestCoreReportThisMonth(t *testing.T) {
	ctx := context.Background()
	Init(ctx)
	CoreReportThisMonth(ctx, consts.CORE_NAME_OUTLINE)

}

func TestStatusCodeUpdate(t *testing.T) {
	s := "300,200"
	codes := strings.Split(s, ",")
	if !utils.Contains(codes, "400") {
		codes = append(codes, "400")
		s = strings.Join(codes, ",")
	}
	fmt.Println(s)
}
