// 模板消息接口.
package templateWX

// 生成模板消息
func GenerateTemplateMessage(ToUser string, TemplateId string, content string) TemplateMessage2 {
	// 生成 data 部分
	data := map[string]DataItem{
		"character_string18": {
			Value: "100",
			//Color: "#000000", // 设置颜色为黑色
		},
		"phrase2": {
			Value: content,
			//Color: "#000000", // 设置颜色为黑色
		},
	}

	// 生成TemplateMessage2结构体
	message := TemplateMessage2{
		ToUser:     ToUser,
		TemplateId: TemplateId,
		URL:        "", // 示例URL
		Data:       data,
	}

	return message
}
