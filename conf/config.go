package conf

import (
	"fmt"
	"os"
	"path/filepath"

	conflib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/conf"
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/yaml.v3"
)

var conf Config

type Config struct {
	Server         Server          `yaml:"server"`
	Logger         conflib.Logger  `yaml:"logger"`
	Metrics        conflib.Metrics `yaml:"metrics"`
	ExternalSecret ExternalSecret  `yaml:"external_secret"`
	Notice         Notice          `yaml:"notice"`
	Feishu         Feishu          `yaml:"feishu"`
	Redis          Redis           `yaml:"redis"`
}

type Redis struct {
	Addrs    []string `yaml:"addrs"`
	Username string   `yaml:"username" env:"REDIS_USERNAME"`
	Password string   `yaml:"password" env:"REDIS_PASSWORD" env-required:"true"`
	UseTls   bool     `yaml:"use_tls"`
}

type Feishu struct {
	AppID        string        `yaml:"app_id"`
	AppSecret    string        `yaml:"app_secret"`
	RedirectURL  string        `yaml:"redirect_url"`
	JWTSecret    string        `yaml:"jwt_secret"`
	CookieDomain string        `yaml:"cookie_domain"`
	Issuer       string        `yaml:"issuer"`        // OIDC issuer，留空则从请求 Host 推断
	OAuthClients []OAuthClient `yaml:"oauth_clients"` // 注册的 OAuth2 客户端
}

// OAuthClient 是一个注册的 OAuth2 接入方。
// 客户端信息静态配置在 YAML 中，变更后重启生效。
type OAuthClient struct {
	ClientID     string   `yaml:"client_id"`
	ClientSecret string   `yaml:"client_secret"`
	Name         string   `yaml:"name"`
	RedirectURIs []string `yaml:"redirect_uris"` // 精确匹配，不支持通配符
}

type Notice struct {
	LingowhaleStabilityWebhook string `yaml:"lingowhale_stability_webhook"`
	// 每日巡检报表独立 webhook；留空时回退到 stability 群
	LingowhaleDailyReportWebhook string `yaml:"lingowhale_daily_report_webhook"`
}

type Server struct {
	Port string `yaml:"port"` // 服务端口
	Name string `yaml:"name"`
}

type ExternalSecret struct {
	DeeplangSlsFcSecret DeeplangSlsFcSecret `yaml:"deeplang_sls_fc_secret"`
	DeeplangDbFcSecret  DeeplangDbFcSecret  `yaml:"deeplang_db_fc_secret"`
}

type DeeplangSlsFcSecret struct {
	BaseUrl string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

type DeeplangDbFcSecret struct {
	BaseUrl string `yaml:"base_url"`
	Token   string `yaml:"token"`
}

// 配置文件路径
const ConfigPath = "./conf/config_%s.yaml"

func GetConfig() Config {
	return conf
}

func InitConfig() {
	env := os.Getenv(constslib.ModeEnvName)
	if env == "" {
		env = "dev"
	}

	configPath := fmt.Sprintf(ConfigPath, env)
	configPath = filepath.Join(GetProjectPath(), configPath)
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
	hlog.Info("read config done")
}

var projPath = ""

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
		info, err := os.Stat(filepath.Join(cwd, "go.mod"))
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
