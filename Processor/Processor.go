// 处理收到的信息事件
package Processor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/hoshinonyaruko/gensokyo-wxmp/config"
	"github.com/hoshinonyaruko/gensokyo-wxmp/handlers"
	"github.com/hoshinonyaruko/gensokyo-wxmp/mylog"
	"github.com/hoshinonyaruko/gensokyo-wxmp/wsclient"
	"github.com/tencent-connect/botgo/dto"
)

type Sender struct {
	Nickname string `json:"nickname"`
	TinyID   string `json:"tiny_id"`
	UserID   int64  `json:"user_id"`
	Role     string `json:"role,omitempty"`
	Card     string `json:"card,omitempty"`
	Sex      string `json:"sex,omitempty"`
	Age      int32  `json:"age,omitempty"`
	Area     string `json:"area,omitempty"`
	Level    string `json:"level,omitempty"`
	Title    string `json:"title,omitempty"`
}

// 频道信息事件
type OnebotChannelMessage struct {
	ChannelID   string      `json:"channel_id"`
	GuildID     string      `json:"guild_id"`
	Message     interface{} `json:"message"`
	MessageID   string      `json:"message_id"`
	MessageType string      `json:"message_type"`
	PostType    string      `json:"post_type"`
	SelfID      int64       `json:"self_id"`
	SelfTinyID  string      `json:"self_tiny_id"`
	Sender      Sender      `json:"sender"`
	SubType     string      `json:"sub_type"`
	Time        int64       `json:"time"`
	Avatar      string      `json:"avatar,omitempty"`
	UserID      int64       `json:"user_id"`
	RawMessage  string      `json:"raw_message"`
	Echo        string      `json:"echo,omitempty"`
}

// 群信息事件
type OnebotGroupMessage struct {
	RawMessage      string      `json:"raw_message"`
	MessageID       int         `json:"message_id"`
	GroupID         int64       `json:"group_id"` // Can be either string or int depending on p.Settings.CompleteFields
	MessageType     string      `json:"message_type"`
	PostType        string      `json:"post_type"`
	SelfID          int64       `json:"self_id"` // Can be either string or int
	Sender          Sender      `json:"sender"`
	SubType         string      `json:"sub_type"`
	Time            int64       `json:"time"`
	Avatar          string      `json:"avatar,omitempty"`
	Echo            string      `json:"echo,omitempty"`
	Message         interface{} `json:"message"` // For array format
	MessageSeq      int         `json:"message_seq"`
	Font            int         `json:"font"`
	UserID          int64       `json:"user_id"`
	RealMessageType string      `json:"real_message_type,omitempty"`  //当前信息的真实类型 group group_private guild guild_private
	IsBindedGroupId bool        `json:"is_binded_group_id,omitempty"` //当前群号是否是binded后的
	IsBindedUserId  bool        `json:"is_binded_user_id,omitempty"`  //当前用户号号是否是binded后的
}

type OnebotGroupMessageS struct {
	RawMessage      string      `json:"raw_message"`
	MessageID       string      `json:"message_id"`
	GroupID         string      `json:"group_id"` // Can be either string or int depending on p.Settings.CompleteFields
	MessageType     string      `json:"message_type"`
	PostType        string      `json:"post_type"`
	SelfID          int64       `json:"self_id"` // Can be either string or int
	Sender          Sender      `json:"sender"`
	SubType         string      `json:"sub_type"`
	Time            int64       `json:"time"`
	Avatar          string      `json:"avatar,omitempty"`
	Echo            string      `json:"echo,omitempty"`
	Message         interface{} `json:"message"` // For array format
	MessageSeq      int         `json:"message_seq"`
	Font            int         `json:"font"`
	UserID          string      `json:"user_id"`
	RealMessageType string      `json:"real_message_type,omitempty"`  //当前信息的真实类型 group group_private guild guild_private
	RealUserID      string      `json:"real_user_id,omitempty"`       //当前真实uid
	RealGroupID     string      `json:"real_group_id,omitempty"`      //当前真实gid
	IsBindedGroupId bool        `json:"is_binded_group_id,omitempty"` //当前群号是否是binded后的
	IsBindedUserId  bool        `json:"is_binded_user_id,omitempty"`  //当前用户号号是否是binded后的
}

