package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chanxuehong/wechat/mp/base"
	"github.com/chanxuehong/wechat/mp/core"
	"github.com/chanxuehong/wechat/mp/menu"
	"github.com/chanxuehong/wechat/mp/message/callback/request"
	"github.com/chanxuehong/wechat/mp/message/callback/response"
	"github.com/fatih/color"
	"github.com/gin-gonic/gin"
	"github.com/hoshinonyaruko/gensokyo-wxmp/Processor"
	"github.com/hoshinonyaruko/gensokyo-wxmp/callapi"
	"github.com/hoshinonyaruko/gensokyo-wxmp/config"
	"github.com/hoshinonyaruko/gensokyo-wxmp/handlers"
	"github.com/hoshinonyaruko/gensokyo-wxmp/idmap"
	"github.com/hoshinonyaruko/gensokyo-wxmp/images"
	"github.com/hoshinonyaruko/gensokyo-wxmp/mylog"
	"github.com/hoshinonyaruko/gensokyo-wxmp/server"
	"github.com/hoshinonyaruko/gensokyo-wxmp/silk"
	"github.com/hoshinonyaruko/gensokyo-wxmp/sys"
	"github.com/hoshinonyaruko/gensokyo-wxmp/template"
	"github.com/hoshinonyaruko/gensokyo-wxmp/url"
	"github.com/hoshinonyaruko/gensokyo-wxmp/webui"
	"github.com/hoshinonyaruko/gensokyo-wxmp/wsclient"
)

var msgIdToEchoMap = make(map[int64]string)
var msgIdToEchoMapMutex = &sync.Mutex{}

var wsClients []*wsclient.WebSocketClient

// UserReplyCountMap 用于存储每个用户的默认回复计数
var UserReplyCountMap sync.Map

var (
	msgHandler        core.Handler
	msgServer         *core.Server
	wechatClient      *core.Client
	accessTokenServer *core.DefaultAccessTokenServer
)

