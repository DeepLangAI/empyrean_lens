# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目简介

**天眼**（empyrean_lens）是一个内部监控/巡检/认证服务，承担三类职责：
1. **告警中转**：接收阿里云 SLS 告警 Webhook 回调，格式化为飞书卡片后转发到对应飞书群机器人。
2. **定时巡检报表**：每日通过 FC（函数计算）定时任务查询 SLS 日志，生成"数据平台每日巡检报表"飞书卡片，推送到运营群。
3. **统一认证中心（SSO）**：基于飞书 OAuth2 + 标准 OIDC 接口的认证中心，供公司内部各服务接入。

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

### 统一认证中心（SSO / OIDC）

**核心机制**：飞书 OAuth2 Authorization Code + PKCE → 签发 JWT → 标准 OIDC 接口对外暴露

#### 文件结构

```
idl/auth.thrift              AuthService + OidcService 接口定义
service/auth.go              AuthService：飞书 OAuth 流程、JWT middleware 单例、Login/Me
service/oidc.go              OidcService：Configuration/UserInfo/Introspect/Token
biz/handler/.../auth/
  auth_service.go            AuthService handler stubs（hz 生成后需手动修复 import）
  oidc_service.go            OidcService handler stubs（hz 生成后需手动修复 import）
biz/router/.../auth/
  middleware.go              各路由的 JWT 保护配置（手动维护）
  auth.go                    路由注册（hz 生成，DO NOT EDIT）
dal/redis/                   Redis 客户端，用于 authorization code 临时存储（TTL 60s）
```

#### 标准 OIDC 登录流程（Authorization Code + PKCE）

```
接入方                       认证中心                        飞书
  │                              │                              │
  ├─ GET /auth/login             │                              │
  │   ?client_id=xxx             │                              │
  │   &redirect_uri=...          │ 校验 client，写临时 cookie   │
  │   &state=xyz                 │ ──────────────────────────► │
  │                              │    飞书授权页                 │
  │                              │ ◄─────────────────────────── │
  │                 302 /auth/callback?code=xxx&state=xyz        │
  │                              │                              │
  │                              │ 换 access_token + 用户信息   │
  │                              │ 签发 JWT，生成 authorization code（Redis TTL 60s）
  │ ◄─── 302 redirect_uri?code=abc&state=xyz ──────────────────│
  │                              │                              │
  ├─ POST /token                 │                              │
  │   client_id + client_secret + code                          │
  │                              │ 验证 client，消费 code（一次性）
  │ ◄─── { access_token, ... } ──│                              │
  │                              │                              │
  ├─ GET /userinfo               │                              │
  │   Authorization: Bearer ...  │                              │
  │ ◄─── { sub, name, picture } ─│                              │
```

#### 标准 OIDC 端点

| 端点 | 方法 | 保护 | 说明 |
|------|------|------|------|
| `/.well-known/openid-configuration` | GET | 公开 | Discovery 文档（RFC 8414） |
| `/auth/login` | GET | 公开 | 授权入口，需 client_id + redirect_uri |
| `/auth/callback` | GET | 公开 | 飞书回调，内部使用 |
| `/token` | POST | 公开（client_secret 验证） | code 换 token（RFC 6749） |
| `/userinfo` | GET | Bearer token | 标准用户信息（OIDC Core §5.3） |
| `/introspect` | POST | 公开（token 在 body） | token 验签（RFC 7662） |
| `/api/auth/me` | GET | Cookie | 内部用，返回业务格式 |
| `/api/auth/logout` | POST | Cookie | 清除 cookie |

#### 客户端注册

接入方信息静态配置在 `conf/config_*.yaml` 中，变更后重启生效：

```yaml
feishu:
  oauth_clients:
    - client_id: "service-b"
      client_secret: "secret-xxx"
      name: "Service B"
      redirect_uris:
        - "https://service-b.lingowhale.com/auth/callback"
```

#### 已知陷阱

- `hz update` 后两个 handler stub 文件都会被重置为错误 import `empyrean_lens/auth`，需手动改回：
  - `auth_service.go`：import `empyrean_lens/service`，删掉 import `empyrean_lens/auth`
  - `oidc_service.go`：import `empyrean_lens/service` + `empyrean_lens/biz/model/empyrean_lens/auth`
- handler 用 `BindAndValidate` 绑定参数后传给 service，service 直接用 req 字段，不再自己读 `c.FormValue`
- service 响应必须用生成的 model struct（`TokenResp`、`IntrospectResp`、`MeResp`、`BaseResp`），不要手写 `map[string]interface{}`
- `c.Get(IdentityKey)` 读取当前用户（`IdentityKey = "user"`），不要硬编码字符串 `"user"`
- OAuth2 错误码用 `service/auth.go` 和 `service/oidc.go` 里定义的常量（`errInvalidClient`、`errInvalidGrant` 等），避免拼写错误导致客户端无法解析
- `tokenTTL = 7 * 24 * time.Hour` 是 JWT 有效期，在 auth middleware 和 TokenResp 里统一用此常量
- `httplib.Do` 默认 POST，飞书 user_info 接口需加 `httplib.WithHttpMethod("GET")`
- OAuth 流程的临时 cookie（state/verifier/client_id/redirect_uri）必须与 callback 在同一域名下
- Authorization code 存 Redis，key 格式 `sso:code:{code}`，TTL 60s，`consumeCode` 读取后立即删除（一次性）

#### 配置项（conf/config_*.yaml feishu 段）

