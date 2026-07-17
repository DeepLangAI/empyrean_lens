#!/usr/bin/env bash
# 判定一次 Claude Code attempt 的结果是否需要换 token 重试。
#
# 输入 env：
#   OUTCOME             base action step 的 outcome（success / failure）
#   EXEC_FILE           base action 输出的 execution_file（可能空 / 不存在）
#   PICKED_TOKEN        本轮使用的 token（仅在 401 时追加到 EXCLUDED_FILE）
#   EXCLUDED_FILE       排除清单文件路径（追加写）
#   ATTEMPT_NUM         本轮 attempt 编号（1 开始）
#   MAX_ATTEMPTS        总 attempt 上限
#   FINAL_EXECUTION_FILE / FINAL_SESSION_ID / FINAL_CONCLUSION：
#                      base action 上一步透出的同名 outputs（可能空）
#
# 写入 $GITHUB_OUTPUT：
#   retry                true=需要再 attempt；false=不再 attempt
#   final_execution_file 只在 retry=false 时填，作为整个 composite action 的 output
#   final_session_id     同上
#   final_conclusion     同上
#   final_outcome        同上（success / failure；failure 时 composite 末尾会 exit 1）
#   reason               人类可读的判定原因（日志用）

set -euo pipefail

OUTCOME="${OUTCOME:-}"
EXEC_FILE="${EXEC_FILE:-}"
PICKED_TOKEN="${PICKED_TOKEN:-}"
EXCLUDED_FILE="${EXCLUDED_FILE:-}"
ATTEMPT_NUM="${ATTEMPT_NUM:-1}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-3}"

# 是否 401 / Invalid bearer token / authentication_failed 三类可重试失败：
# 401 通常是 token 过期 / 被回收，换一个就能恢复。
is_auth_failure() {
  local f="$1"
  [ -z "$f" ] && return 1
  [ ! -f "$f" ] && return 1
  if grep -qE '"api_error_status":[[:space:]]*401' "$f"; then return 0; fi
  if grep -qE '"error":[[:space:]]*"authentication_failed"' "$f"; then return 0; fi
  if grep -qE 'Invalid bearer token' "$f"; then return 0; fi
  return 1
}

write_outputs() {
  local retry="$1"
  local reason="$2"
  # final_* 始终写入，无论 retry / 不 retry。composite outputs 用 "从后往前 fallback"
  # 取最后一次实际跑过 base-action 的 attempt 的 outputs。这样即便后续 attempt 被跳过
  # （pool 用尽 / step skipped），上游也能拿到当前 attempt 的 execution_file 等产物。
  {
    echo "retry=$retry"
    echo "final_execution_file=${FINAL_EXECUTION_FILE:-}"
    echo "final_session_id=${FINAL_SESSION_ID:-}"
    echo "final_conclusion=${FINAL_CONCLUSION:-}"
    echo "final_outcome=$OUTCOME"
    echo "reason=$reason"
  } >> "$GITHUB_OUTPUT"
  echo "attempt $ATTEMPT_NUM: retry=$retry ($reason, outcome=$OUTCOME)" >&2
}

# 成功：不重试
if [ "$OUTCOME" = "success" ]; then
  write_outputs false "success"
  exit 0
fi

# 失败但不是 401 类：不重试，让 composite 末尾 exit 1
if ! is_auth_failure "$EXEC_FILE"; then
  write_outputs false "non-auth failure (no 401 signature in execution_file)"
  exit 0
fi

# 401 类失败：追加 token 到排除清单
if [ -n "$PICKED_TOKEN" ] && [ -n "$EXCLUDED_FILE" ]; then
  printf '%s\n' "$PICKED_TOKEN" >> "$EXCLUDED_FILE"
  echo "appended picked token to excluded list ($EXCLUDED_FILE)" >&2
fi

# 已达 attempt 上限：不再重试
if [ "$ATTEMPT_NUM" -ge "$MAX_ATTEMPTS" ]; then
  write_outputs false "401 auth failure but max_attempts ($MAX_ATTEMPTS) reached"
  exit 0
fi

write_outputs true "401 auth failure; rotating token"
