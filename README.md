# Openxhh

Openxhh 是一个面向小黑盒的 AI 自动回复机器人。它不是只会看见一句 `@机器人` 的关键词脚本，而是尽量把帖子、楼层、图片、上下文和被点名的人一起读进去，再用 OpenAI 兼容接口生成更像真人接话的回复。

项目地址：<https://github.com/Www8881313/Openxhh>

> 推荐部署方式：**VPS / Linux 版本**。  
> Windows 图形化版本目前仍不完善，不建议普通用户优先安装；如果只是想长期稳定运行，请优先使用 VPS 版本。

## 它解决了什么问题

普通机器人最大的问题是“眼睛太短”：用户说“这个怎么样”“楼上说得对吗”“给小菲画一张”，如果机器人只看到当前这一句话，就很容易答非所问。

Openxhh 的目标是让机器人更像一个真的在看帖的人：

- 它会读帖子标题和正文，而不是只看 @ 这一句。
- 它会利用当前评论楼层上下文，理解“楼上”“对方”“这个”指什么。
- 它会把评论图片、帖子图片作为上下文的一部分，让回复更贴合现场。
- 它支持评论区生图，并且不会再把 `已生成：prompt` 这种生硬文本直接丢到评论区。
- 它会清洗显式点名和小黑盒表情控制字符，尽量避免把 `[cube_喜欢]` 之类内容误当用户名。
- 它带 VPS Web UI，可以看最近对话、失败记录、token 消耗和日志，不用一直盯着终端。
- 它默认回复所有 @，也可以手动开启白名单；同时有并发和队列上限，避免普通用户刷 @ 把机器人拖死。

## 主要功能

- 自动检查小黑盒 @ 消息，并调用 OpenAI 兼容接口回复。
- 默认回复所有 @；开启白名单后只回复 owner / 指定 UID。
- 支持最高回复线程、全局待回复队列、单用户待回复队列上限。
- 支持自定义 AI 接口、模型、token 和 prompt。
- 支持 SQLite / PostgreSQL；个人部署推荐 SQLite。
- 支持小黑盒扫码登录，登录态保存到 `cookie.json`。
- 支持帖子 / 评论楼层上下文增强。
- 支持评论图片和帖子图片进入 AI 上下文。
- 支持评论区 @ 生图：`生图`、`画图`、`生成图片`。
- 支持把生成图片写入 VPS 静态图床，并用 `imgs=<图片URL>` 发布顶级带图评论。
- 支持生图后生成自然短回复，避免暴露 prompt。
- 支持显式点名 @：例如“给小菲画一张”“问问楼主”“对小菲说”。
- 支持 VPS Web UI：配置管理、服务控制、日志筛选、最近对话、失败记录、token 统计。

## 最简单安装：VPS / Linux 推荐版

适合 Ubuntu / Debian VPS。下面只安装 **Openxhh 项目版本**，不包含旧仓库、不包含原版机器人。

```bash
sudo mkdir -p /opt/Openxhh
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo bash
cd /opt/Openxhh
sudo ./Openxhh
sudo nano config.json
sudo ./Openxhh -mode login
sudo ./Openxhh -mode start
```

这段命令会做几件事：

1. 把运行目录放到 `/opt/Openxhh`。
2. 从 `Www8881313/Openxhh` 拉取最新源码。
3. 编译 `Openxhh` 主程序。
4. 编译 `Openxhh-webui` VPS 控制台。
5. 保留你的 `config.json`、`cookie.json`、`sql.db` 和日志。
6. 生成配置文件，按你的 AI / 图片接口信息修改配置。
7. 扫码登录小黑盒。
8. 前台启动机器人。

如果你只是想先跑起来，看到这里就够了。下面是更完整的安装、配置、图床、后台运行和默认值说明。

<details>
<summary>完整 VPS 安装教程</summary>

### 1. 准备系统

推荐系统：Ubuntu / Debian。

安装脚本会自动检查并安装 `git`、`curl`、`gcc`、`libsqlite3-dev`、`snapd` 和 Go。如果你的系统没有 `apt-get`，需要手动安装这些依赖后再编译。

先创建运行目录：

```bash
sudo mkdir -p /opt/Openxhh
```

### 2. 拉取并编译 Openxhh

```bash
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo bash
```

默认会生成或更新：

