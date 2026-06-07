# Img App

一个极简的手机端网页应用：前端使用 React + Vite + TypeScript，后端使用 Go。Go 后端从环境变量读取中转站 endpoint 和 API key，通过后端代理调用 `gpt-image-2` 模型，实现文生图和图编辑。

## 当前项目状态

当前项目已经完成了基础初始化：

- 前端：Vite + React + TypeScript，位于项目根目录。
- 样式：Tailwind CSS + daisyUI。
- 后端：Go module，位于 `backend/` 目录。
- 已实现：`GET /api/health` 健康检查。
- 已实现：`POST /api/generate` 文生图代理接口。
- 已实现：`POST /api/edit` 图编辑代理接口。
- 已实现：手机端基础页面和 Vite `/api` 代理。

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
- 文生图可选择输出尺寸。
- 图编辑默认保持原图尺寸，也可以手动选择输出尺寸。
- 调用 `gpt-image-2` 生成图片。
- 在页面展示生成结果。
- 支持保存或下载结果图。

暂时不做用户系统、历史记录、计费、数据库、复杂图片管理。

## 推荐架构

前端不要直接调用中转站，而是：

```txt
React 页面 -> Go 后端 API -> 中转站 gpt-image-2 接口
```

原因：

- 可以避免浏览器跨域问题。
- 可以统一处理上传图片、错误信息和返回格式。
- 可以避免把 API key 暴露给第三方脚本或浏览器网络插件。
- 后续如果要增加鉴权、日志、限流、缓存，也更容易。

当前这个项目由 Go 后端读取 `backend/.env`：

```txt
IMG_API_KEY      必填，中转站 API key
IMG_ENDPOINT     可选，默认 https://img-cn.65535.space
```

前端不展示、不提交 API key。

## 前后端接口设计

先设计两个后端接口：

```txt
POST /api/generate
POST /api/edit
```

### `POST /api/generate`

用于文生图。

前端提交：

```json
{
  "model": "gpt-image-2",
  "prompt": "一只白色猫坐在窗边，柔和自然光",
  "size": "1024x1024",
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

当前按中转站 OpenAI 兼容接口实现。页面里 endpoint 默认是：

```txt
https://img-cn.65535.space
```

后端会自动拼接成：

```txt
POST {endpoint}/v1/images/generations
POST {endpoint}/v1/images/edits
```

如果你想直接输入完整地址，也可以填：

```txt
https://img-cn.65535.space/v1/images/generations
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
fetch('/api/health')
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

先创建 `backend/.env`：

```env
IMG_API_KEY=你的中转站 API key
IMG_ENDPOINT=https://img-cn.65535.space
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

## 下一步

当前已经完成 Go 后端最小服务、文生图接口和图编辑接口：

- `GET /api/health` 返回 `{ "ok": true }`
- 服务监听 `localhost:8080`
- 前端后续通过 Vite 代理访问它
- `POST /api/generate` 会调用兼容 OpenAI Images API 的中转站。
- `POST /api/edit` 会转发 multipart 图片到中转站。

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

后续下一步是用真实 API key 测试文生图和图编辑，再根据中转站实际返回错误做细节调整。
