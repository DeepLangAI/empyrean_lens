package conf

import (
	"empyrean_lens/utils"
	"fmt"
	"os"
	"path/filepath"

	conflib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/conf"
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"github.com/redis/go-redis/v9"
	"gopkg.in/yaml.v3"
)

var conf Config

type Config struct {
	Server            Server                `yaml:"server"`
	MongoEmpyreanlens Mongo                 `yaml:"mongo_empyreanlens"`
	MongoLingo        Mongo                 `yaml:"mongo_lingo"`
	MongoPlugin       Mongo                 `yaml:"mongo_plugin"`
	MongoCollection   Mongo                 `yaml:"mongo_collection"`
	MongoBi           Mongo                 `yaml:"mongo_bi"`
	Redis             *redis.ClusterOptions `yaml:"redis"`
	Logger            conflib.Logger        `yaml:"logger"`
	Lark              Lark                  `yaml:"lark"`
	Oss               OSS                   `yaml:"oss"`
	ShenCe            ShenCe                `yaml:"shence"`
	MnsConfig         MnsConfig             `yaml:"mns_config"`
}

type OSS struct {
	Endpoint     string `yaml:"endpoint"`
	AccessKey    string `yaml:"accessKey"`
	AccessSecret string `yaml:"accessSecret"`
}

type Redis struct {
	Addrs    []string `yaml:"addrs"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"`
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

type Lark struct {
	AppId     string   `yaml:"appId"`
	AppSecret string   `yaml:"appSecret"`
	Auth      LarkAuth `yaml:"auth"`
	JwtSecret string   `yaml:"jwtSecret"`
}

type LarkAuth struct {
	Names       []string `yaml:"names"`
	Emails      []string `yaml:"emails"`
	Mobiles     []string `yaml:"mobiles"`
	EmployeeNos []string `yaml:"employee_nos"`
}

type ShenCe struct {
	Token   string `yaml:"api_key"`
	Project string `yaml:"project"`
	Url     string `yaml:"url"`
}

type MnsConfig struct {
	AccessKeyID     string `yaml:"access_key_id"`
	AccessKeySecret string `yaml:"access_key_secret"`
	Endpoint        string `yaml:"endpoint"`
	QueueName       string `yaml:"queue_name"`
}

// 配置文件路径
const ConfigPath = "./conf/config_%s.yaml"

func GetConfig() Config {
	return conf
}

func GetLark() Lark {
	return conf.Lark
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
