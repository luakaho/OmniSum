# OmniSum - 全平台多媒体 AI 总结系统 (Go版)

一个专为 NAS (oect)打造的全自动多媒体内容总结引擎。结合了 `yt-dlp`、本地 RKNN NPU 算力与 DeepSeek 大模型，支持将 B站 等平台的视频一键转化为结构化 Markdown 报告。

## 核心特性
- **全自动流水线**：智能识别字幕 -> 无字幕自动下载切片 -> 本地 NPU 语音转写 -> 大模型提炼。
- **极致的 Web UI**：内置无框架单页应用，支持深浅主题、多任务并发队列、实时进度流水线。
- **防爬虫对抗**：手搓 WBI 签名算法，内置完整 Cookie 注入与防盗链图片代理。
- **资源自动管家**：一键强行终止后台进程并彻底粉碎缓存，24 小时自动清理陈年垃圾。
- **极致轻量**：纯 Go 语言编写，无头浏览器零依赖。

## 快速部署 (Docker)
1. 克隆项目：
   ```bash
   git clone https://github.com/luakaho/OmniSum.git

2. 配置密钥：
   复制 config.example.json 为 config.json，或直接在宿主机创建 bilibili_cookie.txt 与 deepseek_key.txt。
3. 启动服务：
   ```bash
   docker compose up -d --build
   ```
4. 访问：
   浏览器打开 http://NAS_IP:8081

开发进度

☑ B站 (Bilibili) 模块（字幕提取、NPU兜底、弹幕、楼中楼、并发切片）
☐ GitHub 仓库总结模块（开发中...）
☐ 播客 (小宇宙) 模块
