package consts

const (
	ACCESS_KEY_ID              = "REDACTED"
	ACCESS_KEY_SECRET          = "REDACTED"
	SECURE_TOKEN               = ""
	ENDPOINT                   = "cn-zhangjiakou.log.aliyuncs.com"
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
	HOST_LINGO_BACKEND     = "api.lingoreader.cn"
	HOST_LINGO_PRE_BACKEND = "pre-api.lingoreader.cn"
	HOST_CRAWLER           = "crawler.shenyandayi.com"
	HOST_WCD               = "wcd-v2.deeplang.net"
	HOST_EDU               = "api-edu-arch.shenyandayi.com"
	HOST_QA_BACKEND        = "api-chat.lingoreader.cn"

	HOST_ABSTRACT = "summary.shenyandayi.com"
	HOST_OURLINE  = "outlinecata.shenyandayi.com"
	HOST_OPINION  = "key-opinion.shenyandayi.com"
	//HOST_SUQIN           = "qaucloud-pdfparser.shenyandayi.com"
	HOST_SUQIN           = "pdfparser.shenyandayi.com"
	HOST_QA_RECOMMEND    = "qa-recommend.shenyandayi.com"
	HOST_QA_MAIN         = "qa-main.shenyandayi.com"
	HOST_REPEATER        = "api-repeater.lingoreader.cn"
	HOST_QUERY_EMBEDDING = "search-embedding-v2.shenyandayi.com"
	HOST_MULTI_MODEL     = "ai-infra-service.shenyandayi.com"
)

type API struct {
	Api   string
	Alias string
}

var NGINX_INGRESS_APIS = map[string][]API{
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
	},
	HOST_CRAWLER: {
		{
			Api:   "/crawl",
			Alias: "【数据处理】抓取网页",
		},
	},
	HOST_WCD: {
		{
			Api:   "/wcd-raw",
			Alias: "【数据处理】解析URL",
		},
	},
	HOST_EDU: {
		{
			Api:   "/edu_parse",
			Alias: "【数据处理】最小信息单元",
		},
		//{
		//	Api:   "/positions/list",
		//	Alias: "最小信息单元位置信息获取",
		//},
	},
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

const PASSPORT_HOST = "https://api-passport.lingoreader.cn"
const LINGO_HOST = "https://api.lingoreader.cn"
const PDF_TO_UPLOAD = "assets/1810.04805v2.pdf"
const GRAPH_CONFIG_PATH = "conf/graph.json"
const LOG_DETAIL_TEMPLATE_PATH = "templates/rentention.html"
const REALDATA_TEMPLATE_PATH = "templates/realdata.html"
const OVERVIEW_TEMPLATE_PATH = "templates/overview.html"
const (
	TIMESPAN_TODAY    = 0
	TIMESPAN_WEEK     = 1
	TIMESPAN_MONTH    = 2
	TIMESPAN_LONGTIME = 3
)

const (
	SLOWQUERY_THRESHOLD_FAST = 1.0

	SLOWQUERY_THRESHOLD_ABSTRACT  = 60.0
	SLOWQUERY_THRESHOLD_OUTLINE   = 60.0
	SLOWQUERY_THRESHOLD_VIEWPOINT = 60.0

	SLOWQUERY_THRESHOLD_MULTI_ANALYSIS = 60.0
	SLOWQUERY_THRESHOLD_MULTI_MERGE    = 60.0
	SLOWQUERY_THRESHOLD_MULTI_SUMMARY  = 60.0
	SLOWQUERY_THRESHOLD_MULTI_ETE      = 180.0

	SLOWQUERY_THRESHOLD_QA           = 60.0
	SLOWQUERY_THRESHOLD_QA_RECOMMEND = 10.0
)

const (
	StatusValid   = 0
	StatusDeleted = 1

	StatusUnk     = 0
	StatusSuccess = 1
)

const (
	Channel = "lingo"
)
