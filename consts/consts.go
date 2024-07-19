package consts

const (
	ACCESS_KEY_ID              = "REDACTED"
	ACCESS_KEY_SECRET          = "REDACTED"
	SECURE_TOKEN               = ""
	ENDPOINT                   = "cn-zhangjiakou.log.aliyuncs.com"
	PROJECT_NAME               = "k8s-log-cefd7f8df3eab44a4a8184343c914af94"
	LOG_STORE_NAME             = "business-pod"
	NGINX_LOG_STORE_NAME       = "nginx-ingress"
	MODEL_NGINX_LOG_STORE_NAME = "model-nginx-ingress"
	LOG_QUERY_LIMIT            = 100000

	CORE_NAME_ABSTRACT  = "概述"
	CORE_NAME_OUTLINE   = "大纲"
	CORE_NAME_VIEWPOINT = "viewpoint"
	CORE_NAME_PDFPARSER = "PDFParser"
)

const (
	ALIYUN_LOG_NODE_OUTLINE_AI_COST = "模型生成大纲-结束(模型)"
	//ALIYUN_LOG_NODE_OUTLINE_AI_COST    = "模型生成大纲-结束"
	ALIYUN_LOG_NODE_OUTLINE_ETOE_COST  = "大纲生成完成"
	ALIYUN_LOG_NODE_ABSTRACT_ETOE_COST = "生成概述结束"
	ALIYUN_LOG_NODE_OUTLINE_AI_START   = "后端-开始请求大纲模型"
	ALIYUN_LOG_NODE_ABSTRACT_ETE_COST  = "生成概述结束"
	ALIYUN_LOG_NODE_ABSTRACT_AI_COST   = "模型生成概述"
	ALIYUN_LOG_NODE_QA_DONE_COST       = "回答结束"
	ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST = "观点模型输出完成"
)

var NODE_MAP = map[string]string{
	ALIYUN_LOG_NODE_OUTLINE_AI_COST:    "生成智能大纲模型耗时",
	ALIYUN_LOG_NODE_OUTLINE_ETOE_COST:  "生成智能大纲端到端耗时",
	ALIYUN_LOG_NODE_ABSTRACT_AI_COST:   "生成概述模型耗时",
	ALIYUN_LOG_NODE_ABSTRACT_ETE_COST:  "生成全文速览端到端耗时",
	ALIYUN_LOG_NODE_QA_DONE_COST:       "回答问题端到端耗时",
	ALIYUN_LOG_NODE_VIEWPOINT_ETE_COST: "生成关键信息模型耗时",
}

//var LOG_QUERY_LIMIT int64 = 100000

const (
	HOST_LINGO_BACKEND = "api.lingoreader.cn"
	HOST_CRAWLER       = "crawler.shenyandayi.com"
	HOST_WCD           = "wcd-v2.deeplang.net"
	HOST_EDU           = "api-edu-arch.shenyandayi.com"
	HOST_QA_BACKEND    = "api-chat.lingoreader.cn"

	HOST_ABSTRACT        = "summary.shenyandayi.com"
	HOST_OURLINE         = "outlinecata.shenyandayi.com"
	HOST_OPINION         = "key-opinion.shenyandayi.com"
	HOST_SUQIN           = "qaucloud-pdfparser.shenyandayi.com"
	HOST_QA_RECOMMEND    = "qa-recommend.shenyandayi.com"
	HOST_QA_MAIN         = "qa-main.shenyandayi.com"
	HOST_QUERY_EMBEDDING = "search-embedding-v2.shenyandayi.com"
)

type API struct {
	Api   string
	Alias string
}

var NGINX_INGRESS_APIS = map[string][]API{
	HOST_LINGO_BACKEND: {
		{
			Api:   "/api/plugin/file/add",
			Alias: "上传PDF",
		},
		{
			Api:   "/api/readers/url/upload",
			Alias: "上传URL",
		},

		{
			Api:   "/api/plugin/articles/summary",
			Alias: "全文速览/智能大纲/关键信息",
		},
		{
			Api:   "/api/plugin/articles/summary/list_v2",
			Alias: "刷新模型生成内容",
		},
	},
	HOST_CRAWLER: {
		{
			Api:   "/crawl",
			Alias: "抓取网页",
		},
	},
	HOST_WCD: {
		{
			Api:   "/wcd-raw",
			Alias: "解析URL",
		},
	},
	HOST_EDU: {
		{
			Api:   "/edu_parse",
			Alias: "最小信息单元",
		},
		{
			Api:   "/positions/list",
			Alias: "最小信息单元位置信息获取",
		},
	},
	HOST_QA_BACKEND: {
		{
			Api:   "/api/chat/qa",
			Alias: "问答后端",
		},
		{
			Api:   "/api/chat/recommend",
			Alias: "问题推荐后端",
		},
	},
}

var MODEL_NGINX_INGRESS_APIS = map[string][]API{

	HOST_ABSTRACT: {
		{
			Api:   "/generate",
			Alias: "生成全文速览",
		},
	},
	HOST_OURLINE: {
		{
			Api:   "/generate",
			Alias: "生成智能大纲",
		},
	},
	HOST_OPINION: {
		{
			Api:   "/generate",
			Alias: "生成关键信息",
		},
	},
	HOST_SUQIN: {
		{
			Api:   "/pdfparser",
			Alias: "苏秦PDF解析",
		},
	},
	HOST_QA_RECOMMEND: {
		{
			Api:   "/qa/query_recommend",
			Alias: "问题推荐模型",
		},
	},
	HOST_QA_MAIN: {
		{
			Api:   "/qa/main",
			Alias: "问答模型",
		},
	},
	//HOST_QUERY_EMBEDDING: {
	//	{
	//		Api:   "/get_embedding",
	//		Alias: "问句嵌入",
	//	},
	//},
}
