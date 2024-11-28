package empyrean_lens

import (
	"empyrean_lens/conf"
	"empyrean_lens/dal"
	"testing"
)

func TestSaveGenerateErrlogByDate(t *testing.T) {
	conf.InitConfig()
	dal.Init()
	SaveOnceGenerateErrlogByDate()
}
