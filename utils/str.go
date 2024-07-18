package utils

import (
	"github.com/bytedance/sonic"
	"log"
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
