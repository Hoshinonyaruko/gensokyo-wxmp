package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/hoshinonyaruko/gensokyo-wxmp/callapi"
	"github.com/hoshinonyaruko/gensokyo-wxmp/config"
	"github.com/hoshinonyaruko/gensokyo-wxmp/mylog"
	"github.com/hoshinonyaruko/gensokyo-wxmp/wsclient"
	//xurls是一个从文本提取url的库 适用于多种场景
)

var BotID string
var AppID string

// 定义响应结构体
type ServerResponse struct {
	Data struct {
		MessageID int `json:"message_id"`
	} `json:"data"`
	Message string      `json:"message"`
	RetCode int         `json:"retcode"`
	Status  string      `json:"status"`
	Echo    interface{} `json:"echo"`
}

// SendResponse 向所有连接的WebSocket客户端广播回执信息
func SendResponse(Wsclient []*wsclient.WebSocketClient, err error, message *callapi.ActionMessage) (string, error) {
	// 初始化响应结构体
	response := ServerResponse{}
	response.Data.MessageID = 123 // TODO: 实现messageID转换
	response.Echo = message.Echo
	if err != nil {
		response.Message = err.Error()
		response.RetCode = 0 // 示例中将错误情况也视为操作成功
		response.Status = "ok"
	} else {
		response.Message = "操作成功"
		response.RetCode = 0
		response.Status = "ok"
	}

	// 将响应结构体转换为JSON字符串
	jsonResponse, jsonErr := json.Marshal(response)
	if jsonErr != nil {
		log.Printf("Error marshaling response to JSON: %v", jsonErr)
		return "", jsonErr
	}

	// 准备发送的消息
	messageMap := make(map[string]interface{})
	if err := json.Unmarshal(jsonResponse, &messageMap); err != nil {
		log.Printf("Error unmarshaling JSON response: %v", err)
		return "", err
	}

	// 广播消息到所有Wsclient
	broadcastErr := broadcastMessageToAll(messageMap, Wsclient)
	if broadcastErr != nil {
		log.Printf("Error broadcasting message to all clients: %v", broadcastErr)
		return "", broadcastErr
	}

	log.Printf("发送成功回执: %+v", string(jsonResponse))
	return string(jsonResponse), nil
}

// 方便快捷的发信息函数
func broadcastMessageToAll(message map[string]interface{}, Wsclient []*wsclient.WebSocketClient) error {
	var errors []string

	// 发送到我们作为客户端的Wsclient
	for _, client := range Wsclient {
		//mylog.Printf("第%v个Wsclient", test)
		err := client.SendMessage(message)
		if err != nil {
			errors = append(errors, fmt.Sprintf("error sending private message via wsclient: %v", err))
		}
	}

	// 在循环结束后处理记录的错误
	if len(errors) > 0 {
		return fmt.Errorf(strings.Join(errors, "; "))
	}

	//判断是否填写了反向post地址
	if !allEmpty(config.GetPostUrl()) {
		postMessageToUrls(message)
	}
	return nil
}

// allEmpty checks if all the strings in the slice are empty.
func allEmpty(addresses []string) bool {
	for _, addr := range addresses {
		if addr != "" {
			return false
		}
	}
	return true
}

// 将map转化为json string
func ConvertMapToJSONString(m map[string]interface{}) (string, error) {
	// 使用 json.Marshal 将 map 转换为 JSON 字节切片
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		log.Printf("Error marshalling map to JSON: %v", err)
		return "", err
	}

	// 将字节切片转换为字符串
	jsonString := string(jsonBytes)
	return jsonString, nil
}

// 上报信息给反向Http
func postMessageToUrls(message map[string]interface{}) {
	// 获取上报 URL 列表
	postUrls := config.GetPostUrl()

	// 检查 postUrls 是否为空
	if len(postUrls) > 0 {

		// 转换 message 为 JSON 字符串
		jsonString, err := ConvertMapToJSONString(message)
		if err != nil {
			mylog.Printf("Error converting message to JSON: %v", err)
			return
		}

		for _, url := range postUrls {
			// 创建请求体
			reqBody := bytes.NewBufferString(jsonString)

			// 创建 POST 请求
			req, err := http.NewRequest("POST", url, reqBody)
			if err != nil {
				mylog.Printf("Error creating POST request to %s: %v", url, err)
				continue
			}

			// 设置请求头
			req.Header.Set("Content-Type", "application/json")
			// 设置 X-Self-ID
			selfid := config.GetWxAppId()
			req.Header.Set("X-Self-ID", selfid)

			// 发送请求
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				mylog.Printf("Error sending POST request to %s: %v", url, err)
				continue
			}

			// 处理响应
			defer resp.Body.Close()
			// 可以添加更多的响应处理逻辑，如检查状态码等

			mylog.Printf("Posted to %s successfully", url)
		}
	}
}

