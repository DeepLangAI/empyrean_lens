package conf

import (
	"fmt"
	"os"
	"path/filepath"

	"empyrean_lens/fc/utils"

	conflib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/conf"
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/yaml.v3"
)

var conf Config

type Config struct {
	Logger conflib.Logger `yaml:"logger"`
	Api    Api            `yaml:"api"`
	MNS    MNS            `yaml:"mns"`
}

type Api struct {
	SaveLinkTrace        string `yaml:"save_link_trace"`
	BatchSaveLinkTrace   string `yaml:"batch_save_link_trace"`
	BatchUpdateLinkTrace string `yaml:"batch_update_link_trace"`
}

type FcTimer struct {
	Name string `yaml:"name"`
}

type MNS struct {
	SaveLinkTrace        FcTimer `yaml:"save_link_trace"`
	BatchSaveLinkTrace   FcTimer `yaml:"batch_save_link_trace"`
	BatchUpdateLinkTrace FcTimer `yaml:"batch_update_link_trace"`
}

// 配置文件路径
const ConfigPath = "./output/conf/config_%s.yaml"

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
	// 确保日志文件所在目录已创建
	ensureDirExists(filepath.Dir(conf.Logger.LogPath))
}

func ensureDirExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			panic(err)
		}
	}
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
