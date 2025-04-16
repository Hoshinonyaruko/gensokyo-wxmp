package chatbot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/hoshinonyaruko/gensokyo-wxmp/config"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// API 返回结果结构体
type SensitiveContentResult struct {
	Error  interface{} `json:"error"`
	Result [][]float64 `json:"result"`
}

// 构造 JWT 的数据结构
type SensitiveContentRequest struct {
	Uid  string `json:"uid"`
	Data struct {
		Q     string `json:"q"`
		Model string `json:"model,omitempty"`
	} `json:"data"`
}

// 使用 JWT 加密参数
func signData(secretKey string, reqData SensitiveContentRequest) (string, error) {
	// 创建 JWT Token 对象
	claims := jwt.MapClaims{
		"uid":  reqData.Uid,
		"data": reqData.Data,
	}

	// 创建 JWT Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// 使用 EncodingAESKey 对 token 进行签名
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", errors.Wrap(err, "failed to sign JWT")
	}
	return signedToken, nil
}

// 已弃用
// 调用敏感内容接口
func CheckSensitiveContent(text string, userid string) (*SensitiveContentResult, error) {
	// 获取配置
	//appID := config.GetChatBotAppid()
	token := config.GetChatBotToken()
	encodingAESKey := config.GetChatBotAesKey()

	// 生成请求数据
	requestData := SensitiveContentRequest{
		Uid: userid, // 随机生成一个用户ID，可以根据实际需求替换
	}
	requestData.Data.Q = text
	requestData.Data.Model = "cnn" // 使用默认的 CNNN 模型

	// 使用 JWT 对数据进行加密
	signedData, err := signData(encodingAESKey, requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %v", err)
	}

	// 构造 API 请求
	apiURL := fmt.Sprintf("https://chatbot.weixin.qq.com/openapi/nlp/sensitive/%s", token)
	reqBody := fmt.Sprintf("query=%s", signedData)
	req, err := http.NewRequest("POST", apiURL, bytes.NewBufferString(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	defer resp.Body.Close()
	// 读取并打印响应体内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	// 输出响应体内容，方便调试
	fmt.Printf("Response body: %s\n", body)

	// 解析返回结果
	var result SensitiveContentResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	return &result, nil
}