```text
/opt/Openxhh/Openxhh
/opt/Openxhh/Openxhh-webui
```

脚本会保留以下文件，不会覆盖你的登录态和数据库：

```text
/opt/Openxhh/config.json
/opt/Openxhh/cookie.json
/opt/Openxhh/sql.db
/opt/Openxhh/log/
```

### 3. 生成配置文件

```bash
cd /opt/Openxhh
sudo ./Openxhh
```

首次运行会生成 `config.json` 并退出。编辑配置：

```bash
sudo nano /opt/Openxhh/config.json
```

你至少需要填：

- `xhh.owner`：你的小黑盒数字 UID。
- `ai.model`：AI 回复模型名。
- `ai.baseUrl`：OpenAI 兼容 Chat Completions 完整地址。
- `ai.token`：AI 回复接口 token。
- 如果要生图，再填 `image.baseUrl`、`image.token`、`image.externalDir`、`image.externalBaseUrl`。

### 4. 扫码登录

```bash
cd /opt/Openxhh
sudo ./Openxhh -mode login
```

扫码成功后会生成：

```text
/opt/Openxhh/cookie.json
```

### 5. 前台试运行

```bash
cd /opt/Openxhh
sudo ./Openxhh -mode start
```

确认能正常收到 @、回复和写日志后，再配置 systemd 后台运行。

</details>

<details>
<summary>config.json 配置示例</summary>

个人部署推荐 SQLite。下面是可恢复默认值的完整示例，你可以从这里复制字段修复被误改的配置。

```json
{
  "xhh": {
    "checkTime": 60,
    "replyTime": 30,
    "maxReplyThreads": 3,
    "maxPendingReplies": 50,
    "maxPendingRepliesPerUser": 5,
    "enableWhitelist": false,
    "owner": "你的小黑盒数字UID；多个 UID 用英文逗号分隔",
    "deviceID": "",
    "baseUrl": "https://api.xiaoheihe.cn",
    "webver": "2.5",
    "version": "999.0.4"
  },
  "database": {
    "type": "sqlite",
    "db": "",
    "host": "",
    "port": "",
    "user": "",
    "passwd": ""
  },
  "ai": {
    "model": "你的模型名",
    "prompt": "请根据评论内容自然回复。",
    "baseUrl": "你的 OpenAI 兼容 /v1/chat/completions 地址",
    "token": "你的 AI API Token"
  },
  "image": {
    "model": "gpt-image-2",
    "baseUrl": "你的 OpenAI 兼容 /v1/images/generations 地址",
    "token": "你的图片 API Token",
    "size": "1024x1024",
    "responseFormat": "b64_json",
    "outputDir": "images",
    "uploadMode": "external",
    "externalDir": "/var/www/xhh-images",
    "externalBaseUrl": "http://你的VPS公网IP/xhh-images",
    "promptRefine": false,
    "promptModel": "",
    "promptBaseUrl": "",
    "promptToken": "",
    "promptMaxChars": 1000
  }
}
```

配置要点：

- 默认 `xhh.enableWhitelist=false`，机器人会回复所有 @。
- 如果你只想让 owner / 指定用户触发回复，把 `xhh.enableWhitelist` 改成 `true`。
- `xhh.owner` 填小黑盒数字 UID，不是昵称；多个 UID 用英文逗号分隔。
- 即使白名单关闭，`xhh.owner` 仍会作为 owner 身份上下文注入给 AI。
- 开启白名单后，`xhh.owner` 会自动被允许，不需要重复配置。
- `xhh.maxReplyThreads` 控制同一轮最高回复并发，默认 `3`。
- `xhh.maxPendingReplies` 控制普通用户全局待回复队列上限，默认 `50`。
- `xhh.maxPendingRepliesPerUser` 控制单个普通用户待回复队列上限，默认 `5`。
- owner 不受 `maxPendingReplies` 和 `maxPendingRepliesPerUser` 限制。
- `ai.baseUrl` 要填完整的 Chat Completions 地址，例如 `https://example.com/v1/chat/completions`。
- `image.baseUrl` 要填完整的 Images Generations 地址，例如 `https://example.com/v1/images/generations`。
- `image.responseFormat` 默认 `b64_json`，适合直接拿到图片内容再写入图床目录。
- `image.uploadMode` 默认 `external`，推荐搭配 VPS 静态目录使用。
- `image.externalBaseUrl` 必须是完整公网 URL，例如 `http://你的VPS公网IP/xhh-images`；末尾有没有 `/` 都可以。
- `promptRefine=true` 后，可以用文本模型先把用户口语化生图请求整理成更适合图片模型的提示词。
- 保存配置后需要重启 `Openxhh` 服务才会生效。

