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
