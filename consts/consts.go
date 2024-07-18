package consts

const (
	ACCESS_KEY_ID        = "REDACTED"
	ACCESS_KEY_SECRET    = "REDACTED"
	SECURE_TOKEN         = ""
	ENDPOINT             = "cn-zhangjiakou.log.aliyuncs.com"
	PROJECT_NAME         = "k8s-log-cefd7f8df3eab44a4a8184343c914af94"
	LOG_STORE_NAME       = "business-pod"
	NGINX_LOG_STORE_NAME = "nginx-ingress"
	LOG_QUERY_LIMIT      = 100000

	CORE_NAME_ABSTRACT  = "概述"
	CORE_NAME_OUTLINE   = "大纲"
	CORE_NAME_VIEWPOINT = "viewpoint"
	CORE_NAME_PDFPARSER = "PDFParser"
)

const (
	//ALIYUN_LOG_NODE_OUTLINE_AI_COST = "模型生成大纲-结束(模型)"
	ALIYUN_LOG_NODE_OUTLINE_AI_COST    = "模型生成大纲-结束"
	ALIYUN_LOG_NODE_OUTLINE_ETOE_COST  = "大纲生成完成"
	ALIYUN_LOG_NODE_ABSTRACT_ETOE_COST = "生成概述结束"
	ALIYUN_LOG_NODE_OUTLINE_AI_START   = "后端-开始请求大纲模型"
	ALIYUN_LOG_NODE_ABSTRACT_ETE_COST  = "生成概述结束"
	ALIYUN_LOG_NODE_ABSTRACT_AI_COST   = "模型生成概述"
)

var NODE_MAP = map[string]string{
	ALIYUN_LOG_NODE_OUTLINE_AI_COST:   "生成大纲模型耗时",
	ALIYUN_LOG_NODE_OUTLINE_ETOE_COST: "生成大纲端到端耗时",
	ALIYUN_LOG_NODE_ABSTRACT_AI_COST:  "生成概述模型耗时",
	ALIYUN_LOG_NODE_ABSTRACT_ETE_COST: "生成概述端到端耗时",
}

//var LOG_QUERY_LIMIT int64 = 100000

var CORE_APIS = map[string]string{
	"/api/plugin/file/add":                 "上传PDF",
	"/api/readers/url/upload":              "上传URL",
	"/api/plugin/articles/summary":         "概述/大纲/观点",
	"/api/plugin/articles/summary/list_v2": "刷新模型生成内容",
}
