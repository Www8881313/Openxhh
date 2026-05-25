# Openxhh

Openxhh 是一个面向小黑盒的 AI 自动回复机器人。它不是只会看见一句 `@机器人` 的关键词脚本，而是尽量把帖子、楼层、图片、上下文和被点名的人一起读进去，再用 OpenAI 兼容接口生成更像真人接话的回复。

项目地址：<https://github.com/Www8881313/Openxhh>

> 推荐部署方式：**VPS / Linux 版本**。  
> Windows 图形化版本目前仍不完善，不建议普通用户优先安装；如果只是想长期稳定运行，请优先使用 VPS 版本。

## 它解决了什么问题

普通机器人最大的问题是"眼睛太短"：用户说"这个怎么样""楼上说得对吗""给小菲画一张"，如果机器人只看到当前这一句话，就很容易答非所问。

Openxhh 的目标是让机器人更像一个真的在看帖的人：

- 它会读帖子标题、正文、评论楼层和图片，而不是只看 @ 这一句。
- 它会先用 AI 路由判断用户真实意图，再决定普通回复、生图、重新生成或忽略。
- 它支持模型联网搜索，普通文字回复默认可以带着最新信息回答。
- 它支持评论区生图，并且不会再把 `已生成：prompt` 这种生硬文本直接丢到评论区。
- 它可以按低频、限额、dry-run 的方式自动刷帖回复，先观察再决定是否真实发评。
- 它会记录机器人发出的评论和别人对机器人的回复，方便在 VPS Web UI 里追踪互动。
- 它会清洗显式点名、小黑盒表情和富文本 HTML，尽量避免把 `[cube_喜欢]` 或链接标签误当内容。
- 它带 VPS Web UI，可以看记录、楼层、失败、token 消耗和日志，不用一直盯着终端。
- 它默认回复所有 @，也可以手动开启白名单；同时有并发和队列上限，避免普通用户刷 @ 把机器人拖死。

## 主要功能

- 自动检查小黑盒 @ 消息，并调用 OpenAI 兼容接口回复。
- 所有 @ 消息先经过 AI 意图路由，再决定普通回复、生图、看图生图、重新生成或忽略。
- 普通文字回复默认支持模型联网搜索，可配置搜索强度和是否强制联网。
- 默认回复所有 @；开启白名单后只回复 owner / 指定 UID。
- `xhh.maxReplyThreads` 限制普通用户回复并发，owner 不占用普通用户线程槽位。
- 支持全局待回复队列、单用户待回复队列上限。
- 支持自定义 AI 接口、模型、token 和 prompt。
- 支持 SQLite / PostgreSQL；个人部署推荐 SQLite。
- 支持小黑盒扫码登录，登录态保存到 `cookie.json`。
- 支持帖子 / 评论楼层上下文增强，查看记录时可定位当前评论所在整层楼。
- 支持评论图片和帖子图片进入 AI 上下文；楼层弹窗可展示头像、时间、图片缩略图和小黑盒表情。
- 支持评论区 @ 生图：`生图`、`画图`、`生成图片`。
- 支持把生成图片上传到小黑盒官方 COS 图床，并用 `imgs=<图片URL>` 发布顶级带图评论。
- 支持生图后生成自然短回复，避免暴露 prompt。
- 支持显式点名 @：例如"给小菲画一张""问问楼主""对小菲说"。
- 支持自动刷帖回复：默认关闭、默认 dry-run，可配置间隔、每轮上限、每日上限和专用 Prompt。
- 支持消息流追踪：通过小黑盒通知接口（list_type=0）实时同步"评论我的"，每 60 秒一次；"我评论的"自动记录。
- 支持 VPS Web UI：配置管理、服务控制、日志筛选、@ 回复记录、机器人发言记录、评论我的、失败记录、token 统计。

## 一键安装（VPS / Linux）

适合 Ubuntu / Debian VPS。一条命令完成编译、安装和创建 systemd 服务。

```bash
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/setup.sh | sudo bash
```

脚本会自动：安装构建依赖、拉取源码、编译主程序和 Web UI、生成默认配置、创建并启动 systemd 服务。

安装完成后，三步即可运行：

**第一步：填写配置（二选一）**

方式 A — Web UI 配置（推荐）：

打开 `http://你的VPS公网IP:29173`，首次登录密码查看：

```bash
sudo journalctl -u Openxhh-webui -n 50 --no-pager
```

