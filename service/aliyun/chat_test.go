package aliyun

import (
	"bufio"
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"empyrean_lens/dal"
	"empyrean_lens/utils"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"testing"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Delta struct {
	Content string `json:"content"`
}

type Choice struct {
	Index        int     `json:"index"`
	Delta        Delta   `json:"delta"`
	FinishReason *string `json:"finish_reason"` // 使用指针以处理可能为 null 的情况
}

type ChatCompletionChunk struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

func TestChat(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	//prompt := BuildLogAnlzPrompt(ctx, "ehqErshgqTmiqwBgLqRcj", "2024-08-15")
	//prompt := BuildLogAnlzPrompt(ctx, "66bc36719a11d738b9ec3112", "2024-08-14")
	//prompt := BuildLogAnlzPrompt(ctx, "66c2aa692d65bf63af3ca27f", "2024-08-19")
	//prompt := BuildLogAnlzPrompt(ctx, "66cdc5dace30fa9ead513fe0", "2024-08-27")
	prompt := BuildLogAnlzPrompt(ctx, "N-hmQD1Wf2qv6a6TP9bET", "2024-08-28")

	client := &http.Client{}
	requestData := Request{
		Model: consts.ModelName,
		Messages: []Message{
			{
				Role: "system",
				Content: `你是一个程序员，也是个专业的日志分析者。请根据日志内容，梳理出链路调用过程，以及在哪些环节出现了异常，并给出详细的分析，不需要你给出解决建议。
以下是一些关于日志字段的背景知识：
__tag__:_container_name_，容器名称，也可以认为是服务名称
`,
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	fmt.Println(prompt)
	var data = strings.NewReader(utils.JSONMarshal(requestData))
	//req, err := http.NewRequest("POST", "https://api.moonshot.cn/v1/chat/completions", data)
	req, err := http.NewRequest("POST", consts.ChatApi, data)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+consts.ChatSecret)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	inputReader := bufio.NewReader(resp.Body)
	respBuilder := strings.Builder{}
	for {
		line, err := inputReader.ReadString('\n')
		if err == io.EOF {
			break
		}
		// 利用正则，找到{}及中的内容
		pattern := regexp.MustCompile(`\{.*\}`)
		find := pattern.Find([]byte(line))
		if string(find) == "" {
			continue
		}

		chunk := ChatCompletionChunk{}
		utils.JSONUnMarshal([]byte(find), &chunk)
		//hlog.CtxInfof(ctx, chunk.Choices[0].Delta.Content)
		fmt.Print(chunk.Choices[0].Delta.Content)
		respBuilder.WriteString(chunk.Choices[0].Delta.Content)
	}
	fmt.Println("========================================")
	fmt.Println(respBuilder.String())
}

func TestBuildLogAnlzPrompt(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()

	prompt := BuildLogAnlzPrompt(ctx, "8Rqx02qDUW-UfUJ0WnuXq", "2024-08-15")
	fmt.Println(prompt)
}