// 私聊信息事件
type OnebotPrivateMessage struct {
	RawMessage      string        `json:"raw_message"`
	MessageID       int           `json:"message_id"` // Can be either string or int depending on logic
	MessageType     string        `json:"message_type"`
	PostType        string        `json:"post_type"`
	SelfID          int64         `json:"self_id"` // Can be either string or int depending on logic
	Sender          PrivateSender `json:"sender"`
	SubType         string        `json:"sub_type"`
	Time            int64         `json:"time"`
	Avatar          string        `json:"avatar,omitempty"`
	Echo            string        `json:"echo,omitempty"`
	Message         interface{}   `json:"message"`                     // For array format
	MessageSeq      int           `json:"message_seq"`                 // Optional field
	Font            int           `json:"font"`                        // Optional field
	UserID          int64         `json:"user_id"`                     // Can be either string or int depending on logic
	RealMessageType string        `json:"real_message_type,omitempty"` //当前信息的真实类型 group group_private guild guild_private
	IsBindedUserId  bool          `json:"is_binded_user_id,omitempty"` //当前用户号号是否是binded后的
}

// onebotv11标准扩展
type OnebotInteractionNotice struct {
	GroupID    int64                  `json:"group_id,omitempty"`
	NoticeType string                 `json:"notice_type,omitempty"`
	PostType   string                 `json:"post_type,omitempty"`
	SelfID     int64                  `json:"self_id,omitempty"`
	SubType    string                 `json:"sub_type,omitempty"`
	Time       int64                  `json:"time,omitempty"`
	UserID     int64                  `json:"user_id,omitempty"`
	Data       *dto.WSInteractionData `json:"data,omitempty"`
}

type PrivateSender struct {
	Nickname string `json:"nickname"`
	UserID   int64  `json:"user_id"` // Can be either string or int depending on logic
}

func FoxTimestamp() int64 {
	return time.Now().Unix()
}

// ProcessInlineSearch 处理内联查询
// func (p *Processors) ProcessInlineSearch(data *dto.WSInteractionData) error {
// 转换appid
// var userid64 int64
// var GroupID64 int64
// var err error
// var fromgid, fromuid string
// if data.GroupOpenID != "" {
// 	fromgid = data.GroupOpenID
// 	fromuid = data.GroupMemberOpenID
// } else {
// 	fromgid = data.ChannelID
// 	fromuid = "0"
// }
// if config.GetIdmapPro() {
// 	//将真实id转为int userid64
// 	GroupID64, userid64, err = idmap.StoreIDv2Pro(fromgid, fromuid)
// 	if err != nil {
// 		mylog.Fatalf("Error storing ID: %v", err)
// 	}
// 	//当参数不全
// 	_, _ = idmap.StoreIDv2(fromgid)
// 	_, _ = idmap.StoreIDv2(fromuid)
// 	if !config.GetHashIDValue() {
// 		mylog.Fatalf("避坑日志:你开启了高级id转换,请设置hash_id为true,并且删除idmaps并重启")
// 	}
// } else {
// 	// 映射str的GroupID到int
// 	GroupID64, err = idmap.StoreIDv2(fromgid)
// 	if err != nil {
// 		mylog.Errorf("failed to convert ChannelID to int: %v", err)
// 		return nil
// 	}
// 	// 映射str的userid到int
// 	userid64, err = idmap.StoreIDv2(fromuid)
// 	if err != nil {
// 		mylog.Printf("Error storing ID: %v", err)
// 		return nil
// 	}
// }
// notice := &OnebotInteractionNotice{
// 	GroupID:    GroupID64,
// 	NoticeType: "interaction",
// 	PostType:   "notice",
// 	SelfID:     int64(p.Settings.AppID),
// 	SubType:    "create",
// 	Time:       time.Now().Unix(),
// 	UserID:     userid64,
// 	Data:       data,
// }

// //调试
// PrintStructWithFieldNames(notice)

// // Convert OnebotGroupMessage to map and send
// noticeMap := structToMap(notice)

// //上报信息到onebotv11应用端(正反ws)
// p.BroadcastMessageToAll(noticeMap)
// return nil
// }

//return nil

