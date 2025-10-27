package consts

import (
	"empyrean_lens/biz/model/empyrean_lens"
	"time"
)

const (
	BaseContainerName = "lingowhale"
	EduContainerName  = "edu-arch"
	BasePodName       = "lingowhale"
	BaseHostName      = "lingowhale"
	BaseChannelName   = "lingowhale"
	TablePrefix       = "lingowhale_"
	CacheKeyPrefix    = "empyreanLens::" + BaseHostName
	ModelName         = "Atom-7B-Chat"
	ChatApi           = "https://api.atomecho.cn/v1/chat/completions"
	ChatSecret        = "REDACTED"
)
const (
	ACCESS_KEY_ID              = "REDACTED"
	ACCESS_KEY_SECRET          = "REDACTED"
	SECURE_TOKEN               = ""
	ENDPOINT                   = "cn-zhangjiakou.log.aliyuncs.com"
	FC_PROJECT_NAME            = "aliyun-fc-cn-zhangjiakou-81495289-1680-50dc-991e-3d118acdad1d"
	FC_LOG_STORE_NAME          = "function-log"
	PROJECT_NAME               = "k8s-log-cefd7f8df3eab44a4a8184343c914af94"
	BUSINESS_LOG_STORE_NAME    = "business-pod"
	NGINX_LOG_STORE_NAME       = "nginx-ingress"
	MODEL_NGINX_LOG_STORE_NAME = "model-nginx-ingress"
	METRIC_SOTRE_NAME          = "prod-metrics"
	LOG_QUERY_LIMIT            = 100000
	ALIYUN_SLS_API_KEY         = "REDACTED"
	ALIYUN_SLS_API_SECRET      = "REDACTED"

	CORE_NAME_ABSTRACT  = "概述"
	CORE_NAME_OUTLINE   = "大纲"
	CORE_NAME_VIEWPOINT = "viewpoint"
	CORE_NAME_PDFPARSER = "PDFParser"
	CORE_NAME_MULTI     = "多文档"

	CORE_NAME_CHAT           = "chat"
	CORE_NAME_CHAT_RECOMMEND = "chat_recommend"
)

const (
	//ALIYUN_LOG_NODE_OUTLINE_AI_COST = "模型生成大纲-结束(模型)"
	ALIYUN_LOG_NODE_OUTLINE_AI_COST    = "模型生成大纲-结束"
	ALIYUN_LOG_NODE_OUTLINE_ETOE_COST  = "大纲生成完成"
	ALIYUN_LOG_NODE_ABSTRACT_ETOE_COST = "生成概述结束"
	ALIYUN_LOG_NODE_OUTLINE_AI_START   = "后端-开始请求大纲模型"
	ALIYUN_LOG_NODE_ABSTRACT_ETE_COST  = "生成概述结束"
	ALIYUN_LOG_NODE_ABSTRACT_AI_COST   = "模型生成概述"
	ALIYUN_LOG_NODE_QA_DONE_COST       = "回答结束"
	ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST = "观点模型输出完成"

	ALIYUN_LOG_NODE_MULTI_ETE_COST    = "MULTI_ALL_SUCCESS"
	ALIYUN_LOG_NODE_ANALYSIS          = "ANALYSIS"
	ALIYUN_LOG_NODE_ANALYSIS_START    = "analysis_start"
	ALIYUN_LOG_NODE_ANALYSIS_ALL      = "ANALYSIS_ALL" // 单文档分析全部完成
	ALIYUN_LOG_NODE_ANALYSIS_REPEATER = "ANALYSIS_REPEATER"
	ALIYUN_LOG_NODE_MERGE             = "MERGE" // 多文档整合
	ALIYUN_LOG_NODE_MERGE_START       = "merge_start"
	ALIYUN_LOG_NODE_MERGE_REPEATER    = "MERGE_REPEATER"
	ALIYUN_LOG_NODE_MULTI_ALL_SUCCESS = "MULTI_ALL_SUCCESS" // 所有主题生成完毕
	ALIYUN_LOG_NODE_SUMMARY_REPEATER  = "SUMMARY_REPEATER"
	ALIYUN_LOG_NODE_THEME_ALL_SUMMARY = "THEME_ALL_SUMMARY"
	ALIYUN_LOG_NODE_THEME_SUMMARY     = "THEME_SUMMARY"
)

var NODE_MAP = map[string]string{
	ALIYUN_LOG_NODE_OUTLINE_AI_COST:    "【模型】生成智能大纲模型耗时",
	ALIYUN_LOG_NODE_OUTLINE_ETOE_COST:  "【后端】生成智能大纲端到端耗时",
	ALIYUN_LOG_NODE_ABSTRACT_AI_COST:   "【模型】生成全文速览模型耗时",
	ALIYUN_LOG_NODE_ABSTRACT_ETE_COST:  "【后端】生成全文速览端到端耗时",
	ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST: "【模型】生成关键信息模型耗时",
	ALIYUN_LOG_NODE_QA_DONE_COST:       "【后端】回答问题端到端耗时",

	ALIYUN_LOG_NODE_MULTI_ETE_COST:    "【多文档】多文档生成端到端",
	ALIYUN_LOG_NODE_ANALYSIS_REPEATER: "【多文档】模型-单文档解析",
	ALIYUN_LOG_NODE_MERGE_REPEATER:    "【多文档】模型-多文档合并分析",
	ALIYUN_LOG_NODE_SUMMARY_REPEATER:  "【多文档】模型-多文档总结",
}

