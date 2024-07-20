package utils

import (
	"empyrean_lens/consts"
	"fmt"
	"testing"
)

func TestComputeRevent(t *testing.T) {
	data := map[string]int{
		"2024-07-01": 100,
		"2024-07-02": 110,
		"2024-07-08": 130,
		"2024-07-07": 120,
		"2024-07-05": 115,
		"2024-07-06": 105,
		"2024-07-04": 100,
		"2024-07-03": 90,
		"2024-07-09": 140,
		"2024-07-10": 150,
		"2024-07-11": 160,
		"2024-07-12": 170,
	}

	result := ComputeRevent(data)
	for _, value := range result {
		fmt.Printf("Date: %s, Score: %.0f, Day-Over-Day: %.2f, Week-Over-Week: %.2f\n", value.Date, value.Score, value.DayOverDay, value.WeekOverWeek)
	}

}

func TestComputeStability(t *testing.T) {
	data := map[string]float64{
		consts.ERROR_RATE_PARAMETER:       0.2,
		consts.SLOW_SEARCH_RATE_PARAMETER: 0.4,
		consts.PROBE_ERROR_RATE_PARAMETER: 0.3,
	}
	fmt.Println(ComputeStability(data))
}

func TestComputeStablityScore(t *testing.T) {
	f := SystemStablityFactor{
		ApiFailRate:   0.04,
		SlowQueryRate: 0,
		ProbeFailRate: 0,
	}
	score := ComputeStablityScore(f)
	fmt.Println(score)
}
