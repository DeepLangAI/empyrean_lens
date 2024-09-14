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
