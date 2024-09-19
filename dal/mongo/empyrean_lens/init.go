package empyrean_lens

import (
	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/consts"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
)

var (
	probeDatabase *mongo.Database
	mongoClient   *mongo.Client
)

func initTableNames() {
	prefix := consts.TablePrefix
	TableNameApiCost = prefix + "api_cost"
	TableNameApiFailure = prefix + "api_failure"
	TableNameApiProbeLog = prefix + "api_probe"
	TableNameScene = prefix + "scene"
	TableNameSystemScore = prefix + "system_score"
	//TableNameTraceDebugger = prefix + "trace_debugger"
	TableNameTracebackLog = prefix + "traceback_log"
	TableNameOnlineOperation = prefix + "online_operation"
	TableNameScoreBackup = prefix + "score_backup"
}

func Init(ctx context.Context) {
	var (
		mode          = os.Getenv(constslib.ModeEnvName)
		clientOptions *options.ClientOptions
		cfg           = conf.GetConfig().MongoEmpyreanlens
	)
	if mode == constslib.ModeEnvPre || mode == constslib.ModeEnvProd || cfg.Port == "" {
		clientOptions = options.Client().ApplyURI(cfg.Addr)
		clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
	} else {
		url := fmt.Sprintf("mongodb://%s:%s", cfg.Addr, cfg.Port)
		clientOptions = options.Client().ApplyURI(url).SetDirect(true)
		clientOptions.SetMaxPoolSize(cfg.MaxPoolSize)
		clientOptions.SetAuth(options.Credential{
			AuthSource:  cfg.DatabaseName,
			Username:    cfg.Username,
			Password:    cfg.Password,
			PasswordSet: false,
		})
	}

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		panic(fmt.Sprintf("initialize mongodb Connect failed, err: %v", err))
	}

	err = client.Ping(context.TODO(), nil)
	if err != nil {
		panic(fmt.Sprintf("initialize mongodb Ping failed, err: %v", err))
	}
	probeDatabase = client.Database(cfg.DatabaseName)
	mongoClient = client

	initTableNames()
}
