package empyrean_lens

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
)

func TestSaveGenerateErrlogByDate(t *testing.T) {
	type args struct {
		ctx     context.Context
		dateStr string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{"test1", args{context.Background(), "2024-11-26"}, false},
	}

	conf.InitConfig()
	dal.Init()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := SaveGenerateErrlogByDate(tt.args.ctx, tt.args.dateStr); (err != nil) != tt.wantErr {
				t.Errorf("SaveGenerateErrlogByDate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
