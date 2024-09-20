package empyrean_lens

import (
	"fmt"
	"testing"
)

func TestScoreBackupDao_convertToBsonM(t *testing.T) {
	model := ScoreBackupModel{}
	model.Score = 123

	dao := NewScoreBackupDao()
	m, _ := dao.convertToBsonM(model)
	fmt.Printf("%+v", m)
}