func main() {
	//log.Println(http.ListenAndServe(":80", nil))
	// 定义faststart命令行标志。默认为false。
	fastStart := flag.Bool("faststart", false, "start without initialization if set")
	// 解析命令行参数到定义的标志。
	flag.Parse()
	// 检查是否使用了-faststart参数
	if !*fastStart {
		sys.InitBase() // 如果不是faststart模式，则执行初始化
	}
	if _, err := os.Stat("config.yml"); os.IsNotExist(err) {
		var ip string
		var err error
		// 检查操作系统是否为Android
		if runtime.GOOS == "android" {
			ip = "127.0.0.1"
		} else {
			// 获取内网IP地址
			ip, err = sys.GetLocalIP()
			if err != nil {
				log.Println("Error retrieving the local IP address:", err)
				ip = "127.0.0.1"
			}
		}
		// 将 <YOUR_SERVER_DIR> 替换成实际的内网IP地址 确保初始状态webui能够被访问
		configData := strings.Replace(template.ConfigTemplate, "<YOUR_SERVER_DIR>", ip, -1)

		// 将修改后的配置写入 config.yml
		err = os.WriteFile("config.yml", []byte(configData), 0644)
		if err != nil {
			log.Println("Error writing config.yml:", err)
			return
		}

		log.Println("请配置config.yml然后再次运行.")
		log.Print("按下 Enter 继续...")
		bufio.NewReader(os.Stdin).ReadBytes('\n')
		os.Exit(0)
	}
	// 主逻辑
	// 加载配置
	conf, err := config.LoadConfig("config.yml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	sys.SetTitle(conf.Settings.Title)
	webuiURL := config.ComposeWebUIURL(conf.Settings.Lotus)     // 调用函数获取URL
	webuiURLv2 := config.ComposeWebUIURLv2(conf.Settings.Lotus) // 调用函数获取URL

	// 首先注册处理函数
	http.HandleFunc("/wx_callback", wxCallbackHandler)

	// 从配置中获取端口号并转换为字符串
	wxport := strconv.Itoa(conf.Settings.WxPort) // 将 int 类型端口转换为字符串

	// 在新的 goroutine 中启动 HTTP 服务器
	go func() {
		log.Printf("HTTP server is starting on :%s\n", wxport)
		err := http.ListenAndServe(":"+wxport, nil) // 使用 wxport 作为监听端口
		if err != nil {
			log.Fatalf("HTTP server failed to start: %v", err)
		}
	}()

	// wx逻辑
	accessTokenServer = core.NewDefaultAccessTokenServer(conf.Settings.WxAppId, conf.Settings.WxAppSecret, nil)
	wechatClient = core.NewClient(accessTokenServer, nil)

	fmt.Println("目前地址列表:")
	fmt.Println(base.GetCallbackIP(wechatClient))

	mux := core.NewServeMux()
	mux.DefaultMsgHandleFunc(defaultMsgHandler)
	mux.DefaultEventHandleFunc(defaultEventHandler)
	mux.MsgHandleFunc(request.MsgTypeText, textMsgHandler)
	mux.EventHandleFunc(menu.EventTypeClick, menuClickEventHandler)

	msgHandler = mux
	msgServer = core.NewServer(conf.Settings.WxOriId, conf.Settings.WxAppId, conf.Settings.WxToken, conf.Settings.WxEncodedAESKey, msgHandler, nil)

	// 启动多个WebSocket客户端的逻辑
	if !allEmpty(conf.Settings.WsAddress) {
		wsClientChan := make(chan *wsclient.WebSocketClient, len(conf.Settings.WsAddress))
		errorChan := make(chan error, len(conf.Settings.WsAddress))
		// 定义计数器跟踪尝试建立的连接数
		attemptedConnections := 0
		for _, wsAddr := range conf.Settings.WsAddress {
			if wsAddr == "" {
				continue // Skip empty addresses
			}
			attemptedConnections++ // 增加尝试连接的计数
			go func(address string) {
				retry := config.GetLaunchReconectTimes()
				wxappidint := config.ExtractAndTruncateDigits(conf.Settings.WxAppId)
				wsClient, err := wsclient.NewWebSocketClient(address, wxappidint, retry)
				if err != nil {
					log.Printf("Error creating WebSocketClient for address(连接到反向ws失败) %s: %v\n", address, err)
					errorChan <- err
					return
				}
				wsClientChan <- wsClient
			}(wsAddr)
		}
		// 获取连接成功后的wsClient
		for i := 0; i < attemptedConnections; i++ {
			select {
			case wsClient := <-wsClientChan:
				wsClients = append(wsClients, wsClient)
			case err := <-errorChan:
				log.Printf("Error encountered while initializing WebSocketClient: %v\n", err)
			}
		}

		// 确保所有尝试建立的连接都有对应的wsClient
		if len(wsClients) == 0 {
			log.Println("Error: Not all wsClients are initialized!(反向ws未设置或全部连接失败)")
		} else {
			log.Println("All wsClients are successfully initialized.")
		}
	} else if conf.Settings.EnableWsServer {
		log.Println("只启动正向ws")

	}

	//创建idmap服务器 数据库
	idmap.InitializeDB()
	//创建webui数据库
	webui.InitializeDB()
	defer idmap.CloseDB()
	defer webui.CloseDB()

	//图片上传 调用次数限制
	rateLimiter := server.NewRateLimiter()
	// 根据 lotus 的值选择端口
	var serverPort string
	if !conf.Settings.Lotus {
		serverPort = conf.Settings.Port
	} else {
		serverPort = conf.Settings.BackupPort
	}
	var r *gin.Engine

	if config.GetDeveloperLog() { // 是否启动调试状态
		r = gin.Default()

	} else {
		r = gin.New()
		r.Use(gin.Recovery()) // 添加恢复中间件，但不添加日志中间件

	}
	r.GET("/getid", server.GetIDHandler)
	r.GET("/updateport", server.HandleIpupdate)
	r.POST("/uploadpic", server.UploadBase64ImageHandler(rateLimiter))
	r.POST("/uploadrecord", server.UploadBase64RecordHandler(rateLimiter))
	r.Static("/channel_temp", "./channel_temp")

	//webui和它的api
	webuiGroup := r.Group("/webui")
	{
		webuiGroup.GET("/*filepath", webui.CombinedMiddleware())
		webuiGroup.POST("/*filepath", webui.CombinedMiddleware())
		webuiGroup.PUT("/*filepath", webui.CombinedMiddleware())
		webuiGroup.DELETE("/*filepath", webui.CombinedMiddleware())
		webuiGroup.PATCH("/*filepath", webui.CombinedMiddleware())
	}

	//正向http api
	// http_api_address := config.GetHttpAddress()
	// if http_api_address != "" {
	// 	mylog.Println("正向http api启动成功,监听" + http_api_address + "若有需要,请对外放通端口...")
	// 	HttpApiGroup := hr.Group("/")
	// 	{
	// 		HttpApiGroup.GET("/*filepath", httpapi.CombinedMiddleware())
	// 		HttpApiGroup.POST("/*filepath", httpapi.CombinedMiddleware())
	// 		HttpApiGroup.PUT("/*filepath", httpapi.CombinedMiddleware())
	// 		HttpApiGroup.DELETE("/*filepath", httpapi.CombinedMiddleware())
	// 		HttpApiGroup.PATCH("/*filepath", httpapi.CombinedMiddleware())
	// 	}
	// }

	r.POST("/url", url.CreateShortURLHandler)
	r.GET("/url/:shortURL", url.RedirectFromShortURLHandler)
	//if config.GetIdentifyFile() {
	// appIDStr := config.GetAppIDStr()
	// fileName := appIDStr + ".json"
	// r.GET("/"+fileName, func(c *gin.Context) {
	// 	content := fmt.Sprintf(`{"bot_appid":%d}`, config.GetAppID())
	// 	c.Header("Content-Type", "application/json")
	// 	c.String(200, content)
	// })

	// // 调用 config.GetIdentifyAppids 获取 appid 数组
	// identifyAppids := config.GetIdentifyAppids()

	// // 如果 identifyAppids 不是 nil 且有多个元素
	// if len(identifyAppids) >= 1 {
	// 	// 从数组中去除 config.GetAppID() 来避免重复
	// 	var filteredAppids []int64
	// 	for _, appid := range identifyAppids {
	// 		if appid != int64(config.GetAppID()) {
	// 			filteredAppids = append(filteredAppids, appid)
	// 		}
	// 	}

	// 	// 为每个 appid 设置路由
	// 	for _, appid := range filteredAppids {
	// 		fileName := fmt.Sprintf("%d.json", appid)
	// 		r.GET("/"+fileName, func(c *gin.Context) {
	// 			content := fmt.Sprintf(`{"bot_appid":%d}`, appid)
	// 			c.Header("Content-Type", "application/json")
	// 			c.String(200, content)
	// 		})
	// 	}
	// }
	//}

	//创建一个http.Server实例（主服务器）
	httpServer := &http.Server{
		Addr:    "0.0.0.0:" + serverPort,
		Handler: r,
	}

	mylog.Printf("gin运行在%v端口", serverPort)
	// 在一个新的goroutine中启动主服务器
	go func() {
		if serverPort == "443" {
			// 使用HTTPS
			crtPath := config.GetCrtPath()
			keyPath := config.GetKeyPath()
			if crtPath == "" || keyPath == "" {
				log.Fatalf("crt or key path is missing for HTTPS")
				return
			}
			if err := httpServer.ListenAndServeTLS(crtPath, keyPath); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen (HTTPS): %s\n", err)
			}
		} else {
			// 使用HTTP
			if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %s\n", err)
			}
		}
	}()

	// 如果主服务器使用443端口，同时在一个新的goroutine中启动444端口的HTTP服务器 todo 更优解
	if serverPort == "443" {
		go func() {
			// 创建另一个http.Server实例（用于444端口）
			httpServer444 := &http.Server{
				Addr:    "0.0.0.0:444",
				Handler: r,
			}

			// 启动444端口的HTTP服务器
			if err := httpServer444.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen (HTTP 444): %s\n", err)
			}
		}()
	}
	// 创建 httpapi 的http server
	// if http_api_address != "" {
	// 	go func() {
	// 		// 创建一个http.Server实例（Http Api服务器）
	// 		httpServerHttpApi := &http.Server{
	// 			Addr:    http_api_address,
	// 			Handler: hr,
	// 		}
	// 		// 使用HTTP
	// 		if err := httpServerHttpApi.ListenAndServe(); err != nil && err != http.ErrServerClosed {
	// 			log.Fatalf("http apilisten: %s\n", err)
	// 		}
	// 	}()
	// }

	// 使用color库输出天蓝色的文本
	cyan := color.New(color.FgCyan)
	cyan.Printf("欢迎来到Gensokyo, 控制台地址: %s\n", webuiURL)
	cyan.Printf("%s\n", template.Logo)
	cyan.Printf("欢迎来到Gensokyo, 公网控制台地址(需开放端口): %s\n", webuiURLv2)

	// 使用通道来等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞主线程，直到接收到信号
	<-sigCh

	// 关闭 WebSocket 连接
	// wsClients 是一个 *wsclient.WebSocketClient 的切片
	for _, client := range wsClients {
		err := client.Close()
		if err != nil {
			log.Printf("Error closing WebSocket connection: %v\n", err)
		}
	}

	// 关闭BoltDB数据库
	url.CloseDB()
	idmap.CloseDB()

	// 使用一个5秒的超时优雅地关闭Gin服务器
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
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

