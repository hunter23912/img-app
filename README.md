# Img App

一个极简的手机端网页应用：前端使用 React + Vite + TypeScript，后端使用 Go。后端代理调用中转站的 `gpt-image-2-lite` 模型，实现文生图和图编辑；图片服务 endpoint 和 API key 可以在网页中配置并保存到 SQLite。

## 当前项目状态

当前项目已经完成了基础初始化：

- 前端：Vite + React + TypeScript，位于项目根目录。
- 样式：Tailwind CSS + daisyUI。
- 后端：Go module，位于 `backend/` 目录。
- 已实现：`GET /api/health` 健康检查。
- 已实现：SQLite 持久化的 `GET /api/history?limit=5&cursor=...` 和 `DELETE /api/history/{taskID}`。
- 已实现：SQLite 持久化的共享提示词预设接口和首次启动种子迁移。
- 已实现：`POST /api/generate` 文生图代理接口。
- 已实现：`POST /api/edit` 图编辑代理接口。
- 已实现：手机端基础页面和 Vite `/api` 代理。
- 已实现：`POST /api/compress` 本地单图图片压缩。
- 已实现：`POST /api/compress/batch` 本地批量图片压缩并打包 zip。
- 已实现：`POST /api/watermark/remove` 本地遮罩去水印。

目录大致如下：

```txt
img-app/
  src/                 React 前端源码
  public/              前端静态资源
  package.json         前端依赖和脚本
  vite.config.ts       Vite 配置
  backend/
    go.mod             Go 后端模块
```

注意：`go mod init backend` 或 `go mod init img-app/backend` 只会在当前目录生成 `go.mod`，不会自动创建 `backend` 文件夹。因此正确顺序是先创建并进入后端目录，再执行 Go 初始化：

```powershell
mkdir backend
cd backend
go mod init img-app/backend
```

## 目标功能

第一版只做必要功能，适合手机上使用：

- 输入文生图 prompt。
- 上传一张原图用于图编辑。
- 文生图和图编辑均可选择模型、尺寸（比例）和分辨率。
- 图编辑额外支持使用原图尺寸；SeedVR2-7B 保持原图比例并提供 1K/2K/4K 选项。
- 默认调用 `gpt-image-2-lite` 生成图片，默认输出 9:16 的 1K 尺寸（`720x1280`）；9:16 的 2K 尺寸为 `1440x2560`。
- 在页面展示生成结果。
- 支持保存或下载结果图。

暂时不做用户系统、计费和复杂图片管理。

图片压缩当前也走本地算法：前端上传图片和参数，Go 后端用标准库解码图片，再按原尺寸重编码为 JPG 或 PNG。JPG 使用质量参数控制体积，PNG 使用最高压缩等级重编码并剥离原始元数据。批量压缩会复用同一套压缩逻辑，把成功处理的图片和 `manifest.json` 明细打进 zip 返回。

去水印当前优先走本地算法：前端上传原图和涂抹生成的 mask，Go 后端在本机做遮罩邻域扩散修复并返回 PNG。这个方案不消耗中转站额度，也不会把图片发给外部模型；适合小面积文字水印、纯色或渐变背景。复杂纹理、人脸、文字内容被水印覆盖时，效果会明显弱于专业图像修复模型。

## 推荐架构

前端不要直接调用中转站，而是：

```txt
React 页面 -> Go 后端 API -> 中转站 gpt-image-2-lite 接口
```

原因：

- 可以避免浏览器跨域问题。
- 可以统一处理上传图片、错误信息和返回格式。
- 浏览器只请求本应用后端，避免直接处理跨域中转站请求；API key 的配置和转发集中在本应用接口中。
- 后续如果要增加鉴权、日志、限流、缓存，也更容易。

当前项目支持进程环境变量、`.env` 和网页配置三种来源：

```txt
IMG_API_KEY      可选，中转站 API key
IMG_ENDPOINT     可选，默认 https://task-api-1-cn.65535.space
APP_DB_PATH      可选，默认 data/img-app.db（相对于启动后端时的工作目录）
```

后端启动时会读取当前工作目录下的 `.env`；从项目根目录启动时也会兼容读取 `backend/.env`。进程环境变量优先于 `.env`。缺少 API key 时后端仍会启动并提供健康检查和配置接口，但生成或编辑请求会返回配置错误。endpoint 为空时使用内置默认值。

