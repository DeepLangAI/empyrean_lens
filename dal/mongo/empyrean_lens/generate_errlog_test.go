package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"reflect"
	"testing"
)

func TestGenerateErrLogDao_SaveBatch(t *testing.T) {
	type args struct {
		ctx    context.Context
		models []GenerateErrLogModel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{"test1", args{context.Background(), []GenerateErrLogModel{
			{
				Date:          "2023-07-01",
				EntryId:       "entry_id",
				FailureReason: "failure_reason",
				Ip:            "ip",
				IpRegion:      "ip_region",
				Time:          "time",
				TraceId:       "trace_id",
				UserId:        "user_id",
			},
		}}, false},
	}

	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &GenerateErrLogDao{}
			if err := self.SaveBatch(tt.args.ctx, tt.args.models); (err != nil) != tt.wantErr {
				t.Errorf("SaveBatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateErrLogDao_CountGenerateErrLogsByTimeRange(t *testing.T) {
	type args struct {
		ctx       context.Context
		startDate string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]int64
		wantErr bool
	}{
		// TODO: Add test cases.
		{"test1", args{context.Background(), "2024-11-11"}, map[string]int64{"2024-11-11": 1}, false},
	}

	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &GenerateErrLogDao{}
			got, err := self.CountGenerateErrLogsByTimeRange(tt.args.ctx, tt.args.startDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("CountGenerateErrLogsByTimeRange() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CountGenerateErrLogsByTimeRange() got = %v, want %v", got, tt.want)
			}
		})
	}
}