登录后进入「配置管理」，填写 owner UID、AI 模型、接口地址和 Token。

方式 B — 命令行配置：

```bash
sudo nano /opt/Openxhh/config.json
```

至少填写 `xhh.owner`、`ai.model`、`ai.baseUrl`、`ai.token`。

**第二步：扫码登录**

```bash
cd /opt/Openxhh && sudo ./Openxhh -mode login
```

**第三步：启动机器人**

```bash
sudo systemctl start Openxhh
```

如果只想更新已有安装（不重新配置），使用更新脚本：

```bash
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo bash
```

<details>
<summary>手动编译安装（不用一键脚本）</summary>

如果你不想用一键脚本，可以手动操作：

```bash
# 1. 创建目录并编译
sudo mkdir -p /opt/Openxhh
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/update-installed.sh | sudo bash

# 2. 生成并编辑配置
cd /opt/Openxhh
sudo ./Openxhh
sudo nano config.json

# 3. 扫码登录
sudo ./Openxhh -mode login

# 4. 前台试运行
sudo ./Openxhh -mode start
```

手动安装时，systemd 服务需要自行创建，参见下方"systemd 后台运行"折叠块。

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
    "minRequestInterval": 0.5,
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
    "baseUrl": "你的 OpenAI 兼容 /v1/chat/completions 或 /v1/responses 地址",
    "token": "你的 AI API Token",
    "webSearch": true,
    "forceWebSearch": false,
    "searchContextSize": "medium"
  },
  "feedReply": {
    "enabled": false,
    "interval": 900,
    "maxPerRun": 1,
    "maxPerDay": 10,
    "dryRun": true,
    "prompt": "你正在作为小黑盒用户回复帖子。请结合帖子内容写一句自然、有信息量、不像机器人的短评论；如果帖子不适合回复，或容易引战、广告、抽奖、敏感内容，请只输出 SKIP。"
  },
  "image": {
    "model": "gpt-image-2",
    "baseUrl": "你的 OpenAI 兼容 /v1/images/generations 地址",
    "token": "你的图片 API Token",
    "size": "1024x1024",
    "responseFormat": "b64_json",
    "outputDir": "images",
    "uploadMode": "cos",
    "externalDir": "",
    "externalBaseUrl": "",
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
- `xhh.maxReplyThreads` 控制普通用户最高回复并发，默认 `3`；owner 不占用普通用户线程槽位。
- `xhh.maxPendingReplies` 控制普通用户全局待回复队列上限，默认 `50`。
- `xhh.maxPendingRepliesPerUser` 控制单个普通用户待回复队列上限，默认 `5`。
- owner 不受 `maxPendingReplies` 和 `maxPendingRepliesPerUser` 限制。
- `xhh.minRequestInterval` 控制小黑盒 API 全局最小请求间隔，默认 `0.5` 秒；并发请求自动排队，避免瞬间大量请求触发 403。
- `ai.baseUrl` 要填完整的 OpenAI 兼容地址，例如 `https://example.com/v1/chat/completions` 或 `https://example.com/v1/responses`。
- `ai.webSearch=true` 表示普通文字回复默认启用模型联网搜索。
- `ai.forceWebSearch=false` 表示不强制每次回复都必须调用搜索工具。
- `ai.searchContextSize` 可填 `low` / `medium` / `high`，默认 `medium`。
- `feedReply.enabled=false` 表示自动刷帖默认关闭；`feedReply.dryRun=true` 表示默认只记录不真实发评论。
- `image.baseUrl` 要填完整的 Images Generations 地址，例如 `https://example.com/v1/images/generations`。
- `image.responseFormat` 默认 `b64_json`，程序会把 base64 解码成图片 bytes 后上传。
- `image.uploadMode` 推荐填 `cos`，使用小黑盒官方图床；当前版本即使填 `external` / `static` 也会优先走官方图床。
- `image.externalDir` 和 `image.externalBaseUrl` 仅保留给旧的 VPS 静态图床备用方案，推荐留空。
- `promptRefine=true` 后，可以用文本模型先把用户口语化生图请求整理成更适合图片模型的提示词。
- 保存配置后需要重启 `Openxhh` 服务才会生效。

</details>

<details>
<summary>VPS Web UI 配置管理</summary>

VPS Web UI 提供"配置管理"页，可以读取和保存：

```text
/opt/Openxhh/config.json
```

它支持编辑：

- 小黑盒配置：检查间隔、回复间隔、请求间隔、白名单开关、owner UID、普通用户并发、队列上限。
- 数据库配置：SQLite / PostgreSQL。
- AI 回复配置：模型、Chat Completions / Responses URL、token、回复 prompt、联网搜索开关和搜索上下文强度。
- 自动刷帖配置：开关、dry-run、刷帖间隔、每轮上限、每日上限和专用 Prompt。
- 图片能力配置：图片模型、Images Generations URL、图片 token、输出格式和上传模式；外部图床目录仅作为旧方案备用。
- Prompt 优化配置：是否启用、模型、URL、token、最大字符数。

保存后只会写入 `config.json`，不会自动让运行中的机器人重新读取。改完配置后请在 Web UI 点"重启服务"，或执行：

```bash
sudo systemctl restart Openxhh
```

</details>

<details>
<summary>VPS IP 被小黑盒接口拦截时使用 Cloudflare Worker 中转</summary>

如果你的 VPS 出站 IP 被小黑盒接口拒绝访问，可以把 `xhh.baseUrl` 指向 Cloudflare Worker，让 Openxhh 的小黑盒 API 请求从 Worker 转发出去。

注意：这只能解决"VPS 出站 IP 被拒"的问题；如果是账号、Cookie、请求频率或行为特征触发风控，仍然需要重新登录、降低频率或检查配置。

### 1. 在 VPS 上准备 Worker 目录

```bash
mkdir -p ~/openxhh-worker
cd ~/openxhh-worker
```

如果 VPS 没有 Node.js，先安装：

```bash
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs
node -v
npm -v
```

### 2. 创建 Worker 脚本

```bash
cat > xhh-cloudflare-worker.js <<'EOF'
const DEFAULT_UPSTREAM = "https://api.xiaoheihe.cn";

function normalizePrefix(prefix) {
  const value = (prefix || "").trim();
  if (!value || value === "/") {
    return "";
  }
  return value.startsWith("/") ? value.replace(/\/+$/, "") : `/${value.replace(/\/+$/, "")}`;
}

function forwardedHeaders(request) {
  const headers = new Headers(request.headers);
  for (const name of [
    "host",
    "cf-connecting-ip",
    "cf-ipcountry",
    "cf-ray",
    "cf-visitor",
    "x-forwarded-for",
    "x-forwarded-proto",
    "x-real-ip",
  ]) {
    headers.delete(name);
  }
  return headers;
}

export default {
  async fetch(request, env) {
    if (!["GET", "POST", "HEAD"].includes(request.method)) {
      return new Response("Method not allowed", { status: 405 });
    }

    const prefix = normalizePrefix(env.PROXY_PATH_PREFIX);
    if (!prefix) {
      return new Response("Missing PROXY_PATH_PREFIX", { status: 500 });
    }

    const url = new URL(request.url);
    if (url.pathname !== prefix && !url.pathname.startsWith(`${prefix}/`)) {
      return new Response("Not found", { status: 404 });
    }

    const upstreamBase = new URL(env.UPSTREAM_ORIGIN || DEFAULT_UPSTREAM);
    const upstream = new URL(upstreamBase);
    upstream.pathname = url.pathname.slice(prefix.length) || "/";
    upstream.search = url.search;

    const init = {
      method: request.method,
      headers: forwardedHeaders(request),
      redirect: "manual",
    };
    if (!["GET", "HEAD"].includes(request.method)) {
      init.body = request.body;
    }

    return fetch(upstream.toString(), init);
  },
};
EOF

node --check xhh-cloudflare-worker.js
```

### 3. 登录或配置 Cloudflare Token

有浏览器环境时可以直接登录：

```bash
npx wrangler login
```

如果 VPS 不方便打开浏览器，在 Cloudflare 后台创建一个可编辑 Workers 的 API Token，然后在 VPS 上设置：

```bash
export CLOUDFLARE_API_TOKEN='你的 Cloudflare API Token'
```

不要把这个 token 写进公开文件。

### 4. 部署 Worker

```bash
npx wrangler deploy xhh-cloudflare-worker.js --name openxhh-xhh-proxy --compatibility-date 2026-05-20
```

第一次部署如果提示注册 `workers.dev` 子域名，按提示输入一个英文子域名即可。部署成功后会看到类似：

```text
https://openxhh-xhh-proxy.你的子域名.workers.dev
```

### 5. 设置秘密路径

生成一个随机路径：

```bash
node -e "console.log('/xhh-' + require('crypto').randomBytes(16).toString('hex'))"
```

把输出保存下来，例如：

```text
/xhh-a1b2c3d4e5f678901234567890abcdef
```

写入 Worker secret：

```bash
npx wrangler secret put PROXY_PATH_PREFIX --name openxhh-xhh-proxy
```

提示输入值时，粘贴刚生成的完整路径。

### 6. 测试 Worker 转发

把下面的地址替换成你的 Worker 地址和秘密路径：

```bash
curl -4 -I "https://openxhh-xhh-proxy.你的子域名.workers.dev/xhh-a1b2c3d4e5f678901234567890abcdef/account/get_qrcode_url/"
```

正常情况下不应该返回：

- `404 Not found`：路径前缀不对。
- `500 Missing PROXY_PATH_PREFIX`：Worker secret 没设置成功。
- TLS handshake failure：新注册的 `workers.dev` 子域名证书可能还没生效，等几分钟再试。

如果返回 `200`、`403` 或小黑盒 JSON，通常说明已经转发到小黑盒接口；`403` 还需要结合日志判断是否为账号、Cookie 或频率问题。

### 7. 修改 Openxhh 配置

打开 VPS Web UI：

```text
配置管理 → 小黑盒配置 → API Base URL
```

把默认值：

```text
https://api.xiaoheihe.cn
```

改成：

```text
https://openxhh-xhh-proxy.你的子域名.workers.dev/xhh-a1b2c3d4e5f678901234567890abcdef
```

也可以直接改 `/opt/Openxhh/config.json`：

```json
"baseUrl": "https://openxhh-xhh-proxy.你的子域名.workers.dev/xhh-a1b2c3d4e5f678901234567890abcdef"
```

保存后重启主服务：

```bash
sudo systemctl restart Openxhh
sudo journalctl -u Openxhh -f
```

如果日志里不再出现小黑盒 HTTP `403`，说明 Worker 中转已经生效。若仍然失败，优先检查 `xhh.baseUrl` 是否保存成功、Worker secret 是否一致、Cookie 是否过期，以及 `checkTime` / `replyTime` 是否过低。

</details>

<details>
<summary>systemd 后台运行</summary>

一键安装脚本会自动创建 systemd 服务。如果手动安装，需要自行创建：

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

VPS Web UI 当前重点是运维面板和记录排查。它适合用来：

- 查看 Openxhh 服务是否运行；
- 启动、停止、重启机器人服务；
- 读取和保存 `config.json`；
- 查看总 token、最近 1 小时 token、最近 24 小时 token；
- 查看 @ 回复记录，并可从记录打开当前评论所在整层楼；
- 查看机器人发言记录，包括 AI 回复、图片回复、图片发帖和自动刷帖；
- 查看"评论我的"，通过小黑盒通知接口实时同步评论回复、@ 提及和帖子评论；
- 查看楼层评论里的头像、评论时间、图片缩略图和小黑盒表情；
- 查看 AI 失败回复和异常发送状态；
- 管理日志，支持全部 / 报错 / 用户提问 / AI 回复 / 图片生图 / 自动刷帖筛选；
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

## 自动刷帖回复

自动刷帖回复是独立能力，不会混入 @ 回复队列。它会按配置低频读取帖子，让 AI 判断是否适合回复，并把结果记录到数据库和 VPS Web UI。

默认策略很保守：

- `feedReply.enabled=false`：默认关闭，需要你明确开启。
- `feedReply.dryRun=true`：默认只记录"如果要回复会说什么"，不会真实发评论。
- `feedReply.interval=900`：默认 15 分钟一轮。
- `feedReply.maxPerRun=1`：默认每轮最多处理 1 条。
- `feedReply.maxPerDay=10`：默认每天最多 10 条。
- `feedReply.prompt`：自动刷帖专用 Prompt，和 @ 回复 Prompt 分开。

建议先保持 dry-run 跑一段时间，在 VPS Web UI 的机器人发言记录里看回复质量和跳过原因；确认没问题后，再把 `feedReply.dryRun` 改成 `false`。

## 图片图床配置说明

评论区生图要能发到小黑盒，核心是让小黑盒能访问生成后的图片。当前推荐方案是 **小黑盒官方图床**：

```text
Openxhh 调用图片模型生成图片
→ 如果图片接口返回 b64_json，先解码成图片 bytes
→ Openxhh 通过小黑盒官方 COS 上传接口获取 key 和临时 STS 凭证
→ PUT 到小黑盒官方 COS
→ 拿到 https://imgheybox.max-c.com/... 图片 URL
→ 评论接口把这个 URL 写入 imgs 字段
```

推荐配置：

```json
{
  "image": {
    "uploadMode": "cos",
    "externalDir": "",
    "externalBaseUrl": ""
  }
}
```

说明：

- `uploadMode` 推荐填 `cos`，明确使用小黑盒官方图床。
- 当前版本会把 `external` / `static` 作为旧配置兼容值处理，因此旧配置也会优先走小黑盒图床。
- `externalDir` 和 `externalBaseUrl` 不再是推荐必填项，正常使用可以留空。
- 真实上传依赖小黑盒登录态，运行目录下必须有有效的 `cookie.json`。
- 小黑盒官方图床返回的 URL 通常是 `https://imgheybox.max-c.com/web/bbs/...`。

<details>
<summary>旧方案：VPS / Nginx 静态图床（备用）</summary>

旧方案仍可作为排障或回退思路，但不再推荐作为默认路径。它的流程是：

```text
Openxhh 生成图片
→ 写入 VPS 本地目录 /var/www/xhh-images
→ Nginx 把 /xhh-images/ 暴露成公网 URL
→ Openxhh 把 http://你的VPS公网IP/xhh-images/xxx.png 发到评论 imgs 字段
```

如果要恢复旧方案，需要保证：

- `externalDir` 是服务器本地目录，Openxhh 对它有写入权限。
- `externalBaseUrl` 是公网访问地址，小黑盒能直接访问。
- Nginx 的 `alias` 目录必须和 `externalDir` 一致。
- `externalBaseUrl` 必须是完整 URL，包含 `http://` 或 `https://`。

最小 Nginx location 示例：

```nginx
location /xhh-images/ {
    alias /var/www/xhh-images/;
    add_header Access-Control-Allow-Origin *;
}
```

测试公网访问：

```bash
curl -I http://你的VPS公网IP/xhh-images/test.txt
```

如果返回 `200 OK`，说明旧图床公网访问正常。

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
go run ./cmd/test_image_comment 你的测试帖子link_id "图片测试" "https://imgheybox.max-c.com/web/bbs/示例图片.png"
```

验证本地图片上传到小黑盒官方图床并可选发布评论：

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
- 运行目录下有有效的 `cookie.json`，否则官方图床上传会要求重新登录。
- 图片 URL 是公网 URL，不是 `localhost`、`127.0.0.1`、内网 IP 或只在自己电脑可访问的地址。
- 用手机 4G/5G 或无登录浏览器能直接打开图片 URL。
- `curl -I 图片URL` 返回 `200 OK`。
- 如果别人看不到测试图片，优先检查官方图床上传是否成功，再检查 `link_id`。

</details>

## 为什么推荐 VPS 版

VPS 版本更适合长期运行，因为它的核心需求是"稳定在线"：

- 不依赖桌面窗口是否开着；
- 可以用 systemd 自动拉起；
- 日志、数据库、cookie 都在固定目录；
- 更新脚本能直接覆盖主程序和 VPS Web UI；
- Web UI 能远程看状态、改配置、看 token、筛日志、定位失败；
- Web UI 能打开 @ 回复、机器人发言和评论我的对应楼层，排查上下文更方便；
- 出问题时更容易用 `journalctl` 和 Web UI 日志筛选排查。

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
4. 点击"扫码登录"，使用小黑盒 App 扫码。
5. 日志提示 Cookie 已保存后，点击"启动"。

更多说明见 [docs/windows.md](docs/windows.md)。

</details>

## 安全建议

- `config.json` 包含 AI token。
- `cookie.json` 是小黑盒登录态。
- `sql.db` 里可能包含运行记录。
- 不要把这些文件上传到 GitHub。
- 不要把 `checkTime` 和 `replyTime` 调得太低，容易触发平台风控；建议保持 `checkTime=60`、`replyTime=30` 或更保守。
- VPS Web UI 不要全网裸奔，至少用云安全组限制来源 IP。
- 使用小黑盒官方图床时，不要上传敏感图片；图片会通过公网 CDN URL 暴露。
- 如果恢复旧 VPS 静态图床，外部图床目录不要放敏感文件。

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
# 一键安装（首次）
curl -fsSL https://raw.githubusercontent.com/Www8881313/Openxhh/main/scripts/setup.sh | sudo bash

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

# 手动清理 7 天前的评论缓存（释放 SQLite 空间）
# 注意：程序已内置自动清理，每天凌晨 3 点清理 7 天前的旧缓存
# 以下命令仅在需要立即手动清理时使用
sqlite3 /opt/Openxhh/sql.db "DELETE FROM xhh_comment_cache WHERE updated_at < strftime('%s','now','-7 days');"
sqlite3 /opt/Openxhh/sql.db "DELETE FROM xhh_post_cache WHERE updated_at < strftime('%s','now','-7 days');"
sqlite3 /opt/Openxhh/sql.db "VACUUM;"

# 定期自动清理缓存（建议加入 crontab，每周执行一次）
# crontab -e 后加入：
# 0 4 * * 0 sqlite3 /opt/Openxhh/sql.db "DELETE FROM xhh_comment_cache WHERE updated_at < strftime('%s','now','-7 days');" && sqlite3 /opt/Openxhh/sql.db "DELETE FROM xhh_post_cache WHERE updated_at < strftime('%s','now','-7 days');" && sqlite3 /opt/Openxhh/sql.db "VACUUM;"
```

<details>
<summary>数据库查询常用命令</summary>

以下命令适用于 VPS 默认 SQLite 部署，数据库路径为 `/opt/Openxhh/sql.db`。如果路径不同，请替换为实际路径。

```bash
DB=/opt/Openxhh/sql.db

# ===== 回复统计 =====

# 总 @ 回复数（机器人实际回复了多少条）
sqlite3 $DB "SELECT COUNT(*) FROM at WHERE reply=true;"

# 总 @ 收到数（所有被 @ 的消息，含未回复）
sqlite3 $DB "SELECT COUNT(*) FROM at;"

# 未回复数
sqlite3 $DB "SELECT COUNT(*) FROM at WHERE reply=false;"

# ===== 发出消息统计（outbound_messages）=====

# 总发出数
sqlite3 $DB "SELECT COUNT(*) FROM outbound_messages;"

# 按来源分类统计（ai_reply / image_reply / feed_reply 等）
sqlite3 $DB "SELECT source, COUNT(*) FROM outbound_messages GROUP BY source;"

# 最近 24 小时发出数
sqlite3 $DB "SELECT COUNT(*) FROM outbound_messages WHERE created_at >= strftime('%s','now','-1 day');"

# 最近 10 条发出的消息
sqlite3 $DB -header -column "SELECT source, link_id, substr(text,1,40) as text, datetime(created_at,'unixepoch','localtime') as time FROM outbound_messages ORDER BY created_at DESC LIMIT 10;"

# ===== 收到消息统计（inbound_messages）=====

# 总收到数
sqlite3 $DB "SELECT COUNT(*) FROM inbound_messages;"

# 按来源分类（at_comment / notification / reply_to_bot 等）
sqlite3 $DB "SELECT source, COUNT(*) FROM inbound_messages GROUP BY source;"

# 最近 24 小时收到数
sqlite3 $DB "SELECT COUNT(*) FROM inbound_messages WHERE created_at >= strftime('%s','now','-1 day');"

# 最近 10 条收到的消息
sqlite3 $DB -header -column "SELECT source, user_name, substr(text,1,40) as text, datetime(created_at,'unixepoch','localtime') as time FROM inbound_messages ORDER BY created_at DESC LIMIT 10;"

# ===== 自动刷帖统计（feed_reply_records）=====

# 按状态分类（sent / dry_run / failed / skipped）
sqlite3 $DB "SELECT status, COUNT(*) FROM feed_reply_records GROUP BY status;"

# 成功发送数
sqlite3 $DB "SELECT COUNT(*) FROM feed_reply_records WHERE status='sent';"

# 失败数
sqlite3 $DB "SELECT COUNT(*) FROM feed_reply_records WHERE status='failed';"

# 最近 10 条失败详情
sqlite3 $DB -header -column "SELECT link_id, substr(title,1,30) as title, reason, datetime(replied_at,'unixepoch','localtime') as time FROM feed_reply_records WHERE status='failed' ORDER BY replied_at DESC LIMIT 10;"

# ===== 评论缓存 =====

# 缓存的帖子数
sqlite3 $DB "SELECT COUNT(*) FROM xhh_post_cache;"

# 缓存的评论数
sqlite3 $DB "SELECT COUNT(*) FROM xhh_comment_cache;"

# ===== 数据库大小 =====

ls -lh /opt/Openxhh/sql.db
```

</details>

## 默认配置速查

这些值会在缺失或为 0 时自动补齐，或是本文推荐的恢复值。误改后可以按这里恢复。

| 配置项 | 默认 / 推荐值 | 说明 |
| --- | --- | --- |
| `xhh.checkTime` | `60` | 检查 @ 间隔，秒 |
| `xhh.replyTime` | `30` | 回复间隔，秒 |
| `xhh.minRequestInterval` | `0.5` | 小黑盒 API 全局最小请求间隔，秒；并发请求自动排队 |
| `xhh.maxReplyThreads` | `3` | 普通用户最高回复并发；owner 不占用普通用户线程槽位 |
| `xhh.maxPendingReplies` | `50` | 普通用户全局待回复队列上限 |
| `xhh.maxPendingRepliesPerUser` | `5` | 普通用户单用户待回复队列上限 |
| `xhh.enableWhitelist` | `false` | 默认关闭白名单，回复所有 @ |
| `xhh.baseUrl` | `https://api.xiaoheihe.cn` | 小黑盒 API 地址；VPS 出站 IP 被拒时可改为 Cloudflare Worker 地址 |
| `xhh.webver` | `2.5` | 小黑盒 Web 版本字段 |
| `xhh.version` | `999.0.4` | 小黑盒版本字段 |
| `database.type` | `sqlite` | 个人部署推荐 SQLite |
| `ai.prompt` | `请根据评论内容自然回复。` | VPS Web UI 默认回复策略 |
| `ai.webSearch` | `true` | 普通文字回复默认启用模型联网搜索 |
| `ai.forceWebSearch` | `false` | 不强制每次回复都必须调用搜索工具 |
| `ai.searchContextSize` | `medium` | 搜索上下文强度，可填 `low` / `medium` / `high` |
| `feedReply.enabled` | `false` | 自动刷帖默认关闭 |
| `feedReply.interval` | `900` | 自动刷帖轮询间隔，秒 |
| `feedReply.maxPerRun` | `1` | 每轮最多处理帖子数 |
| `feedReply.maxPerDay` | `10` | 每天最多处理帖子数 |
| `feedReply.dryRun` | `true` | 默认只记录，不真实发评论 |
| `feedReply.prompt` | 自动刷帖专用 Prompt | 控制主动回复风格和 `SKIP` 判断 |
| `image.model` | `gpt-image-2` | 图片模型默认值 |
| `image.size` | `1024x1024` | 图片尺寸默认值 |
| `image.responseFormat` | `b64_json` | 图片接口输出格式 |
| `image.outputDir` | `images` | 本地生成图片临时目录 |
| `image.uploadMode` | `cos` | 推荐使用小黑盒官方图床 |
| `image.externalDir` | 空 | 旧 VPS 静态图床备用字段，推荐留空 |
| `image.externalBaseUrl` | 空 | 旧 VPS 静态图床备用字段，推荐留空 |
| `image.promptRefine` | `false` | 默认不启用图片 prompt 优化 |
| `image.promptMaxChars` | `1000` | 图片 prompt 优化输入最大字符数 |
| VPS Web UI 端口 | `29173` | 默认访问 `http://服务器IP:29173` |
| VPS Web UI 服务名 | `Openxhh-webui` | 更新脚本默认识别和重启 |
| 机器人服务名 | `Openxhh` | 更新脚本默认识别和重启 |

说明：当前推荐小黑盒官方图床，`image.externalDir` 和 `image.externalBaseUrl` 只在恢复旧 VPS 静态图床时才需要填写；自动刷帖默认关闭且 dry-run，建议先在 Web UI 观察记录再真实发评。

## 免责声明

本项目仅供个人学习和自用。自动化访问、自动回复、自动生图和频繁请求都可能触发平台风控。请自行控制频率，并遵守小黑盒相关规则。

## 致谢

感谢 [SomeOvO/xhhRobot](https://github.com/SomeOvO/xhhRobot) 原项目提供早期基础思路与实现参考。也感谢所有测试、反馈和提出建议的朋友。
