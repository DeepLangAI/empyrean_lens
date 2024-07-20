package utils

import (
	"empyrean_lens/consts"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

var projPath = ""

func KeysOfMap[T comparable, V any](dict map[T]V) []T {
	keys := make([]T, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}

func ValuesOfMap[T comparable, V any](dict map[T]V) []V {
	values := make([]V, 0, len(dict))
	for _, v := range dict {
		values = append(values, v)
	}
	return values
}

func GetApiAlias(hostName, apiName string) string {
	if apiName == "" {
		return apiName
	}
	for host, apis := range consts.NGINX_INGRESS_APIS {
		if hostName != host {
			continue
		}
		for _, api := range apis {
			if api.Api == apiName {
				return api.Alias
			}
		}
	}
	for host, apis := range consts.MODEL_NGINX_INGRESS_APIS {
		if hostName != host {
			continue
		}
		for _, api := range apis {
			if api.Api == apiName {
				return api.Alias
			}
		}
	}
	return apiName
}

func TimeSub(t time.Time) string {
	return fmt.Sprintf("%.4f s", time.Now().Sub(t).Seconds())
}
func GetProjectPath() string {
	if projPath != "" {
		return projPath
	}
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// 从当前工作目录向上遍历，寻找main.go文件
	for {
		info, err := os.Stat(filepath.Join(cwd, "main.go"))
		if err == nil && !info.IsDir() {
			// 找到main.go，返回当前目录作为项目路径
			return cwd
		} else if !os.IsNotExist(err) {
			// 其他错误
			return ""
		}

		// 如果没找到，尝试进入上一级目录
		cwd = filepath.Dir(cwd)
		if cwd == "/" || cwd == "" {
			// 如果到达根目录仍然没找到，返回错误
			return ""
		}
	}

	// 不应达到这里，但为了编译器的满意度，返回一个空字符串
	return ""
}