//下面是测试时候固定代码
//发私信给机器人4条机器人不回,就不能继续发了

// timestamp := time.Now().Unix() // 获取当前时间的int64类型的Unix时间戳
// timestampStr := fmt.Sprintf("%d", timestamp)

// dm := &dto.DirectMessage{
// 	GuildID:    GuildID,
// 	ChannelID:  ChannelID,
// 	CreateTime: timestampStr,
// }

// PrintStructWithFieldNames(dm)

// // 发送默认回复
// toCreate := &dto.MessageToCreate{
// 	Content: "默认私信回复",
// 	MsgID:   data.ID,
// }
// _, err = p.Api.PostDirectMessage(
// 	context.Background(), dm, toCreate,
// )
// if err != nil {
// 	mylog.Println("Error sending default reply:", err)
// 	return nil
// }

// 打印结构体的函数
func PrintStructWithFieldNames(v interface{}) {
	val := reflect.ValueOf(v)

	// 如果是指针，获取其指向的元素
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	// 确保我们传入的是一个结构体
	if typ.Kind() != reflect.Struct {
		mylog.Println("Input is not a struct")
		return
	}

	// 迭代所有的字段并打印字段名和值
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		value := val.Field(i)
		mylog.Printf("%s: %v\n", field.Name, value.Interface())
	}
}

// 将结构体转换为 map[string]interface{}
func structToMap(obj interface{}) map[string]interface{} {
	out := make(map[string]interface{})
	j, _ := json.Marshal(obj)
	json.Unmarshal(j, &out)
	return out
}

