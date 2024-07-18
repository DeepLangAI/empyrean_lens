package conf

import (
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var conf Config

type Config struct {
	Server          Server `yaml:"server"`
	LogTemplatePath string `yaml:"logTemplatePath"`
}

type Server struct {
	Port string `yaml:"port"` // 服务端口
	Name string `yaml:"name"`
}

// 配置文件路径
const ConfigPath = "./conf/config_%s.yaml"

func GetConfig() Config {
	return conf
}

func InitConfig() {
	env := os.Getenv(constslib.ModeEnvName)
	if env == "" {
		env = "test"
	}

	configPath := fmt.Sprintf(ConfigPath, env)
	hlog.Info("read config from ", configPath)
	dataBytes, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	if err = yaml.Unmarshal(dataBytes, &conf); err != nil {
		panic("conf.yaml配置文件读取失败:" + err.Error())
	}
	pd, err := getProjectPath()
	if err != nil {
		panic(err)
	}
	conf.LogTemplatePath = filepath.Join(pd, "templates/rentention.html")
}

func TestInit() {
	// 绝对路径
	dirPath, _ := getProjectPath()
	filePath := fmt.Sprintf("./conf/config_test.yaml")
	configPath := filepath.Join(dirPath, filePath)
	hlog.Info("read config from ", configPath)
	dataBytes, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	if err = yaml.Unmarshal(dataBytes, &conf); err != nil {
		panic("conf.yaml配置文件读取失败:" + err.Error())
	}
}

func getProjectPath() (string, error) {
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// 从当前工作目录向上遍历，寻找main.go文件
	for {
		info, err := os.Stat(filepath.Join(cwd, "main.go"))
		if err == nil && !info.IsDir() {
			// 找到main.go，返回当前目录作为项目路径
			return cwd, nil
		} else if !os.IsNotExist(err) {
			// 其他错误
			return "", err
		}

		// 如果没找到，尝试进入上一级目录
		cwd = filepath.Dir(cwd)
		if cwd == "/" || cwd == "" {
			// 如果到达根目录仍然没找到，返回错误
			return "", fmt.Errorf("无法找到项目根目录")
		}
	}

	// 不应达到这里，但为了编译器的满意度，返回一个空字符串
	return "", nil
}
