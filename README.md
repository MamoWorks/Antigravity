# Antigravity

一个高性能的 API 代理服务器，提供 OpenAI、Claude 和 Gemini 兼容接口。

## 功能特性

- **多 API 兼容**：支持 OpenAI、Claude 和 Gemini API 格式
- **流式响应**：支持 SSE (Server-Sent Events) 流式输出
- **自动令牌刷新**：内置 OAuth 令牌自动刷新机制
- **多平台支持**：提供 Windows、macOS、Linux 多平台预编译二进制文件
- **Docker 部署**：支持 Docker 和 Docker Compose 一键部署
- **轻量级设计**：基于 scratch 镜像，极小体积
- **多架构支持**：支持 AMD64 和 ARM64 架构

## 快速开始

### 使用预编译二进制文件

从 [Releases](https://github.com/MamoCode/Antigravity/releases) 下载对应平台的二进制文件：

### 使用 Docker

**使用 Docker Compose (推荐):**
```bash
docker compose -f docker/docker-compose.yml up -d
```

**使用 Docker CLI:**
```bash
docker pull ghcr.io/mamocode/antigravity:latest
docker run -d -p 8000:8000 --name antigravity ghcr.io/mamocode/antigravity:latest
```

### 从源码构建

**环境要求:**
- Go 1.21 或更高版本

**构建步骤:**
```bash
# 克隆仓库
git clone https://github.com/MamoCode/Antigravity.git
cd antigravity

# 安装依赖
go mod download

# 构建
go build -o antigravity ./cmd/server

# 运行
./antigravity
```

## 配置

### 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| `PORT` | 服务监听端口 | `8000` |

**示例:**
```bash
PORT=8080 ./antigravity
```

### 认证

使用 `Authorization` header 传递 refresh_token：

```
Authorization: Bearer <your_refresh_token>
```

**Token 格式**: 直接使用 refresh_token (rt)

## API 端点

### OpenAI 兼容接口

**获取模型列表:**
```http
GET /v1/models
```

**聊天补全:**
```http
POST /v1/chat/completions
Content-Type: application/json
Authorization: Bearer <refresh_token>

{
  "model": "gpt-4",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "stream": true
}
```

### Claude 兼容接口

**创建消息:**
```http
POST /v1/messages
Content-Type: application/json
Authorization: Bearer <refresh_token>

{
  "model": "claude-3-5-sonnet-20241022",
  "messages": [
    {"role": "user", "content": "Hello"}
  ],
  "max_tokens": 1024
}
```

**计算 Token:**
```http
POST /v1/messages/count_tokens
```

### Gemini 兼容接口

**获取模型列表:**
```http
GET /v1beta/models
```

**生成内容:**
```http
POST /v1beta/models/{model}:generateContent
POST /v1beta/models/{model}:streamGenerateContent
```

## 开发

### 项目结构

```
antigravity/
├── cmd/
│   └── server/          # 主程序入口
├── internal/
│   ├── auth/           # 认证和中间件
│   ├── handler/        # 请求处理器
│   ├── model/          # 数据模型
│   ├── proxy/          # 代理逻辑和格式转换
│   └── router/         # 路由配置
├── docker/             # Docker 配置
├── .github/workflows/  # CI/CD 配置
└── .version           # 版本信息
```
