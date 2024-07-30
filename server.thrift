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
    1: string start_time
    2: string end_time
}

struct ApiFailureInfoResp{
    1: i32 code
    2: string msg
    3: list<ApiFailureInfoRespData> data
}

struct ApiFailureInfoRespData{
    1: string date
    2: string api_name
    3: string host_name
    4: i32 num_err_req
    5: i32 num_total_req
    6: double err_percent
    7: i32 num_3xx
    8: i32 num_4xx
    9: i32 num_5xx
}

# Api耗时信息
struct ApiCostReq{
    1: string start_time
    2: string end_time
}

struct ApiCostResp{
    1: i32 code
    2: string msg
    3: list<ApiCostRespData> data
}

struct ApiCostRespData{
    1: string date
    2: string api_name
    3: i32 req_count
    4: double avg_cost

    5: double share_0_1
    6: double share_1_3
    7: double share_3_5
    8: double share_5_10
    9: double share_10_20
    10: double share_20_30
    11: double share_30_50
    12: double share_50_100
    13: double share_100_inf
}

# 刷新缓存数据库
struct DbRefreshReq{
    1: i32 timespan
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
}

struct WriteProbeReq{
    1: list<ProbeLog> data
}
struct WriteProbeResp{
    1: i32 code
    2: string msg
}

service Rentention{
   EmptyResp LogRender(1: EmptyReq req) (api.get="/api/log/report")
   EmptyResp OverviewRender(1: EmptyReq req) (api.get="/api/log/overview")

   //  用于提供前后端分离接口
   RealtimeScoreResp SystemRealtimeScore(1: EmptyReq req) (
       api.get="/api/v1/report/realtime"
   )
   DailyScoreResp SystemDailyScore(1: DailyScoreReq req) (
       api.get="/api/v1/report/daily/score"
   )
   ApiFailureInfoResp SystemDailyApiFailureInfo(1: DailyApiFailureInfoReq req) (
       api.get="/api/v1/report/daily/failure"
   )
   ApiCostResp SystemDailyApiCost(1: ApiCostReq req) (
       api.get="/api/v1/report/daily/cost"
   )

   DbRefreshResp SystemDbRefresh(1: DbRefreshReq req) (
       api.post="/api/v1/report/db/refresh"
   )
   WriteProbeResp WriteProbeLogs(1: WriteProbeReq req) (
       api.post="/api/v1/report/db/write_probe"
   )
}