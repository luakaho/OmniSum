# OmniSum - 全平台多媒体 AI 总结系统 (Go版)

一个专为 NAS (oect)全自动多媒体内容总结引擎。结合了 `yt-dlp`、本地 RKNN NPU 算力与 DeepSeek 大模型，支持将 B站 等平台的视频一键转化为结构化 Markdown 报告。

## 核心特性
- **全自动流水线**：智能识别字幕 -> 无字幕自动下载切片 -> 本地 NPU 语音转写 -> 大模型提炼。
- **极致的 Web UI**：内置无框架单页应用，支持深浅主题、多任务并发队列、实时进度流水线。
- **防爬虫对抗**：手搓 WBI 签名算法，内置完整 Cookie 注入与防盗链图片代理。
- **资源自动管家**：一键强行终止后台进程并彻底粉碎缓存，24 小时自动清理陈年垃圾。
- **极致轻量**：纯 Go 语言编写，无头浏览器零依赖。

## 前置依赖 (非常重要)

本项目并非开箱即用的单体软件，它需要外部环境和本地算力的配合。在部署前，请确保你的宿主机 (NAS) 具备以下条件：

1. **硬件要求**：ARM64 架构设备。若需使用 NPU 语音转写兜底功能，需搭载瑞芯微 NPU（如 RK3566, RK3588 等）。
2. **系统命令行工具**：宿主机必须已安装 `yt-dlp` 和 `ffmpeg`，且已加入系统环境变量。
3. **NPU 驱动**：宿主机的 Linux 内核需已包含瑞芯微 NPU 驱动（通常表现为存在 `/dev/dri` 设备节点）。
4. **SenseVoice 语音转写服务**：本项目不包含语音识别模型，需要单独在本地运行一个支持 RKNN 的语音转写服务（默认监听 8099 端口）。

## 快速部署 (Docker)

1. 克隆项目：
   ```bash
   git clone https://github.com/luakaho/OmniSum.git
   cd OmniSum

   2. 配置密钥 (在项目根目录创建)：
   · bilibili_cookie.txt（可选，填 B 站 SESSDATA，防限流）
   · deepseek_key.txt（必填，DeepSeek API Key）
   · github_token.txt（可选，防 GitHub API 限流）
3. 启动主服务：
   ```bash
   docker compose up -d --build
   ```
4. 访问：
   浏览器打开 http://NAS_IP:8081

NPU 语音转写服务 (SenseVoice) 部署指南

如果 B 站视频没有字幕，系统会自动下载音频并切片，发送给本地的 NPU 服务（SenseVoice）进行转写。该服务必须支持 OpenAI 的接口格式。

1. 编写 SenseVoice 的 docker-compose.yml

参考以下配置（以 RK3566 为例）：

```yaml
version: '3.8'
services:
  sensevoice:
    # 需使用支持 RKNN 的镜像
    image: sensevoice-rknn-sensevoice-rknn 
    container_name: sensevoice
    restart: unless-stopped
    ports:
      - "8099:8000"             # 宿主机的 8099 映射到容器的 8000
    devices:
      - /dev/dri:/dev/dri       # 关键：将宿主机 NPU 设备节点挂载进容器
    volumes:
      - ./models:/models        # 挂载模型目录，具体路径需结合你的镜像实际情况
```

2. 网络连通性配置 (非常重要)

由于 OmniSum 主项目 (npu.go) 默认请求的地址是 http://127.0.0.1:8099，如果 OmniSum 也是用 Docker 部署的，请务必注意以下两种网络配置方案：

· 方案 A（推荐）：将 OmniSum 的 docker-compose.yml 增加一行 network_mode: host，这样容器内的 127.0.0.1 才能直接访问宿主机上的 NPU 服务。
· 方案 B：修改 OmniSum 容器的环境变量 NPU_ENDPOINT（如 - NPU_ENDPOINT=http://192.168.x.x:8099），将其指向宿主机的局域网 IP。

开发进度

☑ B站 NPU兜底、弹幕、楼中楼、并发切片

☐ GitHub 仓库总结模块（开发中...）

☐ 播客 (小宇宙) 模块