const (
	// 后端服务
	HOST_LINGO_PUBLIC_BACKEND     = "api-public.lingowhale.com"
	HOST_LINGO_INNER_BACKED       = "api-inner.lingowhale.com"
	HOST_LINGO_PRE_PUBLIC_BACKEND = "pre-api-public.lingowhale.com"

	HOST_LINGO_BACKEND     = "api.lingowhale.com"
	HOST_LINGO_PRE_BACKEND = "pre-api.lingowhale.com"
	HOST_QA_BACKEND        = "chat.lingowhale.com"
	HOST_QA_PRE_BACKEND    = "pre-chat.lingowhale.com"
	HOST_REPEATER          = "api-repeater.lingowhale.com"
	HOST_PRE_REPEATER      = "pre-api-repeater.lingowhale.com"

	HOST_ZHILIAO_BACKEND           = "api-public.zhiliao.news"
	HOST_ZHILIAO_PRE_BACKEND       = "pre-api-public.zhiliao.news"
	HOST_ZHILIAO_INNER_BACKEND     = "api-inner.zhiliao.news"
	HOST_ZHILIAO_INNER_PRE_BACKEND = "pre-api-inner.zhiliao.news"

	// 后端数据服务
	HOST_CRAWLER     = "crawler.shenyandayi.com"
	HOST_PRE_CRAWLER = "pre-crawler.shenyandayi.com"
	HOST_WCD         = "wcd-v2.deeplang.net"
	HOST_PRE_WCD     = "pre-wcd-v2.deeplang.net"
	HOST_EDU         = "api-edu-arch.shenyandayi.com"
	HOST_PRE_EDU     = "pre-api-edu-arch.shenyandayi.com"

	// 模型服务
	HOST_ABSTRACT        = "summary.shenyandayi.com"
	HOST_OURLINE         = "outlinecata.shenyandayi.com"
	HOST_OPINION         = "key-opinion.shenyandayi.com"
	HOST_SUQIN           = "pdfparser.shenyandayi.com"
	HOST_QA_RECOMMEND    = "qa-recommend.shenyandayi.com"
	HOST_QA_MAIN         = "qa-main.shenyandayi.com"
	HOST_QUERY_EMBEDDING = "search-embedding-v2.shenyandayi.com"
	HOST_MULTI_MODEL     = "ai-infra-service.shenyandayi.com"
)
const (
	AVALIABILITY_TASK_LOCK = "empyrean_lens::availability_task_lock"
)

type API struct {
	Api   string
	Alias string
}

const (
	INDEX_PATH     = "/public/index.html"
	FORBIDDEN_PATH = "/public/403.html"
	HOME_PATH      = "/"
	// DOMAIN_PATH    = "cb92-163-123-192-24.ngrok-free.app"
	DOMAIN_PATH           = "empyrean-lens.lingowhale.com"
	LARK_COOKIE           = "lark_cookie"
	LARK_USERNAME         = "lark_username"
	LARK_MOBILE           = "lark_mobile"
	LARK_EMAIL            = "lark_email"
	LARK_EMPLOYEE_NO      = "lark_employee_no"
	LARK_AUTH_EXPIRE_TIME = 72 * time.Hour
)

