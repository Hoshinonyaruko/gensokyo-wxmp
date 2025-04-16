package base

import (
	"encoding/json"

	"github.com/chanxuehong/wechat/mp/core"
)

// MsgSecCheck 执行消息安全检测
func MsgSecCheck(clt *core.Client, openID, content string, scene int, title, nickname, signature string) (shortURL string, err error) {
	const url = "https://api.weixin.qq.com/wxa/msg_sec_check?access_token="

	// 构造请求参数
	var request = struct {
		OpenID    string `json:"openid"`              // 用户的 OpenID
		Scene     int    `json:"scene"`               // 场景枚举值
		Content   string `json:"content"`             // 需要检测的文本内容
		Version   int    `json:"version"`             // 接口版本号
		Title     string `json:"title,omitempty"`     // 文本标题
		Nickname  string `json:"nickname,omitempty"`  // 用户昵称
		Signature string `json:"signature,omitempty"` // 个性签名，仅在资料类场景有效
	}{
		OpenID:    openID,
		Scene:     scene,
		Content:   content,
		Version:   2,
		Title:     title,
		Nickname:  nickname,
		Signature: signature,
	}

	// 定义一个结构体来接收返回的数据
	var result struct {
		core.Error
		Result struct {
			Suggest string `json:"suggest"` // 综合建议
			Label   int    `json:"label"`   // 综合标签
		} `json:"result"`
		Detail []struct {
			Strategy string `json:"strategy"` // 策略类型
			ErrCode  int    `json:"errcode"`  // 错误码
			Suggest  string `json:"suggest"`  // 建议
			Label    int    `json:"label"`    // 标签
			Prob     int    `json:"prob"`     // 置信度
			Keyword  string `json:"keyword"`  // 命中的自定义关键词
		} `json:"detail"`
		TraceID string `json:"trace_id"` // 唯一请求标识
	}

	// 将请求数据编码为 JSON
	reqBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// 发送 POST 请求并获取返回的结果
	err = clt.PostJSON(url, reqBody, &result)
	if err != nil {
		return "", err
	}

	// 检查接口返回的错误码
	if result.ErrCode != core.ErrCodeOK {
		err = &result.Error
		return "", err
	}

	// 返回结果
	shortURL = result.Result.Suggest // 示例：返回综合建议
	return
}
