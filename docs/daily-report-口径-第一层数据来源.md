# 每日巡检报表口径 · 三层验证（一层定稿 / 二层初版含悬置 / 三层双日闭环）

> 逐指标验证于 2026-07-08（样本日 2026-07-07，环比基线 07-06），全部 SQL 实跑过并交叉勾稽。
> 统计窗口约定：报表日 00:00–24:00（北京时间）。SLS 默认 log_store=`business-pod`，网关层用 `nginx-ingress`。

## 0. 通用方法论（所有 SQL 必须遵守，血泪换来的）

1. **关键词只做粗筛，判定全部写在 SQL `like` 精确匹配里。** SLS 关键词层有分词陷阱：`add` 命中不了 `add_wechat_article`（下划线不分词），会造成假阴性；like 模式要带路由结尾逗号（如 `%.../resource/add,%`）防前缀误匹配。
2. **计数锚点用短日志。** 带全文 body 的 RequestRout 长日志会被采集端截丢约 10%（越长越容易丢）；计数用 ResponseRath 或 nginx 访问日志，长日志只用于从 body 抽字段（当抽样看待）。
3. **大时间窗结果要拆片验证。** SLS 扫描不完整时不报错、静默返回偏小值。全天数字用两个半天相加核对一次。
4. **`total` 字段是关键词命中数，不是 SQL 结果。** 曾把 438,476（关键词命中）当成接收总量，真值是 SQL 聚合的 283,200。
5. **分清「任务量纲」和「URL/篇量纲」。** 失败重试会把日志条数膨胀 4~30 倍。一次性抓取（公众号文章、图片）按 URL 去重数篇数；周期轮询（RSS feed）按条数数任务。每个指标必须写明量纲。
6. **用 `uid` 划分渠道，不用域名。** crawl 日志 extra 里的 `uid`：`resource_server`=公众号自采链路、`1`=订阅抓取（含 RSS 源产出的微信 URL）、其余散 uid=用户手动触发。
7. **两个接收接口的 JSON 格式不统一**：`/resource/add` 无空格（`"source":14`），`send_add_resource_msg` 有空格（`"source": 11`）。regexp 一律写 `"key":\s*(...)`。
8. **任何 section 查询结果为 0 行/全 null 时，报表显示「⚠️ 口径疑似失效」而不是 0。**（`crawl_timing_normal mp.weixin.qq.com` 已静默失效数月的教训。）

## 1.1 供应商推送（清博 / 人民网）

**链路**（两家推的全是公众号文章，同一个外部入口；术语约定：收货=暂存原文，加工=入库处理）：

```
① 到达    nginx 网关（client_ip 归属供应商）           📊 推送量
② 受理    feed 当场回 Respcode:0                        📊 接收成功率 = ②/①（同步口径，
          ↑ 分子勿用下游任何计数——队列积压会伪装成丢失（07-08 实测教训）
③ 收货    resource add_wechat_article：原文（含全文HTML）写 lingowhale_plugin.resource
          （状态 Init）+ 发 MNS 消息。此处不判重，重复副本照单全收
④ 排队    MNS 队列。积压不丢只延迟 → 报表「⏳在途」注记；07-08 人民网爆推积压 3.1 万至次日
⑤ 加工    消费者调 /resource/add（语义=开始加工，非再次入库）
          入口两道闸：URL 处理锁（秒级连推拦截）+ preCheck 查库（ID/URL/md5）
          📊 2.2 矩阵「⓪入口拦截」行 = 全链路最大流失点（拦重复副本，非丢失）
⑥ 流水线  同一条 resource 记录逐阶段推进 → Ready       📊 2.2 矩阵各阶段行
⑦ 上架    通知下游写 lingowhale.content_info            📊 3.1 有效入库
```

