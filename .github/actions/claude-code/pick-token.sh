#!/usr/bin/env bash
# 从 token pool（多行 `token|weight` / 行首 `#` 注释 / 空行忽略）按 weight 加权随机
# 选一个 OAuth token，支持通过 EXCLUDED_FILE 排除上一轮 401 的 token，触发
# 重试时换一个未使用的。
#
# 输入 env：
#   TOKEN_POOL_INPUT       多行 token pool（可空）
#   OAUTH_TOKEN_FALLBACK   pool 为空时的单 token 回落（可空）
#   EXCLUDED_FILE          排除清单文件路径（每行一个 token，文件不存在视作空集）
#
# 输出：
#   stdout：选中的 token（不带换行尾）
#   stderr：日志
#
# 退出码：
#   0  成功选出
#   1  pool 与 fallback 同时为空
#   2  所有候选都已被排除（重试用尽）

set -euo pipefail

declare -a TOKENS=()
declare -a WEIGHTS=()

if [ -n "${TOKEN_POOL_INPUT:-}" ]; then
  while IFS= read -r line; do
    # trim 前后空白
    line="${line#"${line%%[![:space:]]*}"}"
    line="${line%"${line##*[![:space:]]}"}"
    [ -z "$line" ] && continue
    case "$line" in \#*) continue ;; esac

    if [[ "$line" == *"|"* ]]; then
      tok="${line%%|*}"
      wt="${line##*|}"
    else
      tok="$line"
      wt="1"
    fi

    if ! [[ "$wt" =~ ^[1-9][0-9]*$ ]]; then
      echo "::warning::invalid weight '$wt', defaulting to 1" >&2
      wt=1
    fi

    TOKENS+=("$tok")
    WEIGHTS+=("$wt")
  done <<< "$TOKEN_POOL_INPUT"
fi

if [ "${#TOKENS[@]}" -eq 0 ]; then
  if [ -z "${OAUTH_TOKEN_FALLBACK:-}" ]; then
    echo "::error::neither token_pool nor oauth_token provided" >&2
    exit 1
  fi
  TOKENS=("$OAUTH_TOKEN_FALLBACK")
  WEIGHTS=(1)
fi

# 读排除清单。文件不存在 / 为空时视作没有排除。
declare -A EXCLUDED_SET=()
if [ -n "${EXCLUDED_FILE:-}" ] && [ -f "${EXCLUDED_FILE}" ]; then
  while IFS= read -r ex || [ -n "$ex" ]; do
    ex="${ex#"${ex%%[![:space:]]*}"}"
    ex="${ex%"${ex##*[![:space:]]}"}"
    [ -z "$ex" ] && continue
    EXCLUDED_SET["$ex"]=1
  done < "$EXCLUDED_FILE"
fi

declare -a CAND_TOKENS=()
declare -a CAND_WEIGHTS=()
for i in "${!TOKENS[@]}"; do
  tok="${TOKENS[$i]}"
  if [ -z "${EXCLUDED_SET[$tok]:-}" ]; then
    CAND_TOKENS+=("$tok")
    CAND_WEIGHTS+=("${WEIGHTS[$i]}")
  fi
done

if [ "${#CAND_TOKENS[@]}" -eq 0 ]; then
  echo "::warning::all ${#TOKENS[@]} tokens already excluded; no candidate left" >&2
  exit 2
fi

total=0
for w in "${CAND_WEIGHTS[@]}"; do total=$((total + w)); done

r=$((RANDOM % total))
acc=0
picked_idx=0
for i in "${!CAND_TOKENS[@]}"; do
  acc=$((acc + CAND_WEIGHTS[i]))
  if [ "$r" -lt "$acc" ]; then
    picked_idx=$i
    break
  fi
done

echo "picked candidate idx=$picked_idx weight=${CAND_WEIGHTS[$picked_idx]} candidates=${#CAND_TOKENS[@]} pool_size=${#TOKENS[@]} excluded=${#EXCLUDED_SET[@]}" >&2

# 不带尾换行
printf '%s' "${CAND_TOKENS[$picked_idx]}"
