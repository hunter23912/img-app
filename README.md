# Img App

一个面向手机端的图片生成应用。前端使用 React、Vite 和 TypeScript，后端使用 Go，负责调用 OpenAI Images API 兼容的图片中转站。

## 当前功能

- 文生图和图编辑。
- 模型选择：固定提供以下三个模型：
  - gpt-image-2
  - grok-imagine-image-2.0
  - gemini3.1-flash-image
- 支持手动添加和删除自定义模型。自定义模型保存在 SQLite 中，并按当前中转站配置隔离。
- 支持保存多个中转站配置，并手动切换当前配置。
- 提示词预设：内置预设、创建、编辑、删除，以及从旧版 localStorage 迁移自定义预设。
- 生成历史：SQLite 保存最近 50 条成功的生成或编辑记录，支持分页和删除。
- 结果下载：支持 PNG 原图和指定质量的 JPG 转换，透明区域转 JPG 时使用白色背景。
- 支持 OpenAI Images 兼容的同步接口，以及带 `task_id` 的异步图片接口；异步接口会自动轮询任务状态。
- 外部图片优先由手机直接读取和转换，跨域不允许时自动回退到 Go 后端；远程结果会在后台缓存到 SQLite。
- 支持浅色、深色主题和移动端布局。

当前版本不会自动请求中转站的 /v1/models。/api/models 是本应用自己的模型管理接口，用来返回三个固定模型和当前配置下保存的自定义模型；它不是中转站的模型查询接口。

## 项目结构

~~~text
img-app/
├── src/
│   ├── api/              前端 API 请求
│   ├── components/       页面组件和表单组件
│   ├── constants/        模型、尺寸等固定选项
│   ├── hooks/            健康检查、历史、配置、模型和预设状态
│   ├── types/            TypeScript 类型
│   └── utils/             提示词预设工具
├── public/               静态资源
├── backend/
│   ├── main.go           后端启动和依赖组装入口
│   ├── internal/
│   │   ├── config/       环境变量和运行配置
│   │   ├── store/        SQLite、迁移和持久化操作
│   │   ├── provider/     中转站图片接口调用
│   │   ├── imageops/     图片来源校验、解码、下载和格式转换
│   │   ├── history/      无数据库场景的内存历史记录
│   │   ├── httpapi/      路由、handler、响应和中间件
│   │   └── logging/      日志初始化
│   ├── go.mod
│   └── go.sum
├── vite.config.ts        Vite 开发配置和 API 代理
├── package.json
├── deploy.sh             构建并上传脚本
└── test.http             中转站接口调试示例
~~~

backend/main.go 只负责启动流程。新增 HTTP 接口时放入 backend/internal/httpapi，数据库操作放入 store，中转站请求放入 provider，不要把具体业务重新堆回 main.go。

## 快速开始

### 1. 安装依赖

需要 Node.js、pnpm 和 Go 1.25 或更高版本。

~~~bash
pnpm install
~~~

### 2. 配置后端

可以在 backend/.env 中配置以下变量。这个文件已被 Git 忽略，不要把真实 API key 提交到仓库。

~~~dotenv
IMG_ENDPOINT=https://task-api-1-cn.65535.space
IMG_API_KEY=你的中转站APIKey
APP_ADDR=localhost:8080
APP_DB_PATH=data/img-app.db
~~~

变量说明：

| 变量 | 必填 | 说明 |
| --- | --- | --- |
| IMG_ENDPOINT | 否 | 图片中转站地址，默认 https://task-api-1-cn.65535.space |
| IMG_API_KEY | 否 | 中转站 API key；未配置时后端仍可启动，但不能生成或编辑图片 |
| APP_ADDR | 否 | 后端监听地址，默认 localhost:8080 |
| APP_DB_PATH | 否 | SQLite 路径，默认是启动后端时工作目录下的 data/img-app.db |

进程环境变量优先于 .env。后端从 backend/ 目录启动时读取 backend/.env；从项目根目录启动时也会兼容读取 backend/.env。

Endpoint 建议填写中转站 API 基础地址，例如：

~~~text
https://example.com
https://example.com/v1
https://example.com/gateway/v1
~~~

当 endpoint 没有路径时，后端会自动补上 /v1；当 endpoint 已包含路径时，则保留现有路径，只追加 /images/...：

~~~text
endpoint = https://example.com
POST https://example.com/v1/images/generations
POST https://example.com/v1/images/edits