// // 信息处理函数
// func parseMessageContent(paramsMessage callapi.ParamsContent, message callapi.ActionMessage, client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI) (string, map[string][]string) {
// 	messageText := ""

// 	switch message := paramsMessage.Message.(type) {
// 	case string:
// 		mylog.Printf("params.message is a string\n")
// 		messageText = message
// 	case []interface{}:
// 		//多个映射组成的切片
// 		mylog.Printf("params.message is a slice (segment_type_koishi)\n")
// 		for _, segment := range message {
// 			segmentMap, ok := segment.(map[string]interface{})
// 			if !ok {
// 				continue
// 			}

// 			segmentType, ok := segmentMap["type"].(string)
// 			if !ok {
// 				continue
// 			}

// 			segmentContent := ""
// 			switch segmentType {
// 			case "text":
// 				segmentContent, _ = segmentMap["data"].(map[string]interface{})["text"].(string)
// 			case "image":
// 				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
// 				segmentContent = "[CQ:image,file=" + fileContent + "]"
// 			case "voice":
// 				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
// 				segmentContent = "[CQ:record,file=" + fileContent + "]"
// 			case "record":
// 				fileContent, _ := segmentMap["data"].(map[string]interface{})["file"].(string)
// 				segmentContent = "[CQ:record,file=" + fileContent + "]"
// 			case "at":
// 				qqNumber, _ := segmentMap["data"].(map[string]interface{})["qq"].(string)
// 				segmentContent = "[CQ:at,qq=" + qqNumber + "]"
// 			case "markdown":
// 				mdContent, ok := segmentMap["data"].(map[string]interface{})["data"]
// 				if ok {
// 					if mdContentMap, isMap := mdContent.(map[string]interface{}); isMap {
// 						// mdContent是map[string]interface{}，按map处理
// 						mdContentBytes, err := json.Marshal(mdContentMap)
// 						if err != nil {
// 							mylog.Printf("Error marshaling mdContentMap to JSON:%v", err)
// 						}
// 						encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
// 						segmentContent = "[CQ:markdown,data=" + encoded + "]"
// 					} else if mdContentStr, isString := mdContent.(string); isString {
// 						// mdContent是string
// 						if strings.HasPrefix(mdContentStr, "base64://") {
// 							// 如果以base64://开头，直接使用
// 							segmentContent = "[CQ:markdown,data=" + mdContentStr + "]"
// 						} else {
// 							// 处理实体化后的JSON文本
// 							mdContentStr = strings.ReplaceAll(mdContentStr, "&amp;", "&")
// 							mdContentStr = strings.ReplaceAll(mdContentStr, "&#91;", "[")
// 							mdContentStr = strings.ReplaceAll(mdContentStr, "&#93;", "]")
// 							mdContentStr = strings.ReplaceAll(mdContentStr, "&#44;", ",")

// 							// 将处理过的字符串视为JSON对象，进行序列化和编码
// 							var jsonMap map[string]interface{}
// 							if err := json.Unmarshal([]byte(mdContentStr), &jsonMap); err != nil {
// 								mylog.Printf("Error unmarshaling string to JSON:%v", err)
// 							}
// 							mdContentBytes, err := json.Marshal(jsonMap)
// 							if err != nil {
// 								mylog.Printf("Error marshaling jsonMap to JSON:%v", err)
// 							}
// 							encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
// 							segmentContent = "[CQ:markdown,data=" + encoded + "]"
// 						}
// 					}
// 				} else {
// 					mylog.Printf("Error marshaling markdown segment to interface,contain type but data is nil.")
// 				}
// 			}

// 			messageText += segmentContent
// 		}
// 	case map[string]interface{}:
// 		//单个映射
// 		mylog.Printf("params.message is a map (segment_type_trss)\n")
// 		messageType, _ := message["type"].(string)
// 		switch messageType {
// 		case "text":
// 			messageText, _ = message["data"].(map[string]interface{})["text"].(string)
// 		case "image":
// 			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
// 			messageText = "[CQ:image,file=" + fileContent + "]"
// 		case "voice":
// 			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
// 			messageText = "[CQ:record,file=" + fileContent + "]"
// 		case "record":
// 			fileContent, _ := message["data"].(map[string]interface{})["file"].(string)
// 			messageText = "[CQ:record,file=" + fileContent + "]"
// 		case "at":
// 			qqNumber, _ := message["data"].(map[string]interface{})["qq"].(string)
// 			messageText = "[CQ:at,qq=" + qqNumber + "]"
// 		case "markdown":
// 			mdContent, ok := message["data"].(map[string]interface{})["data"]
// 			if ok {
// 				if mdContentMap, isMap := mdContent.(map[string]interface{}); isMap {
// 					// mdContent是map[string]interface{}，按map处理
// 					mdContentBytes, err := json.Marshal(mdContentMap)
// 					if err != nil {
// 						mylog.Printf("Error marshaling mdContentMap to JSON:%v", err)
// 					}
// 					encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
// 					messageText = "[CQ:markdown,data=" + encoded + "]"
// 				} else if mdContentStr, isString := mdContent.(string); isString {
// 					// mdContent是string
// 					if strings.HasPrefix(mdContentStr, "base64://") {
// 						// 如果以base64://开头，直接使用
// 						messageText = "[CQ:markdown,data=" + mdContentStr + "]"
// 					} else {
// 						// 处理实体化后的JSON文本
// 						mdContentStr = strings.ReplaceAll(mdContentStr, "&amp;", "&")
// 						mdContentStr = strings.ReplaceAll(mdContentStr, "&#91;", "[")
// 						mdContentStr = strings.ReplaceAll(mdContentStr, "&#93;", "]")
// 						mdContentStr = strings.ReplaceAll(mdContentStr, "&#44;", ",")

