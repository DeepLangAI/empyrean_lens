package utils

import (
	"fmt"
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