var NGINX_INGRESS_APIS = map[string][]API{
	HOST_LINGO_PUBLIC_BACKEND: {
		{
			Api:   "/api/feed/v1/lingowhale_daily/list",
			Alias: "【今日TAB】【后端】自动拉取日报列表",
		},
		{
			Api:   "/api/feed/v1/lingowhale_daily/get",
			Alias: "【今日TAB】【后端】打开日报详情",
		},
		{
			Api:   "/api/feed/v1/topic/list",
			Alias: "【今日TAB】【后端】我的专题聚览列表",
		},
		{
			Api:   "/api/feed/v1/topic/get",
			Alias: "【今日TAB】【后端】打开专题详情",
		},
		{
			Api:   "/api/feed/v1/feed/topic",
			Alias: "【今日TAB】【后端】专题feed流",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/list",
			Alias: "【订阅TAB】【后端】我的订阅列表",
		},
		{
			Api:   "/api/feed/v2/feed/subscription",
			Alias: "【订阅TAB】【后端】订阅频道feed",
		},
		{
			Api:   "/api/feed/v1/search/list",
			Alias: "【订阅TAB】【后端】频道下内容搜索",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/search",
			Alias: "【订阅TAB】【后端】我创建的频道",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/upsert",
			Alias: "【订阅TAB】【后端】订阅频道",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/upsert",
			Alias: "【订阅TAB】【后端】创建频道",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/category",
			Alias: "【发现页】【后端】频道广场分类",
		},
		{
			Api:   "/api/feed/v1/feed/recommend",
			Alias: "【发现页】【后端】文章推荐",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/get",
			Alias: "【发现页】【后端】频道详情",
		},
		{
			Api:   "/api/feed/v1/subscription/upsert",
			Alias: "【订阅】【后端】添加订阅源",
		},
		{
			Api:   "/api/feed/v1/feed/subscription",
			Alias: "【订阅】【后端】用户订阅feed流",
		},
		{
			Api:   "/api/feed/v1/resource/get",
			Alias: "【订阅】【后端】获取资源详情",
		},
		{
			Api:   "/api/feed/v1/behavior/report",
			Alias: "【订阅】【后端】用户行为上报",
		},
		{
			Api:   "/api/feed/v1/subscription/user_list",
			Alias: "【订阅】【后端】获取用户已订阅数据源信息",
		},
		{
			Api:   "/api/feed/v1/info_source/search",
			Alias: "【订阅】【后端】获取信源分类列表",
		},
		{
			Api:   "/api/feed/v1/subscription/delete",
			Alias: "【订阅】【后端】删除用户订阅",
		},
		{
			Api:   "/api/feed/v1/search_history/delete",
			Alias: "【订阅】【后端】删除搜索记录",
		},
		{
			Api:   "/api/feed/v1/subscription/get",
			Alias: "【订阅】【后端】获取订阅信息",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/check",
			Alias: "【订阅】【后端】频道check",
		},
	},
	//HOST_LINGO_PRE_PUBLIC_BACKEND: {
	//	{
	//		Api:   "/api/feed/v1/subscription/upsert",
	//		Alias: "【订阅】【后端】添加订阅源",
	//	},
	//},
	HOST_LINGO_BACKEND: {
		// {
		// 	Api:   "/api/plugin/file/status",
		// 	Alias: "【异步状态】【后端】获取文件上传状态",
		// },
		// {
		// 	Api:   "/api/plugin/file/batch/status",
		// 	Alias: "【异步状态】【后端】批量获取文件上传状态",
		// },
		// {
		// 	Api:   "/api/readers/url/status",
		// 	Alias: "【异步状态】【后端】URL解析状态",
		// },
		// {
		// 	Api:   "/api/multi/detail",
		// 	Alias: "【异步状态】【后端】多文档状态",
		// },
		// {
		// 	Api:   "/api/readers/parse/status",
		// 	Alias: "【异步状态】【后端】URL/PDF/MULTI解析状态",
		// },
		{
			Api:   "/api/plugin/file/add",
			Alias: "【数据处理】【后端】上传PDF",
		},
		{
			Api:   "/api/readers/url/upload",
			Alias: "【数据处理】【后端】上传URL[web,小程序,插件]",
		},
		{
			Api:   "/api/readers/url/content/upload",
			Alias: "【数据处理】【后端】上传URL[小助手等]",
		},
		{
			Api:   "/api/plugin/articles/summary",
			Alias: "【单文档】【后端】全文速览/智能大纲/关键信息",
		},
		{
			Api:   "/api/plugin/articles/summary/list_v2",
			Alias: "【单文档】【后端】刷新模型生成内容(list_v2)",
		},
		{
			Api:   "/api/novel_form/get",
			Alias: "【阅读器】【后端】新内容形态",
		},
		{
			Api:   "/api/novel_form/resource/get",
			Alias: "【阅读器】【后端】资源详情",
		},
		{
			Api:   "/api/novel_form/feed/recommend",
			Alias: "【阅读器】【后端】更多内容推荐",
		},
		{
			Api:   "/api/readers/resource/copy",
			Alias: "【阅读器】【后端】拷贝",
		},
		{
			Api:   "/api/plugin/articles/summary_inc/outline",
			Alias: "【单文档】【后端】大纲增量生成",
		},
	},
	//HOST_LINGO_PRE_BACKEND: {
	//	{
	//		Api:   "/api/plugin/file/add",
	//		Alias: "【数据处理】【后端】上传PDF",
	//	},
	//	{
	//		Api:   "/api/readers/url/upload",
	//		Alias: "【数据处理】【后端】上传URL[web,小程序,插件]",
	//	},
	//	{
	//		Api:   "/api/readers/url/content/upload",
	//		Alias: "【数据处理】【后端】上传URL[小助手等]",
	//	},
	//	{
	//		Api:   "/api/plugin/articles/summary",
	//		Alias: "【单文档】【后端】全文速览/智能大纲/关键信息",
	//	},
	//	{
	//		Api:   "/api/plugin/articles/summary/list_v2",
	//		Alias: "【单文档】【后端】刷新模型生成内容(list_v2)",
	//	},
	//},
	HOST_CRAWLER: {
		{
			Api:   "/crawl",
			Alias: "【数据处理】抓取网页",
		},
	},
	//HOST_PRE_CRAWLER: {
	//	{
	//		Api:   "/crawl",
	//		Alias: "【数据处理】抓取网页",
	//	},
	//},
	HOST_WCD: {
		{
			Api:   "/wcd-raw",
			Alias: "【数据处理】解析URL",
		},
	},
	//HOST_PRE_WCD: {
	//	{
	//		Api:   "/wcd-raw",
	//		Alias: "【数据处理】解析URL",
	//	},
	//},
	HOST_EDU: {
		{
			Api:   "/edu_parse",
			Alias: "【数据处理】最小信息单元",
		},
	},
	//HOST_PRE_EDU: {
	//	{
	//		Api:   "/edu_parse",
	//		Alias: "【数据处理】最小信息单元",
	//	},
	//},
	HOST_QA_BACKEND: {
		{
			Api:   "/api/chat/qa",
			Alias: "【问答】【后端】问答",
		},
		{
			Api:   "/api/chat/recommend",
			Alias: "【问答】【后端】问题推荐",
		},
	},
	//HOST_QA_PRE_BACKEND: {
	//	{
	//		Api:   "/api/chat/qa",
	//		Alias: "【问答】【后端】问答",
	//	},
	//	{
	//		Api:   "/api/chat/recommend",
	//		Alias: "【问答】【后端】问题推荐",
	//	},
	//},
	HOST_REPEATER: {
		{
			Api:   "/doc/single/analyze",
			Alias: "【多文档】1[中继服务]单文档分析",
		},
		{
			Api:   "/doc/multi/analyze",
			Alias: "【多文档】2[中继服务]多文档整合",
		},
		{
			Api:   "/doc/multi/outline",
			Alias: "【多文档】3[中继服务]多文档总结",
		},
		{
			Api:   "/api/repeater/abstract",
			Alias: "【单文档】1[中继服务]全文速览",
		},
		{
			Api:   "/api/repeater/outline",
			Alias: "【单文档】2[中继服务]智能大纲",
		},
		{
			Api:   "/api/repeater/viewpoint",
			Alias: "【单文档】3[中继服务]关键信息",
		},
	},
	//HOST_PRE_REPEATER: {
	//	{
	//		Api:   "/doc/single/analyze",
	//		Alias: "【多文档】1[中继服务]单文档分析",
	//	},
	//	{
	//		Api:   "/doc/multi/analyze",
	//		Alias: "【多文档】2[中继服务]多文档整合",
	//	},
	//	{
	//		Api:   "/doc/multi/outline",
	//		Alias: "【多文档】3[中继服务]多文档总结",
	//	},
	//	{
	//		Api:   "/api/repeater/abstract",
	//		Alias: "【单文档】1[中继服务]全文速览",
	//	},
	//	{
	//		Api:   "/api/repeater/outline",
	//		Alias: "【单文档】1[中继服务]智能大纲",
	//	},
	//	{
	//		Api:   "/api/repeater/viewpoint",
	//		Alias: "【单文档】1[中继服务]关键信息",
	//	},
	//},

	HOST_ZHILIAO_BACKEND: {
		{
			Api:   "/api/topic/v1/chat/upsert",
			Alias: "【知了追踪】【后端】创建会话",
		},
		{
			Api:   "/api/topic/v1/chat/stream",
			Alias: "【知了追踪】【后端】聊天对话",
		},
		{
			Api:   "/api/topic/v1/topic/upsert",
			Alias: "【知了追踪】【后端】创建话题",
		},
		{
			Api:   "/api/topic/v1/topic/get",
			Alias: "【知了追踪】【后端】获取话题详情",
		},
		{
			Api:   "/api/topic/v1/topic/feed",
			Alias: "【知了追踪】【后端】话题feed",
		},
		{
			Api:   "/api/topic/v1/topic/all_feed",
			Alias: "【知了追踪】【后端】追踪入口all feed",
		},
		{
			Api:   "/api/topic/v1/topic/edit",
			Alias: "【知了追踪】【后端】编辑话题详情",
		},
		{
			Api:   "/api/topic/v1/topic/poll",
			Alias: "【知了追踪】【后端】话题状态变更",
		},
		{
			Api:   "/api/topic/v1/topic/search",
			Alias: "【知了追踪】【后端】搜索话题",
		},
		{
			Api:   "/api/topic/v1/topic/category",
			Alias: "【知了追踪】【后端】话题分类",
		},
		{
			Api:   "/api/topic/v1/user_sub_topic/upsert",
			Alias: "【知了追踪】【后端】话题点击追踪",
		},
		{
			Api:   "/api/topic/v1/user_sub_topic/get",
			Alias: "【知了追踪】【后端】用户订阅话题列表",
		},
		{
			Api:   "/api/topic/v1/user_sub_topic/del",
			Alias: "【知了追踪】【后端】用户取消追踪",
		},
		{
			Api:   "/api/topic/v1/behavior/report",
			Alias: "【知了追踪】【后端】用户行为",
		},

		{
			Api:   "/api/topic/v1/entry/get",
			Alias: "【知了追踪】【后端】文章详情",
		},
		{
			Api:   "/api/topic/v1/info_source/search",
			Alias: "【知了追踪】【后端】信源搜索",
		},
		{
			Api:   "/api/topic/v1/guide/get_topic",
			Alias: "【知了追踪】【后端】创建页话题推荐",
		},
		{
			Api:   "/api/topic/v1/product/list",
			Alias: "【知了追踪】【后端】查询商品列表",
		},
	},

	HOST_ZHILIAO_INNER_BACKEND: {
		{
			Api:   "/iapi/topic/v1/topic/official_upsert",
			Alias: "【知了追踪】【后端】创建官方话题",
		},
		{
			Api:   "/iapi/topic/v1/topic/feed",
			Alias: "【知了追踪】【后端】算法侧获取话题内容",
		},
		{
			Api:   "/iapi/topic/v1/info_source/search_resource",
			Alias: "【知了追踪】【后端】信源搜索内容池资源",
		},
		{
			Api:   "/iapi/topic/v1/permission/grant",
			Alias: "【知了追踪】【后端】开通权益",
		},
		{
			Api:   "/iapi/topic/v1/permission/retrieve",
			Alias: "【知了追踪】【后端】回收权益",
		},
		{
			Api:   "/iapi/topic/v1/product/upsert",
			Alias: "【知了追踪】【后端】创建商品",
		},
		{
			Api:   "/api/topic/v1/user/permission",
			Alias: "【知了追踪】【后端】查询用户权益",
		},
	},
}