func textMsgHandler(ctx *core.Context) {
	var err error
	log.Printf("收到文本消息:\n%s\n", ctx.MsgPlaintext)
	// 解析信息
	msg := request.GetText(ctx.MixedMsg)

	msgIdToEchoMapMutex.Lock()
	echo, exists := msgIdToEchoMap[msg.MsgId]
	msgIdToEchoMapMutex.Unlock()

	// 如果msgId不存在，则处理消息
	if !exists {
		// 根据配置调用相应的处理函数
		if config.GetGlobalGroupOrPrivate() {
			echo, err = Processor.ProcessGroupMessage(ctx, wsClients)
			if err != nil {
				log.Printf("处理信息出错:\n%v\n", err)
			}
		} else {
			echo, err = Processor.ProcessC2CMessage(ctx, wsClients)
			if err != nil {
				log.Printf("处理信息出错:\n%v\n", err)
			}
		}

		// 存储msgId和echo的映射关系
		msgIdToEchoMapMutex.Lock()
		msgIdToEchoMap[msg.MsgId] = echo
		msgIdToEchoMapMutex.Unlock()
	}
	var message *callapi.ActionMessage
	// 首先获取超时时间和长查询命令列表
	timeout := config.GetTimeOut()
	longQueryCommands := config.GetLongQueryCommands()

	// 检查消息是否以长查询命令开头
	for _, cmd := range longQueryCommands {
		// 如果cmd是空字符串，则跳过不检查
		if cmd == "" {
			continue
		}

		if strings.HasPrefix(ctx.MixedMsg.Content, cmd) {
			// 如果是长查询命令，将超时时间设置为15秒
			timeout = 15
			log.Printf("LongQueryCommand detected: %s", cmd)
			break
		}
	}

	if config.GetTwoWayEcho() {
		// 发送消息给WS接口，并等待响应
		message, err = wsclient.WaitForActionMessage(echo, time.Duration(timeout)*time.Second) // 使用新的超时时间
		if err != nil {
			log.Printf("Error waiting for action message: %v", err)
			// 处理错误...
			message = DefaultReplyIfNeeded(ctx.MixedMsg.MsgHeader.FromUserName)
			if message.Params.Message.(string) == "" {
				log.Printf("默认信息为空,请到config设置,或该用户今日已达到默认回复上限.")
				return
			}
		}
	} else {
		// 发送消息给WS接口，并等待响应
		message, err = wsclient.WaitForGeneralMessage(time.Duration(timeout) * time.Second) // 使用新的超时时间
		if err != nil {
			log.Printf("Error waiting for action message: %v", err)
			// 处理错误...
			message = DefaultReplyIfNeeded(ctx.MixedMsg.MsgHeader.FromUserName)
			if message.Params.Message.(string) == "" {
				log.Printf("默认信息为空,请到config设置,或该用户今日已达到默认回复上限.")
				return
			}
		}
	}

	var strmessage string
	// 尝试将message.Params.Message断言为string类型
	if msgStr, ok := message.Params.Message.(string); ok {
		strmessage = msgStr
	} else {
		// 如果不是string，调用parseMessage函数处理
		strmessage = handlers.ParseMessageContent(message.Params.Message)
	}
	// 调用信息处理函数
	messageType, result, err := ProcessMessage(strmessage, wechatClient)
	if err != nil {
		// 处理错误情况
		// 例如，可以回复一个文本消息表明无法处理该消息
		log.Printf("Error ProcessMessage: %v", err)
		resp := response.NewText(msg.FromUserName, msg.ToUserName, msg.CreateTime, "无法处理您的消息")
		ctx.AESResponse(resp, 0, "", nil) // aes密文回复
		return
	}
	// 根据信息处理函数的返回类型决定如何回复
	switch messageType {
	case 1: // 纯文本信息
		textContent := result.(string) // 类型断言
		resp := response.NewText(msg.FromUserName, msg.ToUserName, msg.CreateTime, textContent)
		ctx.AESResponse(resp, 0, "", nil) // aes密文回复
	case 2: // 纯图片信息
		mediaId := result.(string) // 类型断言
		resp := response.NewImage(msg.FromUserName, msg.ToUserName, msg.CreateTime, mediaId)
		ctx.AESResponse(resp, 0, "", nil) // aes密文回复
	case 3: // 纯语音信息
		mediaId := result.(string) // 类型断言
		resp := response.NewVoice(msg.FromUserName, msg.ToUserName, msg.CreateTime, mediaId)
		ctx.AESResponse(resp, 0, "", nil) // aes密文回复
	case 4: // 图文信息
		articles := result.(response.News).Articles // 类型断言
		resp := response.NewNews(msg.FromUserName, msg.ToUserName, msg.CreateTime, articles)
		ctx.AESResponse(resp, 0, "", nil) // aes密文回复
	// 添加更多case处理其他情况
	default:
		// 未知类型，可以回复一个默认消息
		resp := response.NewText(msg.FromUserName, msg.ToUserName, msg.CreateTime, "未知的消息类型")
		ctx.AESResponse(resp, 0, "", nil) // aes密文回复
	}

	//发送成功回执
	handlers.SendResponse(wsClients, err, message)
}

