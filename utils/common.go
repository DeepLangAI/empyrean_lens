package utils

import (
	"empyrean_lens/biz/model/empyrean_lens"
	"empyrean_lens/consts"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var projPath = ""

func KeysOfMap[T comparable, V any](dict map[T]V) []T {
	keys := make([]T, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	return keys
}

func ValuesOfMap[T comparable, V any](dict map[T]V) []V {
	values := make([]V, 0, len(dict))
	for _, v := range dict {
		values = append(values, v)
	}
	return values
}

func GetApiAlias(hostName, apiName string) string {
	if apiName == "" {
		return apiName
	}
	for host, apis := range consts.NGINX_INGRESS_APIS {
		if hostName != host {
			continue
		}
		for _, api := range apis {
			if api.Api == apiName {
				return api.Alias
			}
		}
	}
	for host, apis := range consts.MODEL_NGINX_INGRESS_APIS {
		if hostName != host {
			continue
		}
		for _, api := range apis {
			if api.Api == apiName {
				return api.Alias
			}
		}
	}
	return apiName
}

func TimeSub(t time.Time) string {
	return fmt.Sprintf("%.4f s", time.Now().Sub(t).Seconds())
}

func GetProjectPath() string {
	if projPath != "" {
		return projPath
	}
	// 获取当前工作目录
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// 从当前工作目录向上遍历，寻找main.go文件
	for {
		info, err := os.Stat(filepath.Join(cwd, "main.go"))
		if err == nil && !info.IsDir() {
			// 找到main.go，返回当前目录作为项目路径
			return cwd
		} else if !os.IsNotExist(err) {
			// 其他错误
			return ""
		}

		// 如果没找到，尝试进入上一级目录
		cwd = filepath.Dir(cwd)
		if cwd == "/" || cwd == "" {
			// 如果到达根目录仍然没找到，返回错误
			return ""
		}
	}

	// 不应达到这里，但为了编译器的满意度，返回一个空字符串
	return ""
}

func AvgSimple(arr []float64, filterZero bool) float64 {
	if len(arr) == 0 {
		return 0
	}
	if len(arr) == 1 {
		return arr[0]
	}
	sum := 0.0
	zeroCnt := 0
	for _, v := range arr {
		sum += v
		if v == 0 {
			zeroCnt += 1
		}
	}
	if sum == 0 {
		return 0
	}
	if !filterZero {
		return sum / float64(len(arr))
	}
	if zeroCnt == len(arr) {
		return 0
	}
	return sum / float64(len(arr)-zeroCnt)
}

func Avg(arr []float64) float64 {
	if len(arr) == 0 {
		return 0
	}
	if len(arr) == 1 {
		return arr[0]
	}
	if len(arr) == 2 {
		return (arr[0] + arr[1]) / 2
	}
	sum := 0.0
	maxVal := -123456.0
	minVal := 123456.0
	for _, v := range arr {
		sum += v

		if v > maxVal {
			maxVal = v
		}
		if v < minVal {
			minVal = v
		}
	}
	sum = sum - maxVal - minVal
	return sum / (float64(len(arr)) - 2)
}

func Set(arr []string) []string {
	cache := map[string]int{}
	for _, v := range arr {
		cache[v] = 1
	}
	result := []string{}
	for k := range cache {
		result = append(result, k)
	}
	return result
}
func FilterEmpty(arr []string) []string {
	result := []string{}
	for _, v := range arr {
		v = strings.TrimSpace(v)
		if v != "" {
			result = append(result, v)
		}
	}
	return result
}

func Sum(arr []float64) float64 {
	sum := 0.0
	for _, v := range arr {
		sum += v
	}
	return sum
}

func StructToMap(v interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	value := reflect.ValueOf(v)
	if value.Kind() == reflect.Struct {
		for i := 0; i < value.NumField(); i++ {
			result[value.Type().Field(i).Name] = value.Field(i).Interface()
		}
	}
	return result
}

// Index returns the index of the first occurrence of v in s,
// or -1 if not present.
func Index[S ~[]E, E comparable](s S, v E) int {
	for i := range s {
		if v == s[i] {
			return i
		}
	}
	return -1
}

type Ordered interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~uintptr |
		~float32 | ~float64 |
		~string
}

func Max[T Ordered](x T, y ...T) T {
	for _, v := range y {
		if v > x {
			x = v
		}
	}
	return x
}

func GetCostFromMesage(msg string) (float64, error) {
	// cost:0.123 seconds
	cost, err := strconv.ParseFloat(findKeyValue(msg, "cost:"), 64)
	if err == nil {
		return cost, err
	}

	// cost 123 ms
	cost, err = strconv.ParseFloat(findKeyValue(msg, "cost "), 64)
	return cost / 1000.0, err
}

func findKeyValue(log, key string) string {
	keyPos := strings.Index(log, key)
	if keyPos == -1 {
		return ""
	}

	valueStart := keyPos + len(key)
	if valueLen := strings.Index(log[valueStart:], " "); valueLen != -1 {
		return strings.TrimSpace(log[valueStart : valueStart+valueLen])
	}

	return strings.TrimSpace(log[valueStart:])
}

func ExtractFcLogInfo(log string) (time.Time, string, string, float64) {
	parts := strings.Split(log, " ")

	timestampStr := parts[0]
	t, _ := time.Parse(time.RFC3339Nano, timestampStr)
	t = t.Local()
	userID := findKeyValue(log, "user_id:")
	traceID := findKeyValue(log, "trace_id:")
	cost, _ := strconv.ParseFloat(findKeyValue(log, "cost:"), 64)

	return t, userID, traceID, cost
}

func IsProbe(host string) bool {
	return host == "47.92.241.26" || host == "47.92.55.166"
}

func GetEntrySource(userID, copyFromResourceId, copyFromEntryId string) int {
	if userID == consts.EntryInfoPreUserID {
		return consts.EntryInfoEntrySourceOperationsPre
	}
	if copyFromResourceId != "" {
		return consts.EntryInfoEntrySourceSubscribe
	}
	if copyFromEntryId != "" {
		return consts.EntryInfoEntrySourceShare
	}
	return consts.EntryInfoEntrySourceUserUpload
}

func GetActionStatus(entryType empyrean_lens.EntryTypeEnum, status1, status2, status3 int) empyrean_lens.ActionStatusEnum {
	switch entryType {
	case empyrean_lens.EntryTypeEnum_WEB:
		if status1 == consts.URLSuccessStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	case empyrean_lens.EntryTypeEnum_FILE:
		if status1 == consts.PDFSuccessStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	case empyrean_lens.EntryTypeEnum_MULTI:
		if status1 == consts.MultiSuccessAnalysisStatus && status2 == consts.MultiSuccessSummaryStatus && status3 == consts.MultiSuccessMergeStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	case empyrean_lens.EntryTypeEnum_SUBSCRIBE_WEB,
		empyrean_lens.EntryTypeEnum_SUBSCRIBE_FILE,
		empyrean_lens.EntryTypeEnum_SUBSCRIBE_MULTI:
		if status1 == consts.SubscribeSuccessStatus {
			return empyrean_lens.ActionStatusEnum_SUCCESS
		}
		return empyrean_lens.ActionStatusEnum_FAIL
	}
	return empyrean_lens.ActionStatusEnum_FAIL
}

func GetDataType(multiID, copyFromResourceId string, entryType empyrean_lens.EntryTypeEnum) int {
	switch entryType {
	case empyrean_lens.EntryTypeEnum_WEB, empyrean_lens.EntryTypeEnum_FILE:
		if copyFromResourceId != "" {
			return consts.EntryInfoDataTypeSingleFromSubscribe
		}
		if multiID != "" {
			return consts.EntryInfoDataTypeMultiSingle
		}
		return consts.EntryInfoDataTypeSingle
	case empyrean_lens.EntryTypeEnum_MULTI:
		if copyFromResourceId != "" {
			return consts.EntryInfoDataTypeMultiFromSubscribe
		}
		return consts.EntryInfoDataTypeMulti
	}
	return consts.EntryInfoDataTypeSingle
}

func IsWebChannel(channel int) bool {
	return Contains([]int{10, 11, 12, 13, 14, 20, 23, 24, 30, 31}, channel)
}

// IPInfo 用于解析 IP-API 返回的 JSON 数据
type IPInfo struct {
	Country    string `json:"country"`    // 国家
	RegionName string `json:"regionName"` // 省份/州
	City       string `json:"city"`       // 城市
	Query      string `json:"query"`      // 查询的 IP
	Status     string `json:"status"`     // 状态 ("success" 或 "fail")
}

// GetIPLocation 查询IP的地理位置信息
// 输入: IP地址字符串
// 输出: 位置信息 (country-region-city) 或 "-"
func GetIPLocation(ip string) string {
	// 调用 IP-API 服务
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=zh-CN", ip)
	resp, err := http.Get(url)
	if err != nil {
		return "-" // 查询失败返回 "-"
	}
	defer resp.Body.Close()

	// 解析返回的 JSON 数据
	var info IPInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "-"
	}

	// 检查查询状态
	if info.Status != "success" {
		return "-"
	}

	// 格式化返回值 country-region-city
	return fmt.Sprintf("%s-%s-%s", info.Country, info.RegionName, info.City)
}
