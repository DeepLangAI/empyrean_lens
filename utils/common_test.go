package utils

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAvgSimple(t *testing.T) {
	avg := AvgSimple([]float64{
		15.6,
		12.3,
		0, 0, 0, 0,
	}, true)
	fmt.Println(avg)
}

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestStructToMap(t *testing.T) {
	p := Person{
		Name: "John",
		Age:  30,
	}
	toMap := StructToMap(p)
	fmt.Println(toMap)
}

func TestExtractLogInfo(t *testing.T) {
	ti, tId, uId, c := ExtractLogInfo("2024-10-15T08:09:02.058Z E879A3BDC57912040F754B902065B073 [INFO] 爬取成功 url_id:670e231808f90e2096d279e2 user_id:65bc87cbbfca0462fba60891 url:https://www.163.com/dy/article/JEF4316L051285EO.html trace_id:veyKcXYVsaz74nWPqPfB3 channel_type:20 error_msg:爬取成功 cost:1.222 seconds")
	t.Log(ti)
	t.Log(tId)
	t.Log(uId)
	t.Log(c)
}

func TestGetIPLocation(t *testing.T) {
	type args struct {
		ip string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
		{"test1", args{"0.114.114.114"}, "中国 ips.cn"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, GetIPLocation(tt.args.ip), "GetIPLocation(%v)", tt.args.ip)
		})
	}
}