func defaultMsgHandler(ctx *core.Context) {
	log.Printf("收到消息:\n%s\n", ctx.MsgPlaintext)
	ctx.NoneResponse()
}

func menuClickEventHandler(ctx *core.Context) {
	log.Printf("收到菜单 click 事件:\n%s\n", ctx.MsgPlaintext)

	event := menu.GetClickEvent(ctx.MixedMsg)
	resp := response.NewText(event.FromUserName, event.ToUserName, event.CreateTime, "收到 click 类型的事件")
	//ctx.RawResponse(resp) // 明文回复
	ctx.AESResponse(resp, 0, "", nil) // aes密文回复
}

// 被关注
func defaultEventHandler(ctx *core.Context) {
	var err error
	log.Printf("收到事件:\n%s\n", ctx.MsgPlaintext)
	if ctx.MixedMsg.EventType == "subscribe" {
		// 解析信息
		msg := request.GetText(ctx.MixedMsg)
		// 获取订阅消息类型和可能的消息列表
		subscribeMsgType := config.GetSubscribeMsgType()
		subscribeMsgs := config.GetSubscribeMsgs()
		if len(subscribeMsgs) == 0 {
			log.Printf("No subscribe messages configured.")
			return
		}

		// 从列表中随机选择一条消息
		randomMsg := subscribeMsgs[rand.Intn(len(subscribeMsgs))]

		switch subscribeMsgType {
		case 1:
			log.Printf("Sending subscribe message: %s", randomMsg)
		case 2:
			log.Printf("Assigning random message for further processing: %s", randomMsg)
			// 这将随机消息赋值给ctx.MixedMsg.Content，供后续处理
			ctx.MixedMsg.Content = randomMsg
			//subscribe信息是没有msgid的官方建议用时间做区分
			msgIdToEchoMapMutex.Lock()
			echo, exists := msgIdToEchoMap[msg.CreateTime]
			msgIdToEchoMapMutex.Unlock()

			// 如果msgId不存在，则处理消息
			if !exists {
				// 根据配置调用相应的处理函数
				if config.GetGlobalGroupOrPrivate() {
					echo, err = Processor.ProcessGroupMessage(ctx, wsClients)
					if err != nil {
						log.Printf("处理信息出错:\n%v\n", err)
					}
				} else {
					echo, err = Processor.ProcessC2CMessage(ctx, wsClients)
					if err != nil {
						log.Printf("处理信息出错:\n%v\n", err)
					}
				}

				// 存储msgId和echo的映射关系
				msgIdToEchoMapMutex.Lock()
				msgIdToEchoMap[msg.CreateTime] = echo
				msgIdToEchoMapMutex.Unlock()
			}
			var message *callapi.ActionMessage
			// 首先获取超时时间和长查询命令列表
			timeout := config.GetTimeOut()
			longQueryCommands := config.GetLongQueryCommands()

			// 检查消息是否以长查询命令开头
			for _, cmd := range longQueryCommands {
				// 如果cmd是空字符串，则跳过不检查
				if cmd == "" {
					continue
				}

				if strings.HasPrefix(ctx.MixedMsg.Content, cmd) {
					// 如果是长查询命令，将超时时间设置为15秒
					timeout = 15
					log.Printf("LongQueryCommand detected: %s", cmd)
					break
				}
			}

			if config.GetTwoWayEcho() {
				// 发送消息给WS接口，并等待响应
				message, err = wsclient.WaitForActionMessage(echo, time.Duration(timeout)*time.Second) // 使用新的超时时间
				if err != nil {
					log.Printf("Error waiting for action message: %v", err)
					// 处理错误...
					message = DefaultReplyIfNeeded(ctx.MixedMsg.MsgHeader.FromUserName)
					if message.Params.Message.(string) == "" {
						log.Printf("默认信息为空,请到config设置,或该用户今日已达到默认回复上限.")
						return
					}
				}
			} else {
				// 发送消息给WS接口，并等待响应
				message, err = wsclient.WaitForGeneralMessage(time.Duration(timeout) * time.Second) // 使用新的超时时间
				if err != nil {
					log.Printf("Error waiting for action message: %v", err)
					// 处理错误...
					message = DefaultReplyIfNeeded(ctx.MixedMsg.MsgHeader.FromUserName)
					if message.Params.Message.(string) == "" {
						log.Printf("默认信息为空,请到config设置,或该用户今日已达到默认回复上限.")
						return
					}
				}
			}
			//再次覆盖randomMsg
			// 尝试将message.Params.Message断言为string类型
			if msgStr, ok := message.Params.Message.(string); ok {
				randomMsg = msgStr
			} else {
				// 如果不是string，调用parseMessage函数处理
				randomMsg = handlers.ParseMessageContent(message.Params.Message)
			}
			//发送成功回执
			handlers.SendResponse(wsClients, err, message)
		default:
			//返回200 ok
			ctx.NoneResponse()
			log.Printf("Unsupported subscribe message type: %d", subscribeMsgType)
			return
		}

		// 调用信息处理函数
		messageType, result, err := ProcessMessage(randomMsg, wechatClient)
		if err != nil {
			// 处理错误情况
			// 例如，可以回复一个文本消息表明无法处理该消息
			log.Printf("Error ProcessMessage: %v", err)
			resp := response.NewText(msg.FromUserName, msg.ToUserName, msg.CreateTime, "无法处理您的消息")
			ctx.AESResponse(resp, 0, "", nil) // aes密文回复
			return
		}
		// 根据信息处理函数的返回类型决定如何回复
		switch messageType {
		case 1: // 纯文本信息
			textContent := result.(string) // 类型断言
			resp := response.NewText(msg.FromUserName, msg.ToUserName, msg.CreateTime, textContent)
			ctx.AESResponse(resp, 0, "", nil) // aes密文回复
		case 2: // 纯图片信息
			mediaId := result.(string) // 类型断言
			resp := response.NewImage(msg.FromUserName, msg.ToUserName, msg.CreateTime, mediaId)
			ctx.AESResponse(resp, 0, "", nil) // aes密文回复
		case 3: // 纯语音信息
			mediaId := result.(string) // 类型断言
			resp := response.NewVoice(msg.FromUserName, msg.ToUserName, msg.CreateTime, mediaId)
			ctx.AESResponse(resp, 0, "", nil) // aes密文回复
		case 4: // 图文信息
			articles := result.(response.News).Articles // 类型断言
			resp := response.NewNews(msg.FromUserName, msg.ToUserName, msg.CreateTime, articles)
			ctx.AESResponse(resp, 0, "", nil) // aes密文回复
		// 添加更多case处理其他情况
		default:
			// 未知类型，可以回复一个默认消息
			resp := response.NewText(msg.FromUserName, msg.ToUserName, msg.CreateTime, "未知的消息类型")
			ctx.AESResponse(resp, 0, "", nil) // aes密文回复
		}
	} else {
		//返回200 ok
		ctx.NoneResponse()
		log.Printf("Received non-subscribe event:\n%s\n", string(ctx.MsgPlaintext))
		return
	}
}