</details>

<details>
<summary>VPS Web UI 配置管理</summary>

VPS Web UI 提供“配置管理”页，可以读取和保存：

```text
/opt/Openxhh/config.json
```

它支持编辑：

- 小黑盒配置：检查间隔、回复间隔、白名单开关、owner UID、并发和队列上限。
- 数据库配置：SQLite / PostgreSQL。
- AI 回复配置：模型、Chat Completions URL、token、回复 prompt。
- 图片能力配置：图片模型、Images Generations URL、图片 token、输出格式、上传模式、外部图床目录和访问 URL。
- Prompt 优化配置：是否启用、模型、URL、token、最大字符数。

保存后只会写入 `config.json`，不会自动让运行中的机器人重新读取。改完配置后请在 Web UI 点“重启服务”，或执行：

```bash
sudo systemctl restart Openxhh
```

</details>

<details>
<summary>systemd 后台运行</summary>

创建机器人服务：

```bash
sudo tee /etc/systemd/system/Openxhh.service >/dev/null <<'EOF'
[Unit]
Description=Openxhh
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/Openxhh
ExecStart=/opt/Openxhh/Openxhh -mode start
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now Openxhh
```

查看状态：

```bash
sudo systemctl status Openxhh --no-pager
sudo journalctl -u Openxhh -f
```

</details>

<details>
<summary>VPS Web UI 控制台</summary>

安装脚本会同时编译并更新：

```text
/opt/Openxhh/Openxhh-webui
```

推荐把它作为独立服务运行：

```bash
sudo tee /etc/systemd/system/Openxhh-webui.service >/dev/null <<'EOF'
[Unit]
Description=Openxhh VPS Web UI
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
WorkingDirectory=/opt/Openxhh
ExecStart=/opt/Openxhh/Openxhh-webui -addr :29173 -root /opt/Openxhh -service Openxhh
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now Openxhh-webui
sudo journalctl -u Openxhh-webui -n 50 --no-pager
```

第一次启动会生成随机强密码，日志里会出现：

```text
Openxhh VPS Web UI 已生成随机强密码
登录密码: xxxxxxxx
```

访问：

```text
http://你的VPS公网IP:29173
```

VPS Web UI 当前重点是运维面板。它适合用来：

- 查看 Openxhh 服务是否运行；
- 启动、停止、重启机器人服务；
- 读取和保存 `config.json`；
- 查看最近 20 次对话；
- 查看 AI 失败回复；
- 查看总 token、最近 1 小时 token、最近 24 小时 token；
- 管理日志，支持全部 / 报错 / 用户提问 / AI 回复 / 图片生图筛选；
- 按关键词筛选日志；
- 复制筛选后的全部可见日志；
- 复制选中日志、多行选择复制、Shift 范围选择、暂停刷新。

公网使用建议：

- 只在云安全组放行你自己的 IP；
- 不要把 `29173` 端口裸奔给所有人；
- 保存好第一次生成的随机密码；
- 需要重置密码时，停止 Web UI 后删除 `/opt/Openxhh/webui_auth.json` 再启动。

</details>

<details>
<summary>更新到最新版本</summary>

以后更新仍然只需要执行：

```bash
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo bash
```

默认会更新：

- `/opt/Openxhh/Openxhh`
- `/opt/Openxhh/Openxhh-webui`

并尝试重启：

- `Openxhh`
- `Openxhh-webui`

如果你的安装目录或服务名不同，可以这样指定：

```bash
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo env INSTALL_DIR=/你的安装目录 SERVICE_NAME=你的机器人服务名 WEBUI_SERVICE_NAME=你的WebUI服务名 bash
```

如果只想更新主程序，不更新 VPS Web UI：

```bash
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo env INSTALL_WEBUI=0 bash
```

</details>

## 外部图床配置说明

评论区生图要能发到小黑盒，核心是让小黑盒能访问生成后的图片。当前推荐方案是：

