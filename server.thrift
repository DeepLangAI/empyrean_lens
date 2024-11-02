namespace go empyrean_lens

struct BaseResp {
    1: i32 code
    2: string msg
}

struct EmptyReq{}
struct EmptyResp{}

# 获取系统可用性分数
struct DailyScoreReq {
    1: string start_time
    2: string end_time
}

struct DailyScoreRespData {
    1: string date
    2: i32 score
    3: double day_over_day  # [0, 100]
    4: double week_over_week # [0, 100]
    5: double fail_rate
    6: double slow_rate
    7: double probe_fail_rate
    8: i32 num_traceback
    9: i32 total_req
}

struct DailyScoreResp {
    1: i32 code
    2: string msg
    3: list<DailyScoreRespData> data
}

# 获取系统实时分数等信息
struct Metric{
    1: i32 value
    2: double day_over_day # [0, 100]
    3: double week_over_week # [0, 100]
}

struct RealtimeScoreRespData{
    1: Metric score
    2: Metric num_total_req
    3: Metric num_err_req
    4: Metric avg_resp_cost
    5: Metric slow_query_percent
    6: Metric num_probe_err_req
}

struct RealtimeScoreResp{
    1: i32 code
    2: string msg
    3: RealtimeScoreRespData data
}

# Api失败相关信息
struct DailyApiFailureInfoReq{
    1: string date_begin
    2: string date_end
}

struct ApiFailureInfoResp{
    1: i32 code
    2: string msg
    3: list<ApiFailureInfoRespData> data
}

struct ApiFailureInfoRespData{
    1: string date
    2: string api_name
    3: string host
    4: i32 num_total_req
    5: i32 num_error_req
    6: double err_percent
    7: i32 num_code_3xx
    8: i32 num_code_4xx
    9: i32 num_code_5xx
    10: string api_path
    11: i32 num_biz_error_req
}

# API错误详情
struct DailyApiFailureDetailReq{
    1: string date_begin
    2: string date_end
    3: string host
    4: string path
    5: i32 code_type // 按nginx日志(0)，还是按业务错误码(1)
}

struct ApiFailureDetailResp{
    1: i32 code
    2: string msg
    3: list<ApiFailureDetailRespData> data
}

struct ApiFailureDetailRespData{
    1: string time
    2: string api_name
    3: string host
    4: string path
    5: string http_code
    6: string user_id
    7: string trace_id
    8: string client_ip
    9: i32 biz_code
    10: string biz_msg
}

# 慢查询相关信息
struct DailyApiSlowInfoReq{
    1: string date_begin
    2: string date_end
}

struct ApiSlowInfoResp{
    1: i32 code
    2: string msg
    3: list<ApiSlowInfoRespData> data
}

struct ApiSlowInfoRespData{
    1: string date
    2: i32 num_total_req
    3: i32 num_slow_req
    4: double api_avg_cost
    5: string host
    6: string api_name
    7: i32 num_error_req
    8: list<string> slow_details // list[map[str, any]]
    9: list<string> fail_details // list[map[str, any]]
}

# Api探针信息
struct ApiProbeReq{
    1: string date_begin
    2: string date_end
}

struct ApiProbeResp{
    1: i32 code
    2: string msg
    3: list<ApiProbeRespData> data
}

struct ApiProbeRespData{
    1: string date
    2: string scene
    3: i32 num_total_req
    4: i32 num_success_req
    5: i32 num_correct_req
    6: double avg_cost
}

# Api探针信息，详情
struct ProbeLogDetailReq{
    1: string date_begin
    2: string date_end
    3: string scene
    4: bool not_correct
    5: bool not_success
}

struct ProbeLogDetailResp{
    1: i32 code
    2: string msg
    3: list<ProbeLogDetailRespData> data
}

struct ProbeLogDetailRespData{
    1: string date
    2: string time
    3: string scene
    4: i32 data_source
    5: i32 http_code
    6: i32 business_code
    7: string msg
    8: bool success
    9: bool correct
    10: double cost
    11: string host
    12: string api_path
    13: string trace_id
}

# 全链路日志
struct EndToEndTraceReq{
    1: string date_begin
    2: string date_end
    3: string trace_id
    4: string level
}

struct EndToEndTraceResp{
    1: i32 code
    2: string msg
    3: list<EndToEndTraceRespData> data
}