- 两库分工：lingowhale_plugin.resource = 仓库+车间（Init→Ready 状态机）；content_info = 成品货架。
- 被⑤拦截的重复副本，其③的暂存记录永久滞留 Init（死数据，人民网 ~4万条×全文HTML/日）→ 优化项：③处加 URL 判重或定期清理。

**source 枚举**（`/resource/add` body）：1=Subscription（订阅）、11=FromMonitoring（自采集）、12=Renminwang（人民网）、14=Qingbo（清博）、15=XiaoYuZhou（小宇宙）。

| 指标 | 口径 | 07-07 实测（环比） |
|---|---|---|
| 推送量 | resource-go RequestRout `/resource/add` 按 source 分组（语义归因，不依赖 IP） | 人民网 116,140（+2.5%）· 清博 38,899（+2.9%） |
| 接收成功率 | **同步受理口径**：nginx status=200 ÷ 到达（按 client_ip 拆供应商）。分子严禁用下游计数（积压→伪丢失） | 人民网 99.995%（6 条 502）· 清博 100% |
| 推送时效 | body `pub_time`（unix 秒）→ 日志时间差，服务端 approx_percentile | 人民网 P50 15.9m / P90 47.7m / P99 3.8h · 清博 P50 23.5m / P90 55.5m / P99 4.1h |

```sql
-- 推送量
RequestRout and resource and add |
select regexp_extract(message, '"source":([0-9]+)', 1) as src, count(*) as cnt
from log where message like '%RequestRout:/iapi/resource/v1/resource/add,%'
group by src order by cnt desc limit 10

-- 接收成功率（log_store=nginx-ingress）
wechat_article |
select client_ip, count(*) as total, count_if(status = 200) as ok
from log where url = '/api/feed/v1/resource/wechat_article/add'
group by client_ip order by total desc limit 10

-- 推送时效（分钟）
RequestRout and resource and add |
select regexp_extract(message, '"source":([0-9]+)', 1) as src,
  round(approx_percentile(greatest(__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint), 0), 0.50)/60.0, 1) as p50,
  round(approx_percentile(greatest(__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint), 0), 0.90)/60.0, 1) as p90,
  round(approx_percentile(greatest(__time__ - cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint), 0), 0.99)/60.0, 1) as p99
from log where message like '%/iapi/resource/v1/resource/add%'
  and cast(regexp_extract(message, '"pub_time":([0-9]+)', 1) as bigint) > 0
group by src order by src limit 10
```

**勾稽断言**：nginx 量 ≈ feed ResponseRath（Respcode:0）量 ≈ `/resource/add` source=12+14 量，容差 0.5%（07-07 三层为 155,057−6 = 155,051 = 155,039+16 级别，全部咬合）。

**语义注意**：
- 供应商**没有重试/补推机制**（业务确认）→ 网关非 200 ≈ 文章真实丢失；且 502 请求的载荷无处可查（nginx 不记 body），丢了不知道丢的是谁。对账机制（推送带唯一 ID 进 header）值得立项。
- 时效 P99 长尾是供应商自身延迟（真实健康信号），阈值告警有意义。
- 时效样本有偏：只能从长日志抽 pub_time（丢长文），覆盖率 ~72%。根治方案：让接收端在 ResponseRath 或独立短日志里带 pub_time + source。
- 老模板"人民网 66,220"是**处理端（去重后）口径**；真实接收 116K，前置去重损耗 37% 原来完全不可见。

## 1.2 公众号自采集（wechat_spider）

**链路**：

```
wechat_spider（集群外 10.0.4.3，独立 Mongo 库 wechat-spider）
  ① 监控 1,918 个公众号（target_account 表），发现新文章 → article 表
  ② 批量查重 POST /iapi/feed/v1/monitor/article_exist（语鲸已有=covered，不推）
  ③ 只推 uncovered：POST /iapi/resource/v1/resource/send_add_resource_msg
     payload={source_uniq_id, author, url, entry_type:7, source:11}（只有链接，无正文）
  → resource-go 受理 → 调爬虫抓正文（crawl 日志 uid=resource_server）
  → 抓到 html 回灌 /resource/add（source=11）→ 入库管线
另有独立回灌脚本（enqueue_backfill.py 等）直推 ③ 接口，不写 article 表。
```

