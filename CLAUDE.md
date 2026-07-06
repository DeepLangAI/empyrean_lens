# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

**天眼**（empyrean_lens）是一个内部监控/巡检/认证服务，承担三类职责：
1. **告警中转**：接收阿里云 SLS 告警 Webhook 回调，格式化为飞书卡片后转发到对应飞书群机器人。
2. **定时巡检报表**：每日通过 FC（函数计算）定时任务查询 SLS 日志，生成"数据平台每日巡检报表"飞书卡片，推送到运营群。
3. **统一认证中心（SSO）**：基于飞书 OAuth2 + JWT 的轻量 SSO，供公司内部各服务接入。

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

# 根据 IDL 更新生成代码（model + router，不覆盖已有 handler）
make server
# 注意：hz update 后 biz/handler/empyrean_lens/auth/auth_service.go 会被重置为错误 import
# 需手动修正为 import "empyrean_lens/service"，不要 import "empyrean_lens/auth"
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
| 内容抓取总览 | `crawl_normal OR crawl_error` + `extra like '%"domain"%'` | 避免 wrapper ERROR 重复计数 |
| 抓取耗时 P50/P90/P95/P99 | `crawl_timing_normal`，regexp_extract extra 里 `crawl_took` | 单位秒，136K+ 样本/天 |
| 公众号（自采代理）耗时 | 同上 + 全文搜索 `mp.weixin.qq.com` | 仅代理路径，不含第三方 |
| 微信第三方服务 | `kakalong weixin OR tikhub weixin OR renmin weixin` | count_if success/fail |
| 图片抓取 | `async_crawl_img end`，regexp_extract message 里 `cost` | P50/P90/P95/P99 |
| 图片最终失败 | `所有方式均失败` | 每张图片仅一条 |
| Top 失败站点 | `crawl_error` + regexp_extract extra 里 `domain` | TOP 5 |
| 数据生成服务 | `lingowhale-repeater-go-prod` 容器 OutRequest + ERROR | UNION ALL，按 service_type 分组 |

**重要注意事项：**
- 每次失败产生两条日志，必须用 `where extra like '%"domain"%'` 过滤 wrapper 条目
- `crawl_normal` 的 HTTP 200 不代表内容有效（微信反爬返回 200 + 拦截页）
- Kakalong 成功的微信抓取不进入 `crawl_normal`（提前 return，绕过装饰器）
- nginx-ingress `/crawl_img` 只是极少数同步路径，主路径是 `async_crawl_img` 消息队列

### 统一认证中心（SSO）

**核心机制**：飞书 OAuth2 Authorization Code + PKCE → 签发 JWT → httpOnly Cookie 或 redirect 带 token

```
service/auth.go              OAuth 逻辑 + AuthMiddleware() 单例 + Login/Me/Verify 方法
biz/router/.../auth/
  middleware.go              _authMw() 应用 JWT 验证
  auth.go                    路由注册（hz 生成，DO NOT EDIT）
biz/handler/.../auth/
  auth_service.go            stubs，全部委托 service（hz 重新生成后需手动修复 import）
```

**认证流程：**
1. `/auth/login?redirect=<url>` → 写 state/verifier/redirect cookie → 跳飞书授权页
2. `/auth/callback?code=&state=` → 校验 state → 换 access_token → 拉用户信息 → 签发 JWT
3. JWT 写入 httpOnly Cookie（同域服务自动携带）或带 token 跳回 redirect URL（跨域服务）

**三种接入方式：**

| 方式 | 适用场景 | 做法 |
|------|---------|------|
| Cookie 共享 | `*.lingowhale.com` 同父域 | 共享 JWTSecret，本地验证 cookie |
| Redirect + Token | 跨域服务 | 带 `?redirect=` 参数，接收 token 后存本地 cookie |
| `/api/auth/verify` | 后端/移动端 | `Authorization: Bearer <token>` 调验证接口 |

**已知陷阱：**
- `redirect` query param 会被 URL 编码，cookie 读出来后必须 `url.QueryUnescape` 再用
- `allowed_redirects` 白名单比较的是 `u.Host`（不含 scheme），填 `lingowhale.com` 可匹配所有子域
- OAuth 流程中 state cookie 必须与 callback 在**同一域名**下，不能混用 localhost 和 ngrok
- `httplib.Do` 默认 POST，GET 接口（如飞书 user_info）需加 `httplib.WithHttpMethod("GET")`
- `hz update` 后 handler stub 会被重置为错误 import `empyrean_lens/auth`，需手动改回 `empyrean_lens/service`

**配置项（conf/config_*.yaml feishu 段）：**
```yaml
feishu:
  app_id: ""
  app_secret: ""
  redirect_url: ""        # 飞书回调地址，需在开发者后台安全设置添加
  jwt_secret: ""          # 32+ 字符，各接入服务共享此值可本地验证
  cookie_domain: ""       # 生产填 .lingowhale.com，本地留空
  allowed_redirects:      # 跨域 redirect 白名单，填父域名可匹配所有子域
    - "lingowhale.com"
    - "deeplang.ai"
    - "shenyandayi.com"
```

**路由保护状态：**
- 公开：`/auth/login`、`/auth/callback`、`/ping`、`/api/v1/notice_webhook/sls_notice`、`assets/` 静态资源
- 需登录：`/api/auth/me`、`/api/auth/verify`、`/api/auth/logout`、`/`、其他 SPA 路由

### 环境配置

通过 `MODE_ENV` 环境变量选择配置文件，对应 `conf/config_<MODE_ENV>.yaml`。未设置时默认读取 `config_test.yaml`。

- dev：本地开发，SLS 用 QA 代理，飞书 redirect_url 填 ngrok 地址
- test：`qa-empyrean-lens.lingowhale.com`，SLS 用 QA 代理
- pre：`pre-empyrean-lens.lingowhale.com`，SLS 用生产代理
- prod：`empyrean-lens.lingowhale.com`，SLS 用生产代理

### Handler 规范

Handler 函数由 hz 生成骨架，手动实现。约定：
- **不使用 `c.BindAndValidate`**：当 body 是 JSON 数组时会失败，直接用 `c.Request.Body()` 读取原始字节。
- 所有响应均返回 HTTP 200，错误信息编码在 JSON body 的 `code`/`msg` 字段中。

### Service 规范

- Service 方法签名：`(ctx, req) → (*middleware.BaseResp, *consts.BizCode)`
- 业务错误用 `consts.BizCode`；HTTP 层面始终返回 200。
- 出站 HTTP 请求统一使用 `httplib.Do`（来自内部 go_lib），不使用标准库 `net/http`。
  - GET 接口需加 `httplib.WithHttpMethod("GET")`，否则默认 POST。
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
| JWT | `github.com/hertz-contrib/jwt`（基于 golang-jwt/v4） |
| 日志/配置/中间件 | `go_lib`（内部库，路径 `codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib`） |
| FC 运行时 | `github.com/aliyun/fc-runtime-go-sdk` |