网页的“配置”页面可以保存多个中转站配置，并手动切换当前配置。配置存储在 SQLite 的 `image_profiles` 表中；旧版单配置数据会在升级时迁移为“默认配置”。API key 按个人使用场景明文保存在 SQLite 中，数据库文件会设置为仅当前用户可读写，日志不会输出 key。没有激活配置时，后端继续使用环境变量或默认 endpoint。

SQLite 数据库由后端使用纯 Go 的 `modernc.org/sqlite` 驱动自动创建和迁移。部署时请保留数据库文件；正式环境建议使用独立绝对路径，例如 `APP_DB_PATH=/var/lib/img-app/img-app.db`。

### 后续部署

可以运行：

```bash
./deploy.sh
```

但当前脚本只执行前端构建、后端交叉编译和 `scp` 上传，不负责登录服务器或重启远端进程。执行完成后，需要按照服务器上的进程管理方式重启后端，例如：

```bash
ssh server 'sudo systemctl restart img-app-backend'
```

如果使用 `supervisor`、`pm2`、Docker 或手工后台进程，请替换为对应的重启命令。请确认远端服务的工作目录、`.env`/进程环境变量和 `APP_DB_PATH` 仍然存在；升级时只替换二进制和前端 `dist`，不要删除 SQLite 数据库文件。

## 前后端接口设计

后端接口：

```txt
GET /api/history
DELETE /api/history
GET /api/settings/image
PUT /api/settings/image
GET /api/image-profiles
POST /api/image-profiles
PUT /api/image-profiles/{id}
DELETE /api/image-profiles/{id}
POST /api/image-profiles/{id}/activate
POST /api/generate
POST /api/edit
POST /api/download/image
```

### `GET /api/history` 和 `DELETE /api/history`

历史记录由 SQLite 保存最近 50 条生成/编辑任务，所有访问同一个后端的设备共享这份列表。前端首屏请求 5 条，使用 `cursor` 继续分页。普通结果保存 HTTPS 图片 URL；SeedVR2-7B 返回的 base64 图片结果会持久化在后端，并通过 `/api/history/{taskID}/image` 按需读取，避免历史分页把多张大图一次性传到手机端。

查询返回：

```json
{
	"tasks": [{
		"id": "task-id",
		"mode": "generate",
		"status": "succeeded",
		"image": "https://example.com/image.png",
		"error": ""
	}],
	"next_cursor": "...",
	"has_more": false
}
```

删除时使用 `DELETE /api/history/{taskID}`，成功返回 `204 No Content`。

提示词预设使用 `GET/POST /api/presets`、`PUT/DELETE /api/presets/{id}` 和 `POST /api/presets/import`。数据库首次创建时自动写入 7 条内置预设；浏览器旧版本 localStorage 中的自定义预设会尝试迁移一次。主题跟随系统，并在用户切换后保存在当前浏览器。

### `GET /api/settings/image` 和 `PUT /api/settings/image`

读取或保存当前生效的图片服务配置：

```json
{
  "endpoint": "https://task-api-1-cn.65535.space",
  "api_key": "你的中转站 API key"
}
```

`PUT` 时 endpoint 必须是合法的 `http` 或 `https` 地址；两个字段都可以为空。保存为空表示清除 SQLite 覆盖值，重新使用 `.env` 或进程环境变量。响应会返回当前生效值，网页打开时会自动读取它。

### `/api/image-profiles`

用于保存和切换多个中转站。第一个新建的配置会自动激活；激活其他配置使用 `POST /api/image-profiles/{id}/activate`。当前激活配置不能直接删除，必须先切换到其他配置。配置列表会返回完整 API key，适合当前单用户自托管场景。

### `POST /api/generate`

用于文生图。

前端提交：

```json
{
  "model": "gpt-image-2-lite",
  "prompt": "一只白色猫坐在窗边，柔和自然光",
  "size": "720x1280",
  "quality": "auto"
}
```

后端负责：

- 校验 prompt 是否为空。
- 拼接中转站图片生成接口地址。
- 使用 `Authorization: Bearer <IMG_API_KEY>` 调用中转站。
- 返回图片 URL 或 base64 数据给前端。

### `POST /api/edit`

用于图编辑。

前端提交：

- prompt
- image 文件
- 可选 mask 文件
- size 可选；为空时后端不传该字段，中转站按默认处理

后端负责：

- 接收 multipart form-data。
- 将图片和 prompt 转发给中转站图编辑接口。
- 返回编辑后的图片结果。

### `POST /api/download/image`

用于下载生成结果。请求体为：

```json
{
  "source": "当前图片 URL 或 data URL",
  "format": "jpg",
  "quality": 95
}
```

