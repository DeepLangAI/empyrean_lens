package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var projPath = ""

func TimeSub(t time.Time) string {
	return fmt.Sprintf("%.4f s", time.Since(t).Seconds())
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
}

func GetQueueNameFromSubject(subject string) string {
	strList := strings.Split(subject, "/")
	if len(strList) == 2 {
		return strList[1]
	}
	return subject
}
