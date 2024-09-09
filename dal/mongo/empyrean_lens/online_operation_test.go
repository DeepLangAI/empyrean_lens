package empyrean_lens

import (
	"empyrean_lens/consts"
	"fmt"
	"testing"
	"time"
)

func TestDate(t *testing.T) {
	fmt.Println(time.Now())
	fmt.Println(time.Now().Local())
	tm, e := time.ParseInLocation(consts.DateHourMinSecTemplate, "2024-09-09 16:08:37", time.Local)
	if e != nil {
		t.Error(e)
	}
	fmt.Println(tm)
}
