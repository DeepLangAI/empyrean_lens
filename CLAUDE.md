# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

**天眼**（empyrean_lens）是一个内部监控告警中转服务，负责接收来自阿里云 SLS（日志服务）的告警 Webhook 回调，将告警数据格式化为飞书卡片消息后转发给对应的飞书群机器人。

## Commands

```bash
# 本地开发启动（端口 :19001）
MODE_ENV=dev go run .

# 运行测试
go test ./...

# 运行单个测试
go test ./service/ -run TestSlsNoticeWebhook -v

# 编译检查
go build ./...

# 根据 IDL 更新生成代码（model + router，不覆盖 handler）
make server
```

## 架构

### 请求链路

```
IDL (idl/*.thrift)
  └─ hz update ──► biz/model/         生成的 thrift 结构体（DO NOT EDIT）
                   biz/router/        生成的路由注册（DO NOT EDIT）
                   biz/handler/       生成脚手架，后续手动维护
                        └─ service/   业务逻辑（手动编写）
```

`hz update` 会重新生成 `biz/model/` 和 `biz/router/`，**不会覆盖** `biz/handler/` 下已存在的文件。

### 环境配置

通过 `MODE_ENV` 环境变量选择配置文件，对应 `conf/config_<MODE_ENV>.yaml`。未设置时默认读取 `config_test.yaml`。

### Handler 规范

Handler 函数由 hz 生成骨架，手动实现。约定：
- **不使用 `c.BindAndValidate`**：当 body 是 JSON 数组时会失败，直接用 `c.Request.Body()` 读取原始字节。
- 所有响应均返回 HTTP 200，错误信息编码在 JSON body 的 `code`/`msg` 字段中。

### Service 规范

- Service 方法签名：`(ctx, req) → (*middleware.BaseResp, *consts.BizCode)`
- 业务错误用 `consts.BizCode`；HTTP 层面始终返回 200。
- 出站 HTTP 请求统一使用 `httplib.Do`（来自内部 go_lib），不使用标准库 `net/http`。
- JSON 序列化统一用 `github.com/bytedance/sonic`，不用 `encoding/json`。
- 需要保留 JSON key 原始顺序时，用 `gjson.ForEach` 提取 key 列表，再用 `map[string]string` 存数据。

## 关键依赖

| 用途 | 包 |
|------|----|
| HTTP 框架 | `github.com/cloudwego/hertz` |
| 内部 HTTP 客户端 | `go_lib/httplib`（封装了 hertz client、TLS、重试中间件） |
| JSON | `github.com/bytedance/sonic` |
| JSON 路径查询（保序） | `github.com/tidwall/gjson` |
| 重试 | `github.com/avast/retry-go` |
| 日志/配置/中间件 | `go_lib`（内部库，路径 `codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib`） |