struct EndToEndTraceRespData{
    1: string log_store_name
    2: string trace_id
    3: string user_id
    4: string time
    5: string msg
    6: string host
    7: string api_path
    8: double cost
    9: string client_ip
    10: string ua
    11: string channel
    12: map<string, string> origin_log
}


# 用户全链路日志
struct EndToEndUserTraceReq{
    1: string time_begin
    2: string time_end
    3: string user_id
}

struct EndToEndUserTraceResp{
    1: i32 code
    2: string msg
    3: list<EndToEndUserTraceRespData> data
}
struct EndToEndUserTraceRespData{
    1: string trace_id
    2: string time_begin
    3: string time_end
    4: i32 num_total_logs
    5: map<string, list<EndToEndTraceRespData>> scene_logs
}

# Traceback日志
struct TracebackReq{
    1: string date_begin
    2: string date_end
}

struct TracebackResp{
    1: i32 code
    2: string msg
    3: list<TracebackRespData> data
}

struct TracebackRespData{
    1: string exc_info
    2: string msg
    3: string trace_id
    4: string user_id
    5: string time
    6: map<string, string> origin_log
}

# 刷新缓存数据库
struct DbRefreshReq{
    1: i32 timespan
    2: bool rm
}
struct DbRefreshResp{
    1: i32 code
    2: string msg
    3: DbRefreshData data
}
struct DbRefreshData{
}

# 写入探针日志
struct ProbeLog{
    1: string scene
    2: string api
    3: string host
    4: bool is_core
    5: bool success
    6: bool correct
    7: double cost
    8: i32 data_source // 0: api, 1: ui
    9: i32 business_code
    10: i32 http_code
    11: string trace_id
    12: string msg
}

struct WriteProbeReq{
    1: list<ProbeLog> data
}
struct WriteProbeResp{
    1: i32 code
    2: string msg
}

# 数据库清理操作
struct DbTidyReq{
//    1: i32 timespan
}

// 查询用户信息
struct UInfoReq{
    1: string phone
    2: string uid
}
struct UInfoResp{
    1: i32 code
    2: string msg
    3: UInfoRespData data
}
struct UInfoRespData{
    1: string phone
    2: string uid
    3: string nickname
}
// 用户活动信息
struct UserActionInfoReq{
    1: list<string> uids
}
struct UserActionInfoResp{
    1: i32 code
    2: string msg
    3: list<UserActionInfoRespData> data
}
struct UserActionInfoRespData{
    1: string uid
    2: string action
    3: string time
    4: UserActionDetail detail
}
struct UserActionDetail{
    1: string result_status
    2: string failure_reason
    3: string file_id
    4: string action
    5: string action_channel
    6: double cost_seconds
}


// 查询请求量趋势
struct RequestTrendReq{
    1: string date
}
struct RequestTrendResp{
    1: i32 code
    2: string msg
    3: RequestTrendRespData data
}
struct RequestTrendRespData{
    1: list<RequestTrendRespDataItem> data0
    2: list<RequestTrendRespDataItem> data1
    3: list<RequestTrendRespDataItem> data7
}
struct RequestTrendRespDataItem{
    1: string time
    3: i32 req_count
    4: i32 score
    5: double fail_rate
    6: double slow_rate
    7: double probe_fail_rate
    8: double probe_fail_count
    9: i32 num_trace_back
}

// 上报：上线数据
struct UploadOnlineOperationReq{
    1: list<UploadOnlineOperationReqData> data
}
struct UploadOnlineOperationReqData{
	1: string builder // 'lixinliang',
	2: string branch_name // 'v2.0.7',
	3: string commit_message // 'log不截断超长URL 测试：PRE 环境验证通过',
	4: string domain_name // 'wcd.deeplang.net',
	5: string time // '2024-01-17 11:25:28',
	6: string app_name // 'Wcd_Python_Prod',
	7: string remote_url // 'https://codeup.aliyun.com/deeplang/biz-data/web-content-distill.git'
}
struct UploadOnlineOperationResp{
    1: i32 code
    2: string msg
    3: UploadOnlineOperationRespData data
}
struct UploadOnlineOperationRespData{
    1: i32 success_cnt
    2: list<string> fail_msgs
}

struct AuthReq{
    1: string code
    2: string state
}

