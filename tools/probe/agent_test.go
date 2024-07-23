package probe

import (
	"context"
	"empyrean_lens/consts"
	"empyrean_lens/utils"
	"path/filepath"
	"testing"
)

func TestAgent_Login(t *testing.T) {
	ctx := context.Background()
	agent := NewAgent()
	agent.Login(ctx)
	agent.Logout(ctx)
}

func TestUploadFile(t *testing.T) {
	ctx := context.Background()
	filePath := "assets/1810.04805v2.pdf"
	filePath = filepath.Join(utils.GetProjectPath(), filePath)
	agent := NewAgent()
	agent.Login(ctx)
	fileId, err := uploadFile(ctx, consts.LINGO_HOST+"/api/plugin/file/add", filePath, agent.Headers)
	if err != nil {
		t.Errorf("UploadFile failed: %v", err)
	} else {
		t.Logf("UploadFile success, fileId: %s", fileId)
	}
}