// 						// 将处理过的字符串视为JSON对象，进行序列化和编码
// 						var jsonMap map[string]interface{}
// 						if err := json.Unmarshal([]byte(mdContentStr), &jsonMap); err != nil {
// 							mylog.Printf("Error unmarshaling string to JSON:%v", err)
// 						}
// 						mdContentBytes, err := json.Marshal(jsonMap)
// 						if err != nil {
// 							mylog.Printf("Error marshaling jsonMap to JSON:%v", err)
// 						}
// 						encoded := base64.StdEncoding.EncodeToString(mdContentBytes)
// 						messageText = "[CQ:markdown,data=" + encoded + "]"
// 					}
// 				}
// 			} else {
// 				mylog.Printf("Error marshaling markdown segment to interface,contain type but data is nil.")
// 			}
// 		}
// 	default:
// 		mylog.Println("Unsupported message format: params.message field is not a string, map or slice")
// 	}
// 	//处理at
// 	messageText = transformMessageTextAt(messageText)

// 	//mylog.Printf(messageText)

// 	// 正则表达式部分
// 	var localImagePattern *regexp.Regexp
// 	var localRecordPattern *regexp.Regexp
// 	if runtime.GOOS == "windows" {
// 		localImagePattern = regexp.MustCompile(`\[CQ:image,file=file:///([^\]]+?)\]`)
// 	} else {
// 		localImagePattern = regexp.MustCompile(`\[CQ:image,file=file://([^\]]+?)\]`)
// 	}
// 	if runtime.GOOS == "windows" {
// 		localRecordPattern = regexp.MustCompile(`\[CQ:record,file=file:///([^\]]+?)\]`)
// 	} else {
// 		localRecordPattern = regexp.MustCompile(`\[CQ:record,file=file://([^\]]+?)\]`)
// 	}
// 	httpUrlImagePattern := regexp.MustCompile(`\[CQ:image,file=http://(.+)\]`)
// 	httpsUrlImagePattern := regexp.MustCompile(`\[CQ:image,file=https://(.+)\]`)
// 	base64ImagePattern := regexp.MustCompile(`\[CQ:image,file=base64://(.+)\]`)
// 	base64RecordPattern := regexp.MustCompile(`\[CQ:record,file=base64://(.+)\]`)
// 	httpUrlRecordPattern := regexp.MustCompile(`\[CQ:record,file=http://(.+)\]`)
// 	httpsUrlRecordPattern := regexp.MustCompile(`\[CQ:record,file=https://(.+)\]`)
// 	mdPattern := regexp.MustCompile(`\[CQ:markdown,data=base64://(.+)\]`)

// 	patterns := []struct {
// 		key     string
// 		pattern *regexp.Regexp
// 	}{
// 		{"local_image", localImagePattern},
// 		{"url_image", httpUrlImagePattern},
// 		{"url_images", httpsUrlImagePattern},
// 		{"base64_image", base64ImagePattern},
// 		{"base64_record", base64RecordPattern},
// 		{"local_record", localRecordPattern},
// 		{"url_record", httpUrlRecordPattern},
// 		{"url_records", httpsUrlRecordPattern},
// 		{"markdown", mdPattern},
// 	}

// 	foundItems := make(map[string][]string)
// 	for _, pattern := range patterns {
// 		matches := pattern.pattern.FindAllStringSubmatch(messageText, -1)
// 		for _, match := range matches {
// 			if len(match) > 1 {
// 				foundItems[pattern.key] = append(foundItems[pattern.key], match[1])
// 			}
// 		}
// 		// 移动替换操作到这里，确保所有匹配都被处理后再进行替换
// 		messageText = pattern.pattern.ReplaceAllString(messageText, "")
// 	}
// 	//最后再处理Url
// 	messageText = transformMessageTextUrl(messageText, message, client, api, apiv2)

// 	// for key, items := range foundItems {
// 	// 	fmt.Printf("Key: %s, Items: %v\n", key, items)
// 	// }
// 	return messageText, foundItems
// }

// func isIPAddress(address string) bool {
// 	return net.ParseIP(address) != nil
// }