// 上线单查询
struct OnlineOperationReq{
    1: string time_begin
    2: string time_end
    3: string app_name
}
struct OnlineOperationResp{
    1: i32 code
    2: string msg
    3: list<OnlineOperationRespData> data
}
struct OnlineOperationRespData{
    1: string app_name
    2: list<UploadOnlineOperationReqData> detail
}

// ========================================================
//               链路追踪相关接口定义
// ========================================================

// 用户行为
enum UserActionStatus {
    Success = 1 // 成功
    Fail = 2 // 失败
    Slow = 3 // 慢查询
    Timeout = 4 // 超时失败
}

// 资源渠道的枚举值
enum ChannelType {
    All = 0,  // 全部渠道

    PdfPc = 10,  // 语鲸web首页pdf上传button
    PdfPlugin = 11,  // 插件阅读面板button
    PdfReader = 12,  // pdf阅读器button
    PdfPcDb = 13,  // 语鲸web个人库上传button
    PdfWebReader = 14,  // web阅读器button

    UrlPc = 20,  // 语鲸web首页url输入框
    UrlPlugin = 21,  // 插件
    UrlPluginMenu = 22,  // 插件右键菜单
    UrlPcDb = 23,  // 语鲸web个人库上传button
    UrlReader = 24,  // web阅读器button

    WechatUrl = 30,  // url语鲸小助手
    WechatPdf = 31,  // pdf语鲸小助手

    MiniUrl = 40,  // 语鲸小程序url
    MiniPdf = 41,  // 语鲸小程序pdf

    MiniLingoUrl = 42,  // 灵狗小程序url
    MiniLingoPdf = 43,  // 灵狗小程序pdf

    WechatLingoUrl = 50,  // url灵狗小助手
    WechatLingoPdf = 51,  // pdf灵狗小助手

    DesktopUrl = 60,  // 桌面端url
    DesktopPdf = 61,  // 桌面端pdf

    WebLingoUrl = 70,  // url 灵狗web 端
    WebLingoPdf = 71,  // pdf 灵狗web 端
    WebLingoMulti = 72  // multi 灵狗web 端
}

// 实体类型
enum EntryTypeEnum{
    WORD = 1  # 词
    QUOTE = 2  # 句
    COLL = 3  # 搭配
    COLL_QUOTE = 4  # 句子搭配
    SUMMARY = 5  # 概述
    OUTLINE = 6  # 大纲
    WEB = 7  # 网页
    EXTRACT = 8  # 摘录
    FILE = 10  # pdf
    VIEWPOINT = 11  # 关键观点
    MULTI = 12  # 多文档总结
}

enum ActionStatusEnum {
    UNK = -1
    SUCCESS = 0 // 表示正常完成
    SLOW_SUCCESS = 1 //表示完成但比较慢（对齐稳定性指标的慢查询指标
    SLOW_FAIL = 2 // 模块超时失败
    FAIL = 3 // 模块执行失败
    UNREACHEAD = 4 // 未执行
    WORTHLESS = 5 // 无意义
}

enum LinkNodeTypeEnum{
    UPLOAD_FINISH = 1  // 上传完成
    CRAWLER_FINISH = 2  // 抓取完成
    WCD_PARSE_FINISH = 3  // wcd解析完成
    SUQIN_PARSE_FINISH = 4  // 苏秦解析完成
    TEXT_PARSE_FINISH = 5  // text-parse完成
    EDU_PARSE_FINISH = 6  // edu-parse完成
    SUMMARY_FINISH = 7  // 概述生成
    KEY_INFO_FINISH = 8  // 关键信息生成
    OUTLINE_FINISH = 9  // 大纲生成
    MULTI_ANALYSIS_FINISH = 10  // 单文档解析
    MULTI_TOPIC_FINISH = 11  // 主题生成
    MULTI_OUTLINE_FINISH = 12  // 大纲生成
}

// 用户行为查询
struct UserActionReq {
    1: string query // 关键词、url、uid、entry-id、multiid等。如果为空则表示不限制
    2: string start_time // consts.DateHourMinSecTemplate
    3: string end_time // consts.DateHourMinSecTemplate
    4: list<ActionStatusEnum> status // 如果为UNK则表示所有状态
    5: i64 skip // 分页查询，跳过多少条数据，0起
    6: i64 limit // 分页查询，每页多少条数据
    7: bool only_external // 仅外部用户
}

