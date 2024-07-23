package probe

import (
	"bytes"
	"errors"
	"github.com/bytedance/sonic"
	"github.com/cloudwego/hertz/pkg/common/hlog"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

import "context"

type FileUploadResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		FileId string `json:"file_id"`
	} `json:"data"`
}

func uploadFile(ctx context.Context, url, filePath string, headers map[string]string) (string, error) {
	fileName := filepath.Base(filePath)
	fields := map[string]string{
		"file_name":    fileName,
		"channel_type": "71",
	}

	// 创建一个新的multipart表单
	reqBody := &bytes.Buffer{}
	writer := multipart.NewWriter(reqBody)

	// 添加文件字段
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	fileWriter, err := writer.CreateFormFile("upload_file", fileName)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return "", err
	}

	// 添加其他表单字段
	for key, value := range fields {
		err = writer.WriteField(key, value)
		if err != nil {
			return "", err
		}
	}

	// 关闭multipart表单
	err = writer.Close()
	if err != nil {
		return "", err
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return "", err
	}

	// 设置请求头
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if headers != nil {
		for k, v := range headers {
			req.Header.Set(k, v)
		}
	}

	// 发送请求
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http UploadFile] get fail, err:%v, params:%v", err, fields)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		hlog.CtxErrorf(ctx, "[Http UploadFile] status code not 200,  code:%d, params:%v", resp.StatusCode, fields)
		return "", errors.New("status code not 200")
	}

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Get] read respBody fail, err:%v, params:%v", err, fields)
		return "", err
	}
	respData := FileUploadResp{}
	// 解析响应
	err = sonic.Unmarshal(respBody, &respData)
	if err != nil {
		hlog.CtxErrorf(ctx, "[Http Get] json unmarshal fail, err:%v, params:%v", err, fields)
		return "", err
	}
	if respData.Data.FileId != "" {
		return respData.Data.FileId, nil
	}
	return "", errors.New("file_id is empty")
}

func apiFailed(err error, resp *RestResp) bool {
	if err != nil || resp.Code != 0 {
		return true
	}
	return false
}