// wxCallbackHandler 是处理回调请求的 http handler.
//  1. 不同的 web 框架有不同的实现
//  2. 一般一个 handler 处理一个公众号的回调请求(当然也可以处理多个, 这里我只处理一个)
func wxCallbackHandler(w http.ResponseWriter, r *http.Request) {
	msgServer.ServeHTTP(w, r, nil)
}

// ProcessMessage 处理信息并归类
func ProcessMessage(input string, clt *core.Client) (int, interface{}, error) {
	// 正则表达式定义
	httpUrlImagePattern := regexp.MustCompile(`\[CQ:image,file=http://(.+?)\]`)
	httpsUrlImagePattern := regexp.MustCompile(`\[CQ:image,file=https://(.+?)\]`)
	base64ImagePattern := regexp.MustCompile(`\[CQ:image,file=base64://(.+?)\]`)
	base64RecordPattern := regexp.MustCompile(`\[CQ:record,file=base64://(.+?)\]`)
	httpUrlRecordPattern := regexp.MustCompile(`\[CQ:record,file=http://(.+?)\]`)
	httpsUrlRecordPattern := regexp.MustCompile(`\[CQ:record,file=https://(.+?)\]`)

	// 检查是否含有base64编码的图片或语音信息
	var err error
	if base64ImagePattern.MatchString(input) || base64RecordPattern.MatchString(input) {
		input, err = processInput(input)
		if err != nil {
			log.Printf("processInput出错:\n%v\n", err)
		}
		log.Printf("处理后的base64编码的图片或语音信息:\n%v\n", input)
	}

	// 检查是否为纯文本信息
	if !httpUrlImagePattern.MatchString(input) && !httpsUrlImagePattern.MatchString(input) && !httpUrlRecordPattern.MatchString(input) && !httpsUrlRecordPattern.MatchString(input) {
		// 使用正则表达式匹配并替换[CQ:at,qq=x]格式的信息
		cqAtPattern := regexp.MustCompile(`\[CQ:at,qq=\d+\]`)
		// 将匹配到的部分替换为空字符串
		filteredInput := cqAtPattern.ReplaceAllString(input, "")

		// 返回过滤后的纯文本信息
		return 1, filteredInput, nil
	}

	// 图片信息处理
	if httpUrlImagePattern.MatchString(input) || httpsUrlImagePattern.MatchString(input) {
		// 合并匹配到的所有图片URL
		httpImageUrls := httpUrlImagePattern.FindAllStringSubmatch(input, -1)
		httpsImageUrls := httpsUrlImagePattern.FindAllStringSubmatch(input, -1)

		// 通过前缀重新构造完整的图片URL
		var imageUrls []string
		for _, match := range httpImageUrls {
			imageUrls = append(imageUrls, "http://"+match[1])
		}
		for _, match := range httpsImageUrls {
			imageUrls = append(imageUrls, "https://"+match[1])
		}

		if len(imageUrls) == 1 {
			// 单图片信息
			mediaId, err := images.ProcessInput(imageUrls[0], clt, "png")
			if err != nil {
				return 0, nil, err
			}
			return 2, mediaId, nil // 纯图片信息
		} else {
			// 图文信息
			var articles []response.Article
			for _, url := range imageUrls {
				articles = append(articles, response.Article{
					PicURL: url, // 这里的url已经是包含正确协议头的完整URL
				})
			}
			news := response.News{
				ArticleCount: len(articles),
				Articles:     articles,
			}
			return 4, news, nil // 图文信息
		}
	}

	// 语音信息处理
	if httpUrlRecordPattern.MatchString(input) || httpsUrlRecordPattern.MatchString(input) {
		// 初始化变量用于存放处理后的URL
		var recordUrl string

		// 查找匹配的HTTP URL
		httpRecordMatches := httpUrlRecordPattern.FindAllStringSubmatch(input, -1)
		if len(httpRecordMatches) > 0 {
			// 取第一个匹配项，并添加HTTP前缀
			recordUrl = "http://" + httpRecordMatches[0][1]
		}

		// 查找匹配的HTTPS URL
		httpsRecordMatches := httpsUrlRecordPattern.FindAllStringSubmatch(input, -1)
		if len(httpsRecordMatches) > 0 {
			// 如果已经找到HTTP URL，优先处理HTTPS URL
			recordUrl = "https://" + httpsRecordMatches[0][1]
		}

		// 如果找到了语音URL
		if recordUrl != "" {
			mediaId, err := images.ProcessInput(recordUrl, clt, "mp3") // 确保ProcessInput可以处理语音URL
			if err != nil {
				mylog.Printf("Failed to ProcessInput record data[%v]: %v", recordUrl, err)
				return 0, nil, err
			}
			return 3, mediaId, nil // 纯语音信息
		}
	}

	// 如果没有匹配到任何已知格式，返回错误
	return 0, nil, errors.New("unknown message format")
}

