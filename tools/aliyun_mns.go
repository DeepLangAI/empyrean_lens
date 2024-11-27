package tools

import (
	"empyrean_lens/conf"
	"empyrean_lens/utils"
	"encoding/base64"

	ali_mns "github.com/aliyun/aliyun-mns-go-sdk"
)

func BatchSendMsg(queueName string, msgList []interface{}) error {
	// 链接mns queue
	client := ali_mns.NewAliMNSClientWithConfig(ali_mns.AliMNSClientConfig{
		EndPoint:        conf.GetConfig().MnsConfig.Endpoint,
		AccessKeyId:     conf.GetConfig().MnsConfig.AccessKeyID,
		AccessKeySecret: conf.GetConfig().MnsConfig.AccessKeySecret,
	})
	queue := ali_mns.NewMNSQueue(queueName, client)
	// 创建消息
	mnsMsgList := []ali_mns.MessageSendRequest{}
	for _, msg := range msgList {
		mnsMsgList = append(mnsMsgList, ali_mns.MessageSendRequest{
			MessageBody: base64.StdEncoding.EncodeToString([]byte(utils.JSONMarshal(msg))),
			Priority:    1,
		})
	}
	// 10条为一批
	for i := 0; i < len(mnsMsgList); i += 10 {
		var newMsgList []ali_mns.MessageSendRequest
		if i+10 <= len(mnsMsgList) {
			newMsgList = mnsMsgList[i : i+10]
		} else {
			newMsgList = mnsMsgList[i:]
		}
		_, err := queue.BatchSendMessage(newMsgList...)
		if err != nil {
			return err
		}
	}
	return nil
}