```text
Openxhh 生成图片
→ 写入 VPS 本地目录 /var/www/xhh-images
→ Nginx 把 /xhh-images/ 暴露成公网 URL
→ Openxhh 把 http://你的VPS公网IP/xhh-images/xxx.png 发到评论 imgs 字段
```

也就是说，配置里这两个字段必须对应：

```json
{
  "image": {
    "uploadMode": "external",
    "externalDir": "/var/www/xhh-images",
    "externalBaseUrl": "http://你的VPS公网IP/xhh-images"
  }
}
```

- `externalDir` 是服务器本地目录，Openxhh 会把图片文件写到这里。
- `externalBaseUrl` 是公网访问地址，小黑盒会通过这个 URL 读取图片。
- Nginx 的 `alias` 目录必须和 `externalDir` 一致。
- `externalBaseUrl` 必须是完整 URL，包含 `http://` 或 `https://`。
- `externalBaseUrl` 末尾有没有 `/` 都可以，程序会自动拼接文件名。

<details>
<summary>用 Nginx 配置静态图床</summary>

### 1. 安装 Nginx

```bash
sudo apt update
sudo apt install -y nginx
```

### 2. 创建图片目录

```bash
sudo mkdir -p /var/www/xhh-images
sudo chown -R root:www-data /var/www/xhh-images
sudo chmod 775 /var/www/xhh-images
```

Openxhh 服务如果不是 root 运行，需要确保它对 `/var/www/xhh-images` 有写入权限。

### 3. 添加 Nginx location

如果你使用默认站点，可以编辑：

```bash
sudo nano /etc/nginx/sites-available/default
```

在 `server { ... }` 里加入：

```nginx
location /xhh-images/ {
    alias /var/www/xhh-images/;
    add_header Access-Control-Allow-Origin *;
}
```

注意：

- `location /xhh-images/` 末尾有 `/`。
- `alias /var/www/xhh-images/` 末尾也有 `/`。
- 两边路径要一一对应。

### 4. 放一张测试图

```bash
sudo bash -c 'printf test > /var/www/xhh-images/test.txt'
sudo nginx -t
sudo systemctl reload nginx
```

### 5. 测试公网访问

```bash
curl -I http://你的VPS公网IP/xhh-images/test.txt
```

如果返回 `200 OK`，说明公网图床可访问。

### 6. 回到 Openxhh 配置

把 `config.json` 里的图片配置改成：

```json
{
  "image": {
    "uploadMode": "external",
    "externalDir": "/var/www/xhh-images",
    "externalBaseUrl": "http://你的VPS公网IP/xhh-images"
  }
}
```

然后重启机器人：

```bash
sudo systemctl restart Openxhh
```

</details>

## 评论区生图

Openxhh 支持在评论区直接 @ 机器人生图，例如：

```text
@机器人 生图 一只穿赛博朋克外套的猫，站在霓虹街头
@机器人 给小菲画一张雨夜里的机甲少女
```

它会尽量把生图请求整理成适合图片模型的 prompt，同时把回复文案处理得更自然：

- 不再直接输出 `已生成：prompt`。
- AI 短回复异常时会降级成 `图片来了喵`。
- 如果生图命令点名了目标用户，会同时 @ 触发者和目标用户。
- 会按 `data-user-id` 去重，避免重复 @。
- 会清洗 `[cube_*]` 等小黑盒表情控制字符。

<details>
<summary>生图验证命令</summary>

下面命令里的 `你的测试帖子link_id` 必须换成你自己的小黑盒测试帖子 ID。不要直接复制别人的 `link_id`：评论会发到别人的帖子下面，或者因为帖子权限、可见性、状态不同导致测试失败。

验证命令识别和 Form Data，不调用真实生图接口：

```bash
go run ./cmd/dry_run_image_comment \
  -comment_id 123 \
  -link_id 你的测试帖子link_id \
  -root_id 123 \
  -userid 你的ownerUID \
  -text "@机器人 生图 一只赛博朋克猫"
```

调用真实生图接口但不上传、不发评论：

```bash
go run ./cmd/dry_run_image_comment \
  -comment_id 123 \
  -link_id 你的测试帖子link_id \
  -root_id 123 \
  -userid 你的ownerUID \
  -text "@机器人 生图 一只赛博朋克猫" \
  -mock_image=false
```

验证已有图片 URL 能否发带图评论：