// 方便快捷的发信息函数
func BroadcastMessageToAll(message map[string]interface{}, Wsclient []*wsclient.WebSocketClient) error {
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
		PostMessageToUrls(message)
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

// 上报信息给反向Http
func PostMessageToUrls(message map[string]interface{}) {
	// 获取上报 URL 列表
	postUrls := config.GetPostUrl()

	// 检查 postUrls 是否为空
	if len(postUrls) > 0 {

		// 转换 message 为 JSON 字符串
		jsonString, err := handlers.ConvertMapToJSONString(message)
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

// func (p *Processors) HandleFrameworkCommand(messageText string, data interface{}, Type string) error {
// 	// 正则表达式匹配转换后的 CQ 码
// 	cqRegex := regexp.MustCompile(`\[CQ:at,qq=\d+\]`)

// 	// 使用正则表达式替换所有的 CQ 码为 ""
// 	cleanedMessage := cqRegex.ReplaceAllString(messageText, "")

// 	// 去除字符串前后的空格
// 	cleanedMessage = strings.TrimSpace(cleanedMessage)
// 	if cleanedMessage == "t" {
// 		// 生成临时指令
// 		tempCmd := handleNoPermission()
// 		mylog.Printf("临时bind指令: %s 可忽略权限检查1次,或将masterid设置为空数组", tempCmd)
// 	}
// 	var err error
// 	var now, new, newpro1, newpro2 string
// 	var nowgroup, newgroup string
// 	var realid, realid2 string
// 	var guildid, guilduserid string
// 	switch v := data.(type) {
// 	case *dto.WSGroupATMessageData:
// 		realid = v.Author.ID
// 	case *dto.WSATMessageData:
// 		realid = v.Author.ID
// 		guildid = v.GuildID
// 		guilduserid = v.Author.ID
// 	case *dto.WSMessageData:
// 		realid = v.Author.ID
// 		guildid = v.GuildID
// 		guilduserid = v.Author.ID
// 	case *dto.WSDirectMessageData:
// 		realid = v.Author.ID
// 	case *dto.WSC2CMessageData:
// 		realid = v.Author.ID
// 	}

// 	switch v := data.(type) {
// 	case *dto.WSGroupATMessageData:
// 		realid2 = v.GroupID
// 	case *dto.WSATMessageData:
// 		realid2 = v.ChannelID
// 	case *dto.WSMessageData:
// 		realid2 = v.ChannelID
// 	case *dto.WSDirectMessageData:
// 		realid2 = v.ChannelID
// 	case *dto.WSC2CMessageData:
// 		realid2 = "group_private"
// 	}

// 	// 获取MasterID数组
// 	masterIDs := config.GetMasterID()
// 	// 根据realid获取new(用户id)
// 	now, new, err = idmap.RetrieveVirtualValuev2(realid)
// 	if err != nil {
// 		mylog.Printf("根据realid获取new(用户id) 错误:%v", err)
// 	}
// 	// 根据realid获取new(群id)
// 	nowgroup, newgroup, err = idmap.RetrieveVirtualValuev2(realid2)
// 	if err != nil {
// 		mylog.Printf("根据realid获取new(群id)错误:%v", err)
// 	}
// 	// idmaps-pro获取群和用户id
// 	if config.GetIdmapPro() {
// 		newpro1, newpro2, err = idmap.RetrieveVirtualValuev2Pro(realid2, realid)
// 		if err != nil {
// 			mylog.Printf("idmaps-pro获取群和用户id 错误:%v", err)
// 		}
// 	}
// 	// 检查真实值或虚拟值是否在数组中
// 	var realValueIncluded, virtualValueIncluded bool

// 	// 如果 masterIDs 数组为空，则这两个值恒为 true
// 	if len(masterIDs) == 0 {
// 		realValueIncluded = true
// 		virtualValueIncluded = true
// 	} else {
// 		// 否则，检查真实值或虚拟值是否在数组中
// 		realValueIncluded = contains(masterIDs, realid)
// 		virtualValueIncluded = contains(masterIDs, new)
// 	}

// 	//unlock指令
// 	if Type == "guild" && strings.HasPrefix(cleanedMessage, config.GetUnlockPrefix()) {
// 		dm := &dto.DirectMessageToCreate{
// 			SourceGuildID: guildid,
// 			RecipientID:   guilduserid,
// 		}
// 		cdm, err := p.Api.CreateDirectMessage(context.TODO(), dm)
// 		if err != nil {
// 			mylog.Printf("unlock指令创建dm失败:%v", err)
// 		}
// 		msg := &dto.MessageToCreate{
// 			Content: "欢迎使用Gensokyo框架部署QQ机器人",
// 			MsgType: 0,
// 			MsgID:   "",
// 		}
// 		_, err = p.Api.PostDirectMessage(context.TODO(), cdm, msg)
// 		if err != nil {
// 			mylog.Printf("unlock指令发送失败:%v", err)
// 		}
// 	}

// 	// me指令处理逻辑
// 	if strings.HasPrefix(cleanedMessage, config.GetMePrefix()) {
// 		if err != nil {
// 			// 发送错误信息
// 			SendMessage(err.Error(), data, Type, p.Api, p.Apiv2)
// 			return err
// 		}
// 		// 发送成功信息
// 		if config.GetIdmapPro() {
// 			// 构造清晰的对应关系信息
// 			userMapping := fmt.Sprintf("当前真实值（用户）/当前虚拟值（用户） = [%s/%s]", realid, newpro2)
// 			groupMapping := fmt.Sprintf("当前真实值（群/频道）/当前虚拟值（群/频道） = [%s/%s]", realid2, newpro1)

// 			// 构造 bind 指令的使用说明
// 			bindInstruction := fmt.Sprintf("bind 指令: %s 当前虚拟值(用户) 目标虚拟值(用户) [当前虚拟值(群/频道) 目标虚拟值(群/频道)]", config.GetBindPrefix())

// 			// 发送整合后的消息
// 			message := fmt.Sprintf("idmaps-pro状态:\n%s\n%s\n%s", userMapping, groupMapping, bindInstruction)
// 			SendMessage(message, data, Type, p.Api, p.Apiv2)
// 		} else {
// 			SendMessage("目前状态:\n当前真实值(用户) "+now+"\n当前虚拟值(用户) "+new+"\n当前真实值(群/频道) "+nowgroup+"\n当前虚拟值(群/频道) "+newgroup+"\nbind指令:"+config.GetBindPrefix()+" 当前虚拟值"+" 目标虚拟值", data, Type, p.Api, p.Apiv2)
// 		}
// 		return nil
// 	}

// 	fields := strings.Fields(cleanedMessage)

// 	// 首先确保消息不是空的，然后检查是否是有效的临时指令
// 	if len(fields) > 0 && isValidTemporaryCommand(fields[0]) {
// 		// 执行 bind 操作
// 		if config.GetIdmapPro() {
// 			err := performBindOperationV2(cleanedMessage, data, Type, p.Api, p.Apiv2, newpro1)
// 			if err != nil {
// 				mylog.Printf("bind遇到错误:%v", err)
// 			}
// 		} else {
// 			err := performBindOperation(cleanedMessage, data, Type, p.Api, p.Apiv2)
// 			if err != nil {
// 				mylog.Printf("bind遇到错误:%v", err)
// 			}
// 		}
// 		return nil
// 	}

// 	// 如果不是临时指令，检查是否具有执行bind操作的权限并且消息以特定前缀开始
// 	if (realValueIncluded || virtualValueIncluded) && strings.HasPrefix(cleanedMessage, config.GetBindPrefix()) {
// 		// 执行 bind 操作
// 		if config.GetIdmapPro() {
// 			err := performBindOperationV2(cleanedMessage, data, Type, p.Api, p.Apiv2, newpro1)
// 			if err != nil {
// 				mylog.Printf("bind遇到错误:%v", err)
// 			}
// 		} else {
// 			err := performBindOperation(cleanedMessage, data, Type, p.Api, p.Apiv2)
// 			if err != nil {
// 				mylog.Printf("bind遇到错误:%v", err)
// 			}
// 		}
// 		return nil
// 	} else if strings.HasPrefix(cleanedMessage, config.GetBindPrefix()) {
// 		// 生成临时指令
// 		tempCmd := handleNoPermission()
// 		mylog.Printf("您没有权限,使用临时指令：%s 忽略权限检查,或将masterid设置为空数组", tempCmd)
// 		SendMessage("您没有权限,请配置config.yml或查看日志,使用临时指令", data, Type, p.Api, p.Apiv2)
// 	}

// 	//link指令
// 	if Type == "group" && strings.HasPrefix(cleanedMessage, config.GetLinkPrefix()) {
// 		md, kb := generateMdByConfig()
// 		SendMessageMd(md, kb, data, Type, p.Api, p.Apiv2)
// 	}

// 	return nil
// }

// // 生成由两个英文字母构成的唯一临时指令
// func generateTemporaryCommand() (string, error) {
// 	bytes := make([]byte, 1) // 生成1字节的随机数，足以表示2个十六进制字符
// 	if _, err := rand.Read(bytes); err != nil {
// 		return "", err // 处理随机数生成错误
// 	}
// 	command := hex.EncodeToString(bytes)[:2] // 将1字节转换为2个十六进制字符
// 	return command, nil
// }

// // 生成并添加一个新的临时指令
// func handleNoPermission() string {
// 	idmap.MutexT.Lock()
// 	defer idmap.MutexT.Unlock()

// 	cmd, _ := generateTemporaryCommand()
// 	idmap.TemporaryCommands = append(idmap.TemporaryCommands, cmd)
// 	return cmd
// }

// // 检查指令是否是有效的临时指令
// func isValidTemporaryCommand(cmd string) bool {
// 	idmap.MutexT.Lock()
// 	defer idmap.MutexT.Unlock()

// 	for i, tempCmd := range idmap.TemporaryCommands {
// 		if tempCmd == cmd {
// 			// 删除已验证的临时指令
// 			idmap.TemporaryCommands = append(idmap.TemporaryCommands[:i], idmap.TemporaryCommands[i+1:]...)
// 			return true
// 		}
// 	}
// 	return false
// }

// // 执行 bind 操作的逻辑
// func performBindOperation(cleanedMessage string, data interface{}, Type string, p openapi.OpenAPI, p2 openapi.OpenAPI) error {
// 	// 分割指令以获取参数
// 	parts := strings.Fields(cleanedMessage)
// 	if len(parts) != 3 {
// 		mylog.Printf("bind指令参数错误\n正确的格式" + config.GetBindPrefix() + " 当前虚拟值 新虚拟值")
// 		return nil
// 	}

// 	// 将字符串转换为 int64
// 	oldRowValue, err := strconv.ParseInt(parts[1], 10, 64)
// 	if err != nil {
// 		return err
// 	}

// 	newRowValue, err := strconv.ParseInt(parts[2], 10, 64)
// 	if err != nil {
// 		return err
// 	}

// 	// 调用 UpdateVirtualValue
// 	err = idmap.UpdateVirtualValuev2(oldRowValue, newRowValue)
// 	if err != nil {
// 		SendMessage(err.Error(), data, Type, p, p2)
// 		return err
// 	}
// 	now, new, err := idmap.RetrieveRealValuev2(newRowValue)
// 	if err != nil {
// 		SendMessage(err.Error(), data, Type, p, p2)
// 	} else {
// 		SendMessage("绑定成功,目前状态:\n当前真实值 "+new+"\n当前虚拟值 "+now, data, Type, p, p2)
// 	}

// 	return nil
// }

// func performBindOperationV2(cleanedMessage string, data interface{}, Type string, p openapi.OpenAPI, p2 openapi.OpenAPI, GroupVir string) error {
// 	// 分割指令以获取参数
// 	parts := strings.Fields(cleanedMessage)

// 	// 检查参数数量
// 	if len(parts) < 3 || len(parts) > 5 {
// 		mylog.Printf("bind指令参数错误\n正确的格式: " + config.GetBindPrefix() + " 当前虚拟值(用户) 新虚拟值(用户) [当前虚拟值(群) 新虚拟值(群)]")
// 		return nil
// 	}

// 	// 当前虚拟值 用户
// 	oldVirtualValue1, err := strconv.ParseInt(parts[1], 10, 64)
// 	if err != nil {
// 		return err
// 	}
// 	//新的虚拟值 用户
// 	newVirtualValue1, err := strconv.ParseInt(parts[2], 10, 64)
// 	if err != nil {
// 		return err
// 	}

// 	// 设置默认值
// 	var oldRowValue, newRowValue int64

// 	// 如果提供了第3个和第4个参数，则解析它们
// 	if len(parts) > 3 {
// 		oldRowValue, err = parseOrDefault(parts[3], GroupVir)
// 		if err != nil {
// 			return err
// 		}

// 		newRowValue, err = parseOrDefault(parts[4], GroupVir)
// 		if err != nil {
// 			return err
// 		}
// 	} else {
// 		// 如果没有提供这些参数，则直接使用 GroupVir
// 		oldRowValue, err = strconv.ParseInt(GroupVir, 10, 64)
// 		if err != nil {
// 			return err
// 		}
// 		newRowValue = oldRowValue // 使用相同的值
// 	}
// 	// 调用 UpdateVirtualValue(兼顾老转换)
// 	err = idmap.UpdateVirtualValuev2(oldVirtualValue1, newVirtualValue1)
// 	if err != nil {
// 		SendMessage(err.Error(), data, Type, p, p2)
// 		return err
// 	}
// 	// 调用 UpdateVirtualValuev2Pro
// 	err = idmap.UpdateVirtualValuev2Pro(oldRowValue, newRowValue, oldVirtualValue1, newVirtualValue1)
// 	if err != nil {
// 		SendMessage(err.Error(), data, Type, p, p2)
// 		return err
// 	}

// 	now, new, err := idmap.RetrieveRealValuesv2Pro(newRowValue, newVirtualValue1)
// 	if err != nil {
// 		SendMessage(err.Error(), data, Type, p, p2)
// 	} else {
// 		newVirtualValue1Str := strconv.FormatInt(newRowValue, 10)
// 		newVirtualValue2Str := strconv.FormatInt(newVirtualValue1, 10)
// 		SendMessage("绑定成功,目前状态:\n当前真实值(群)"+now+"\n当前真实值(用户)"+new+"\n当前虚拟值(群)"+newVirtualValue1Str+"当前虚拟值(用户)"+newVirtualValue2Str, data, Type, p, p2)
// 	}

// 	return nil
// }

// // parseOrDefault 将字符串转换为int64，如果无法转换或为0，则使用默认值
// func parseOrDefault(s string, defaultValue string) (int64, error) {
// 	value, err := strconv.ParseInt(s, 10, 64)
// 	if err == nil && value != 0 {
// 		return value, nil
// 	}

// 	return strconv.ParseInt(defaultValue, 10, 64)
// }

// // contains 检查数组中是否包含指定的字符串
// func contains(slice []string, item string) bool {
// 	for _, a := range slice {
// 		if a == item {
// 			return true
// 		}
// 	}
// 	return false
// }

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
// 		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
// 		if _, err := api.PostMessage(context.TODO(), msg.ChannelID, textMsg); err != nil {
// 			mylog.Printf("发送文本信息失败: %v", err)
// 			return err
// 		}

// 	case "group":
// 		// 处理群组消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
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
// 		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
// 		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, textMsg); err != nil {
// 			mylog.Printf("发送文本信息失败: %v", err)
// 			return err
// 		}

// 	case "group_private":
// 		// 处理群组私聊消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		textMsg, _ := handlers.GenerateReplyMessage(msg.ID, nil, messageText, msgseq+1)
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

// // SendMessageMd  发送Md消息根据不同的类型
// func SendMessageMd(md *dto.Markdown, kb *keyboard.MessageKeyboard, data interface{}, messageType string, api openapi.OpenAPI, apiv2 openapi.OpenAPI) error {
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
// 		Message := &dto.MessageToCreate{
// 			Content:  "markdown",
// 			MsgID:    msg.ID,
// 			MsgSeq:   msgseq,
// 			Markdown: md,
// 			Keyboard: kb,
// 			MsgType:  2, //md信息
// 		}
// 		Message.Timestamp = time.Now().Unix() // 设置时间戳
// 		if _, err := api.PostMessage(context.TODO(), msg.ChannelID, Message); err != nil {
// 			mylog.Printf("发送文本信息失败: %v", err)
// 			return err
// 		}

// 	case "group":
// 		// 处理群组消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		Message := &dto.MessageToCreate{
// 			Content:  "markdown",
// 			MsgID:    msg.ID,
// 			MsgSeq:   msgseq,
// 			Markdown: md,
// 			Keyboard: kb,
// 			MsgType:  2, //md信息
// 		}
// 		Message.Timestamp = time.Now().Unix() // 设置时间戳
// 		_, err := apiv2.PostGroupMessage(context.TODO(), msg.GroupID, Message)
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
// 		Message := &dto.MessageToCreate{
// 			Content:  "markdown",
// 			MsgID:    msg.ID,
// 			MsgSeq:   msgseq,
// 			Markdown: md,
// 			Keyboard: kb,
// 			MsgType:  2, //md信息
// 		}
// 		Message.Timestamp = time.Now().Unix() // 设置时间戳
// 		if _, err := apiv2.PostDirectMessage(context.TODO(), dm, Message); err != nil {
// 			mylog.Printf("发送文本信息失败: %v", err)
// 			return err
// 		}

// 	case "group_private":
// 		// 处理群组私聊消息
// 		msgseq := echo.GetMappingSeq(msg.ID)
// 		echo.AddMappingSeq(msg.ID, msgseq+1)
// 		Message := &dto.MessageToCreate{
// 			Content:  "markdown",
// 			MsgID:    msg.ID,
// 			MsgSeq:   msgseq,
// 			Markdown: md,
// 			Keyboard: kb,
// 			MsgType:  2, //md信息
// 		}
// 		Message.Timestamp = time.Now().Unix() // 设置时间戳
// 		_, err := apiv2.PostC2CMessage(context.TODO(), msg.Author.ID, Message)
// 		if err != nil {
// 			mylog.Printf("发送文本私聊信息失败: %v", err)
// 			return err
// 		}

// 	default:
// 		return errors.New("未知的消息类型")
// 	}

// 	return nil
// }

// // 更新映射的辅助函数
// func updateMappings(userid64, vuinValue, GroupID64, idValue int64) error {
// 	if err := idmap.UpdateVirtualValuev2(userid64, vuinValue); err != nil {
// 		mylog.Printf("Error UpdateVirtualValuev2 for vuin: %v", err)
// 		return err
// 	}
// 	if err := idmap.UpdateVirtualValuev2(GroupID64, idValue); err != nil {
// 		mylog.Printf("Error UpdateVirtualValuev2 for gid: %v", err)
// 		return err
// 	}
// 	return nil
// }

// // GenerateAvatarURL 生成根据给定 userID 和随机 q 值组合的 QQ 头像 URL
// func GenerateAvatarURL(userID int64) (string, error) {
// 	// 使用 crypto/rand 生成更安全的随机数
// 	n, err := rand.Int(rand.Reader, big.NewInt(5))
// 	if err != nil {
// 		return "", err
// 	}
// 	qNumber := n.Int64() + 1 // 产生 1 到 5 的随机数

// 	// 构建并返回 URL
// 	return fmt.Sprintf("http://q%d.qlogo.cn/g?b=qq&nk=%d&s=640", qNumber, userID), nil
// }

// // 生成link卡片
// func generateMdByConfig() (md *dto.Markdown, kb *keyboard.MessageKeyboard) {
// 	//相关配置获取
// 	mdtext := config.GetLinkText()
// 	mdtext = "\r" + mdtext
// 	CustomTemplateID := config.GetCustomTemplateID()
// 	linkBots := config.GetLinkBots()
// 	imgURL := config.GetLinkPic()

// 	//超过16个时候随机显示
// 	if len(linkBots) > 16 {
// 		linkBots = getRandomSelection(linkBots, 16)
// 	}

// 	//组合 mdParams
// 	var mdParams []*dto.MarkdownParams
// 	if imgURL != "" {
// 		height, width, err := images.GetImageDimensions(imgURL)
// 		if err != nil {
// 			mylog.Printf("获取图片宽高出错")
// 		}
// 		imgDesc := fmt.Sprintf("图片 #%dpx #%dpx", width, height)
// 		// 创建 MarkdownParams 的实例
// 		mdParams = []*dto.MarkdownParams{
// 			{Key: "img_dec", Values: []string{imgDesc}},
// 			{Key: "img_url", Values: []string{imgURL}},
// 			{Key: "text_end", Values: []string{mdtext}},
// 		}
// 	} else {
// 		mdParams = []*dto.MarkdownParams{
// 			{Key: "text_end", Values: []string{mdtext}},
// 		}
// 	}

// 	// 组合模板 Markdown
// 	md = &dto.Markdown{
// 		CustomTemplateID: CustomTemplateID,
// 		Params:           mdParams,
// 	}

// 	// 创建自定义键盘
// 	customKeyboard := &keyboard.CustomKeyboard{}
// 	var currentRow *keyboard.Row
// 	var buttonCount int

// 	for _, bot := range linkBots {
// 		parts := strings.SplitN(bot, "-", 3)
// 		if len(parts) < 3 {
// 			continue // 跳过无效的格式
// 		}
// 		name := parts[2]
// 		botuin := parts[1]
// 		botappid := parts[0]
// 		boturl := handlers.BuildQQBotShareLink(botuin, botappid)

// 		button := &keyboard.Button{
// 			RenderData: &keyboard.RenderData{
// 				Label:        name,
// 				VisitedLabel: name,
// 				Style:        1, // 蓝色边缘
// 			},
// 			Action: &keyboard.Action{
// 				Type:          0,                             // 链接类型
// 				Permission:    &keyboard.Permission{Type: 2}, // 所有人可操作
// 				Data:          boturl,
// 				UnsupportTips: "请升级新版手机QQ",
// 			},
// 		}

// 		// 如果当前行为空或已满（4个按钮），则创建一个新行
// 		if currentRow == nil || buttonCount == 4 {
// 			currentRow = &keyboard.Row{}
// 			customKeyboard.Rows = append(customKeyboard.Rows, currentRow)
// 			buttonCount = 0
// 		}

// 		// 将按钮添加到当前行
// 		currentRow.Buttons = append(currentRow.Buttons, button)
// 		buttonCount++
// 	}

// 	// 创建 MessageKeyboard 并设置其 Content
// 	kb = &keyboard.MessageKeyboard{
// 		Content: customKeyboard,
// 	}

// 	return md, kb
// }

// func getRandomSelection(slice []string, max int) []string {
// 	if len(slice) <= max {
// 		return slice
// 	}

// 	selected := make(map[int]bool)
// 	var result []string
// 	for len(result) < max {
// 		index, _ := rand.Int(rand.Reader, big.NewInt(int64(len(slice))))
// 		idx := int(index.Int64())
// 		if !selected[idx] {
// 			selected[idx] = true
// 			result = append(result, slice[idx])
// 		}
// 	}
// 	return result
// }
