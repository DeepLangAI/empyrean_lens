package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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
func TestGetIPLocation1(t *testing.T) {
	ip := "223.87.43.219"
	location := GetIPLocation(ip)
	assert.True(t, location != "-")
	fmt.Println(location)
}
