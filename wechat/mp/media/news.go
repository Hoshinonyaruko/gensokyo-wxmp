package media

import (
	"os"

	"github.com/chanxuehong/wechat/mp/core"
)

type Article struct {
	ThumbMediaId     string `json:"thumb_media_id"`               // 必须; 图文消息缩略图的 media_id, 可以在上传多媒体文件接口中获得
	Title            string `json:"title"`                        // 必须; 图文消息的标题
	Author           string `json:"author,omitempty"`             // 可选; 图文消息的作者
	Digest           string `json:"digest,omitempty"`             // 可选; 图文消息的摘要
	Content          string `json:"content"`                      // 必须; 图文消息页面的内容, 支持HTML标签
	ContentSourceURL string `json:"content_source_url,omitempty"` // 可选; 在图文消息页面点击"阅读原文"后的页面
	ShowCoverPic     int    `json:"show_cover_pic"`               // 可选; 是否显示封面, 1为显示, 0为不显示, 默认为不显示
}

type News struct {
	Articles []Article `json:"articles,omitempty"`
}

// MediaUploadResponse 用于接收上传媒体文件后的响应
type MediaUploadResponse struct {
	core.Error
	Type      string `json:"type"`
	MediaID   string `json:"media_id"`
	CreatedAt int64  `json:"created_at"`
}

// UploadNews 创建图文消息素材, 返回的素材一般用于群发消息.
func UploadNews(clt *core.Client, news *News) (info *MediaInfo, err error) {
	const incompleteURL = "https://api.weixin.qq.com/cgi-bin/media/uploadnews?access_token="

	var result struct {
		core.Error
		MediaInfo
	}
	if err = clt.PostJSON(incompleteURL, news, &result); err != nil {
		return
	}
	if result.ErrCode != core.ErrCodeOK {
		err = &result.Error
		return
	}
	info = &result.MediaInfo
	return
}

// UploadMedia 上传媒体文件到微信服务器
func UploadMedia(clt *core.Client, accessToken, mediaType, filename string) (*MediaUploadResponse, error) {
	const baseURL = "https://api.weixin.qq.com/cgi-bin/media/upload"

	// 打开文件
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 准备表单字段
	fields := []core.MultipartFormField{
		{
			IsFile:   true,
			Name:     "media",
			FileName: filename, // 注意：这里的filename不是文件路径，而是multipart/form-data中filename的值
			Value:    file,
		},
		// 如果需要其他字段，可以在这里添加
	}

	// 构建完整URL
	incompleteURL := baseURL + "?access_token=" + accessToken + "&type=" + mediaType

	// 响应结构
	var result MediaUploadResponse

	// 发送请求
	if err := clt.PostMultipartForm(incompleteURL, fields, &result); err != nil {
		return nil, err
	}
	if result.ErrCode != core.ErrCodeOK {
		return nil, &result.Error
	}

	return &result, nil
}
