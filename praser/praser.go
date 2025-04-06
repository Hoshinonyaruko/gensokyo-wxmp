package praser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hoshinonyaruko/gensokyo-wxmp/mylog"
)

func ParseMessageContent(message interface{}, removeMDPic bool) string {
	messageText := ""

	switch message := message.(type) {
	case string:
		mylog.Printf("params.message is a string\n")
		messageText = message
	case []interface{}:
		//多个映射组成的切片
		mylog.Printf("params.message is a slice (segment_type_koishi)\n")
		for _, segment := range message {
			segmentMap, ok := segment.(map[string]interface{})
			if !ok {
				continue
			}

			segmentType, ok := segmentMap["type"].(string)
			if !ok {
				continue
			}

			segmentContent := ""
			switch segmentType {
			case "text":
				segmentContent, _ = segmentMap["data"].(map[string]interface{})["text"].(string)
			case "image":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:image,file=" + fileContent + "]"
			case "voice":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:record,file=" + fileContent + "]"
			case "record":
				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
				segmentContent = "[CQ:record,file=" + fileContent + "]"
			case "at":
				qqNumber, _ := segmentMap["data"].(map[string]interface{})["qq"].(string)
				segmentContent = "[CQ:at,qq=" + qqNumber + "]"
			case "markdown":
				mdContent, ok := segmentMap["data"].(map[string]interface{})["data"]
				if ok {
					if mdContentMap, isMap := mdContent.(map[string]interface{}); isMap {
						// mdContent是map[string]interface{}，按map处理
						mdContentBytes, err := json.Marshal(mdContentMap)
						if err != nil {
							mylog.Printf("Error marshaling mdContentMap to JSON:%v", err)
						}
						encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
						segmentContent = "[CQ:markdown,data=" + encoded + "]"
					} else if mdContentStr, isString := mdContent.(string); isString {
						// mdContent是string
						if strings.HasPrefix(mdContentStr, "base64://") {
							// 如果以base64://开头，直接使用
							segmentContent = "[CQ:markdown,data=" + mdContentStr + "]"
						} else {
							// 处理实体化后的JSON文本
							mdContentStr = strings.ReplaceAll(mdContentStr, "&amp;", "&")
							mdContentStr = strings.ReplaceAll(mdContentStr, "&#91;", "[")
							mdContentStr = strings.ReplaceAll(mdContentStr, "&#93;", "]")
							mdContentStr = strings.ReplaceAll(mdContentStr, "&#44;", ",")

							// 将处理过的字符串视为JSON对象，进行序列化和编码
							var jsonMap map[string]interface{}
							if err := json.Unmarshal([]byte(mdContentStr), &jsonMap); err != nil {
								mylog.Printf("Error unmarshaling string to JSON:%v", err)
							}
							mdContentBytes, err := json.Marshal(jsonMap)
							if err != nil {
								mylog.Printf("Error marshaling jsonMap to JSON:%v", err)
							}
							encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
							segmentContent = "[CQ:markdown,data=" + encoded + "]"
						}
					}
				} else {
					mylog.Printf("Error marshaling markdown segment to interface,contain type but data is nil.")
				}
			}

			messageText += segmentContent
		}
	case map[string]interface{}:
		//单个映射
		mylog.Printf("params.message is a map (segment_type_trss)\n")
		messageType, _ := message["type"].(string)
		switch messageType {
		case "text":
			messageText, _ = message["data"].(map[string]interface{})["text"].(string)
		case "image":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:image,file=" + fileContent + "]"
		case "voice":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:record,file=" + fileContent + "]"
		case "record":
			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
			messageText = "[CQ:record,file=" + fileContent + "]"
		case "at":
			qqNumber, _ := message["data"].(map[string]interface{})["qq"].(string)
			messageText = "[CQ:at,qq=" + qqNumber + "]"
		case "markdown":
			mdContent, ok := message["data"].(map[string]interface{})["data"]
			if ok {
				if mdContentMap, isMap := mdContent.(map[string]interface{}); isMap {
					// mdContent是map[string]interface{}，按map处理
					mdContentBytes, err := json.Marshal(mdContentMap)
					if err != nil {
						mylog.Printf("Error marshaling mdContentMap to JSON:%v", err)
					}
					fmt.Print("开始处理md:" + string(mdContentBytes))
					messageText, err = parseMDData(mdContentBytes)
					if err != nil {
						fmt.Print(err)
					}
					// 是否移除md图片(搞不懂wx怎么发图文,在只能把图文信息的文字抽出来作为历史信息的时候,发历史信息就要移除图片.)
					if !removeMDPic {
						messageText = ConvertMarkdownToCQImage(messageText)
					} else {
						messageText = RemoveMarkdownImages(messageText)
					}
					fmt.Print(messageText)
				} else if mdContentStr, isString := mdContent.(string); isString {
					// mdContent是string
					if strings.HasPrefix(mdContentStr, "base64://") {
						// 如果以base64://开头，直接使用
						messageText = "[CQ:markdown,data=" + mdContentStr + "]"
					} else {
						// 处理实体化后的JSON文本
						mdContentStr = strings.ReplaceAll(mdContentStr, "&amp;", "&")
						mdContentStr = strings.ReplaceAll(mdContentStr, "&#91;", "[")
						mdContentStr = strings.ReplaceAll(mdContentStr, "&#93;", "]")
						mdContentStr = strings.ReplaceAll(mdContentStr, "&#44;", ",")

						// 将处理过的字符串视为JSON对象，进行序列化和编码
						var jsonMap map[string]interface{}
						if err := json.Unmarshal([]byte(mdContentStr), &jsonMap); err != nil {
							mylog.Printf("Error unmarshaling string to JSON:%v", err)
						}
						mdContentBytes, err := json.Marshal(jsonMap)
						if err != nil {
							mylog.Printf("Error marshaling jsonMap to JSON:%v", err)
						}
						encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
						messageText = "[CQ:markdown,data=" + encoded + "]"
					}
				}
			} else {
				mylog.Printf("Error marshaling markdown segment to interface,contain type but data is nil.")
			}
		}
	default:
		mylog.Println("Unsupported message format: params.message field is not a string, map or slice")
	}
	return messageText
}