// processInput 处理含有Base64编码的图片和语音信息的字符串
func processInput(input string) (string, error) {
	// 定义正则表达式
	base64ImagePattern := regexp.MustCompile(`\[CQ:image,file=base64://(.+?)\]`)
	base64RecordPattern := regexp.MustCompile(`\[CQ:record,file=base64://(.+?)\]`)

	// 处理Base64编码的图片
	input = processBase64Media(input, base64ImagePattern, images.UploadBase64ImageToServer, "image")

	// 处理Base64编码的语音
	input = processBase64Media(input, base64RecordPattern, images.UploadBase64RecordToServer, "record")

	return input, nil
}

// processBase64Media 处理并替换Base64编码的媒体信息
func processBase64Media(input string, pattern *regexp.Regexp, uploadFunc func(string) (string, error), mediaType string) string {
	matches := pattern.FindAllStringSubmatch(input, -1)
	for _, match := range matches {
		base64Data := match[1] // 获取Base64编码数据
		decodedData, err := base64.StdEncoding.DecodeString(base64Data)
		if err != nil {
			mylog.Printf("Failed to decode base64 data: %v", err)
			continue
		}

		// 特殊处理语音数据
		if mediaType == "record" && !silk.IsAMRorSILK(decodedData) {
			decodedData = silk.EncoderSilk(decodedData)
			mylog.Printf("Audio transcoding")
			//mylog.Printf("不是amr格式但是不转码.")
		} else {
			mylog.Printf("pic or amr")
		}

		// 将解码的数据重新编码为Base64并上传
		encodedData := base64.StdEncoding.EncodeToString(decodedData)
		url, err := uploadFunc(encodedData)
		if err != nil {
			mylog.Printf("Failed to upload base64 data: %v", err)
			continue
		}
		// 根据媒体类型构造替换格式
		var cqFormat string
		if mediaType == "image" {
			cqFormat = `[CQ:image,file=%s]`
		} else if mediaType == "record" {
			cqFormat = `[CQ:record,file=%s]`
		}

		// 替换原始Base64编码信息为URL
		input = strings.Replace(input, match[0], fmt.Sprintf(cqFormat, url), 1)

	}
	return input
}

