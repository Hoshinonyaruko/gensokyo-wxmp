package template

const ConfigTemplate = `
version: 1
settings:
  #反向ws设置
  ws_address: ["ws://<YOUR_WS_ADDRESS>:<YOUR_WS_PORT>"] # WebSocket服务的地址 支持多个["","",""]
  ws_token: ["","",""]                                 # 连接wss地址时服务器所需的token,按顺序一一对应,如果是ws地址,没有密钥,请留空.
  wxAppId: "12345"                                       # 可选; 公众号的AppId, 如果设置了值则安全模式时该Server只能处理 AppId 为该值的公众号的消息(事件);
  wxToken: "<YOUR_APP_TOKEN>"                          # 必须; 公众号用于验证签名的token;
  wxAppSecret : ""                                     # 开发者密码是校验公众号开发者身份的密码，具有极高的安全性。切记勿把密码直接交给第三方开发者或直接存储在代码中。如需第三方代开发公众号，请使用授权方式接入。
  wxEncodedAESKey: "<YOUR_CLIENT_SECRET>"              # 可选; aes加密解密key, 43字节长(base64编码, 去掉了尾部的'='), 安全模式必须设置;
  wxOriId: ""                                          # 可选; 公众号的原始ID(微信公众号管理后台查看), 如果设置了值则该Server只能处理 ToUserName 为该值的公众号的消息(事件);
  wxPort: 80                                           # 默认:80 非80端口需配置nginx 将80端口的任意path 指向所配置端口的/wx_callback端点
  forwardPort : ["","",""]                             # 配置后,会监听/数字 然后将收到的webhook转发到本地的数字端口的/wx_callback 实现一拖n          

  timeOut : 4                                          # 等待反向ws信息超时时间,默认4秒,当超时时,可以触发默认回复,引导用户。
  long_query_commands : ["","",""]                     # 允许一些长时间指令等待更长时间,如gpt,ai绘图,vits语音指令(微信服务器每5秒重试发一次,一共3次,所以最长等待15秒。)
  subscribe_msg_type : 0                               # 默认0 代表不发欢迎信息 1=从subscribe_msgs随机 2=随机一个subscribe_msgs发给反向ws
  subscribe_msgs : ["","",""]                          # 设置需要发送给用户或服务端的欢迎信息(发给服务端,比如 /help 然后会把服务端的返回回复发给用户)

  global_group_or_private: true                      # 公众号收到的是群聊信息还是私聊 默认true=群
  array: false                                       # 连接trss云崽请开启array
  hash_id : false                                    # 使用hash来进行idmaps转换,可以让user_id不是123开始的递增值

  server_dir: "<YOUR_SERVER_DIR>"                    # 提供图片上传服务的服务器(图床)需要带端口号. 如果需要发base64图,需为公网ip,且开放对应端口
  port: "15630"                                      # idmaps和图床对外开放的端口号
  backup_port : "5200"                               # 当totus为ture时,port值不再是本地webui的端口,使用lotus_Port来访问webui

  lotus: false                                       # lotus特性默认为false,当为true时,将会连接到另一个lotus为false的gensokyo。
                                                     # 使用它提供的图床和idmaps服务(场景:同一个机器人在不同服务器运行,或内网需要发送base64图)。
                                                     # 如果需要发送base64图片,需要设置正确的公网server_dir和开放对应的port
  lotus_password : ""                                # lotus鉴权 设置后,从gsk需要保持相同密码来访问主gsk

  #增强配置项                                           

  image_sizelimit : 0               #代表kb 腾讯api要求图片1500ms完成传输 如果图片发不出 请提升上行或设置此值 默认为0 不压缩
  image_limit : 100                 #每分钟上传的最大图片数量,可自行增加
  master_id : ["1","2"]             #群场景尚未开放获取管理员和列表能力,手动从日志中获取需要设置为管理,的user_id并填入(适用插件有权限判断场景)
  record_bitRate : "128k"           #语音文件的比特率 128kbps
  card_nick : ""                    #默认为空,连接mirai-overflow时,请设置为非空,这里是机器人对用户称谓,为空为插件获取,mirai不支持
  
  reconnect_times : 100             #反向ws连接失败后的重试次数,希望一直重试,可设置9999
  heart_beat_interval : 10          #反向ws心跳间隔 单位秒 推荐5-10
  launch_reconnect_times : 1        #启动时尝试反向ws连接次数,建议先打开应用端再开启gensokyo,因为启动时连接会阻塞webui启动,默认只连接一次,可自行增大
  native_ob11 : false               #如果你的机器人收到事件报错,请开启此选项增加兼容性
  ramdom_seq : false                #当多开gensokyo时,如果遇到群信息只能发出一条,请开启每个gsk的此项.(建议使用一个gsk连接多个应用)
  url_to_qrimage : false            #将信息中的url转换为二维码单独作为图片发出,需要同时设置  #SSL配置类 机器人发送URL设置 的 transfer_url 为 true visible_ip也需要为true
  qr_size : 200                     #二维码尺寸,单位像素
  default_content : ["",""]         #微信公众号机器人是自由触发,当后端没有返回数据时,用户易流失,可设置多个随机的兜底回复,提示帮助指令,文档等.如果发不出,请减小timeOut的值
  default_daily_reply_limit: 3      #设置0=关闭 每天对一位用户发送default_content的次数上限,发送太多默认回复容易让用户厌烦,尤其是老用户.
  
  oss_type : 0                      #请完善后方具体配置 完成#腾讯云配置...,0代表配置server dir port服务器自行上传(省钱),1,腾讯cos存储桶 2,百度oss存储桶 3,阿里oss存储桶
  self_introduce : ["",""]          #自我介绍,可设置多个随机发送,当不为空时,机器人被邀入群会发送自定义自我介绍 需手动添加新textintent   - "GroupAddRobotEventHandler"   - "GroupDelRobotEventHandler"

  #正向ws设置
  ws_server_path : "ws"             #默认监听0.0.0.0:port/ws_server_path 若有安全需求,可不放通port到公网,或设置ws_server_token 若想监听/ 可改为"",若想监听到不带/地址请写nil
  enable_ws_server: true            #是否启用正向ws服务器 监听server_dir:port/ws_server_path
  ws_server_token : "12345"         #正向ws的token 不启动正向ws可忽略 可为空

  #SSL配置类 机器人发送URL设置

  identify_file : true               #自动生成域名校验文件,在q.qq.com配置信息URL,在server_dir填入自己已备案域名,正确解析到机器人所在服务器ip地址,机器人即可发送链接
  identify_appids : []               #默认不需要设置,完成SSL配置类+server_dir设置为域名+完成备案+ssl全套设置后,若有多个机器人需要过域名校验(自己名下)可设置,格式为,整数appid,组成的数组
  crt : ""                           #证书路径 从你的域名服务商或云服务商申请签发SSL证书(qq要求SSL) 
  key : ""                           #密钥路径 Apache（crt文件、key文件）示例: "C:\\123.key" \需要双写成\\
  transfer_url : true                #默认开启,关闭后自理url发送,配置server_dir为你的域名,配置crt和key后,将域名/url和/image在q.qq.com后台通过校验,自动使用302跳转处理机器人发出的所有域名.

  #日志类

  developer_log : false             #开启开发者日志 默认关闭
  log_level : 1                     # 0=debug 1=info 2=warning 3=error 默认1
  save_logs : false                 #自动储存日志

  #webui设置

  server_user_name : "useradmin"    #默认网页面板用户名
  server_user_password : "admin"    #默认网页面板密码

  #指令过滤类

  white_prefix_mode : false         #公域 过审用 指令白名单模式开关 如果审核严格 请开启并设置白名单指令 以白名单开头的指令会被通过,反之被拦截
  v_white_prefix_mode : true        #虚拟二级指令(虚拟前缀)的白名单,默认开启,有二级指令帮助的效果
  white_prefixs : [""]              #可设置多个 比如设置 机器人 测试 则只有信息以机器人 测试开头会相应 remove_prefix remove_at 需为true时生效
  white_bypass : []                 #格式[1,2,3],白名单不生效的群或用户(私聊时),用于设置自己的灰度沙箱群/灰度沙箱私聊,避免开发测试时反复开关白名单的不便,请勿用于生产环境.
  
  white_bypass_reverse : false      #反转white_bypass的效果,可仅在white_bypass应用white_prefix_mode,场景:您的不同用户群,可以开放不同层次功能,便于您的运营和规化(测试/正式环境)
  No_White_Response : ""            #默认不兜底,强烈建议设置一个友善的兜底回复,告知审核机器人已无隐藏指令,如:你输入的指令不对哦,@机器人来获取可用指令

  black_prefix_mode : false         #公私域 过审用 指令黑名单模式开关 过滤被审核打回的指令不响应 无需改机器人后端
  black_prefixs : [""]              #可设置多个 比如设置 查询 则查询开头的信息均被拦截 防止审核失败
  alias : ["",""]                   #两两成对,指令替换,"a","b","c","d"代表将a开头替换为b开头,c开头替换为d开头.
  enters : ["",""]                  #自动md卡片点击直接触发,小众功能,满足以下条件:应用端支持双向echo+设置了visual_prefixs和whiteList

  visual_prefixs :                  #虚拟前缀 与white_prefixs配合使用 处理流程自动忽略该前缀 remove_prefix remove_at 需为true时生效
  - prefix: ""                      #虚拟前缀开头 例 你有3个指令 帮助 测试 查询 将 prefix 设置为 工具类 后 则可通过 工具类 帮助 触发机器人
    whiteList: [""]                 #开关状态取决于 white_prefix_mode 为每一个二级指令头设计独立的白名单
    No_White_Response : ""          #带有*代表不忽略掉,但是应用二级白名单的普通指令   如果  whiteList=邀请机器人 就是邀请机器人按钮                    
  - prefix: ""
    whiteList: [""]
    No_White_Response : "" 
  - prefix: ""
    whiteList: [""]
    No_White_Response : "" 

  #开发增强类

  send_error : true                 #将报错用文本发出,避免机器人被审核报无响应
  url_pic_transfer : false          #把图片url(任意来源图链)变成你备案的白名单url 需要较高上下行+ssl+自备案域名+设置白名单域名(暂时不需要)
  idmap_pro : false                 #需开启hash_id配合,高级id转换增强,可以多个真实值bind到同一个虚拟值,对于每个用户,每个群\私聊\判断私聊\频道,都会产生新的虚拟值,但可以多次bind,bind到同一个数字.数据库负担会变大.
  send_delay : 300                  #单位 毫秒 默认300ms 可以视情况减少到100或者50
  string_ob11 : false

  title : "Gensokyo © 2023 - Hoshinonyaruko"              #程序的标题 如果多个机器人 可根据标题区分
  custom_bot_name : "Gensokyo全域机器人"                   #自定义机器人名字,会在api调用中返回,默认Gensokyo全域机器人
 
  twoway_echo : false               #是否采用双向echo,根据机器人选择,獭獭\早苗 true 红色问答\椛椛 或者其他 请使用 false
  forward_msg_limit : 3             #发送折叠转发信息时的最大限制条数 若要发转发信息 请设置lazy_message_id为true
  transform_api_ids : true          #对get_group_menmber_list\get_group_member_info\get_group_list生效,是否在其中返回转换后的值(默认转换,不转换请自行处理插件逻辑,比如调用gsk的http api转换)
  
  #bind指令类 

  bind_prefix : "/bind"             #需设置   #增强配置项  master_id 可触发
  me_prefix : "/me"                 #需设置   #增强配置项  master_id 可触发
  unlock_prefix : "/unlock"         #频道私信卡住了? gsk可以帮到你 在任意子频道发送unlock 你会收到来自机器人的频道私信
  link_prefix : "/link"             #友情链接配置 配置custom_template_id后可用(https://www.yuque.com/km57bt/hlhnxg/tzbr84y59dbz6pib)
  link_bots : ["",""]               #发送友情链接时 下方按钮携带的机器人 格式 "appid-qq-name","appid-qq-name"
  link_text : ""                    #友情链接文本 不可为空!
  link_pic : ""                     #友情链接图片 可为空 需url图片 可带端口 不填可能会有显示错误

  #穿透\cos\oss类配置(可选!)
  frp_port : "0"                    #不使用请保持为0,frp的端口,frp有内外端口,请在frp软件设置gensokyo的port,并将frp显示的对外端口填入这里

  #HTTP API配置

  #正向http
  http_address: ""                  #http监听地址 与websocket独立 示例:0.0.0.0:5700 为空代表不开启
  http_version : 11                 #暂时只支持11
  http_timeout: 5                   #反向 HTTP 超时时间, 单位秒，<5 时将被忽略

  #反向http
  post_url: [""]                    #反向HTTP POST地址列表 为空代表不开启 示例:http://192.168.0.100:5789
  post_secret: [""]                 #密钥
  post_max_retries: [3]             #最大重试,0 时禁用
  post_retries_interval: [1500]     #重试时间,单位毫秒,0 时立即

  #腾讯云配置
  t_COS_BUCKETNAME : ""             #存储桶名称
  t_COS_REGION : ""                 #COS_REGION 所属地域()内的复制进来 可以在控制台查看 https://console.cloud.tencent.com/cos5/bucket, 关于地域的详情见 https://cloud.tencent.com/document/product/436/6224
  t_COS_SECRETID : ""               #用户的 SecretId,建议使用子账号密钥,授权遵循最小权限指引，降低使用风险。子账号密钥获取可参考 https://cloud.tencent.com/document/product/598/37140
  t_COS_SECRETKEY : ""              #用户的 SECRETKEY 请腾讯云搜索 api密钥管理 生成并填写.妥善保存 避免泄露
  t_audit : false                   #是否审核内容 请先到控制台开启

  #百度云配置
  b_BOS_BUCKETNAME : ""             #百度智能云-BOS控制台-Bucket列表-需要选择的存储桶-域名发布信息-完整官方域名-填入 形如 hellow.gz.bcebos.com
  b_BCE_AK : ""                     #百度 BCE的 AK 获取方法 https://cloud.baidu.com/doc/BOS/s/Tjwvyrw7a 
  b_BCE_SK : ""                     #百度 BCE的 SK 
  b_audit : 0                       #0 不审核 仅使用oss, 1 使用oss+审核, 2 不使用oss 仅审核

  #阿里云配置
  a_OSS_EndPoint : ""               #形如 https://oss-cn-hangzhou.aliyuncs.com 这里获取 https://oss.console.aliyun.com/bucket/oss-cn-shenzhen/sanaee/overview
  a_OSS_BucketName : ""             #要使用的桶名称,上方EndPoint不包含这个名称,如果有,请填在这里
  a_OSS_AccessKeyId : ""            #阿里云控制台-最右上角点击自己头像-AccessKey管理-然后管理和生成
  a_OSS_AccessKeySecret : ""
  a_audit : false                   #是否审核图片 请先开通阿里云内容安全需企业认证。具体操作 请参见https://help.aliyun.com/document_detail/69806.html

  #腾讯对话开放平台配置
  chatbot_appid : ""                #https://chatbot.weixin.qq.com/
  chatbot_token : ""
  chatbot_aeskey : ""
`
const Logo = `
'                                                                                                      
'    ,hakurei,                                                      ka                                  
'   ho"'     iki                                                    gu                                  
'  ra'                                                              ya                                  
'  is              ,kochiya,    ,sanae,    ,Remilia,   ,Scarlet,    fl   and  yu        ya   ,Flandre,   
'  an      Reimu  'Dai   sei  yas     aka  Rei    sen  Ten     shi  re  sca    yu      ku'  ta"     "ko  
'  Jun        ko  Kirisame""  ka       na    Izayoi,   sa       ig  Koishi       ko   mo'   ta       ga  
'   you.     rei  sui   riya  ko       hi  Ina    baI  'ran   you   ka  rlet      komei'    "ra,   ,sa"  
'     "Marisa"      Suwako    ji       na   "Sakuya"'   "Cirno"'    bu     sen     yu''        Satori  
'                                                                                ka'                   
'                                                                               ri'                    
`