// 结构体定义
type Markdown struct {
	TemplateID       int
	CustomTemplateID string
	Params           []*MarkdownParams
	Content          string
}

// MarkdownParams markdown 模版参数 键值对
type MarkdownParams struct {
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type MessageKeyboard struct {
	ID      string
	Content *CustomKeyboard
}

type CustomKeyboard struct {
	Rows []*Row
}

type Row struct {
	Buttons []*Button
}

type Button struct {
	ID         string
	RenderData *RenderData
	Action     *Action
}

type RenderData struct {
	Label        string
	VisitedLabel string
	Style        int
}

type Action struct {
	Type int
	Data string
}

// 去除 <qqbot-at-user id="..." /> 格式的文本
func removeAtUserTags(content string) string {
	// 正则表达式匹配 <qqbot-at-user id="..." />
	re := regexp.MustCompile(`<qqbot-at-user id="[^"]*" />`)

	// 使用正则表达式替换匹配的标签为 ""（即去除）
	content = re.ReplaceAllString(content, "")

	return content
}

// 使用正则表达式替换Markdown中的命令模式
func parseMarkdownContent(content string) string {
	// 先去除 <qqbot-at-user id="..." /> 标签
	content = removeAtUserTags(content)

	// 正则表达式匹配 <qqbot-cmd-input ... /> 标签
	re := regexp.MustCompile(`<qqbot-cmd-input[^>]*text='([^']*)'[^>]*show='([^']*)'[^>]*/>`)

	// 使用正则表达式替换匹配的标签
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取text和show属性的值
		reText := regexp.MustCompile(`text='([^']*)'`)
		reShow := regexp.MustCompile(`show='([^']*)'`)

		// 从匹配的字符串中提取text和show的值
		text := reText.FindStringSubmatch(match)[1]
		show := reShow.FindStringSubmatch(match)[1]

		// 构造HTML链接
		return fmt.Sprintf(`<a href="weixin://bizmsgmenu?msgmenucontent=%s&msgmenuid=0">%s</a>`, text, show)
	})

	return content
}

// 使用正则表达式替换Markdown中的命令模式
func parseMarkdownContentV2(content string) string {
	// 先去除 <qqbot-at-user id="..." /> 标签
	content = removeAtUserTags(content)

	// 正则表达式匹配 <qqbot-cmd-input ... /> 标签
	re := regexp.MustCompile(`<qqbot-cmd-input[^>]*text='([^']*)'[^>]*show='([^']*)'[^>]*/>`)

	// 使用正则表达式替换匹配的标签
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取text和show属性的值
		reText := regexp.MustCompile(`text='([^']*)'`)
		//reShow := regexp.MustCompile(`show='([^']*)'`)

		// 从匹配的字符串中提取text和show的值
		text := reText.FindStringSubmatch(match)[1]
		//show := reShow.FindStringSubmatch(match)[1]

		// 构造HTML链接
		return text
	})

	return content
}

// 处理键盘内容，生成文本
func parseKeyboardContent(keyboard *CustomKeyboard) string {
	var result []string
	for _, row := range keyboard.Rows {
		var rowContent []string
		for _, button := range row.Buttons {
			// 跳过没有label或没有action data的按钮
			if button == nil || button.RenderData == nil || button.Action == nil {
				continue
			}

			label := button.RenderData.Label
			actionData := button.Action.Data
			buttonText := fmt.Sprintf(`<a href="weixin://bizmsgmenu?msgmenucontent=%s&msgmenuid=0">%s</a>`, actionData, label)
			rowContent = append(rowContent, buttonText)
		}

		// 如果当前行没有有效的按钮，则跳过
		if len(rowContent) > 0 {
			result = append(result, strings.Join(rowContent, " "))
		}
	}
	return strings.Join(result, "\n")
}

