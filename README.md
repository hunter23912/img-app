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

请求成功后返回：

~~~json
{
  "image": "https://example.com/image.png"
}
~~~

中转站返回 b64_json 时，后端会转换为 data URL。成功结果会写入 SQLite 历史记录；外部 URL 会加入后端信任列表，供下载接口校验。

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
| image | 是 | 原图文件 |
| mask | 否 | 可选遮罩文件，供 API 调用者使用 |

图编辑请求会以 multipart 形式转发给中转站的 /v1/images/edits。

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

中转站返回的 HTTPS 图片保存为 URL；返回 base64 图片时，历史列表中的 image 会指向 /api/history/{taskID}/image，避免一次性把多个大图放进分页响应。

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

format 支持 png 和 jpg。外部图片 URL 必须是 HTTPS，且必须是当前后端生成或恢复的可信来源；data URL 可以直接处理。JPG 质量范围为 1-100，默认 95。

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
- 生成和编辑使用同步等待模式，后端请求超时约为 330 秒。
- 没有用户系统、登录、鉴权、限流和计费功能，不适合直接暴露到公网。
- 不会自动查询中转站 /v1/models；模型列表需要使用固定模型或手动添加自定义模型。
- 历史中的外部图片仍依赖中转站 URL 的有效期；base64 结果才会由本地数据库保存图片数据。

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
