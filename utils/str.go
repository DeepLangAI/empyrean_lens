package utils

import (
	"empyrean_lens/consts"
	"encoding/base64"
	"log"
	"strings"

	"github.com/cloudwego/hertz/pkg/common/hlog"

	"github.com/bytedance/sonic"
)

func JSONMarshal(v interface{}) string {
	byt, err := sonic.Marshal(v)
	if err != nil {
		log.Println(err)
		return ""
	}
	return string(byt)
}

func JSONUnMarshal(data []byte, v interface{}) interface{} {
	err := sonic.Unmarshal(data, v)
	if err != nil {
		println(err.Error())
		return nil
	}
	return v
}

func Contains[S ~[]E, E comparable](s S, v E) bool {
	return Index(s, v) != -1
}

func DecodeMIME(encodedStr string) string {
	// 解析MIME编码的字符串
	//_, _, err := mime.ParseMediaType(encodedStr)
	//if err != nil {
	//	return encodedStr
	//}

	// 去掉前面的头信息
	parts := strings.Split(encodedStr, "?")
	if len(parts) != 5 {
		return encodedStr
	}

	if parts[1] != "utf-8" || parts[2] != "b" {
		return encodedStr
	}

	// 解码base64内容
	data, err := base64.StdEncoding.DecodeString(parts[3])
	if err != nil {
		return encodedStr
	}

	return string(data)
}

func TranslateJsonIO(data string, needProcess bool) string {
	var jsonData any
	if err := sonic.Unmarshal([]byte(data), &jsonData); err != nil {
		hlog.Errorf("Error parsing JSON: %v", err)
		if len(data) > consts.DataTooLongUpper {
			return consts.DataTooLongUpperErrMsg
		}
		return data
	}

	// 处理 JSON
	var processedData any
	if needProcess {
		processedData = processJSON(jsonData)
	} else {
		processedData = jsonData
	}

	// 转换回 JSON 字符串
	result, err := sonic.Marshal(processedData)
	if err != nil {
		hlog.Errorf("Error converting JSON back to string: %v", err)
		if len(data) > consts.DataTooLongUpper {
			return consts.DataTooLongUpperErrMsg
		}
		return data
	}
	return string(result)
}

// 递归处理 JSON 数据
func processJSON(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, value := range v {
			v[key] = processJSON(value)
		}
	case []interface{}:
		for i, value := range v {
			v[i] = processJSON(value)
		}
	case string:
		if len(v) > consts.DataTooLongUpper {
			return consts.DataTooLongUpperErrMsg
		}
	}
	return data
}