endpoint = https://example.com/gateway/v1
POST https://example.com/gateway/v1/images/generations
POST https://example.com/gateway/v1/images/edits
~~~

### 3. 启动 Go 后端

~~~bash
cd backend
go run .
~~~

默认监听 http://localhost:8080。

### 4. 启动前端

另开一个终端，在项目根目录执行：

~~~bash
pnpm dev
~~~

打开 Vite 输出的地址，通常是 http://localhost:5173。开发服务器会把前端的 /api 请求代理到 localhost:8080。

## 配置和数据

网页的“配置”页面可以保存多个中转站配置。第一个新建的配置会自动启用，之后可以手动切换。启用的配置优先于环境变量和单配置设置；没有启用配置时，后端使用环境变量或网页保存的单配置设置。

SQLite 数据库会自动创建并执行迁移，主要保存：

- 中转站配置和 API key。
- 自定义模型。
- 提示词预设。
- 图片生成历史及必要的 base64 图片数据。

数据库文件默认位于 backend/data/img-app.db，部署或升级时必须保留。后端会尝试将数据库文件权限设置为仅当前用户可读写。正式部署建议使用独立的绝对路径，例如：

~~~dotenv
APP_DB_PATH=/var/lib/img-app/img-app.db
~~~

当前配置接口会返回完整 API key，适合单用户、自托管场景。若要公开部署，应在反向代理或应用层增加认证、访问控制和限流。

## 接口

所有接口都由 Go 后端提供。错误响应统一为：

~~~json
{
  "error": "错误信息"
}
~~~

### 接口总览

| 方法 | 路径 | 作用 |
| --- | --- | --- |
| GET | /api/health | 健康检查和 API key 配置状态 |
| GET / PUT | /api/settings/image | 读取或保存单配置图片服务设置 |
| GET / POST | /api/image-profiles | 获取或新增中转站配置 |
| PUT / DELETE | /api/image-profiles/{id} | 编辑或删除中转站配置 |
| POST | /api/image-profiles/{id}/activate | 切换当前中转站 |
| GET / POST | /api/models | 获取或添加模型 |
| DELETE | /api/models/{id} | 删除自定义模型 |
| GET | /api/history | 分页获取成功的生成历史 |
| GET | /api/history/{taskID}/image | 获取历史中的 base64 图片 |
| DELETE | /api/history/{taskID} | 删除一条历史记录 |
| GET / POST | /api/presets | 获取或创建提示词预设 |
| PUT / DELETE | /api/presets/{id} | 编辑或删除提示词预设 |
| POST | /api/presets/import | 批量导入提示词预设 |
| POST | /api/generate | 文生图 |
| POST | /api/edit | 图编辑 |
| POST | /api/download/image | 下载或转换图片 |

### 模型管理

固定模型始终由后端返回，不能删除：

~~~text
gpt-image-2
grok-imagine-image-2.0
gemini3.1-flash-image
~~~

添加自定义模型：

~~~http
POST /api/models
Content-Type: application/json

{"model":"vendor/custom-image"}
~~~

自定义模型名会去除首尾空格，同一中转站配置下不允许重复。切换中转站后，只能使用固定模型和该配置保存的自定义模型。

### 文生图

~~~http
POST /api/generate
Content-Type: application/json

{
  "model": "gpt-image-2",
  "prompt": "一只白色猫坐在窗边，柔和自然光",
  "size": "720x1280",
  "quality": "auto"
}
~~~

当前网页提供 1:1 和 9:16 两种比例，以及 1k、2k 两种分辨率：

| 比例 | 1k | 2k |
| --- | --- | --- |
| 1:1 | 1024x1024 | 2048x2048 |
| 9:16 | 720x1280 | 1440x2560 |

使用 `?stream=false` 时请求成功后返回：

~~~json
{
  "image": "/api/history/{taskID}/image"
}
~~~

默认请求返回 Server-Sent Events（SSE）：后端先返回 started 事件，收到上游部分图片时转发 partial_image 事件，最终返回 completed 事件。上游不支持 SSE 时，后端会把普通 JSON 结果包装成 completed 事件。请求 `?stream=false` 可兼容旧版 JSON 响应。

#### 异步接口接入

部分中转站（例如 Mikoto）不会在一次请求中直接返回图片，而是先创建任务：