| 指标 | 口径 | 07-07 实测 |
|---|---|---|
| 推送量（到达，含回灌） | SLS：`send_add_resource_msg` RequestRout 数 | 31,137（07-06: 6,135；其中 16–19 点回灌 ≈25,817） |
| 接收量 | 同路由 ResponseRath 且 Respcode:0 | 31,137（100%） |
| 常规链路日推送 | spider 库 `article` 按 push_time 当日计数（回灌不写此表，天然干净） | 5,304 |
| 采集时效 P50/P90/P99 | spider 库 `push_time − publish_time` 分位 | 10.1 min / 6.9 h / 22.5 h |
| 覆盖账号数 | spider 库：当日有 article 的 distinct target_account / target_account 总数 | 1,121 / 1,918 |
| 供应商重叠率 | spider 库 `exist=true` 占比 | 9.5% |

```sql
-- 推送量/接收量（SLS）
send_add_resource_msg |
select
  count_if(message like '%RequestRout:/iapi/resource/v1/resource/send_add_resource_msg,%'
           and message like '%"source": 11%')  as push_cnt,
  count_if(message like '%ResponseRath:/iapi/resource/v1/resource/send_add_resource_msg,%'
           and message like '%Respcode:0,%')   as accept_cnt
from log
```

```javascript
// spider 库（直连 Mongo，连接串在 wechat_spider/config/__init__.py 的 MongoConfig）
// 时效：db.article.find({publish_time: 当日}) 取 push_time-publish_time 算分位
// 账号：db.target_account.count() 为分母；当日 article distinct target_account 为分子
```

**回灌识别（勾稽断言）**：SLS 到达量 / spider 库 push_time 当日计数 > 1.5 → 报表自动标注「当日有历史回灌，二三层环比会连带波动」。

**附表：正文抓取通道健康**（uid=resource_server，URL 去重量纲）

| 通道 | 07-07（URL 级） |
|---|---|
| 整体 | 成功 15,295 / 尝试 ~17,402 ≈ 95%（净失败需扣除重试后成功） |
| Kakalong | 成功 15,975 / 净失败 2,499 = 86.5%（含 435 篇失败后重试成功） |
| Tikhub | 9 篇全挂（老口径"301 次失败"是重试膨胀 33 倍） |
| Renmin 代理 | 53 篇成功 |

**坑**：
- CLAUDE.md 原注释「Kakalong 成功不进 crawl_normal」**已过时**（2026-07 验证 15,975 里 15,819 都进了）；两者相加会双计。
- 失败日志膨胀 7.9~33 倍，任何按条数算的成功率都不可信（07-07 条数口径 53.9% vs URL 真实 86.5%）。
- 推送量语义是「请求次数」；spider 超时重试会一篇计多次，精确篇数按 body `source_uniq_id` 去重。
- 待业务确认：uid=resource_server 是否只服务自采集（若也给其他渠道补抓，附表要再拆）。
- **wechat-spider 库时间字段是北京时间墙钟按 UTC 类型存的（naive）**（2026-07-13 验证：最新 push_time 比真实 UTC 超前 8h）。日窗口必须注入墙钟边界（当日 00:00:00Z），不能用真 UTC 边界（前日 16:00:00Z）——引擎 Query 加 `MongoNaiveCST: true`。修正前采集量少计约 3%（07-12：3,194 → 3,296）。`lingowhale.content_info` 验证过是真 UTC，不受影响。
- 采集量口径 2026-07-13 拍板改为**发布日**（publish_time），与 spider 侧日报对齐（目标：下掉 spider 报告，由本报表 cover）。注意该口径有补采长尾：报表 08:00 生成时约九成已采到，数值是当时截面（07-12 例：08:00 截面 ~3,061，10:00 ~3,339，收敛后 3,414）；环比时两天都是同时点截面，可比。