```bash
go run ./cmd/test_image_comment 你的测试帖子link_id "图片测试" "http://你的VPS公网IP/xhh-images/test.png"
```

验证本地图片上传到外部图床并可选发布评论：

```bash
go run ./cmd/test_xhh_image_upload_comment \
  -file ./images/example.png \
  -link_id 你的测试帖子link_id \
  -reply_id -1 \
  -root_id -1 \
  -text "图片测试" \
  -publish=true
```

测试带图评论前确认：

- `link_id` 已换成你自己的小黑盒测试帖子 ID。
- 图片 URL 是公网 URL，不是 `localhost`、`127.0.0.1`、内网 IP 或只在自己电脑可访问的地址。
- 用手机 4G/5G 或无登录浏览器能直接打开图片 URL。
- `curl -I 图片URL` 返回 `200 OK`。
- `image.externalDir` 和 Nginx `alias` 指向同一个目录。
- `image.externalBaseUrl` 和浏览器访问的图片 URL 前缀一致。
- 如果别人看不到测试图片，优先检查公网图床，而不是先怀疑 `link_id`。

</details>

## 为什么推荐 VPS 版

VPS 版本更适合长期运行，因为它的核心需求是“稳定在线”：

- 不依赖桌面窗口是否开着；
- 可以用 systemd 自动拉起；
- 日志、数据库、cookie 都在固定目录；
- 更新脚本能直接覆盖主程序和 VPS Web UI；
- Web UI 能远程看状态、改配置、看 token、筛日志、定位失败；
- 出问题时更容易用 `journalctl` 排查。

如果你只是想让机器人一直在小黑盒里工作，VPS 是主线版本。

## Windows 版本：目前不建议优先安装

Windows 图形化安装包还在完善中。它适合测试本地桌面控制台，但不建议作为主要部署方式。

如果你只是普通使用，请优先使用上面的 VPS 版本。Windows 版本可能遇到：

- 桌面窗口 / WebView2 环境差异；
- 扫码登录和本地控制台兼容性问题；
- 长期后台运行不如 VPS 稳定；
- 日志和服务管理不如 Linux systemd 清晰。

<details>
<summary>仍然想测试 Windows 版</summary>

可以下载 Release 中的：

```text
Openxhh-Setup-x64.exe
```

安装后会放置：

```text
C:\Program Files\Openxhh\Openxhh.exe
C:\Program Files\Openxhh\Openxhh-webui.exe
C:\ProgramData\Openxhh\config.json
C:\ProgramData\Openxhh\cookie.json
C:\ProgramData\Openxhh\sql.db
C:\ProgramData\Openxhh\log\
```

第一次使用：

1. 启动 Openxhh 控制台。
2. 保存页面显示的随机控制台密码，并登录本地控制台。
3. 在配置向导中填写 owner UID、AI 接口、模型和 token；如需只回复 owner / 指定用户，再开启白名单。
4. 点击“扫码登录”，使用小黑盒 App 扫码。
5. 日志提示 Cookie 已保存后，点击“启动”。

更多说明见 [docs/windows.md](docs/windows.md)。

</details>

## 安全建议

- `config.json` 包含 AI token。
- `cookie.json` 是小黑盒登录态。
- `sql.db` 里可能包含运行记录。
- 不要把这些文件上传到 GitHub。
- 不要把 `checkTime` 和 `replyTime` 调得太低，容易触发平台风控；建议保持 `checkTime=60`、`replyTime=30` 或更保守。
- VPS Web UI 不要全网裸奔，至少用云安全组限制来源 IP。
- 外部图床目录不要放敏感文件，它会通过公网 URL 暴露。

建议限制运行目录权限：

```bash
sudo chmod 600 /opt/Openxhh/config.json /opt/Openxhh/cookie.json /opt/Openxhh/sql.db 2>/dev/null || true
sudo chmod 700 /opt/Openxhh
```

## 回滚

更新脚本会自动备份旧二进制，文件名类似：

```text
/opt/Openxhh/Openxhh.bak-20260517-120000
/opt/Openxhh/Openxhh-webui.bak-20260517-120000
```

如需回滚主程序：

```bash
sudo systemctl stop Openxhh
sudo cp /opt/Openxhh/Openxhh.bak-时间戳 /opt/Openxhh/Openxhh
sudo chmod +x /opt/Openxhh/Openxhh
sudo systemctl start Openxhh
```