// // at处理
// func transformMessageTextAt(messageText string) string {
// 	// 首先，将AppID替换为BotID
// 	messageText = strings.ReplaceAll(messageText, AppID, BotID)

// 	// 去除所有[CQ:reply,id=数字] todo 更好的处理办法
// 	replyRE := regexp.MustCompile(`\[CQ:reply,id=\d+\]`)
// 	messageText = replyRE.ReplaceAllString(messageText, "")

// 	// 使用正则表达式来查找所有[CQ:at,qq=数字]的模式
// 	re := regexp.MustCompile(`\[CQ:at,qq=(\d+)\]`)
// 	messageText = re.ReplaceAllStringFunc(messageText, func(m string) string {
// 		submatches := re.FindStringSubmatch(m)
// 		if len(submatches) > 1 {
// 			realUserID, err := idmap.RetrieveRowByIDv2(submatches[1])
// 			if err != nil {
// 				// 如果出错，也替换成相应的格式，但使用原始QQ号
// 				mylog.Printf("Error retrieving user ID: %v", err)
// 				return "<@!" + submatches[1] + ">"
// 			}

// 			// 在这里检查 GetRemoveBotAtGroup 和 realUserID 的长度
// 			if config.GetRemoveBotAtGroup() && len(realUserID) == 32 {
// 				return ""
// 			}

// 			return "<@!" + realUserID + ">"
// 		}
// 		return m
// 	})
// 	return messageText
// }

// // 链接处理
// func transformMessageTextUrl(messageText string, message callapi.ActionMessage, client callapi.Client, api openapi.OpenAPI, apiv2 openapi.OpenAPI) string {
// 	// 是否处理url
// 	if config.GetTransferUrl() {
// 		// 判断服务器地址是否是IP地址
// 		serverAddress := config.GetServer_dir()
// 		isIP := isIPAddress(serverAddress)
// 		VisualIP := config.GetVisibleIP()

// 		// 使用xurls来查找和替换所有的URL
// 		messageText = xurls.Relaxed.ReplaceAllStringFunc(messageText, func(originalURL string) string {
// 			// 当服务器地址是IP地址且GetVisibleIP为false时，替换URL为空
// 			if isIP && !VisualIP {
// 				return ""
// 			}

// 			// // 如果启用了URL到QR码的转换
// 			// if config.GetUrlToQrimage() {
// 			// 	// 将URL转换为QR码的字节形式
// 			// 	qrCodeGenerator, _ := qrcode.New(originalURL, qrcode.High)
// 			// 	qrCodeGenerator.DisableBorder = true
// 			// 	qrSize := config.GetQrSize()
// 			// 	pngBytes, _ := qrCodeGenerator.PNG(qrSize)
// 			// 	//pngBytes 二维码图片的字节数据
// 			// 	base64Image := base64.StdEncoding.EncodeToString(pngBytes)
// 			// 	picmsg := processActionMessageWithBase64PicReplace(base64Image, message)
// 			// 	ret := callapi.CallAPIFromDict(client, api, apiv2, picmsg)
// 			// 	mylog.Printf("发送url转图片结果:%v", ret)
// 			// 	// 从文本中去除原始URL
// 			// 	return "" // 返回空字符串以去除URL
// 			// }

// 			// 根据配置处理URL
// 			if config.GetLotusValue() {
// 				// 连接到另一个gensokyo
// 				shortURL := url.GenerateShortURL(originalURL)
// 				return shortURL
// 			} else {
// 				// 自己是主节点
// 				shortURL := url.GenerateShortURL(originalURL)
// 				// 使用getBaseURL函数来获取baseUrl并与shortURL组合
// 				return url.GetBaseURL() + "/url/" + shortURL
// 			}
// 		})
// 	}
// 	return messageText
// }

// processActionMessageWithBase64PicReplace 将原有的callapi.ActionMessage内容替换为一个base64图片
// func processActionMessageWithBase64PicReplace(base64Image string, message callapi.ActionMessage) callapi.ActionMessage {
// 	newMessage := createCQImageMessage(base64Image)
// 	message.Params.Message = newMessage
// 	return message
// }

// // createCQImageMessage 从 base64 编码的图片创建 CQ 码格式的消息
// func createCQImageMessage(base64Image string) string {
// 	return "[CQ:image,file=base64://" + base64Image + "]"
// }

// 处理at和其他定形文到onebotv11格式(cq码)
// func RevertTransformedText(data interface{}, msgtype string, api openapi.OpenAPI, apiv2 openapi.OpenAPI, vgid int64, vuid int64, whitenable bool) string {
// 	var msg *dto.Message
// 	var menumsg bool
// 	var messageText string
// 	switch v := data.(type) {
// 	case *dto.WSGroupATMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSATMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSDirectMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSC2CMessageData:
// 		msg = (*dto.Message)(v)
// 	default:
// 		return ""
// 	}
// 	menumsg = false
// 	//单独一个空格的信息的空格用户并不希望去掉
// 	if msg.Content == " " {
// 		menumsg = true
// 		messageText = " "
// 	}