```yaml
feishu:
  app_id: ""           # 飞书 App ID
  app_secret: ""       # 飞书 App Secret
  redirect_url: ""     # 飞书回调地址（需在开发者后台安全设置添加）
  jwt_secret: ""       # JWT 签名密钥，32+ 字符
  cookie_domain: ""    # 生产填 .lingowhale.com，本地留空
  issuer: ""           # OIDC issuer URL，留空则从请求 Host 推断
  oauth_clients: []    # 注册的 OAuth2 客户端列表
```

#### 路由保护状态

- **公开**：`/auth/login`、`/auth/callback`、`/token`、`/introspect`、`/.well-known/openid-configuration`、`/ping`、`/api/v1/notice_webhook/sls_notice`、`assets/` 静态资源
- **Bearer token**：`/userinfo`（MiddlewareFunc 保护）
- **Cookie**：`/api/auth/me`、`/api/auth/logout`、`/`、其他 SPA 路由

#### 待优化项（已知问题，暂未实现）

| 优先级 | 问题 | 说明 |
|--------|------|------|
| 高 | **SecureCookie 未开启** | `SecureCookie: false`，生产 HTTPS 环境应改为 `true`，否则 cookie 可能通过 HTTP 泄露 |
| 高 | **Token 无法吊销** | JWT 签发后 7 天内始终有效，员工离职/账号异常无法立即踢出。需 Redis 黑名单：logout 时写入，middleware 验证时检查 |
| 中 | **只有认证（AuthN），没有授权（AuthZ）** | 解决了"你是谁"，未解决"你能做什么"。所有登录用户权限相同，缺少角色/权限体系（RBAC） |
| 中 | **没有 Refresh Token** | 7 天过期后必须重新登录。标准做法：短期 access_token（1-2h）+ 长期 refresh_token（30d）无感续期 |
| 中 | **关键接口无限流** | `/auth/login`、`/token`、`/introspect` 无速率限制，可被暴力请求 |
| 中 | **client_secret 明文存 YAML** | 生产应接入 Secret 管理（KMS/Vault）或 bcrypt hash 存储 |
| 低 | **没有 JWKS 端点（非对称签名）** | 当前 HMAC 共享密钥，持有 `jwt_secret` 的服务可自行签发 token。RSA + JWKS 可让接入方只有公钥，只能验证不能伪造 |
| 低 | **没有单点登出（SLO）** | 认证中心登出后其他服务的 session 仍有效，缺少跨服务登出通知机制 |
| 低 | **客户端注册需重启** | 新增接入方需改 YAML + 重启，接入方多了后运维低效，可考虑管理 API 或数据库存储 |
| 低 | **没有审计日志** | 缺少"谁在什么时间从哪个服务登录/登出"的记录，合规场景需要 |

### 环境配置

通过 `MODE_ENV` 环境变量选择配置文件，对应 `conf/config_<MODE_ENV>.yaml`。未设置时默认读取 `config_test.yaml`。

- dev：本地开发，SLS 用 QA 代理，飞书 redirect_url 填 ngrok 地址
- test：`qa-empyrean-lens.lingowhale.com`，SLS 用 QA 代理
- pre：`pre-empyrean-lens.lingowhale.com`，SLS 用生产代理
- prod：`empyrean-lens.lingowhale.com`，SLS 用生产代理

### Handler 规范

Handler 函数由 hz 生成骨架，手动实现。约定：
- **form/query 参数**：用 `c.BindAndValidate(&req)` 绑定后传给 service，service 直接用 req 字段
- **JSON body 为数组时**：`BindAndValidate` 会失败，改用 `c.Request.Body()` 读取原始字节
- **Legacy 告警 Webhook**：所有响应均返回 HTTP 200，错误信息编码在 JSON body 的 `code`/`msg` 字段中
- **Auth / OIDC 端点**：返回标准 HTTP 状态码（400/401/500），错误格式遵循 RFC 6749（`{"error":"..."}`）

### Service 规范

**Legacy 服务（SlsNoticeWebhook 等）**：
- 方法签名：`(ctx, req) → (*middleware.BaseResp, *consts.BizCode)`
- 业务错误用 `consts.BizCode`；HTTP 层面始终返回 200

**Auth / OIDC 服务（AuthService、OidcService）**：
- 方法签名：`(ctx context.Context, c *app.RequestContext, req *model.XxxReq)`，直接写响应
- 响应必须使用 IDL 生成的 model struct（`TokenResp`、`IntrospectResp`、`MeResp`、`BaseResp`）
- 错误响应使用 `service/auth.go` 和 `service/oidc.go` 中定义的错误码常量
- 返回标准 HTTP 状态码，不强制 200

**通用规范**：
- 出站 HTTP 请求统一使用 `httplib.Do`（来自内部 go_lib），不使用标准库 `net/http`
  - GET 接口需加 `httplib.WithHttpMethod("GET")`，否则默认 POST
- JSON 序列化统一用 `github.com/bytedance/sonic`，不用 `encoding/json`
- 需要保留 JSON key 原始顺序时，用 `gjson.ForEach` 提取 key 列表，再用 `map[string]string` 存数据

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
| JWT（hertz 中间件） | `github.com/hertz-contrib/jwt`（基于 golang-jwt/v4） |
| JWT（手动验签，/introspect 用） | `github.com/golang-jwt/jwt/v4` |
| Redis | `github.com/redis/go-redis/v9`（via go_lib，用于 authorization code 存储） |
| 日志/配置/中间件 | `go_lib`（内部库，路径 `codeup.aliyun.com/deeplang/lingowhale/lingowhale_backend/go_lib`） |
| FC 运行时 | `github.com/aliyun/fc-runtime-go-sdk` |