struct UserActionResp {
    1: i64 code
    2: string msg
    3: UserActionRespData data
}

struct UserActionRespData {
    1: bool has_next // 是否有下一页
    2: list<UserActionRespRow> rows // 表中每行数据
}

struct ResourceInfo{
    1: string entry_id
    2: EntryTypeEnum entry_type
    3: string title
    4: string url
}

struct UserActionRespRow {
    1: string user_id
    2: string create_time // DateHourMinSecTemplate
    3: string channel
    4: string action_name
    5: string title
    6: list<ResourceInfo> resources
    7: double cost // seconds
    8: ActionStatusEnum status
    9: EntryTypeEnum entry_type
    10: string entry_id
}

// 节点链路图
typedef string NodeId
struct TraceLinkGraph {
    1: list<GraphNode> nodes
    2: map<NodeId, list<NodeId>> edges
}

struct GraphNode {
    1: NodeId id // 节点id，可用bson.objectid来生成，方便查询
    2: string name // 节点名称，如上传完成、抓取完成、多文档合并等
    3: LinkNodeTypeEnum type // 节点类型
    4: string enter_time // 节点接收到请求的时间戳。DateHourMinSecTemplate
    5: string finish_time // 节点处理完成的时间戳。DateHourMinSecTemplate
    6: ActionStatusEnum status // 节点状态，如成功、失败、超时等
    7: string trace_id // trace id
}

// 单文档链路查询
struct DocLinkTraceReq {
    1: string entry_id
    2: EntryTypeEnum entry_type
}

struct DocLinkTraceResp {
    1: i64 code
    2: string msg
    3: DocLinkTraceRespData data
}

struct DocLinkTraceRespData {
    1: TraceLinkGraph link_graph
    2: double cost // end to end cost, seconds
    3: string entry_id
    4: EntryTypeEnum entry_type
    5: string title
}

// 多文档链路查询
struct MultiDocLinkTraceReq {
    1: string multi_id
}

struct MultiDocLinkTraceResp {
    1: i64 code
    2: string msg
    3: MultiDocLinkTraceRespData data
}

struct Article {
    1: EntryTypeEnum entry_type
    2: string entry_id
    3: NodeId start_id
    4: TraceLinkGraph graph
}

struct MultiDocLinkTraceRespData {
    1: TraceLinkGraph graph // 关于多文档本身的链路图，如多文档合并、主题生成、多文档大纲生成完成等，有可能退化为链表。
    2: double cost // end to end cost, seconds
    3: list<Article> articles // 多文档中包含的文档列表
    4: string title
}

// 链路中某节点的日志查询
struct LinkNodeLogReq {
    1: string entry_id
    2: EntryTypeEnum entry_type
    3: LinkNodeTypeEnum node_type
}

struct LinkNodeLogResp {
    1: i64 code
    2: string msg
    3: LinkNodeLogRespData data
}
struct LinkNodeLogRespData {
    1: list<ApiLog> logs
    2: double cost // end to end cost, seconds
    3: string trace_id
}

struct ApiLog {
    1: string path // 如 "/api/v1/link_trace/user_actions"
    2: string host // 如 "api-repeater.lingowhale.com"
    3: string method // 如 "GET", "POST"
    4: i64 http_code // 如 200
    5: i64 biz_code // business code，业务响应码
    6: i64 biz_msg // business message，业务响应信息
    7: string input // 请求参数
    8: string output // 响应结果
    9: string enter_time // DateHourMinSecTemplate
    10: string finish_time // DateHourMinSecTemplate
    11: string error_msg // 错误响应信息
}

// 查wcd在oss上传的详细日志
struct WcdOssDetalReq{
    1: string entry_id
    2: string trace_id
}

struct WcdOssDetalResp{
    1: i64 code
    2: string msg
    3: list<WcdOssDetalRespData> data
}

struct WcdOssDetalRespData{
    1: string raw_html
    2: string parsed_html
    3: string text_parser_labels
    4: string conclusion
}


