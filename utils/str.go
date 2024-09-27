package utils

import (
	"encoding/base64"
	"github.com/bytedance/sonic"
	"log"
	"strings"
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

func Contains(data []string, target string) bool {
	for _, item := range data {
		if item == target {
			return true
		}
	}
	return false

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