var MODEL_NGINX_INGRESS_APIS = map[string][]API{

	HOST_ABSTRACT: {
		{
			Api:   "/generate",
			Alias: "【单文档】【模型】生成全文速览",
		},
	},
	HOST_OURLINE: {
		{
			Api:   "/generate",
			Alias: "【单文档】【模型】生成智能大纲",
		},
	},
	HOST_OPINION: {
		{
			Api:   "/generate",
			Alias: "【单文档】【模型】生成关键信息",
		},
	},
	HOST_SUQIN: {
		{
			Api:   "/pdfparser",
			Alias: "【数据处理】苏秦PDF解析",
		},
	},
	HOST_QA_RECOMMEND: {
		{
			Api:   "/qa/query_recommend",
			Alias: "【问答】【模型】问题推荐模型",
		},
	},
	HOST_QA_MAIN: {
		{
			Api:   "/qa/main",
			Alias: "【问答】【模型】问答模型",
		},
	},
	//HOST_QUERY_EMBEDDING: {
	//	{
	//		Api:   "/get_embedding",
	//		Alias: "问句嵌入",
	//	},
	//},
	HOST_MULTI_MODEL: {
		{
			Api:   "/multi-doc/single-doc-analysis",
			Alias: "【多文档】1[模型服务]单文档分析",
		},
		{
			Api:   "/multi-doc/doc-merge",
			Alias: "【多文档】2[模型服务]多文档整合",
		},
		{
			Api:   "/multi-doc/doc-summary",
			Alias: "【多文档】3[模型服务]多文档总结",
		},
	},
}