// DefaultReplyIfNeeded 发送默认回复（如果需要）
func DefaultReplyIfNeeded(fromUserName string) *callapi.ActionMessage {
	// 获取今天的日期字符串
	today := time.Now().Format("2006-01-02")
	userKey := fmt.Sprintf("%s_%s", fromUserName, today)

	// 获取用户今天的回复计数
	value, ok := UserReplyCountMap.Load(userKey)
	var count int
	if ok {
		count = value.(int)
	}

	// 检查是否已经达到每日回复限制
	if count >= config.GetDefaultDailyReplyLimit() {
		// 构造ActionMessage类型的消息
		actionMessage := &callapi.ActionMessage{
			Action: "send_group_msg",
			Params: callapi.ParamsContent{
				Message: "",
			},
		}
		return actionMessage // 达到限制，返回空字符串
	}

	// 未达到限制，增加计数并选择一个默认回复
	count++
	UserReplyCountMap.Store(userKey, count)

	defaultReplies := config.GetDefaultContent()
	if len(defaultReplies) == 0 {
		log.Println("No default content available.")
		// 构造ActionMessage类型的消息
		actionMessage := &callapi.ActionMessage{
			Action: "send_group_msg",
			Params: callapi.ParamsContent{
				Message: "",
			},
		}
		return actionMessage
	}

	// 随机选择一个默认回复发送
	reply := GetRandomReply(defaultReplies)

	// 构造ActionMessage类型的消息
	actionMessage := &callapi.ActionMessage{
		Action: "send_group_msg",
		Params: callapi.ParamsContent{
			Message: reply,
		},
	}

	fmt.Println("发送默认回复:", reply)

	return actionMessage
}

// GetRandomReply 从提供的回复列表中随机选择一个回复
func GetRandomReply(replies []string) string {
	if len(replies) == 0 {
		return ""
	}
	// 生成一个replies切片长度范围内的随机索引
	index := rand.Intn(len(replies))
	// 返回随机选中的回复
	return replies[index]
}
