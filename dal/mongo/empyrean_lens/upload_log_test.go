package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"reflect"
	"testing"
	"time"
)

func TestUploadLogModelDao_Save(t *testing.T) {
	type args struct {
		ctx   context.Context
		model UploadLogModel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test",
			args: args{
				ctx: context.Background(),
				model: UploadLogModel{
					UserId:        "ly",
					Date:          "2024-08-15",
					FailureReason: "test",
					FileName:      "test",
					FileSize:      1024,
					FileType:      "test",
					Ip:            "test",
					Time:          "2024-08-15",
					TraceId:       "test",
					UpdateTime:    time.Now(),
				},
			},
		},
	}
	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &UploadLogModelDao{}
			if err := self.Save(tt.args.ctx, tt.args.model); (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadLogModelDao_SaveBatch(t *testing.T) {
	type args struct {
		ctx    context.Context
		models []UploadLogModel
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test",
			args: args{
				ctx: context.Background(),
				models: []UploadLogModel{
					{
						UserId:        "ly",
						Date:          "2024-08-15",
						FailureReason: "test",
						FileName:      "test",
						FileSize:      1024,
						FileType:      "test",
						Ip:            "test",
						Time:          "2024-08-15",
						TraceId:       "test",
						UpdateTime:    time.Now(),
					},
				},
			},
		},
	}

	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &UploadLogModelDao{}
			if err := self.SaveBatch(tt.args.ctx, tt.args.models); (err != nil) != tt.wantErr {
				t.Errorf("SaveBatch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadLogModelDao_CountLogsByStartDate(t *testing.T) {
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
		{
			name: "test",
			args: args{
				ctx:       context.Background(),
				startDate: "2024-11-15",
			},
			want:    nil,
			wantErr: false,
		},
	}

	ctx := context.Background()
	conf.InitConfig()
	Init(ctx)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			self := &UploadLogModelDao{}
			got, err := self.CountLogsByTimeRange(tt.args.ctx, tt.args.startDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("CountLogsByStartDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CountLogsByStartDate() got = %v, want %v", got, tt.want)
			}
		})
	}
}
