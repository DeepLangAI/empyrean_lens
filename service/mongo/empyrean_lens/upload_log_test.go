package empyrean_lens

import (
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
)

func TestSaveUploadLogByDate(t *testing.T) {
	conf.InitConfig()
	dal.Init()
	SaveOnceUploadLogByDate()
}