## 1.3 网站/RSS 抓取（订阅渠道，uid=1）

**任务从哪来**：subscription 服务按 `subscription.sub` 表（16,204 个活跃信源）× 各自 `frequency`（自适应，均值 ~9h）自动排程轮询；信源清单上游是语鲸产品的订阅/运营配置。**渠道边界按 uid=1 划分，含 RSS 源产出的微信 URL**（与 1.2 按 uid 互斥）。

| 指标 | 口径 | 07-07 实测 |
|---|---|---|
| 抓取任务总量 | crawl 日志条数（轮询语义：每次轮询=一个任务，**不做 URL 去重**） | 143,784 |
| 任务成功率 | crawl_normal 占比 | 89.9% |
| 内容覆盖 | 成功/失败去重 URL 数 | 52,062 / 3,174 |
| 调度完成率 | 实际任务数 ÷ Σ(1440/frequency)（分母 Mongo 现算） | 143,784 / 166,900 ≈ 86% |
| 新文时效 | `/resource/add` source=1 的 pub_time 时效，**只统计 lag≤24h** | P50 2.1h / P90 7.8h / ≤4h 占比 82.5%（样本 69,777） |
| 历史回补量 | source=1 中 lag>24h 的篇数 | 35,436（占 34%） |

```sql
-- 任务量/成功率/内容覆盖
crawl_normal OR crawl_error |
select count_if(message='crawl_normal') as ok_tasks, count_if(message='crawl_error') as fail_tasks,
  count(distinct case when message='crawl_normal' then regexp_extract(extra,'"url":"([^"]*)"',1) end) as ok_urls,
  count(distinct case when message='crawl_error'  then regexp_extract(extra,'"url":"([^"]*)"',1) end) as fail_urls
from log where extra like '%"domain"%' and extra like '%"uid":"1"%'

-- Top 失败站点（双维度）
crawl_normal OR crawl_error |
select regexp_extract(extra,'"domain":"([^"]*)"',1) as domain,
  count_if(message='crawl_error') as fail_tasks, count(*) as total_tasks,
  round(count_if(message='crawl_error')*100.0/count(*),1) as fail_pct,
  count(distinct case when message='crawl_error' then regexp_extract(extra,'"url":"([^"]*)"',1) end) as fail_urls
from log where extra like '%"domain"%' and extra like '%"uid":"1"%'
group by domain having count_if(message='crawl_error') > 0
order by fail_tasks desc limit 8

-- 新文时效（lag≤24h 截断）
RequestRout and resource and add |
select count(*) as fresh_cnt,
  round(approx_percentile(lag_min,0.50),0) as p50, round(approx_percentile(lag_min,0.90),0) as p90,
  round(count_if(lag_min<=240)*100.0/count(*),1) as le4h_pct
from (select (__time__ - cast(regexp_extract(message,'"pub_time":([0-9]+)',1) as bigint))/60.0 as lag_min
      from log where message like '%/iapi/resource/v1/resource/add%' and message like '%"source":1,%'
        and cast(regexp_extract(message,'"pub_time":([0-9]+)',1) as bigint) > 0)
where lag_min >= 0 and lag_min <= 1440
```

**时效评价框架**：实际 P50 对照调度预算（≈平均 frequency/2）。实际 ≤ 预算 → 🟢（想更快是产品决策：加密轮询）；实际 > 预算 2 倍 → 🔴 管线积压。07-07：2.1h vs ~4.5h，健康。

**Top 失败站点（07-07）**：fail_urls 列区分「整站挂」vs「个别源挂」——rsshub.app 99.9%/484 源=整站挂（连续 ≥5 天）；52pojie.cn 87.8% 但仅 13 源、linux.do 100%/10 源=个别源死循环。

