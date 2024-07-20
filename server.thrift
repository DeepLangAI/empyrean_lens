namespace go empyrean_lens

struct BaseResp {
    1: i32 code
    2: string msg
}

struct RenderReq{}
struct RenderResp{}


service Rentention{
   RenderResp LogRender(1: RenderReq req) (api.get="/api/log/report")
   RenderResp OverviewRender(1: RenderReq req) (api.get="/api/log/overview")
}