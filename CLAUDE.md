# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

**天眼**（empyrean_lens）是一个内部监控/巡检服务，承担两类职责：
1. **告警中转**：接收阿里云 SLS 告警 Webhook 回调，格式化为飞书卡片后转发到对应飞书群机器人。
2. **定时巡检报表**：每日通过 FC（函数计算）定时任务查询 SLS 日志，生成"数据平台每日巡检报表"飞书卡片，推送到运营群。

## Commands

```bash
# 本地开发启动（端口 :19001）
MODE_ENV=dev go run .

# 运行测试
go test ./...

# 运行单个测试（告警 Webhook）
go test ./service/ -run TestSlsNoticeWebhook -v

# 运行定时巡检报表测试（实际发送飞书卡片）
MODE_ENV=dev go test ./fc/cron/handlers/ -run TestLingowhaleStability_Handle -v -timeout 120s

# 编译检查
go build ./...

# 根据 IDL 更新生成代码（model + router，不覆盖 handler）
make server
```

## 架构

### 告警中转链路

```
IDL (idl/*.thrift)
  └─ hz update ──► biz/model/         生成的 thrift 结构体（DO NOT EDIT）
                   biz/router/        生成的路由注册（DO NOT EDIT）
                   biz/handler/       生成脚手架，后续手动维护
                        └─ service/   业务逻辑（手动编写）
```

`hz update` 会重新生成 `biz/model/` 和 `biz/router/`，**不会覆盖** `biz/handler/` 下已存在的文件。

### 定时巡检报表链路

```
FC 定时触发（每日 08:00）
  └─ fc/cron/main.go          入口，路由到对应 handler
       └─ handlers/lingowhale-stability.go
            ├─ 并发 SLS 查询（http/sls.go → 内部 SLS 代理 FC）
            └─ 飞书卡片构建 + Webhook 发送（retry-go 重试 3 次）
```

**巡检报表数据来源（均为 business-pod logstore，前一日 00:00–24:00）：**

| 报表节 | SLS 查询 | 关键字段 |
|--------|----------|----------|
| 内容抓取总览 | `crawl_normal OR crawl_error` + `extra like '%"domain"%'` | message=crawl_normal/crawl_error，避免 wrapper ERROR 重复计数 |
| 抓取耗时 P50/P90/P95/P99 | `crawl_timing_normal`，regexp_extract extra 里 `crawl_took` | 单位秒，136K+ 样本/天 |
| 公众号（自采代理）耗时 | 同上 + 全文搜索 `mp.weixin.qq.com` | 仅代理路径，不含 Kakalong 等第三方 |
| 微信第三方服务 | `kakalong weixin OR tikhub weixin OR renmin weixin` | count_if success/fail，success rate 直接在 SQL 算 |
| 图片抓取 | `async_crawl_img end`，regexp_extract message 里 `cost` | 全量（356K+/天），P50/P90/P95/P99 |
| 图片最终失败 | `所有方式均失败` | 每张图片仅一条，success = total - failures |
| Top 失败站点 | `crawl_error` + regexp_extract extra 里 `domain` | 按失败量降序 TOP 5 |
| 数据生成服务 | `lingowhale-repeater-go-prod` 容器 OutRequest + ERROR | UNION ALL 统计调用量/失败量/成功率，按 service_type 分组 |

**重要注意事项：**
- `crawl_normal`/`crawl_error` 每次失败会产生两条日志（wrapper ERROR + log_for_analysis INFO），必须用 `where extra like '%"domain"%'` 过滤掉 wrapper 条目
- `crawl_normal` 的 `status=CrawlNormal` 只代表 HTTP 200，不代表内容有效（微信反爬会返回 200 + 拦截页）
- Kakalong 成功的微信抓取**不会**进入 `crawl_normal`（在 `_crawl` 里提前 return，绕过了 `_retry_async` 装饰器）
- nginx-ingress 的 `/crawl_img`（833条/天）是极少数同步路径，**不代表**主体图片抓取量；主路径是 `async_crawl_img` 消息队列

### 环境配置

通过 `MODE_ENV` 环境变量选择配置文件，对应 `conf/config_<MODE_ENV>.yaml`。未设置时默认读取 `config_test.yaml`。

各环境 Webhook 地址：
- dev/test：`config_dev.yaml` → 测试群
- pre/prod：`config_pre.yaml` / `config_prod.yaml` → 正式运营群

SLS 查询通过内部 FC 代理（`external_secret.deeplang_sls_fc_secret`），dev/test 用 QA 环境代理，pre/prod 用生产环境代理。

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

### 飞书卡片规范

- schema `2.0`，`wide_screen_mode: true`
- 表格列 `name` 必须是 ASCII 标识符（不能用中文或 `#`），`display_name` 才是显示文本
- 列宽只用 `"auto"`，不支持 `"50px"` 等 CSS 写法
- 行数据 map 的 key 必须与列 `name` 完全一致

## 关键依赖

| 用途 | 包 |
|------|----|
| HTTP 框架 | `github.com/cloudwego/hertz` |
| 内部 HTTP 客户端 | `go_lib/httplib`（封装了 hertz client、TLS、重试中间件） |
| JSON | `github.com/bytedance/sonic` |
| JSON 路径查询（保序） | `github.com/tidwall/gjson` |
| 重试 | `github.com/avast/retry-go` |
| 日志/配置/中间件 | `go_lib`（内部库，路径 `codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib`） |
| FC 运行时 | `github.com/aliyun/fc-runtime-go-sdk` |