**阈值建议**：失败率>80% 且失败源>50 → 🔴 疑似整站故障；失败率>80% 且失败源<20 → 🟡 个别源失效建议换源；连续 ≥3 天升一档；调度完成率 <60% → 🔴 调度异常。
**连续失败天数**：由 Redis 每日快照（`daily-report:{date}`）读近 N 天，不重查历史。

## 1.4 图片抓取

| 指标 | 口径 | 07-07（环比） |
|---|---|---|
| 抓取任务量 | `async_crawl_img end` 条数（任务量纲；日志无 URL 做不了图片去重） | 684,429（+10.7%） |
| 任务失败率 | `所有方式均失败` 条数 ÷ 任务量（同量纲） | 2.2%（失败 15,051，+15.8%） |
| 失败图片数 | 失败日志按 img_url 去重 | 4,232 张 / 374 域名（条数膨胀 3.6 倍） |
| 耗时 | end 日志 cost 分位 | P50 6.2s / P90 15.8s / P95 19.6s / P99 28.6s |
| Top 失败图床 | 失败按 img_url 域名分组 | financialjuice 1,644 · springer 1,003 · chzbgr 982 · vulners 854 |

**坑与建议**：CLAUDE.md「每张图片仅一条」已过时（同图跨任务反复失败）；失败尾部混有广告追踪像素（dable/toast pixel），属上游正文清洗问题非抓取故障；埋点建议——end 日志带上 img_url。阈值：单图床失败 >500 张/天 🟡。

## 跨节勾稽 & 漏斗上半段

`/iapi/resource/v1/resource/add` 是全渠道统一汇聚点（07-07 总计 283,200）：

| source | 渠道 | 07-07 | 对应节 |
|---|---|---|---|
| 12 | 人民网 | 116,140 | 1.1 |
| 1 | 订阅抓取 | 105,213 | 1.3（feed→文章非 1:1，转换比待二层定） |
| 14 | 清博 | 38,899 | 1.1 |
| 11 | 自采集（含回灌） | 22,933 | 1.2（注意≠推送量 31,137，此处是抓完正文的回灌） |
| 15 | 小宇宙 | 15 | 播客 |

---

# 第二层 · 处理与监控层（初版，含悬置点）

> 聚类（SimResourceCluster / 推荐侧 [Dedup]）全部不进报表——聚类是管线内部衍生逻辑，不是数据渠道（2026-07-08 拍板）。

## 2.1 各环节服务健康

| 环节 | 处理量 | 成功率 | P50 | P99 | 口径 |
|---|---|---|---|---|---|
| 内容抓取 | 169,012 任务 | 86.8% | 3.2s | 54.2s | crawl 日志全渠道（环节视角不拆 uid）；耗时 `crawl_timing_normal` |
| 解析（edu_parse） | 221,547 次 | 99.85% | 24.7s | 141s | edu-arch-go `ResponseRath:/edu_parse` Respcode + cost。耗时双峰实锤：21% <500ms（md5 缓存命中）、72% ≥5s（真实全文解析） |
| 入库管线 | 241,214 篇 | 82.8% | 14.4s | 59.6s | `ResourceProcessor end process resource` status + cost（成功路径；失败路径 P99 852s 单独注释） |

**环节间勾稽（已闭合，误差 0.1%）**：解析量 = 进管线量 − 解析前死亡（UrlChecked + 去重 + 抓取失败）。07-07：理论 221,122 vs 实际 resource_server 发起 221,354（散户用户 ~190 次在管线外）。供应商推送和自采抓取**都走解析**（四渠道均有 ResourceParsed 阶段失败为证）。

⚠️ **cost 语义矛盾待确认**：管线整体 P50 14.4s < 解析单环节 P50 24.7s——两个 cost 的起止点语义不同（管线 cost 疑似不含解析等待），跨环节耗时对比暂禁。

