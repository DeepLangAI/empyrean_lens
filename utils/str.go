package utils

import (
	"crypto/md5"
	"empyrean_lens/consts"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"io"
	"log"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

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

func UnMarshalJson(asciiJsonStr string) string {
	// 解码JSON字符串，将ASCII码转换为实际字符，如果失败则返回原字符串
	decodedMsg, err := strconv.Unquote(fmt.Sprintf("\"%v\"", asciiJsonStr))
	if err != nil {
		return asciiJsonStr
	}
	if json.Valid([]byte(decodedMsg)) {
		var result map[string]interface{}
		err = json.Unmarshal([]byte(decodedMsg), &result)
		if err != nil {
			return asciiJsonStr
		}
		decodedBizMsg, err := json.Marshal(result)
		if err != nil {
			return asciiJsonStr
		}
		return string(decodedBizMsg)
	}
	return asciiJsonStr
}

func DecodeMIME(encodedStr string) string {
	// 解析MIME编码的字符串
	//_, _, err := mime.ParseMediaType(encodedStr)
	//if err != nil {
	//	return encodedStr
	//}
	encodedStr = UnMarshalJson(encodedStr)

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

func IsValidQuery(query string) (bool, error) {
	// 判断 query是否为数据库id
	if IsValidObjectID(query) {
		return false, nil
	}
	// 判断 query是否为 url
	if IsValidUrl(query) {
		return false, nil
	}
	// 判断 query是否uid
	if IsAlphanumeric(query) && CharLength(query) >= 24 {
		return false, nil
	}

	return true, nil
}

func IsAlphanumeric(s string) bool {
	// 创建一个正则表达式，匹配仅包含字母和数字的字符串
	re := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return re.MatchString(s)
}

func IsValidUrl(query string) bool {
	if strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://") {
		return true
	}
	return false
}

func CharLength(s string) int {
	return utf8.RuneCountInString(s)
}

func ByteLength(s string) int {
	return len(s)
}

func StrToMd5(str string) string {
	hasher := md5.New()
	io.WriteString(hasher, str) // 写入字符串到MD5哈希器

	// 获取哈希值
	hashBytes := hasher.Sum(nil)
	hashStr := fmt.Sprintf("%x", hashBytes) // 转换为16进制字符串
	return hashStr
}