// 	if !menumsg {
// 		//处理前 先去前后空
// 		messageText = strings.TrimSpace(msg.Content)
// 	}
// 	//mylog.Printf("1[%v]", messageText)

// 	// 将messageText里的BotID替换成AppID
// 	messageText = strings.ReplaceAll(messageText, BotID, AppID)

// 	// 使用正则表达式来查找所有<@!数字>的模式
// 	re := regexp.MustCompile(`<@!(\d+)>`)
// 	// 使用正则表达式来替换找到的模式为[CQ:at,qq=用户ID]
// 	messageText = re.ReplaceAllStringFunc(messageText, func(m string) string {
// 		submatches := re.FindStringSubmatch(m)
// 		if len(submatches) > 1 {
// 			userID := submatches[1]
// 			// 检查是否是 BotID，如果是则直接返回，不进行映射,或根据用户需求移除
// 			if userID == AppID {
// 				if config.GetRemoveAt() {
// 					return ""
// 				} else {
// 					return "[CQ:at,qq=" + AppID + "]"
// 				}
// 			}

// 			// 不是 BotID，进行正常映射
// 			userID64, err := idmap.StoreIDv2(userID)
// 			if err != nil {
// 				//如果储存失败(数据库损坏)返回原始值
// 				mylog.Printf("Error storing ID: %v", err)
// 				return "[CQ:at,qq=" + userID + "]"
// 			}
// 			// 类型转换
// 			userIDStr := strconv.FormatInt(userID64, 10)
// 			// 经过转换的cq码
// 			return "[CQ:at,qq=" + userIDStr + "]"
// 		}
// 		return m
// 	})
// 	//结构 <@!>空格/内容
// 	//如果移除了前部at,信息就会以空格开头,因为只移去了最前面的at,但at后紧跟随一个空格
// 	if config.GetRemoveAt() {
// 		if !menumsg {
// 			//再次去前后空
// 			messageText = strings.TrimSpace(messageText)
// 		}
// 	}
// 	//mylog.Printf("2[%v]", messageText)

// 	// 检查是否需要移除前缀
// 	if config.GetRemovePrefixValue() {
// 		// 移除消息内容中第一次出现的 "/"
// 		if idx := strings.Index(messageText, "/"); idx != -1 {
// 			messageText = messageText[:idx] + messageText[idx+1:]
// 		}
// 	}

// 	// 检查是否启用白名单模式
// 	if config.GetWhitePrefixMode() && whitenable {
// 		// 获取白名单反转标志
// 		whiteBypassRevers := config.GetWhiteBypassRevers()

// 		// 获取白名单例外群数组（现在返回 int64 数组）
// 		whiteBypass := config.GetWhiteBypass()
// 		bypass := false

// 		// 根据 whiteBypassRevers 的值来改变逻辑
// 		if whiteBypassRevers {
// 			// 如果反转白名单效果，只有在白名单数组中的 vgid 或 vuid 才生效
// 			bypass = true // 默认设置为 true，意味着需要白名单检查
// 			for _, id := range whiteBypass {
// 				if id == vgid || id == vuid {
// 					bypass = false // 如果在白名单数组中找到了 vgid 或 vuid，设置为 false
// 					break
// 				}
// 			}
// 		} else {
// 			// 常规逻辑：检查 vgid 是否在白名单例外数组中
// 			for _, id := range whiteBypass {
// 				if id == vgid || id == vuid {
// 					bypass = true
// 					break
// 				}
// 			}
// 		}

// 		// 如果vgid不在白名单例外数组中，则应用白名单过滤
// 		if !bypass {
// 			// 获取白名单数组
// 			whitePrefixes := config.GetWhitePrefixs()
// 			// 加锁以安全地读取 TemporaryCommands
// 			idmap.MutexT.Lock()
// 			temporaryCommands := make([]string, len(idmap.TemporaryCommands))
// 			copy(temporaryCommands, idmap.TemporaryCommands)
// 			idmap.MutexT.Unlock()

// 			// 合并白名单和临时指令
// 			allPrefixes := append(whitePrefixes, temporaryCommands...)
// 			// 默认设置为不匹配
// 			matched := false

// 			// 遍历白名单数组，检查是否有匹配项
// 			for _, prefix := range allPrefixes {
// 				if strings.HasPrefix(messageText, prefix) {
// 					// 找到匹配项，保留 messageText 并跳出循环
// 					matched = true
// 					break
// 				}
// 			}

