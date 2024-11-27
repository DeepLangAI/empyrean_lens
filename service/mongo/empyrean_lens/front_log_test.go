package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"empyrean_lens/dal/mongo/empyrean_lens"
	"reflect"
	"testing"
	"time"
)

func TestGetGenerateErrLogCountByDay(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]int64
		wantErr bool
	}{
		// TODO: Add test cases.
		{"test1", args{context.Background()}, map[string]int64{"2024-11-15": 1}, false},
	}
	conf.InitConfig()
	dal.Init()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGenerateErrLogCountByDay(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGenerateErrLogCountByDay() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGenerateErrLogCountByDay() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetGenerateErrLogInofs(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		args    args
		want    []empyrean_lens.GenerateErrLogModel
		wantErr bool
	}{
		// TODO: Add test cases.
		{"test1", args{context.Background()}, []empyrean_lens.GenerateErrLogModel{{Date: "2024-11-15", EntryId: "test", FailureReason: "test", Ip: "test", IpRegion: "test", Time: "test", TraceId: "test", UpdateTime: time.Now(), UserId: "test"}}, false},
	}
	conf.InitConfig()
	dal.Init()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetGenerateErrLogInofs(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetGenerateErrLogInofs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetGenerateErrLogInofs() got = %v, want %v", got, tt.want)
			}
		})
	}
}
