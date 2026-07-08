package http

import (
	"context"
	"empyrean_lens/conf"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// wechat-spider 库直连客户端（只读）。
// 该库不在 mcp-db 代理的暴露列表里，巡检报表需要它的 article/target_account 表
// （采集时效、覆盖账号数），故直连。连接串配置在 conf 的 wechat_spider_mongo.url。

var (
	spiderClient     *mongo.Client
	spiderClientOnce sync.Once
	spiderClientErr  error
)

func getSpiderDB(ctx context.Context) (*mongo.Database, error) {
	spiderClientOnce.Do(func() {
		url := conf.GetConfig().WechatSpiderMongo.URL
		if url == "" {
			spiderClientErr = fmt.Errorf("wechat_spider_mongo.url 未配置")
			return
		}
		opts := options.Client().
			ApplyURI(url).
			SetReadPreference(readpref.SecondaryPreferred()). // 只读负载走从库
			SetMaxPoolSize(5).
			SetConnectTimeout(10 * time.Second)
		cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		spiderClient, spiderClientErr = mongo.Connect(cctx, opts)
		if spiderClientErr == nil {
			spiderClientErr = spiderClient.Ping(cctx, readpref.SecondaryPreferred())
		}
		if spiderClientErr != nil {
			hlog.CtxErrorf(ctx, "connect wechat-spider mongo failed: %v", spiderClientErr)
		}
	})
	if spiderClientErr != nil {
		return nil, spiderClientErr
	}
	return spiderClient.Database("wechat-spider"), nil
}

// SpiderAggregate 在 wechat-spider 库上执行只读聚合，结果拍平为 []map[string]string
// （与 SLS 查询结果同构，供 report 引擎统一消费）。
// pipelineJSON 为聚合管道的扩展 JSON 文本（支持 {"$date":"..."}）。
func SpiderAggregate(ctx context.Context, collection, pipelineJSON string) ([]map[string]string, error) {
	db, err := getSpiderDB(ctx)
	if err != nil {
		return nil, err
	}
	var pipeline []bson.M
	if err := bson.UnmarshalExtJSON([]byte(pipelineJSON), false, &pipeline); err != nil {
		return nil, fmt.Errorf("bad pipeline json: %w", err)
	}

	qctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cur, err := db.Collection(collection).Aggregate(qctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(qctx)

	var out []map[string]string
	for cur.Next(qctx) {
		var doc bson.M
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		out = append(out, flattenBson(doc))
	}
	return out, cur.Err()
}

// flattenBson 把 bson 文档的顶层字段转为字符串（嵌套值序列化为其字面量表示）。
func flattenBson(doc bson.M) map[string]string {
	row := make(map[string]string, len(doc))
	for k, v := range doc {
		row[k] = bsonValToString(v)
	}
	return row
}

func bsonValToString(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case int:
		return strconv.Itoa(t)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	case time.Time:
		return t.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}
