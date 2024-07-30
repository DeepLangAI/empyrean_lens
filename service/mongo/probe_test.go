package mongo

import (
	"context"
	"empyrean_lens/conf"
	"empyrean_lens/dal"
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

func TestProbeReport(t *testing.T) {
	ctx := context.Background()
	conf.InitConfig()
	dal.Init()
	report, err := ProbeReport(ctx)
	if err != nil {
		t.Errorf("ProbeReport err: %v", err)
	}
	for _, r := range report {
		fmt.Println(r)
	}
}
