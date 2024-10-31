package bi

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"os"

	constslib "codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/consts"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	biCollection *mongo.Database
	mongoClient  *mongo.Client
)

func Init(ctx context.Context) {
	var (
		mode          = os.Getenv(constslib.ModeEnvName)
		clientOptions *options.ClientOptions
		cfg           = conf.GetConfig().MongoBi
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
	biCollection = client.Database(cfg.DatabaseName)
	mongoClient = client
}