service Rentention{
   EmptyResp OverviewRender(1: EmptyReq req) (api.get="/api/log/overview")
   EmptyResp ToolsRender(1: EmptyReq req) (api.get="/api/log/tools")

    // 鉴权
    EmptyResp Auth(1: AuthReq req) (api.get="/api/v1/report/auth")

   //  用于提供前后端分离接口
   RealtimeScoreResp SystemRealtimeScore(1: EmptyReq req) (
       api.get="/api/v1/report/realtime"
   )
   // 系统分数列表
   DailyScoreResp SystemDailyScore(1: DailyScoreReq req) (
       api.get="/api/v1/report/daily/score"
   )
   // 错误率，基于Nginx日志
   ApiFailureInfoResp SystemDailyApiFailureInfo(1: DailyApiFailureInfoReq req) (
       api.get="/api/v1/report/fail/list"
   )
   ApiFailureDetailResp SystemDailyApiFailureDetail(1: DailyApiFailureDetailReq req) (
       api.get="/api/v1/report/fail/detail"
   )


   // 慢查询率，基于业务日志
   ApiSlowInfoResp SystemDailyApiSlowInfo(1: DailyApiSlowInfoReq req) (
       api.get="/api/v1/report/slow/list"
   )
   // 探针错误率，基于探针日志
   ApiProbeResp SystemDailyApiCost(1: ApiProbeResp req) (
       api.get="/api/v1/report/probe/list"
   )
   ProbeLogDetailResp SystemProbeLogDetail(1: ProbeLogDetailReq req) (
       api.get="/api/v1/report/probe/detail"
   )

   // 全链路日志
   EndToEndTraceResp SystemEndToEndTraceLogs(1: EndToEndTraceReq req) (
       api.get="/api/v1/report/trace/list"
   )
   // 用户全链路日志
   EndToEndUserTraceResp SystemEndToEndUserTraceLogs(1: EndToEndUserTraceReq req) (
       api.get="/api/v1/report/user_trace/list"
   )
   // Traceback日志列表
   TracebackResp SysteTracebackLogs(1: TracebackReq req) (
       api.get="/api/v1/report/traceback/list"
   )

   // 系统请求量趋势数据
   RequestTrendResp RequestTrends(1: RequestTrendReq req) (
       api.get="/api/v1/report/trend/request"
   )

   // 用于提供缓存数据库接口
   DbRefreshResp SystemDbTidy(1: DbTidyReq req) (
       api.post="/api/v1/report/db/tidy"
   )
   // 刷新数据库数据
   DbRefreshResp SystemDbRefresh(1: DbRefreshReq req) (
       api.post="/api/v1/report/db/refresh"
   )
   // 查用户信息
   UInfoResp GetUInfo(1: UInfoReq req) (
       api.get="/api/v1/report/user/info"
   )
   /**** 用户活动信息 ****/
   UserActionInfoResp UserActionInfo(1: UserActionInfoReq req) (
       api.post="/api/v1/report/user/user_action_info"
   )

   // 上线单查询
   OnlineOperationResp GetOnlineOperation(1: OnlineOperationReq req) (
       api.get="/api/v1/report/online_operation"
   )

   // 上报数据
   // - 上线数据上报
   BaseResp UploadOnlineOperation(1: UploadOnlineOperationReq req) (
       api.post="/api/v1/report/upload/online_operation"
   )
   // - 探针日志上报
   WriteProbeResp WriteProbeLogs(1: WriteProbeReq req) (
       api.post="/api/v1/report/db/write_probe"
   )
}

service LinkTrace{
    // 查用户行为列表
    UserActionResp UserActions(1: UserActionReq req) (
        api.get="/api/v1/link_trace/user_actions"
    )
    // 查单文档链路
    DocLinkTraceResp DocLinkTrace(1: DocLinkTraceReq req) (
        api.get="/api/v1/link_trace/single_doc"
    )
    // 查多文档链路
    MultiDocLinkTraceResp MultiDocLinkTrace(1: MultiDocLinkTraceReq req) (
        api.get="/api/v1/link_trace/multi_doc"
    )
    // 链路中某节点的日志查询
    LinkNodeLogResp LinkNodeLogs(1: LinkNodeLogReq req) (
        api.get="/api/v1/link_trace/node_logs"
    )
    // wcd节点处理的详情
    WcdOssDetalResp WcdNodeDetail(1: WcdOssDetalReq req) (
        api.get="/api/v1/link_trace/wcd_oss_detail"
    )
}