`format` 支持 `png` 和 `jpg`。PNG 直接返回原始图片数据；JPG 在后端解码后按 `quality`（`1-100`，缺省为 `95`）编码，透明区域使用白色填充。外部图片 URL 仅允许 HTTPS，并且必须是当前 Go 后端刚从生成或编辑接口收到的图片 URL；`data:image/...` 可直接处理。

当前按中转站 OpenAI 兼容接口实现。页面里 endpoint 默认是：

```txt
https://task-api-1-cn.65535.space
```

后端会自动拼接成：

```txt
POST {endpoint}/v1/images/generations
POST {endpoint}/v1/images/edits
```

如果你想直接输入完整地址，也可以填：

```txt
https://task-api-1-cn.65535.space/v1/images/generations
```

中转站同步等待模式最多可能阻塞 5 分钟，Go 后端 HTTP client 当前设置了 330 秒超时。

## 开发步骤

### 1. 整理后端骨架

在 `backend/` 中创建：

```txt
backend/
  go.mod
  main.go
```

`main.go` 先只做三件事：

- 启动 HTTP 服务。
- 提供 `/api/health` 健康检查。
- 开启基本 CORS，方便 Vite 开发环境访问。

### 2. 配置前端代理

在 `vite.config.ts` 中配置 `/api` 代理到 Go 后端，例如：

```ts
server: {
  proxy: {
    '/api': 'http://localhost:8080',
  },
}
```

这样前端可以直接请求：

```ts
fetch("/api/health");
```

不用关心后端实际端口。

### 3. 做手机端基础页面

React 页面先分成几个区域：

- 后端配置状态区。
- 模式切换：文生图 / 图编辑。
- prompt 输入区。
- 图片上传区，仅图编辑模式显示。
- 生成按钮。
- 结果预览区。

当前样式方案使用 Tailwind CSS + daisyUI，主要使用 daisyUI 的 `card`、`input`、`textarea`、`select`、`tabs`、`btn`、`alert` 等组件类名。

### 4. 实现文生图接口

先实现 `/api/generate`：

- 前端把 prompt、size 发送给 Go。
- Go 调用中转站。
- Go 把图片结果返回给 React。
- React 展示生成结果。

这一阶段先把文生图跑通，不处理图编辑。

### 5. 实现图编辑接口

再实现 `/api/edit`：

- 前端上传图片和 prompt。
- Go 解析 multipart form-data。
- Go 转发给中转站。
- React 展示编辑结果。

### 6. 打包与部署

开发阶段：

在启动 Go 后端的同一个 PowerShell 终端中设置环境变量：

```powershell
$env:IMG_API_KEY = "你的中转站 API key"
$env:IMG_ENDPOINT = "https://task-api-1-cn.65535.space"
```

然后启动后端：

```powershell
cd backend
go run .
```

另开一个终端：

```powershell
pnpm dev
```

生产阶段：

```powershell
pnpm build
```

后续可以让 Go 后端托管前端生成的 `dist/` 文件，这样最终只需要启动一个 Go 服务。

## 实现顺序建议

推荐按这个顺序做：

1. 写 `backend/main.go`，跑通 `/api/health`。
2. 配置 Vite 代理，确认 React 能请求 `/api/health`。
3. 重写前端页面，做手机端表单和结果展示。
4. 实现 `/api/generate`。
5. 接入真实中转站，调通文生图。
6. 实现 `/api/edit`。
7. 优化错误提示、加载状态和下载按钮。

不要一开始就同时做文生图、图编辑、部署和美化。先把最短链路跑通：

```txt
页面输入 prompt -> Go 后端 -> 中转站 -> 返回图片 -> 页面展示
```

## 当前接口

当前已经完成这些接口：

- `GET /api/health` 返回 `{ "ok": true }`
- 服务监听 `localhost:8080`
- 前端后续通过 Vite 代理访问它
- `POST /api/generate` 会调用兼容 OpenAI Images API 的中转站。
- `POST /api/edit` 会转发 multipart 图片到中转站。
- `POST /api/compress` 会在本地保持原尺寸重编码单张 JPG/PNG。
- `POST /api/compress/batch` 会在本地批量压缩多张图片，返回 zip 压缩包。
- `POST /api/watermark/remove` 会在本地根据 mask 修复标记区域。

本地运行方式：

```powershell
cd backend
go run .
```

另开一个终端：

```powershell
pnpm dev
```

打开 Vite 输出的本地地址，通常是：

```txt
http://localhost:5173
```

后续下一步是继续优化本地去水印效果，补充更多后端测试，并在需要发布时让 Go 后端直接托管 `dist/`。
