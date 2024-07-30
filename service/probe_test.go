package service

import (
	"fmt"
	"testing"
)

func TestProbeTimespanFailRate(t *testing.T) {
	d := map[string]int{}
	d["1"] += 1
	d["1"] += 1
	d["2"] += 1
	fmt.Println(d)
	fmt.Println(d["1"], d["3"])
}
