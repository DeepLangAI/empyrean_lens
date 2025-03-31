package empyrean_lens

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var TableNameApiStatsDaily = "api_stats_daily"

// ApiType 定义API类型枚举
type ApiType string

const (
	ApiTypeUploadPDF         ApiType = "upload_pdf"
	ApiTypeUploadPDFParsing  ApiType = "upload_pdf_parsing"
	ApiTypeUploadURL         ApiType = "upload_url"
	ApiTypeEduInput          ApiType = "edu_input"
	ApiTypeEduOutput         ApiType = "edu_output"
	ApiTypeEduTree           ApiType = "edu_tree"
	ApiTypeSingleDocOutline  ApiType = "singledoc_outline"
	ApiTypePDFParsing        ApiType = "pdf_parsing"
	ApiTypeTextParse         ApiType = "text_parse"
	ApiTypeWCD               ApiType = "wcd"
	ApiTypeCrawler           ApiType = "crawler"
	ApiTypeCrawlerImg        ApiType = "crawler_img"
	ApiTypeNovelFormGenerate ApiType = "novel_form_generate"
	ApiTypeNovelFormGet      ApiType = "novel_form_get"
	ApiTypeMasterThemeURL    ApiType = "theme:master_theme_url"
	ApiTypeSingleDocURL      ApiType = "theme:single_doc_url"
	ApiTypeKeyOpinion        ApiType = "key-opinion"
)

// ApiStats 接口统计结构
type ApiStats struct {
	ApiType string  `bson:"api_type" json:"api_type"`
	Total   int     `bson:"total" json:"total"`
	Success int     `bson:"success" json:"success"`
	Rate    float64 `bson:"rate" json:"rate"`
}

// ApiStatsDailyModel 每日统计模型
type ApiStatsDailyModel struct {
	Id          primitive.ObjectID `bson:"_id"`
	Date        string             `bson:"date"` // 统计日期 YYYY-MM-DD
	ApiStats    []ApiStats         `bson:"api_stats"`
	CreatedTime time.Time          `bson:"created_time"`
}

type ApiStatsDailyDao struct{}

var apiStatsDailyDao *ApiStatsDailyDao
var apiStatsDailyDaoInitOnce sync.Once

func NewApiStatsDailyDao() *ApiStatsDailyDao {
	apiStatsDailyDaoInitOnce.Do(func() {
		apiStatsDailyDao = &ApiStatsDailyDao{}
	})
	return apiStatsDailyDao
}

// UpsertDailyStats 更新或插入每日统计数据
func (self *ApiStatsDailyDao) UpsertDailyStats(ctx context.Context, stats []ApiStats, date time.Time) error {
	// 统一使用中国时区
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	date = date.In(loc)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	dateStr := date.Format("2006-01-02")

	filter := bson.M{
		"date": dateStr,
	}

	update := bson.M{
		"$set": bson.M{
			"api_stats":    stats,
			"date":         dateStr,
			"created_time": startOfDay,
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := probeDatabase.Collection(TableNameApiStatsDaily).UpdateOne(ctx, filter, update, opts)
	return err
}

// GetDailyStats 获取指定日期范围的统计数据
func (self *ApiStatsDailyDao) GetDailyStats(ctx context.Context, startDate, endDate time.Time) ([]ApiStatsDailyModel, error) {
	// 统一使用中国时区
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	startDate = startDate.In(loc)
	endDate = endDate.In(loc)

	filter := bson.M{}

	// 如果提供了日期范围，则添加日期过滤条件
	if !startDate.IsZero() && !endDate.IsZero() {
		startDateStr := startDate.Format("2006-01-02")
		endDateStr := endDate.Format("2006-01-02")
		filter["date"] = bson.M{
			"$gte": startDateStr,
			"$lte": endDateStr,
		}
	}

	cursor, err := probeDatabase.Collection(TableNameApiStatsDaily).Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []ApiStatsDailyModel
	if err = cursor.All(ctx, &results); err != nil {
		return nil, err
	}

	return results, nil
}

// GetLatestStats 获取最新的统计数据
func (self *ApiStatsDailyDao) GetLatestStats(ctx context.Context) (*ApiStatsDailyModel, error) {
	opts := options.FindOne().SetSort(bson.D{{"date", -1}})
	var result ApiStatsDailyModel
	err := probeDatabase.Collection(TableNameApiStatsDaily).FindOne(ctx, bson.M{}, opts).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // 如果没有记录，返回 nil
		}
		return nil, err
	}
	return &result, nil
}

// ExistsByDate 检查指定日期是否存在记录
func (self *ApiStatsDailyDao) ExistsByDate(ctx context.Context, date string) (bool, error) {
	count, err := probeDatabase.Collection(TableNameApiStatsDaily).CountDocuments(ctx, bson.M{"date": date})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