~~~text
POST /v1/images/generations/async
        ↓
{ "task_id": "img_xxx", "status": "running" }
        ↓ 每 3 秒查询
GET /v1/images/tasks/img_xxx
        ↓
{ "status": "success", "result": { "data": [...] } }
~~~

当 endpoint 是 `api.mikoto.vip`，或用户配置的 endpoint 已以 `/async` 结尾时，后端会自动使用异步路径。生成和编辑分别使用 `/images/generations/async`、`/images/edits/async`，并将 `n` 限制为 1（服务商文档要求异步接口单张生成）。异步创建请求会请求 `response_format: "b64_json"`；如果服务商仍返回 URL，后端也支持读取 `result.data[].url`。

后端在本次请求期间保留创建响应中的 `task_id`，并使用同一个 API key 轮询任务。`queued` 和 `running` 状态继续等待，`success` 从 `result.data` 提取 `url` 或 `b64_json`，`error`/`failed` 读取 `error.message`。轮询遇到 408、429、5xx 或临时网络错误只重试查询，不会重新提交生图请求，避免重复扣费。一次异步调用的总等待上限由 relay 的 10 分钟请求上下文控制，单次状态查询最多等待 60 秒。当前异步等待依附于这次 HTTP 请求；如果后端进程在完成前重启，需要重新提交任务。

后端对前端仍可返回 SSE：先发送 `started`，异步任务完成后发送 `completed`；等待期间每 15 秒发送 SSE 注释心跳，避免反向代理把长时间无数据的连接关闭。使用 `?stream=false` 时，等待完成后直接返回 JSON。

#### 图片结果为什么可能“不显示”

服务商的成功响应可能是以下任一种格式：

