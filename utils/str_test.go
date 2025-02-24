package utils

import (
	"empyrean_lens/consts"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/assert"
)

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

		result := TranslateJsonIO(jsonStr, true)
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

		result := TranslateJsonIO(jsonStr, true)
		assert.True(t, result == "数据解析异常")
	})
	t.Run("成功", func(t *testing.T) {
		// 示例 JSON
		// 读取文件内容
		fpath := "/Users/wh/Documents/DeepLang/empyrean_lens/longJson.json"
		longStr, err := os.ReadFile(fpath)

		assert.Nil(t, err)
		fmt.Println(len(longStr))

		t0 := time.Now()
		result := TranslateJsonIO(string(longStr), true)
		t1 := time.Since(t0)
		fmt.Println(t1)
		fmt.Println(result[:10])
	})
}

func TestUnMarshalJson(t *testing.T) {
	s := "{\\x22code\\x22:600101,\\x22msg\\x22:\\x22\\x22,\\x22status\\x22:\\x22error\\x22,\\x22entry_id\\x22:\\x22\\x22,\\x22edu_tree_nodes\\x22:null,\\x22to_segment\\x22:null}\t"
	output := UnMarshalJson(s)
	fmt.Println(output)
}

func TestParseQuery_ValidObjectID_ReturnsTrue(t *testing.T) {
	query := "507f1f77bcf86cd799439011"
	expected := false
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_InvalidObjectID_ReturnsFalse(t *testing.T) {
	query := "invalid-object-id"
	expected := true
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_ValidURL_ReturnsFalse(t *testing.T) {
	query := "http://example.com"
	expected := false
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_InvalidURL_ReturnsFalse(t *testing.T) {
	query := "ftp://example.com"
	expected := false
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_ValidNumber_ReturnsTrue(t *testing.T) {
	query := "12345abc"
	expected := false
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_ValidUid_ReturnsTrue(t *testing.T) {
	query := "8618120bd7324346a3ba6dae6feb78e6"
	expected := false
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_InvalidNumber_ReturnsFalse(t *testing.T) {
	query := "not-a-number"
	expected := false
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}

func TestParseQuery_validQuery_ReturnsFalse(t *testing.T) {
	query := "世界"
	expected := true
	actual, _ := IsValidQuery(query)
	if actual != expected {
		t.Errorf("ParseQuery(%s) = %v; want %v", query, actual, expected)
	}
}