```sql
-- 解析环节（量/成功率/耗时；注意大域截断：edu_parse 关键词域 88 万条/天，
-- 单发全天聚合必被静默截断（实测骗走 4K），必须拆半相加或用窄域 ResponseRath and edu_parse）
ResponseRath and edu_parse |
select count(*) as resp, count_if(message not like '%Respcode:0,%') as fail,
  round(approx_percentile(cast(regexp_extract(message,'cost: ([0-9]+) ms',1) as double),0.50),0) as p50_ms,
  round(approx_percentile(cast(regexp_extract(message,'cost: ([0-9]+) ms',1) as double),0.99),0) as p99_ms
from log where message like '%ResponseRath:/edu_parse,%'

-- 入库管线
ResourceProcessor |
select regexp_extract(message,'status:(\w+)',1) as status, count(*) as cnt,
  round(approx_percentile(cast(regexp_extract(message,'cost:([0-9.]+)',1) as double),0.50),1) as p50_s,
  round(approx_percentile(cast(regexp_extract(message,'cost:([0-9.]+)',1) as double),0.99),1) as p99_s
from log where message like '%end process resource%' group by status limit 5
```

## 2.2 入库失败原因分类（渠道 × 阶段矩阵）

07-07 实测（各渠道阶段失败之和与 failed 总数勾稽误差 = 0）：

| 渠道 | 处理总量 | 成功 | 去重过滤 | 内容无效/不支持 | 抓取失败 | 生成失败 | 系统异常 |
|---|---|---|---|---|---|---|---|
| Subscription 订阅 | 104,615 | 93,156 | 3,007 | 2,781 | 5,224 | 431 | 16 |
| Renminwang 人民网 | 72,997 | 55,998 | 14,676 | 1,881 | — | 429 | 13 |
| Qingbo 清博 | 38,789 | 35,732 | 1,758 | 1,065 | — | 233 | 1 |
| FromMonitoring 自采 | 24,698 | 13,452 | 3,371 | 6,777 | — | 144 | 29 |

阶段 → 分类映射【暂定，悬置待拍板】：DuplicateChecked→去重过滤；ResourceParsed+Validated→内容无效/不支持；ResourceCrawled→抓取失败（新增列，模板没有）；AuthorParsed+UrlChecked→系统异常。

各渠道病灶不同：人民网 86% 损耗是去重（供应商重复推）；自采 60% 是解析失败（微信反爬拦截页 html worthless）；订阅特有抓取失败（源站挂）。模板空着的"自有网站/RSS"两行合并为 Subscription 一行（管线内拆不开）。

```sql
ResourceProcessor and error |
select regexp_extract(message,'source: ([A-Za-z]+)',1) as source,
  regexp_extract(message,'lastest status: ([A-Za-z]+)',1) as stage, count(*) as cnt
from log where message like '%handle resource error%'
group by source, stage order by source, cnt desc limit 40
```

## 2.3 数据生成服务

- 总量/成功率/失败数：**OutRequest 口径**（模型调用维度），沿用 stability 现有 SQL。07-07：single_abstract 339,060 次 99.61%、single_outline 71,420 次 98.52%、hs_tts 33,596、multi_abstract 16,169、multi_theme 15,971、smart_outline 5,811、model_daily 5,104、multi_summary 682、single_enlightenment 330。
- P99 耗时：**API 请求维度**（`Response rout` cost）：abstract 42.3s / analyze 60.0s / multi_abstract 56.3s / smart_outline 83.8s / daily 72.4s / enlightenment 180s。查询类路由（毫秒级）排除。
- Top 失败原因：level:error 按前缀归三桶——内容不满足条件（正常过滤）≈65% / 解析入参 ≈33% / 模型下游网络 ≈2%。无"模型超时"独立埋点。

## 二层悬置点清单（数据已验证，归类/语义待拍板）

