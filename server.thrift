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


# 系统真实数据
struct RealDataReq{
    1: string date # 2006-01-02
}

struct RealDataResp{
    1: i32 code
    2: string msg
    3: RealDataRespData data
}

struct RealDataRespData{
    1: i32 prebuild_web
    2: i32 upload_web
    3: i32 web
    4: i32 file_in_multi
    5: i32 file_in_single
    6: i32 file
    7: i32 summary_abstract
    8: i32 summary_outline
    9: i32 summary_viewpoint
    10: i32 summary
    11: i32 question
    12: i32 answer
    13: i32 question_recommend
    14: i32 multi
    15: i32 multi_by_theme
}


service Rentention{
   EmptyResp LogRender(1: EmptyReq req) (api.get="/api/log/report")
   EmptyResp OverviewRender(1: EmptyReq req) (api.get="/api/log/overview")
   EmptyResp RealDataRender(1: EmptyReq req) (api.get="/api/log/realdata")

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
   // 慢查询率，基于业务日志
   ApiSlowInfoResp SystemDailyApiSlowInfo(1: DailyApiSlowInfoReq req) (
       api.get="/api/v1/report/slow/list"
   )
   // 探针错误率，基于探针日志
   ApiProbeResp SystemDailyApiCost(1: ApiProbeResp req) (
       api.get="/api/v1/report/probe/list"
   )

   // 用于提供缓存数据库接口
   DbRefreshResp SystemDbTidy(1: DbTidyReq req) (
       api.post="/api/v1/report/db/tidy"
   )
   DbRefreshResp SystemDbRefresh(1: DbRefreshReq req) (
       api.post="/api/v1/report/db/refresh"
   )
   WriteProbeResp WriteProbeLogs(1: WriteProbeReq req) (
       api.post="/api/v1/report/db/write_probe"
   )
   RealDataResp SystemRealData(1: RealDataReq req) (
       api.post="/api/v1/report/db/realdata"
   )
}