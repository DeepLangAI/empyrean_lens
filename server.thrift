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
}

# API错误详情
struct DailyApiFailureDetailReq{
    1: string date_begin
    2: string date_end
    3: string host
    4: string path
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
    2: i32 count
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