// 主函数，将 Markdown 和 Keyboard 合并成文本
func parseMDData(mdData []byte) (string, error) {
	markdown, keyboard, err := parseMDDataPre(mdData)
	if err != nil {
		fmt.Print(err)
	}

	// 处理Markdown内容
	var messageText string
	if markdown != nil {
		messageText = parseMarkdownContent(markdown.Content)
	}

	fmt.Printf("长度为%d\n", len(messageText))

	if len(messageText) > 2047 {
		messageText = parseMarkdownContentV2(markdown.Content)
	}

	// 处理键盘内容，如果存在
	if keyboard != nil && keyboard.Content != nil {
		keyboardText := parseKeyboardContent(keyboard.Content)

		// 如果键盘文本长度超过2037字符，跳过添加键盘内容
		if len(messageText) > 2047 {
			// 不组合键盘按钮
			fmt.Println("键盘内容超过1000字符，跳过组合键盘内容")
		} else {
			// 组合键盘内容
			if len(keyboardText) > 0 {
				if len(messageText) > 0 {
					messageText += "\n\n" // Markdown和键盘之间加个空行
				}
				messageText += keyboardText
			}
		}
	}

	return messageText, nil
}

func parseMDDataPre(mdData []byte) (*Markdown, *MessageKeyboard, error) {
	// 定义一个用于解析 JSON 的临时结构体
	var temp struct {
		Markdown struct {
			CustomTemplateID *string           `json:"custom_template_id,omitempty"`
			Params           []*MarkdownParams `json:"params,omitempty"`
			Content          string            `json:"content,omitempty"`
		} `json:"markdown,omitempty"`
		Keyboard struct {
			ID      string          `json:"id,omitempty"`
			Content *CustomKeyboard `json:"content,omitempty"`
		} `json:"keyboard,omitempty"`
		Rows []*Row `json:"rows,omitempty"`
	}

	// 解析 JSON
	if err := json.Unmarshal(mdData, &temp); err != nil {
		return nil, nil, err
	}

	// 处理 Markdown
	var md *Markdown
	if temp.Markdown.CustomTemplateID != nil {
		// 处理模板 Markdown
		md = &Markdown{
			CustomTemplateID: *temp.Markdown.CustomTemplateID,
			Params:           temp.Markdown.Params,
			Content:          temp.Markdown.Content,
		}
	} else if temp.Markdown.Content != "" {
		// 处理自定义 Markdown
		md = &Markdown{
			Content: temp.Markdown.Content,
		}
	}

	// 处理 Keyboard
	var kb *MessageKeyboard
	if temp.Keyboard.Content != nil {
		// 处理嵌套在 Keyboard 中的 CustomKeyboard
		kb = &MessageKeyboard{
			ID:      temp.Keyboard.ID,
			Content: temp.Keyboard.Content,
		}
	} else if len(temp.Rows) > 0 {
		// 处理顶层的 Rows
		kb = &MessageKeyboard{
			Content: &CustomKeyboard{Rows: temp.Rows},
		}
	} else if temp.Keyboard.ID != "" {
		// 处理嵌套在 Keyboard 中的 ID(当使用按钮模板时)
		kb = &MessageKeyboard{
			ID: temp.Keyboard.ID,
		}
	}

	return md, kb, nil
}

// 将Markdown图片链接转换为CQ:image格式
func ConvertMarkdownToCQImage(text string) string {
	// 定义正则表达式，匹配Markdown图片链接中的URL
	mdImagePattern := regexp.MustCompile(`!\[.*?\]\((http[s]?:\/\/[^\)]+)\)`)

	// 使用正则替换将Markdown图片链接转换为CQ:image格式
	result := mdImagePattern.ReplaceAllStringFunc(text, func(match string) string {
		// 提取URL部分
		url := mdImagePattern.FindStringSubmatch(match)[1]

		// 返回CQ:image格式
		return fmt.Sprintf("[CQ:image,file=%s]", url)
	})

	return result
}

// RemoveMarkdownImages 移除Markdown图片链接
func RemoveMarkdownImages(text string) string {
	// 定义正则表达式，匹配Markdown图片链接
	mdImagePattern := regexp.MustCompile(`!\[.*?\]\((http[s]?:\/\/[^\)]+)\)`)

	// 使用正则替换将Markdown图片链接移除
	result := mdImagePattern.ReplaceAllString(text, "")

	return result
}