1. Validated（Title required，935 条）归"内容无效"还是"系统异常"
2. AuthorParsed/UrlChecked 归"系统异常"是否成立
3. ResourceCrawled 独立成列还是并入其他
4. 前置去重（37%）在 2.2 表还是只在漏斗展示
5. 2.3 两个量纲（OutRequest vs API）混排是否接受
6. service_type ↔ 模板中文任务名对照未核对
7. 失败原因三桶归法未确认
8. cost 字段起止语义（管线 14.4s < 解析 24.7s 矛盾）

---

# 第三层 · 有效入库层（双日验证闭环）

## 3.1 有效入库（剔除聚类 entry_type=12）

| 内容类型 | 07-07 | 07-06 | 口径 |
|---|---|---|---|
| 公众号文章 | 107,006 | — | content_info create_time 当日 + entry_type=7 + orig_url 含 mp.weixin.qq.com |
| 网站文章 | 63,104 | — | 同上，非微信 |
| PDF/文档 | 13,169 | 143 | entry_type=10（07-07 暴增 92 倍 = 新上 PDF 信源，环比灯应亮） |
| **合计** | **183,279** | **145,349** | |

- **有效率** = 有效入库 ÷ 第一层收到推送总量：07-07 = 62.9%，07-06 = 66.2%。（模板样例 84.3% 是拿"进管线量"当分母，藏掉了前置去重和自采中段。）
- **端到端时延**：公众号 P50 ≈16min、82% ≤30min；网站 P50 ≈2.4h、84% ≤24h。P99 因 pub_time 日期精度+历史回补失真，**报表用 ≤30min / ≤4h / ≤24h 三档占比，不报 P99**。
- 聚类勾稽（不进报表但留作口径健康探针）：entry_type=12 新增 = SimResourceCluster 管线成功数，两天均精确相等。

## 3.2 当日漏斗（漏斗顶 = 第一层各节接收量，与报表一层逐节对应）

```
                                07-07（回灌日）      07-06（干净日）
第一层 · 收到推送               291,422              219,418
  供应商(nginx)                 155,057              151,154
  自采推送(spider)               31,137                6,135
  订阅提交                      105,213               62,117
  小宇宙                             15                   12
  ├─ 供应商转发损耗                 −18                  −17   (502 等)
  ├─ 自采中段(抓正文→回灌)       −8,204                 +862   ⚠️ 挂起：两天方向相反，存在第二入口/跨天队列
汇聚 /iapi/resource/v1/resource/add
                                283,200              220,263
  ├─ 前置快速去重               −41,986              −42,586   (无日志，差值推算)
进入管线                        241,214              177,677
  ├─ 管线失败                   −42,761              −29,357
成功处理动作                    198,453              148,320   ← 与"管线量−失败数"分毫不差（两天均 ✅）
  ├─ 当日重复处理               −13,713               −2,094   (回灌日放大 6.5 倍，交叉印证)
去重资源                        184,740              146,226
  ├─ 残差                       −1,461                 −877    (0.8% / 0.6%)
第三层 · 有效入库               183,279              145,349
```

**结论**：漏斗结构两日稳定复现，全链路残差 <1%，勾稽容差定 1%。挂起项：自采中段构成（回灌第二入口，FromMonitoring 管线量两天都 > 回灌量）。

```sql
-- 成功动作 vs 去重资源（重复处理探针）
ResourceProcessor |
select count(*) as success_msgs,
  count(distinct regexp_extract(message,'entry_id: ([^;]+)',1)) as uniq_entries
from log where message like '%end process resource%' and message like '%status:success%'
  and message not like '%source: SimResourceCluster%'
```

```javascript
// 有效入库（Mongo lingowhale.content_info，CST 日窗口换 UTC：前日 16:00Z ~ 当日 16:00Z）
[{"$match":{"create_time":{"$gte":ISODate("T-1 16:00Z"),"$lt":ISODate("T 16:00Z")}}},
 {"$group":{"_id":"$entry_type","count":{"$sum":1}}}]
// 端到端分桶：$project (create_time-pub_time)/60000 → $group by kind(weixin/web) × bucket
```

