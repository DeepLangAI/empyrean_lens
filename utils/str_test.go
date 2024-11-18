package utils

import (
	"empyrean_lens/consts"
	"fmt"
	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
)

func TestJSONMarshal(t *testing.T) {
	p := Person{
		Name: "John",
		Age:  30,
	}
	marshal := JSONMarshal(p)
	t.Log(marshal)
}

func TestDecodeMIME(t *testing.T) {
	s := "=?utf-8?b?5YmN5pa55qih5Z6L5Y2H57qn77yM6K+36YeN5paw5LiK5Lyg5paH56ug6YeN6K+V772e?="
	assert.True(t, DecodeMIME(s) == "前方模型升级，请重新上传文章重试～")
}

func TestContains(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		assert.True(t, Contains([]int{1, 2, 3}, 2))
	})
	t.Run("str", func(t *testing.T) {
		assert.True(t, Contains([]string{"a", "b", "c"}, "b"))
	})
}
func TestActionIO_TranslateJsonIO(t *testing.T) {
	//ctx := context.Background()
	// 示例 JSON
	jsonStr := `{
		"name": "example",
		"details": {
			"description": "This is a very long description that exceeds the length limit%v...",
			"tags": ["gogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogo%v", "json", "example"],
			"nested": {
				"note": "This is a note",
				"longField": "a long string that is way too long%v"
			},
			"number": 12,
			"float": 3.14
		}
	}`
	longStr := make([]byte, consts.DataTooLongUpper)
	for i := range longStr {
		longStr[i] = 'a'
	}
	jsonStr = fmt.Sprintf(jsonStr, string(longStr), string(longStr), string(longStr))

	// 解析 JSON
	var jsonData interface{}
	if err := sonic.Unmarshal([]byte(jsonStr), &jsonData); err != nil {
		log.Fatalf("Error parsing JSON: %v", err)
	}

	// 处理 JSON
	processedData := processJSON(jsonData)

	// 转换回 JSON 字符串
	result, err := sonic.Marshal(processedData)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	// 输出结果
	fmt.Println(string(result))

}
func TestActionIO_TranslateJsonIO1(t *testing.T) {
	//ctx := context.Background()
	t.Run("success", func(t *testing.T) {
		// 示例 JSON
		jsonStr := `{
		"name": "example",
		"details": {
			"description": "This is a very long description that exceeds the length limit%v...",
			"tags": ["gogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogo%v", "json", "example"],
			"nested": {
				"note": "This is a note",
				"longField": "a long string that is way too long%v"
			},
			"number": 12,
			"float": 3.14
		}
	}`
		longStr := make([]byte, consts.DataTooLongUpper)
		for i := range longStr {
			longStr[i] = 'a'
		}
		jsonStr = fmt.Sprintf(jsonStr, string(longStr), string(longStr), string(longStr))

		result := TranslateJsonIO(jsonStr)
		fmt.Println(result)
	})
	t.Run("解析失败", func(t *testing.T) {
		// 示例 JSON
		jsonStr := `{
		"name": "example",
		"details": {
			"description": "This is a very long description that exceeds the length limit%v...",
			"tags": ["gogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogogo%v", "json", "example"],
			"nested": {
				"note": "This is a note",
				"longField": "a long string that is way too long%v"
			},
			"number": 12,
			"float": 3.14,
		}
	}`
		longStr := make([]byte, consts.DataTooLongUpper)
		for i := range longStr {
			longStr[i] = 'a'
		}
		jsonStr = fmt.Sprintf(jsonStr, string(longStr), string(longStr), string(longStr))

		result := TranslateJsonIO(jsonStr)
		assert.True(t, result == "数据解析异常")
	})
	t.Run("成功", func(t *testing.T) {
		// 示例 JSON
		// 读取文件内容
		fpath := "/Users/wh/Documents/DeepLang/empyrean_lens/longJson"
		longStr, err := os.ReadFile(fpath)
		assert.Nil(t, err)

		result := TranslateJsonIO(string(longStr))
		fmt.Println(result)
	})
}
