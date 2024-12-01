package http

import (
	"context"
	"empyrean_lens/conf"
	"reflect"
	"testing"
)

func Test_shenceDal_GetUploadInfoByTime(t *testing.T) {
	type args struct {
		ctx     context.Context
		dateStr string
	}
	tests := []struct {
		name    string
		args    args
		want    []UploadLogInfo
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test",
			args: args{
				ctx:     context.Background(),
				dateStr: "2024-12-01",
			},
			want:    nil,
			wantErr: false,
		},
	}
	conf.InitConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &shenceDal{}
			got, err := s.GetUploadInfoByTime(tt.args.ctx, tt.args.dateStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetUploadInfoByTime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetUploadInfoByTime() got = %v, want %v", got, tt.want)
			}
		})
	}
}