## 待办清单

| 类型 | 事项 |
|---|---|
| 配置 | wechat-spider Mongo 连接串进天眼 conf（直连方案，mcp-db 加库作废；注意密钥管理欠账） |
| 埋点需求 | ① 接收端短日志带 pub_time+source（修 1.1 时效样本偏差）② 图片 end 日志带 img_url ③ 供应商推送带唯一 ID 进 header（502 对账）④ edu_parse 拆轻重路由或响应带解析类型 |
| 待业务确认 | ① uid=resource_server 是否只服务自采集 ② 管线/解析 cost 字段起止语义 |
| ~~挂起疑点~~ 已结案 | ① **"第二入口"不存在**（代码证实 AddResourceHandler 是唯一管线入口）：FromMonitoring 管线量 > 到达量是因为 `end process` 日志的量纲是**处理动作数**（失败后管线内部重试、每次重试都打一条，不产生新 /resource/add 请求，抽样 trace 证实一次到达两条 end process）。自采失败率最高故重试放大最明显。② 自采推送→回灌的日间错位来自 P0 常驻队列（send_add_resource_msg → ResourceInnerP0Queue 即 wechat_spider 专用队列，晚推次日处理属正常）③ 漏斗残差 <1% 精确归因（低优，保持挂起） |
| 精度升级 | **"前置快速去重"可直接观测**：`/resource/add` 的 ResponseRath 有 22% Respcode≠0（07-07 共 62,963 条），前置拒绝写在响应码里——漏斗该层从"差值推算"改为按 Respcode 分布统计 |
| 量纲修正 | 2.2"处理总量"实为**处理动作数**（含管线内重试）；要数资源数按 uniq_id 去重（与三层漏斗口径统一）。报表落地时列名标注清楚 |
| 悬置拍板 | 二层 8 个悬置点（见二层清单）+ 有效率分母口径（决策 A：现按"第一层接收总量"）|
| 框架 | 环比/连续失败天数走 Redis 快照 `daily-report:{date}`（TTL 90d）；引擎 QueryFunc 支持 SLS+Mongo 双源；0 行/全 null 显示口径失效；**大关键词域 section 自动按时间分片查询再聚合**（防 SLS 静默截断）；3.1 环比由快照自动积累 |
| 文档 | CLAUDE.md 两条过时注释待修（Kakalong 绕过 crawl_normal、图片失败每图一条） |


## 附：入库判重机制（2026-07-08 代码钉死，resource_processor.go）

重复副本的拦截/判出共四道，位置和依据不同：

| # | 位置 | 依据 | 结果 | 可观测 |
|---|---|---|---|---|
| 0 | process() 入口 URL 锁 | 同 orig_url 正在处理中（5 分钟锁） | 拒绝，Respcode≠0，**无处理日志** | /resource/add 非零响应码（07-07 全渠道 62,963 条/22%） |
| 1 | preCheck 查库 | source + source_uniq_id 相同 | 成功结束（更新互动数据，不新增） | end process success，acts−uniqs = 「重复更新」 |
| 2 | preCheck 查库 | orig_url + root_path 相同且已入库 | 同上 | 同上 |
| 3 | preCheck / 阶段⑤ | 正文 md5/内容相似（跨 URL 转载） | 阶段⑤失败 310001（已花抓取+解析成本） | 矩阵「⑤内容去重」行 |

- 供应商（人民网）重复是**秒级连推** → 全部死在第 0 道（锁），故 2.2「重复更新」行供应商恒为 —；内部渠道重复隔小时到达 → 死在 1/2 道，计入「重复更新」。
- 人民网到达中约 37% 为重复副本（两日稳定），非丢失；1.1 表已单列「重复拦截」列，推送量−丢失−重复拦截=2.2 处理总量。
- 遗留核对（低优）：非零 Respcode 的错误码构成抽样（理论上混有少量参数非法/黑名单）。