// 需要计算权重的维度
const (
	ERROR_RATE_PARAMETER       = "error_rate"
	SLOW_SEARCH_RATE_PARAMETER = "slow_search_rate"
	PROBE_ERROR_RATE_PARAMETER = "probe_error_rate"
)

// 计算权重相关
const (
	//错误率权重
	ERROR_WEIGHT = 0.3
	//慢查询权重
	SLOW_SEARCH_WEIGHT = 0.3
	//探针权重
	PROBE_WEIGHT = 0.4
)

// 单文档生成类型
const (
	GenerateTypeOverview = iota
	GenerateTypeOutline
	GenerateTypeViewPoint
)

const (
	PDFSuccessStatus           = 8
	URLSuccessStatus           = 0
	MultiSuccessAnalysisStatus = 2
	MultiSuccessMergeStatus    = 2
	MultiSuccessSummaryStatus  = 2
	SubscribeSuccessStatus     = 99
)

const LOG_DETAIL_TEMPLATE_PATH = "templates/rentention.html"
const REALDATA_TEMPLATE_PATH = "templates/realdata.html"
const OVERVIEW_TEMPLATE_PATH = "templates/overview.html"
const TOOLS_TEMPLATE_PATH = "templates/tools.html"
const DATA_SERVICE_TEMPLATE_PATH = "templates/data_service.html"
const (
	TIMESPAN_TODAY    = 0
	TIMESPAN_WEEK     = 1
	TIMESPAN_MONTH    = 2
	TIMESPAN_LONGTIME = 3
)

const (
	SLOWQUERY_THRESHOLD_FAST = 1.0

	SLOWQUERY_THRESHOLD_ABSTRACT  = 150.0
	SLOWQUERY_THRESHOLD_OUTLINE   = 300.0
	SLOWQUERY_THRESHOLD_VIEWPOINT = 150.0

	SLOWQUERY_THRESHOLD_MULTI_ANALYSIS = 100.0
	SLOWQUERY_THRESHOLD_MULTI_MERGE    = 100.0
	SLOWQUERY_THRESHOLD_MULTI_SUMMARY  = 200.0
	SLOWQUERY_THRESHOLD_MULTI_ETE      = 300.0

	SLOWQUERY_THRESHOLD_QA           = 60.0
	SLOWQUERY_THRESHOLD_QA_RECOMMEND = 15.0

	SLOWQUERY_THRESHOLD_COLLECT_SLOW_API = 5.0
	SLOWQUERY_DETAIL_THRESHOLD           = 5
)

const (
	StatusValid   = 0
	StatusDeleted = 1

	StatusUnk             = 0
	StatusSuccess         = 1
	LOG_QUERY_RETRY_TIMES = 3

	ErrorCodeTypeNginx    = 0
	ErrorCodeTypeBusiness = 1
	ErrorCodeTypeAll      = 2
)

const (
	DateTemplate           = "2006-01-02"
	DateHourTemplate       = "2006-01-02 15:00:00"
	DateHourMinuteTemplate = "2006-01-02 15:04:00"
	DateHourMinSecTemplate = "2006-01-02 15:04:05"
	DateTimeTemplate       = "2006-01-02 15:04:05,999"

	UserErrorStartDate = "2024-11-15"
	LinkTraceStartDate = "2024-11-15"
)

const (
	CacheKeyRequestTrend = CacheKeyPrefix + "::requestTrend::%v"
	HttpHeaderChannel    = "Channel"
	NonLoginChannel      = "local"
)

const (
	PDF   = "pdf"
	URL   = "url"
	MULTI = "multi"
)

type EntryType int64

const (
	EntryTypeWEB   = 7
	EntryTypePDF   = 10
	EntryTypeMulti = 12
)

var EntryTypeMap = map[int]string{
	EntryTypeWEB:   URL,
	EntryTypePDF:   PDF,
	EntryTypeMulti: MULTI,
}

const DB_NOT_FOUND_ERR = "not fountd"
const EntryInfoPreUserID = "lingowhale"
const LingowhelaBiDataVersion = "v1"
const DataTooLongUpper = 10000
const DataTooLongUpperErrMsg = "data too long"
const ContentIdxExpire = 86400 * 7 // content索引只保留7天

const (
	EntryInfoEntrySourceUserUpload    = iota // 用户上传
	EntryInfoEntrySourceSubscribe            // 订阅
	EntryInfoEntrySourceOperationsPre        // 运营预置
	EntryInfoEntrySourceShare                // 用户分享
	EntryInfoEntrySourceSummary              // 总结
)

const (
	EntryInfoDataTypeSingle              = iota // 单文档
	EntryInfoDataTypeMultiSingle                // 多文档-单文档
	EntryInfoDataTypeMulti                      // 多文档
	EntryInfoDataTypeSubscribe                  // 订阅
	EntryInfoDataTypeSingleFromSubscribe        // 单文档-订阅
	EntryInfoDataTypeMultiFromSubscribe         // 多文档-订阅
	EntryInfoDataTypeSummary                    // 总结-重新生成
)

// 链路追踪
var SingleFileProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
}
var SubscribeFileProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
}
var SingleWebReaderProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH,
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH,
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH,
	//empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
}
var SubscribeWebReaderProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH,
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH,
	//empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
}
var MultiProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH,
}
var SubscribeMultiProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
}
var MultiFileProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH,
}
var MultiWebReaderProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH,
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH,
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH,
	//empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH,
}

var TotalProcessList = []empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH,
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH,
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH,
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_RETRY_FINISH,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH,
}