// 			// 如果没有匹配项，则将 messageText 置为兜底回复 兜底回复可空
// 			if !matched {
// 				messageText = ""
// 				SendMessage(config.GetNoWhiteResponse(), data, msgtype, api, apiv2)
// 			}
// 		}
// 	}
// 	//mylog.Printf("3[%v]", messageText)
// 	//检查是否启用黑名单模式
// 	if config.GetBlackPrefixMode() {
// 		// 获取黑名单数组
// 		blackPrefixes := config.GetBlackPrefixs()
// 		// 遍历黑名单数组，检查是否有匹配项
// 		for _, prefix := range blackPrefixes {
// 			if strings.HasPrefix(messageText, prefix) {
// 				// 找到匹配项，将 messageText 置为空并停止处理
// 				messageText = ""
// 				break
// 			}
// 		}
// 	}
// 	// 移除以 GetVisualkPrefixs 数组开头的文本
// 	visualkPrefixs := config.GetVisualkPrefixs()
// 	var matchedPrefix *config.VisualPrefixConfig
// 	var isSpecialType bool    // 用于标记是否为特殊类型
// 	var originalPrefix string // 存储原始前缀

// 	// 处理特殊类型前缀
// 	specialPrefixes := make(map[int]string)
// 	for i, vp := range visualkPrefixs {
// 		if strings.HasPrefix(vp.Prefix, "*") {
// 			specialPrefixes[i] = vp.Prefix                                // 保存原始前缀
// 			visualkPrefixs[i].Prefix = strings.TrimPrefix(vp.Prefix, "*") // 移除 '*'
// 		}
// 	}

// 	//从当前信息去掉虚拟前缀(因为是虚拟的),不会实际发给应用端
// 	for i, vp := range visualkPrefixs {
// 		if strings.HasPrefix(messageText, vp.Prefix) {
// 			if _, ok := specialPrefixes[i]; ok {
// 				isSpecialType = true
// 				originalPrefix = specialPrefixes[i] // 恢复原始前缀
// 			}
// 			// 检查 messageText 的长度是否大于 prefix 的长度
// 			if len(messageText) > len(vp.Prefix) {
// 				// 移除找到的前缀 且messageText不为空
// 				if messageText != "" {
// 					messageText = strings.TrimPrefix(messageText, vp.Prefix)
// 					messageText = strings.TrimSpace(messageText)
// 					matchedPrefix = &vp
// 				}
// 				break // 只移除第一个匹配的前缀
// 			}
// 		}
// 	}

// 	// 已经完成了移除前缀等操作,进行aliases替换
// 	aliases := config.GetAlias()
// 	messageText = processMessageText(messageText, aliases)
// 	//mylog.Printf("4[%v]", messageText)
// 	// 检查是否启用二级白名单模式
// 	if config.GetVwhitePrefixMode() && matchedPrefix != nil {
// 		// 获取白名单反转标志
// 		whiteBypassRevers := config.GetWhiteBypassRevers()

// 		// 获取白名单例外群数组（现在返回 int64 数组）
// 		whiteBypass := config.GetWhiteBypass()
// 		bypass := false

// 		// 根据 whiteBypassRevers 的值来改变逻辑
// 		if whiteBypassRevers {
// 			// 如果反转白名单效果，只有在白名单数组中的 vgid 或 vuid 才生效
// 			bypass = true // 默认设置为 true，意味着需要白名单检查
// 			for _, id := range whiteBypass {
// 				if id == vgid || id == vuid {
// 					bypass = false // 如果在白名单数组中找到了 vgid 或 vuid，设置为 false
// 					break
// 				}
// 			}
// 		} else {
// 			// 常规逻辑：检查 vgid 是否在白名单例外数组中
// 			for _, id := range whiteBypass {
// 				if id == vgid || id == vuid {
// 					bypass = true
// 					break
// 				}
// 			}
// 		}

// 		// 如果vgid不在白名单例外数组中，则应用白名单过滤
// 		if !bypass {
// 			allPrefixes := matchedPrefix.WhiteList
// 			idmap.MutexT.Lock()
// 			temporaryCommands := make([]string, len(idmap.TemporaryCommands))
// 			copy(temporaryCommands, idmap.TemporaryCommands)
// 			idmap.MutexT.Unlock()

// 			// 合并虚拟前缀的白名单和临时指令
// 			allPrefixes = append(allPrefixes, temporaryCommands...)
// 			matched := false

// 			// 检查allPrefixes中的每个prefix是否都以*开头
// 			allStarPrefixed := true
// 			for _, prefix := range allPrefixes {
// 				if !strings.HasPrefix(prefix, "*") {
// 					allStarPrefixed = false
// 					break
// 				}
// 			}

