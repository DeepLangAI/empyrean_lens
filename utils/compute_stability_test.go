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
	for date, value := range result {
		fmt.Printf("Date: %s, Score: %d, Day-Over-Day: %.2f, Week-Over-Week: %.2f\n", date, int(value[0]), value[1], value[2])
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