如需回滚 VPS Web UI：

```bash
sudo systemctl stop Openxhh-webui
sudo cp /opt/Openxhh/Openxhh-webui.bak-时间戳 /opt/Openxhh/Openxhh-webui
sudo chmod +x /opt/Openxhh/Openxhh-webui
sudo systemctl start Openxhh-webui
```

## 常用命令速查

```bash
# 进入运行目录
cd /opt/Openxhh

# 生成或补齐配置文件
sudo ./Openxhh

# 扫码登录
sudo ./Openxhh -mode login

# 前台启动机器人
sudo ./Openxhh -mode start

# 查看机器人服务状态
sudo systemctl status Openxhh --no-pager

# 启动 / 停止 / 重启机器人服务
sudo systemctl start Openxhh
sudo systemctl stop Openxhh
sudo systemctl restart Openxhh

# 查看机器人日志
sudo journalctl -u Openxhh -f

# 查看 VPS Web UI 日志
sudo journalctl -u Openxhh-webui -f

# 重启 VPS Web UI
sudo systemctl restart Openxhh-webui

# 更新到最新版本
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo bash

# 重置 VPS Web UI 登录密码
sudo systemctl stop Openxhh-webui
sudo rm -f /opt/Openxhh/webui_auth.json
sudo systemctl start Openxhh-webui
sudo journalctl -u Openxhh-webui -n 50 --no-pager
```

## 默认配置速查

这些值会在缺失或为 0 时自动补齐，或是本文推荐的恢复值。误改后可以按这里恢复。

| 配置项 | 默认 / 推荐值 | 说明 |
| --- | --- | --- |
| `xhh.checkTime` | `60` | 检查 @ 间隔，秒 |
| `xhh.replyTime` | `30` | 回复间隔，秒 |
| `xhh.maxReplyThreads` | `3` | 同一轮最高回复并发 |
| `xhh.maxPendingReplies` | `50` | 普通用户全局待回复队列上限 |
| `xhh.maxPendingRepliesPerUser` | `5` | 单个普通用户待回复队列上限 |
| `xhh.enableWhitelist` | `false` | 默认关闭白名单，回复所有 @ |
| `xhh.baseUrl` | `https://api.xiaoheihe.cn` | 小黑盒 API 地址 |
| `xhh.webver` | `2.5` | 小黑盒 Web 版本字段 |
| `xhh.version` | `999.0.4` | 小黑盒版本字段 |
| `database.type` | `sqlite` | 个人部署推荐 SQLite |
| `ai.prompt` | `请根据评论内容自然回复。` | VPS Web UI 默认回复策略 |
| `image.model` | `gpt-image-2` | 图片模型默认值 |
| `image.size` | `1024x1024` | 图片尺寸默认值 |
| `image.responseFormat` | `b64_json` | 图片接口输出格式 |
| `image.outputDir` | `images` | 本地生成图片临时目录 |
| `image.uploadMode` | `external` | 推荐写入外部静态图床 |
| `image.externalDir` | `/var/www/xhh-images` | 推荐外部图片目录，需手动配置 |
| `image.externalBaseUrl` | `http://你的VPS公网IP/xhh-images` | 推荐外部图片访问 URL，需手动配置为公网可访问地址 |
| `image.promptRefine` | `false` | 默认不启用图片 prompt 优化 |
| `image.promptMaxChars` | `1000` | 图片 prompt 优化输入最大字符数 |
| VPS Web UI 端口 | `29173` | 默认访问 `http://服务器IP:29173` |
| VPS Web UI 服务名 | `Openxhh-webui` | 更新脚本默认识别和重启 |
| 机器人服务名 | `Openxhh` | 更新脚本默认识别和重启 |

说明：`image.externalDir` 和 `image.externalBaseUrl` 不会凭空知道你的服务器公网地址，必须按你的 VPS 实际情况填写；上表给的是本文推荐恢复值。

## 免责声明

本项目仅供个人学习和自用。自动化访问、自动回复、自动生图和频繁请求都可能触发平台风控。请自行控制频率，并遵守小黑盒相关规则。

## 致谢

感谢 [SomeOvO/xhhRobot](https://github.com/SomeOvO/xhhRobot) 原项目提供早期基础思路与实现参考。也感谢所有测试、反馈和提出建议的朋友。