// 			//如果二级指令白名单全部是*(忽略自身,那么不判断二级白名单是否匹配)
// 			if allStarPrefixed {
// 				matched = true
// 			} else {
// 				// 遍历白名单数组，检查是否有匹配项
// 				for _, prefix := range allPrefixes {
// 					trimmedPrefix := prefix
// 					if strings.HasPrefix(prefix, "*") {
// 						// 如果前缀以 * 开头，则移除 *
// 						trimmedPrefix = strings.TrimPrefix(prefix, "*")
// 					} else if strings.HasPrefix(prefix, "&") {
// 						// 如果前缀以 & 开头，则移除 & 并从 trimmedPrefix 前端去除 matchedPrefix.Prefix
// 						trimmedPrefix = strings.TrimPrefix(prefix, "&")
// 						trimmedPrefix = strings.TrimPrefix(trimmedPrefix, matchedPrefix.Prefix)
// 					}

// 					// 从trimmedPrefix中去除前后空格(可能会有bug)
// 					trimmedPrefix = strings.TrimSpace(trimmedPrefix)

// 					if strings.HasPrefix(messageText, trimmedPrefix) {
// 						matched = true
// 						break
// 					}
// 				}
// 			}

// 			// 如果没有匹配项，则将 messageText 置为对应的兜底回复
// 			if !matched {
// 				messageText = ""
// 				SendMessage(matchedPrefix.NoWhiteResponse, data, msgtype, api, apiv2)
// 			}
// 		}
// 	}

// 	// 在返回 messageText 时，根据 isSpecialType 判断是否需要添加原始前缀
// 	if isSpecialType && matchedPrefix != nil {
// 		//移除开头的*
// 		originalPrefix = strings.TrimPrefix(originalPrefix, "*")
// 		messageText = originalPrefix + messageText
// 	}
// 	//mylog.Printf("5[%v]", messageText)
// 	// 如果未启用白名单模式或没有匹配的虚拟前缀，执行默认逻辑

// 	// 处理图片附件
// 	for _, attachment := range msg.Attachments {
// 		if strings.HasPrefix(attachment.ContentType, "image/") {
// 			// 获取文件的后缀名
// 			ext := filepath.Ext(attachment.FileName)
// 			md5name := strings.TrimSuffix(attachment.FileName, ext)

// 			// 检查 URL 是否已包含协议头
// 			var url string
// 			if strings.HasPrefix(attachment.URL, "http://") || strings.HasPrefix(attachment.URL, "https://") {
// 				url = attachment.URL
// 			} else {
// 				url = "http://" + attachment.URL // 默认使用 http，也可以根据需要改为 https
// 			}

// 			imageCQ := "[CQ:image,file=" + md5name + ".image,subType=0,url=" + url + "]"
// 			messageText += imageCQ
// 		}
// 	}
// 	//mylog.Printf("6[%v]", messageText)
// 	return messageText
// }

// // replaceFirstOccurrence 替换字符串中的第一个匹配项
// func replaceFirstOccurrence(s, old, new string) string {
// 	if idx := strings.Index(s, old); idx != -1 {
// 		return s[:idx] + new + s[idx+len(old):]
// 	}
// 	return s
// }

// // processMessageText 处理消息文本
// func processMessageText(messageText string, aliases []string) string {
// 	for i := 0; i < len(aliases); i += 2 {
// 		// 确保别名数组中有成对的元素
// 		if i+1 < len(aliases) {
// 			messageText = replaceFirstOccurrence(messageText, aliases[i], aliases[i+1])
// 		}
// 	}
// 	return messageText
// }

// 将收到的data.content转换为message segment todo,群场景不支持受图片,频道场景的图片可以拼一下
func ConvertToSegmentedMessage(data string) []map[string]interface{} {

	var messageSegments []map[string]interface{}

	// 内容被视为文本部分
	if data != "" {
		textSegment := map[string]interface{}{
			"type": "text",
			"data": map[string]interface{}{
				"text": data,
			},
		}
		messageSegments = append(messageSegments, textSegment)
	}
	//排列
	messageSegments = sortMessageSegments(messageSegments)
	return messageSegments
}

// ConvertToInt64 尝试将 interface{} 类型的值转换为 int64 类型
func ConvertToInt64(value interface{}) (int64, error) {
	switch v := value.(type) {
	case int:
		return int64(v), nil
	case int64:
		return v, nil
	case string:
		return strconv.ParseInt(v, 10, 64)
	default:
		// 当无法处理该类型时返回错误
		return 0, fmt.Errorf("无法将类型 %T 转换为 int64", value)
	}
}

// 排列MessageSegments
func sortMessageSegments(segments []map[string]interface{}) []map[string]interface{} {
	var atSegments, textSegments, imageSegments []map[string]interface{}

	for _, segment := range segments {
		switch segment["type"] {
		case "at":
			atSegments = append(atSegments, segment)
		case "text":
			textSegments = append(textSegments, segment)
		case "image":
			imageSegments = append(imageSegments, segment)
		}
	}

	// 按照指定的顺序合并这些切片
	return append(append(atSegments, textSegments...), imageSegments...)
}

