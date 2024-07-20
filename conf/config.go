package conf

import (
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"empyrean_lens/utils"
	"fmt"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var conf Config

type Config struct {
	Server          Server `yaml:"server"`
	Mongo           Mongo  `yaml:"mongo"`
	LogTemplatePath string `yaml:"logTemplatePath"`
}

type Mongo struct {
	Addr         string `yaml:"addr"`
	Port         string `yaml:"port"`
	DatabaseName string `yaml:"databaseName"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	MaxPoolSize  uint64 `yaml:"maxPoolSize"`
	Shadow       string `yaml:"shadow"`
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
	configPath = filepath.Join(utils.GetProjectPath(), configPath)
	hlog.Info("read config from ", configPath)
	dataBytes, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	if err = yaml.Unmarshal(dataBytes, &conf); err != nil {
		panic("conf.yaml配置文件读取失败:" + err.Error())
	}
	if err != nil {
		panic(err)
	}
	conf.LogTemplatePath = filepath.Join(utils.GetProjectPath(), "templates/rentention.html")
}

func TestInit() {
	// 绝对路径
	dirPath := utils.GetProjectPath()
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
