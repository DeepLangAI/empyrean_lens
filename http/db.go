package http

import (
	"context"
	"empyrean_lens/conf"
	"fmt"

	"codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib/httplib"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
)

// ── 响应类型 ──────────────────────────────────────────────────────────────────

type MongoFindResp struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data []map[string]interface{} `json:"data"`
}

type MongoAggregateResp struct {
	Code int                      `json:"code"`
	Msg  string                   `json:"msg"`
	Data []map[string]interface{} `json:"data"`
}

// ── 请求类型 ──────────────────────────────────────────────────────────────────

// MongoFindReq 对应 proto FindReq，filter/projection/sort 使用 map[string]any 匹配 protobuf Struct。
type MongoFindReq struct {
	DbName         string         `json:"db_name"`
	CollectionName string         `json:"collection_name"`
	Filter         map[string]any `json:"filter,omitempty"`
	Projection     map[string]any `json:"projection,omitempty"`
	Sort           map[string]any `json:"sort,omitempty"`
	Skip           *int32         `json:"skip,omitempty"`
	Limit          *int32         `json:"limit,omitempty"`
}

// MongoAggregateReq 对应 proto AggregateReq。
type MongoAggregateReq struct {
	DbName         string           `json:"db_name"`
	CollectionName string           `json:"collection_name"`
	Pipeline       []map[string]any `json:"pipeline"`
	Projection     map[string]any   `json:"projection,omitempty"`
	Sort           map[string]any   `json:"sort,omitempty"`
	Skip           *int32           `json:"skip,omitempty"`
	Limit          *int32           `json:"limit,omitempty"`
}

// ── 接口调用 ──────────────────────────────────────────────────────────────────

// MongoDbFind 调用 mcp-db POST /api/mongo/query/find 接口。
func MongoDbFind(ctx context.Context, req *MongoFindReq) (*MongoFindResp, error) {
	bodyBytes, _ := sonic.Marshal(req)
	resp := &MongoFindResp{}
	secret := conf.GetConfig().ExternalSecret.DeeplangDbFcSecret
	_, err := httplib.Do(ctx,
		fmt.Sprintf("%s/api/mongo/query/find", secret.BaseUrl),
		map[string]string{
			consts.HeaderContentType:   consts.MIMEApplicationJSON,
			consts.HeaderAuthorization: "Bearer " + secret.Token,
		},
		bodyBytes, resp,
	)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("mongo find error, code: %d, msg: %s", resp.Code, resp.Msg)
	}
	return resp, nil
}

// MongoDbAggregate 调用 mcp-db POST /api/mongo/query/aggregate 接口。
func MongoDbAggregate(ctx context.Context, req *MongoAggregateReq) (*MongoAggregateResp, error) {
	bodyBytes, _ := sonic.Marshal(req)
	resp := &MongoAggregateResp{}
	secret := conf.GetConfig().ExternalSecret.DeeplangDbFcSecret
	_, err := httplib.Do(ctx,
		fmt.Sprintf("%s/api/mongo/query/aggregate", secret.BaseUrl),
		map[string]string{
			consts.HeaderContentType:   consts.MIMEApplicationJSON,
			consts.HeaderAuthorization: "Bearer " + secret.Token,
		},
		bodyBytes, resp,
	)
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("mongo aggregate error, code: %d, msg: %s", resp.Code, resp.Msg)
	}
	return resp, nil
}