// // SendMessage 发送消息根据不同的类型
// func SendMessage(messageText string, data interface{}, messageType string, api openapi.OpenAPI, apiv2 openapi.OpenAPI) error {
// 	// 强制类型转换，获取Message结构
// 	var msg *dto.Message
// 	switch v := data.(type) {
// 	case *dto.WSGroupATMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSATMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSDirectMessageData:
// 		msg = (*dto.Message)(v)
// 	case *dto.WSC2CMessageData:
// 		msg = (*dto.Message)(v)
// 	default:
// 		return nil
// 	}
// 	switch messageType {
// 	case "guild":
// 		// 处理公会消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		textMsg, _ := GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
// 		if _, err := api.PostMessage(context.TODO(), msg.ChannelID, textMsg); err != nil {
// 			mylog.Printf("发送文本信息失败: %v", err)
// 			return err
// 		}

// 	case "group":
// 		// 处理群组消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		textMsg, _ := GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
// 		_, err := apiv2.PostGroupMessage(context.TODO(), msg.GroupID, textMsg)
// 		if err != nil {
// 			mylog.Printf("发送文本群组信息失败: %v", err)
// 			return err
// 		}

// 	case "guild_private":
// 		// 处理私信
// 		timestamp := time.Now().Unix()
// 		timestampStr := fmt.Sprintf("%d", timestamp)
// 		dm := &dto.DirectMessage{
// 			GuildID:    msg.GuildID,
// 			ChannelID:  msg.ChannelID,
// 			CreateTime: timestampStr,
// 		}
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		textMsg, _ := GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
// 		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, textMsg); err != nil {
// 			mylog.Printf("发送文本信息失败: %v", err)
// 			return err
// 		}

// 	case "group_private":
// 		// 处理群组私聊消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		textMsg, _ := GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
// 		_, err := apiv2.PostC2CMessage(context.TODO(), msg.Author.ID, textMsg)
// 		if err != nil {
// 			mylog.Printf("发送文本私聊信息失败: %v", err)
// 			return err
// 		}

// 	default:
// 		return errors.New("未知的消息类型")
// 	}

// 	return nil
// }

// // 将map转化为json string
// func ConvertMapToJSONString(m map[string]interface{}) (string, error) {
// 	// 使用 json.Marshal 将 map 转换为 JSON 字节切片
// 	jsonBytes, err := json.Marshal(m)
// 	if err != nil {
// 		log.Printf("Error marshalling map to JSON: %v", err)
// 		return "", err
// 	}

// 	// 将字节切片转换为字符串
// 	jsonString := string(jsonBytes)
// 	return jsonString, nil
// }

// func parseMDData(mdData []byte) (*dto.Markdown, *keyboard.MessageKeyboard, error) {
// 	// 定义一个用于解析 JSON 的临时结构体
// 	var temp struct {
// 		Markdown struct {
// 			CustomTemplateID *string               `json:"custom_template_id,omitempty"`
// 			Params           []*dto.MarkdownParams `json:"params,omitempty"`
// 			Content          string                `json:"content,omitempty"`
// 		} `json:"markdown,omitempty"`
// 		Keyboard struct {
// 			ID      string                   `json:"id,omitempty"`
// 			Content *keyboard.CustomKeyboard `json:"content,omitempty"`
// 		} `json:"keyboard,omitempty"`
// 		Rows []*keyboard.Row `json:"rows,omitempty"`
// 	}

// 	// 解析 JSON
// 	if err := json.Unmarshal(mdData, &temp); err != nil {
// 		return nil, nil, err
// 	}

// 	// 处理 Markdown
// 	var md *dto.Markdown
// 	if temp.Markdown.CustomTemplateID != nil {
// 		// 处理模板 Markdown
// 		md = &dto.Markdown{
// 			CustomTemplateID: *temp.Markdown.CustomTemplateID,
// 			Params:           temp.Markdown.Params,
// 			Content:          temp.Markdown.Content,
// 		}
// 	} else if temp.Markdown.Content != "" {
// 		// 处理自定义 Markdown
// 		md = &dto.Markdown{
// 			Content: temp.Markdown.Content,
// 		}
// 	}

// 	// 处理 Keyboard
// 	var kb *keyboard.MessageKeyboard
// 	if temp.Keyboard.Content != nil {
// 		// 处理嵌套在 Keyboard 中的 CustomKeyboard
// 		kb = &keyboard.MessageKeyboard{
// 			ID:      temp.Keyboard.ID,
// 			Content: temp.Keyboard.Content,
// 		}
// 	} else if len(temp.Rows) > 0 {
// 		// 处理顶层的 Rows
// 		kb = &keyboard.MessageKeyboard{
// 			Content: &keyboard.CustomKeyboard{Rows: temp.Rows},
// 		}
// 	}

// 	return md, kb, nil
// }

// 将结构体转换为 map[string]interface{}
// func structToMap(obj interface{}) map[string]interface{} {
// 	out := make(map[string]interface{})
// 	j, _ := json.Marshal(obj)
// 	json.Unmarshal(j, &out)
// 	return out
// }