~~~text
data[0].url       外部图片 URL
data[0].b64_json  Base64 图片数据
image/*           直接返回的二进制图片
SSE               流式 partial/final 事件
~~~

后端会把 Base64 和二进制统一为可识别的图片数据，并兼容 URL 结果。URL 可以直接用于 `<img>` 显示；如果服务商返回的是临时 URL、URL 过期、响应被截断或中转站没有允许浏览器读取，直接下载或历史恢复就可能失败。生成完成后，启用 SQLite 时后端会在后台把外部 URL 下载并保存为 Base64，保存成功后历史列表自动改为 `/api/history/{taskID}/image`，不再依赖远程 URL。缓存失败时仍保留原 URL，便于稍后重试。

### 图编辑

~~~http
POST /api/edit
Content-Type: multipart/form-data
~~~

表单字段：

| 字段 | 必填 | 说明 |
| --- | --- | --- |
| model | 否 | 模型名，默认 gpt-image-2 |
| prompt | 是 | 编辑要求 |
| size | 否 | 输出尺寸；网页选择“原图”时使用原图尺寸 |
| quality | 否 | 当前网页传 auto |
| moderation | 否 | 内容审核级别，默认 auto |
| output_format | 否 | 输出格式，默认 png |
| n | 否 | 生成数量，默认 1 |
| image | 是 | 图片文件；可重复提交多张，第一张为主图，其余为参考图，网页最多 4 张 |
| mask | 否 | 遮罩文件 |

手机到 Go 后端使用 multipart 上传，Go 再按 Playground 和 OpenAI Images API 的传统编辑方式，以 multipart 转发到中转站的 `/v1/images/edits` 或 `/v1/images/edits/async`：图片字段可重复提交，后端按顺序转发为 `image[]`，第一张是主图，其余是参考图；遮罩字段为 `mask`。提示词中的 `@1`、`@2` 等编号对应图片顺序，网页输入 `@` 时会提供图片补全。异步编辑创建成功后仍使用同一个任务查询接口轮询；原图不会写入 SQLite，也不需要 `APP_PUBLIC_URL` 或公网 URL。

### 历史记录

~~~http
GET /api/history?limit=5&cursor=...
DELETE /api/history/{taskID}
~~~

limit 范围为 1-5，不传时为 5。返回格式：

~~~json
{
  "tasks": [],
  "next_cursor": "",
  "has_more": false
}
~~~

新图片返回后会先记录任务和原始结果；如果结果是外部 HTTPS URL，后端在后台下载并缓存 Base64。缓存完成后，SQLite 的 `image_url` 保存本地图片数据，历史列表只返回 `/api/history/{taskID}/image`，而 `source_url` 保留原始 URL 用于追踪和兼容旧记录。后台缓存不会阻塞前端的生成完成事件。

旧历史记录中的外部 HTTPS URL 也支持迁移式缓存：第一次查看或下载时，如果本地还没有 Base64，后端会读取远程图片并写回同一条 SQLite 记录；以后从历史路径读取，不再访问中转站。远程链接已经过期或服务端不再提供完整图片时，无法从本地恢复不存在的数据。

### 图片下载

~~~http
POST /api/download/image
Content-Type: application/json

{
  "source": "当前图片 URL 或 data URL",
  "format": "jpg",
  "quality": 95
}
~~~

format 支持 png 和 jpg，输入支持 PNG、JPEG、GIF、WebP。非 PNG 原图在选择 PNG 时会实际转换。外部图片 URL 必须是 HTTPS，且必须是当前后端生成或数据库中的可信来源；data URL 可以直接处理。JPG 质量范围为 1-100，默认 95。日志分别记录首字节、原图读取和格式转换耗时，便于区分远程传输慢与压缩慢。

对于外部 HTTPS 图片，前端下载会优先让手机直接 `fetch` 图片：PNG 直接保存，JPG 使用手机端 Canvas 转换后保存。这样不会再经过“手机 → Go → 中转站 → Go → 手机”的重复传输。若服务商没有返回允许当前网页来源的 CORS，或浏览器无法读取图片，前端自动回退到 `/api/download/image`，由 Go 后端读取并转换。历史路径和 Base64 结果始终直接使用本地后端数据。

第一次下载外部大图可能仍然较慢，因为必须先把完整原图传输到手机或 Go 后端；JPG 编码通常只占很短时间。后端对支持 `Accept-Ranges: bytes` 且大于 2 MB 的图片使用 4 路 Range 并发读取，并记录首字节、原图读取和转换耗时。成功缓存后，后续不同 JPG 质量的下载只读取本地原图，通常会很快。

## 开发和验证

后端包结构调整后，后端仍保持原来的启动和构建方式：

~~~bash
cd backend
go test ./...
go vet ./...
go build -o /tmp/img-app-backend .
cd ..
pnpm lint
pnpm build
git diff --check
~~~

如果只改前端，可以使用：

~~~bash
pnpm lint
pnpm build
~~~

## 部署

执行：

~~~bash
./deploy.sh
~~~

当前脚本会：

1. 构建前端 dist/。
2. 使用 GOOS=linux GOARCH=amd64 编译后端。
3. 通过 scp 上传 dist/、后端二进制和 backend/.env 到脚本中配置的 server SSH 主机。

脚本不会登录服务器重启后端，需要根据实际进程管理方式手动重启，例如：

~~~bash
ssh server 'sudo systemctl restart img-app-backend'
~~~

生产环境建议由 Nginx 托管 dist/，并将 /api 反向代理到 Go 后端。升级时只替换前端文件和后端二进制，不要删除或覆盖 SQLite 数据库。

## 已知限制

- 当前只按 OpenAI Images API 兼容格式调用中转站，不同中转站的额外参数不会自动适配。
- 普通中转站使用同步等待；支持异步路径的中转站会按任务 ID 轮询，单次 relay 请求最长等待约 10 分钟。
- 没有用户系统、登录、鉴权、限流和计费功能，不适合直接暴露到公网。
- 不会自动查询中转站 /v1/models；模型列表需要使用固定模型或手动添加自定义模型。
- 尚未缓存的旧历史图片仍依赖中转站 URL 的有效期；成功读取一次后即可本地保存。

## 常见问题

### “模型列表暂时无法加载，当前仍可使用固定模型”是什么原因？

这是前端请求本应用的 /api/models 失败时显示的降级提示，不是因为没有创建自定义模型。固定模型仍然可以使用。通常检查：

1. Go 后端是否已启动。
2. 后端端口是否仍为 localhost:8080，是否与 vite.config.ts 一致。
3. 浏览器访问 /api/models 是否返回 200。
4. 当前运行的后端是否为最新构建版本。

后端正常启动且数据库可读时，/api/models 会至少返回三个固定模型，即使没有任何自定义模型。

### 自定义模型为什么提示“模型不可用”？

自定义模型按当前中转站配置隔离。切换配置后，之前配置下添加的自定义模型不会出现在当前列表中；请在当前配置下重新添加，或使用固定模型。