var RetryProcessToEntryType = map[empyrean_lens.LinkNodeTypeEnum]empyrean_lens.EntryTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH:        empyrean_lens.EntryTypeEnum_SUMMARY,
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH:        empyrean_lens.EntryTypeEnum_OUTLINE,
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_RETRY_FINISH: empyrean_lens.EntryTypeEnum_OUTLINE,
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH: empyrean_lens.EntryTypeEnum_OUTLINE,
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH:       empyrean_lens.EntryTypeEnum_VIEWPOINT,
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH:  empyrean_lens.EntryTypeEnum_MULTI_OUTLINE,
}

var SingleFileProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:      {empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:  {empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:   {empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH, empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH},
}
var SubscribeFileProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:    {empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:     {empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:      {empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH: {empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH},
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH: {empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH},
}
var SingleWebReaderProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:  {empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH},
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH: {empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH},
	//empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH, empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH},
}
var SubscribeWebReaderProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:        {empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:      {empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:      {empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH, empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH, empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH: {empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH},
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH: {empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH},
}
var MultiProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH: {empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH},
}
var SubscribeMultiProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:   {empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH, empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH: {empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH},
}
var MultiFileProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:      {empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:  {empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:   {empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH},
}
var MultiWebReaderProcessMapping = map[empyrean_lens.LinkNodeTypeEnum][]empyrean_lens.LinkNodeTypeEnum{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:  {empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH},
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH: {empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH},
	//empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH},
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH: {empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH},
}

var LinkNodeTypeName = map[empyrean_lens.LinkNodeTypeEnum]string{
	empyrean_lens.LinkNodeTypeEnum_UPLOAD_FINISH:               "上传完成",
	empyrean_lens.LinkNodeTypeEnum_CRAWLER_FINISH:              "抓取完成",
	empyrean_lens.LinkNodeTypeEnum_WCD_PARSE_FINISH:            "网页解析完成",
	empyrean_lens.LinkNodeTypeEnum_SUQIN_PARSE_FINISH:          "pdf解析完成",
	empyrean_lens.LinkNodeTypeEnum_TEXT_PARSE_FINISH:           "text-parse完成",
	empyrean_lens.LinkNodeTypeEnum_EDU_PARSE_FINISH:            "edu-parse完成",
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_FINISH:              "概述生成",
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_FINISH:             "关键信息生成",
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_FINISH:              "大纲生成",
	empyrean_lens.LinkNodeTypeEnum_MULTI_ANALYSIS_FINISH:       "单文档分析",
	empyrean_lens.LinkNodeTypeEnum_MULTI_TOPIC_FINISH:          "主题生成",
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_FINISH:        "多文档大纲生成",
	empyrean_lens.LinkNodeTypeEnum_SUMMARY_RETRY_FINISH:        "概述重新生成",
	empyrean_lens.LinkNodeTypeEnum_OUTLINE_RETRY_FINISH:        "大纲重新生成",
	empyrean_lens.LinkNodeTypeEnum_KEY_INFO_RETRY_FINISH:       "关键观点重新生成",
	empyrean_lens.LinkNodeTypeEnum_MULTI_OUTLINE_RETRY_FINISH:  "多文档大纲重新生成",
	empyrean_lens.LinkNodeTypeEnum_SUBSCRIBE_NOVEL_FORM_FINISH: "内容新形态生成",
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_FINISH:       "简单大纲生成",
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_FINISH:       "详细大纲生成",
	empyrean_lens.LinkNodeTypeEnum_SIMPLE_OUTLINE_RETRY_FINISH: "简单大纲重新生成",
	empyrean_lens.LinkNodeTypeEnum_DETAIL_OUTLINE_RETRY_FINISH: "详细大纲重新生成",
}

const (
	FilterProbeUser      = "not user_id: 823df25bde18445494b5691222979cd0 not user_id: 6575b44010fcc60ccaf92101 not user_id: e8cc0d425acd4660b36afcbb976a7d97"
	ModelFilterProbeuser = "not content.user_id: 823df25bde18445494b5691222979cd0 not content.user_id: 6575b44010fcc60ccaf92101 not content.user_id: e8cc0d425acd4660b36afcbb976a7d97"
)

const (
	Platform_IOS     = "IOS"
	Platform_Android = "ANDROID"
)

const (
	ModelAutomation_Correlation             = "correlation"
	ModelAutomation_SingleOutline           = "single_outline"
	ModelAutomation_SingleOutlineContinuity = "single_outline_continuity"
)

// Nginx日志里获取超时的接口（慢日志列表）
// 这里的接口Alias（别名）需要严格按照这个格式，逻辑会对Alias进行split，方法为utils/common.go中的SplitAlias()
var NGINX_INGRESS_COLLECT_SLOW_APIS = map[string][]API{
	HOST_LINGO_PUBLIC_BACKEND: {
		{
			Api:   "/api/feed/v1/lingowhale_daily/list",
			Alias: "【今日TAB】【后端】日报列表",
		},
		{
			Api:   "/api/feed/v1/lingowhale_daily/get",
			Alias: "【今日TAB】【后端】日报详情",
		},
		{
			Api:   "/api/feed/v1/topic/list",
			Alias: "【今日TAB】【后端】我的专题",
		},
		{
			Api:   "/api/feed/v1/topic/get",
			Alias: "【今日TAB】【后端】专题详情",
		},
		{
			Api:   "/api/feed/v1/feed/topic",
			Alias: "【今日TAB】【后端】专题详情页文章列表",
		},
		{
			Api:   "/api/feed/v1/topic/subscribe",
			Alias: "【今日TAB】【后端】订阅专题",
		},
		{
			Api:   "/api/feed/v1/topic/unsubscribe",
			Alias: "【今日TAB】【后端】取消订阅专题",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/list",
			Alias: "【订阅TAB】【后端】我的订阅列表",
		},
		{
			Api:   "/api/feed/v2/feed/subscription",
			Alias: "【订阅TAB】【后端】订阅频道feed",
		},
		{
			Api:   "/api/feed/v1/search/list",
			Alias: "【订阅TAB】【后端】频道下内容搜索",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/search",
			Alias: "【订阅TAB】【后端】我创建的频道",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/upsert",
			Alias: "【订阅TAB】【后端】订阅频道",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/delete",
			Alias: "【订阅TAB】【后端】删除订阅关系",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/batch_upsert",
			Alias: "【订阅TAB】【后端】批量订阅频道",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/sort",
			Alias: "【订阅TAB】【后端】订阅列表排序",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/check",
			Alias: "【订阅TAB】【后端】批量检查用户订阅",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/upsert",
			Alias: "【订阅TAB】【后端】创建频道",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/category",
			Alias: "【发现页】【后端】频道广场分类",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/aigc_info",
			Alias: "【发现页】【后端】生成频道信息",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/info_options",
			Alias: "【发现页】【后端】获取频道信息选项",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/add_opml",
			Alias: "【发现页】【后端】导入opml订阅源数据并创建频道",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/parse_status",
			Alias: "【发现页】【后端】rss链接解析以及频道创建状态",
		},
		{
			Api:   "/api/feed/v1/feed/recommend",
			Alias: "【发现页】【后端】文章推荐",
		},
		{
			Api:   "/api/feed/v1/subscription_channel/get",
			Alias: "【发现页】【后端】频道详情",
		},
		{
			Api:   "/api/feed/v1/subscription/upsert",
			Alias: "【订阅】【后端】添加订阅源",
		},
		{
			Api:   "/api/feed/v1/feed/subscription",
			Alias: "【订阅】【后端】用户订阅feed流",
		},
		{
			Api:   "/api/feed/v1/resource/get",
			Alias: "【订阅】【后端】获取资源详情",
		},
		{
			Api:   "/api/feed/v1/behavior/report",
			Alias: "【订阅】【后端】用户行为上报",
		},
		{
			Api:   "/api/feed/v1/subscription/user_list",
			Alias: "【订阅】【后端】获取用户已订阅数据源信息",
		},
		{
			Api:   "/api/feed/v1/info_source/search",
			Alias: "【订阅】【后端】获取信源分类列表",
		},
		{
			Api:   "/api/feed/v1/info_source/category",
			Alias: "【订阅】【后端】获取信源分类列表 (频道版本前老 app 使用)",
		},
		{
			Api:   "/api/feed/v1/subscription/delete",
			Alias: "【订阅】【后端】删除用户订阅",
		},
		{
			Api:   "/api/feed/v1/search_history/delete",
			Alias: "【订阅】【后端】删除搜索记录",
		},
		{
			Api:   "/api/feed/v1/subscription/get",
			Alias: "【订阅】【后端】获取订阅信息",
		},
		{
			Api:   "/api/feed/v1/user_subscribe/check",
			Alias: "【订阅】【后端】频道check",
		},
		{
			Api:   "/api/feed/v1/resource/wechat_author/list",
			Alias: "【订阅】【后端】供应商获取需要抓取的公众号作者",
		},
		{
			Api:   "/api/feed/v1/resource/wechat_author/provider/updateStatus",
			Alias: "【订阅】【后端】供应商状态更新",
		},
		{
			Api:   "/api/feed/v1/resource/wechat_article/add",
			Alias: "【订阅】【后端】添加公众号文章（队列）",
		},
		{
			Api:   "/api/feed/v1/lingowhale_daily/cal",
			Alias: "【语鲸日报】【后端】计算语鲸日报",
		},
		{
			Api:   "/api/feed/v1/lingowhale_daily/get_voice_info",
			Alias: "【语鲸日报】【后端】获取语鲸日报语音详情",
		},
		{
			Api:   "/api/feed/v1/lingowhale_daily/polling",
			Alias: "【语鲸日报】【后端】日报轮训接口",
		},
		{
			Api:   "/api/feed/v1/anonymous_login/bind",
			Alias: "【匿名登录】【后端】匿名登录绑定用户",
		},
		{
			Api:   "/api/feed/v1/op_status/report",
			Alias: "【其他接口】【后端】上报状态",
		},
		{
			Api:   "/api/feed/v1/op_status/get",
			Alias: "【其他接口】【后端】获取状态",
		},
		{
			Api:   "/api/feed/v1/app_version/latest",
			Alias: "【其他接口】【后端】获取app最新版本",
		},
		{
			Api:   "/api/feed/v1/jiguang/bind_device",
			Alias: "【其他接口】【后端】极光设备绑定",
		},
		{
			Api:   "/api/feed/v1/hot/list",
			Alias: "【热点榜单】【后端】热点榜单列表",
		},
		{
			Api:   "/api/feed/v1/search/list",
			Alias: "【搜索】【后端】列表搜索",
		},
		{
			Api:   "/api/feed/v1/search_history/list",
			Alias: "【搜索记录】【后端】搜索历史记录",
		},
		{
			Api:   "/api/feed/v1/search_history/delete",
			Alias: "【搜索记录】【后端】删除历史记录",
		},
		{
			Api:   "/api/feed/v1/utils/file/upload",
			Alias: "【工具类】【后端】上传文件",
		},
		{
			Api:   "/api/feed/v1/share/parse",
			Alias: "【分享】【后端】分享口令解析",
		},
		{
			Api:   "/api/feed/v1/user_config/get",
			Alias: "【用户配置】【后端】获取用户配置",
		},
		{
			Api:   "/api/feed/v1/songsu/sync",
			Alias: "【其他接口】【后端】同步松鼠快看数据",
		},
		{
			Api:   "/api/feed/v1/songsu/status",
			Alias: "【其他接口】【后端】查询松鼠快看是否同步",
		},
		{
			Api:   "/api/feed/v1/model/smart_outline",
			Alias: "【模型生成接口】【后端】模型生成接口",
		},
	},
	HOST_LINGO_BACKEND: {
		{
			Api:   "/api/plugin/file/add",
			Alias: "【数据处理】【后端】上传PDF",
		},
		{
			Api:   "/api/readers/url/upload",
			Alias: "【数据处理】【后端】上传URL[web,小程序,插件]",
		},
		{
			Api:   "/api/readers/url/content/upload",
			Alias: "【数据处理】【后端】上传URL[小助手等]",
		},
		{
			Api:   "/api/plugin/articles/summary",
			Alias: "【单文档】【后端】全文速览/智能大纲/关键信息",
		},
		{
			Api:   "/api/plugin/articles/summary/list_v2",
			Alias: "【单文档】【后端】刷新模型生成内容(list_v2)",
		},
		{
			Api:   "/api/novel_form/get",
			Alias: "【阅读器】【后端】新内容形态",
		},
		{
			Api:   "/api/novel_form/resource/get",
			Alias: "【阅读器】【后端】资源详情",
		},
		{
			Api:   "/api/novel_form/feed/recommend",
			Alias: "【阅读器】【后端】更多内容推荐",
		},
		{
			Api:   "/api/readers/resource/copy",
			Alias: "【阅读器】【后端】拷贝",
		},
		{
			Api:   "/api/plugin/articles/summary_inc/outline",
			Alias: "【单文档】【后端】大纲增量生成",
		},
	},
	HOST_LINGO_INNER_BACKED: {
		{
			Api:   "/iapi/feed/v1/novel_form/get",
			Alias: "【订阅-inner】【后端】获取新内容形态的详情",
		},
		{
			Api:   "/iapi/feed/v1/subscription_channel/search",
			Alias: "【发现页-inner】【后端】内部搜索频道",
		},
		{
			Api:   "/iapi/feed/v1/monitor/article_exist",
			Alias: "【其他接口-inner】【后端】判断微信文章是否存在",
		},
		{
			Api:   "/iapi/feed/v1/share/get",
			Alias: "【分享-inner】【后端】获取分享透传信息",
		},
		{
			Api:   "/iapi/feature/v1/user/upsert",
			Alias: "【用户特征-inner】【后端】更新",
		},
		{
			Api:   "/iapi/feature/v1/user/list",
			Alias: "【用户特征-inner】【后端】列表",
		},
		{
			Api:   "/iapi/feature/v1/item/list",
			Alias: "【物品特征-inner】【后端】列表",
		},
		{
			Api:   "/iapi/feature/v1/item/recent_pos_behavior",
			Alias: "【物品特征-inner】【后端】获取用户最近有积极操作的物品列表",
		},
		{
			Api:   "/iapi/feature/v1/item/cal_feature",
			Alias: "【物品特征-inner】【后端】计算物品特征",
		},
		{
			Api:   "/iapi/feature/v1/item/del_feature",
			Alias: "【物品特征-inner】【后端】删除物品特征",
		},
		{
			Api:   "/iapi/feature/v1/item/search",
			Alias: "【物品特征-inner】【后端】搜索item",
		},
		{
			Api:   "/iapi/feature/v1/item/scan_feature",
			Alias: "【物品特征-inner】【后端】Scan物品特征",
		},
		{
			Api:   "/iapi/feature/v1/item/upsert_dimension",
			Alias: "【物品特征-inner】【后端】更新细分维度",
		},
		{
			Api:   "/iapi/feature/v1/behavior/report",
			Alias: "【用户行为-inner】【后端】行为报告",
		},
		{
			Api:   "/iapi/feature/v1/behavior/cal_feature",
			Alias: "【用户行为-inner】【后端】特征计算",
		},
		{
			Api:   "/iapi/feature/v1/behavior/count",
			Alias: "【用户行为-inner】【后端】统计用户行为",
		},
		{
			Api:   "/iapi/feature/v1/behavior/list",
			Alias: "【用户行为-inner】【后端】行为list",
		},
		{
			Api:   "/iapi/feature/v1/anonymous_login/bind",
			Alias: "【匿名登录-inner】【后端】匿名登录绑定用户",
		},
		{
			Api:   "/iapi/feature/v1/cluster/sim_resource",
			Alias: "【聚类-inner】【后端】相似文章聚类",
		},
		{
			Api:   "/iapi/feature/v1/cluster/search_result",
			Alias: "【聚类-inner】【后端】查找聚类结果",
		},
		{
			Api:   "/iapi/feature/v1/cluster/get_result",
			Alias: "【聚类-inner】【后端】获取聚类结果",
		},
		{
			Api:   "/iapi/feature/v1/cluster/search_cluster_ids_by_center",
			Alias: "【聚类-inner】【后端】根据中心id 查找聚类id",
		},
		{
			Api:   "/iapi/feature/v1/cluster/search_cluster_ids_by_center_entry",
			Alias: "【聚类-inner】【后端】兜底策略：根据中心文章id 查找最新的一条聚类 id",
		},
		{
			Api:   "/iapi/feature/v1/cluster/search_cluster_ids_by_center_entry",
			Alias: "【聚类-inner】【后端】兜底策略：根据中心文章id 查找最新的一条聚类 id",
		},
		{
			Api:   "/iapi/feature/v1/cluster/merge_result",
			Alias: "【聚类-inner】【后端】合并聚类结果",
		},
		{
			Api:   "/iapi/feature/v1/cluster/multi_summary",
			Alias: "【聚类-inner】【后端】通过聚类结果生成多文档",
		},
	},
}
