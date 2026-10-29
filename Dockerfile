# 第一阶段：编译 Go 程序
FROM golang:alpine AS builder
WORKDIR /app
# 把整个项目拷进来
COPY . .
# 编译静态二进制文件
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/summarizer main.go web.go router.go bilibili.go npu.go deepseek.go

# 第二阶段：运行环境 (使用 Alpine，极小极快)
FROM python:3.10-alpine
WORKDIR /app

# Alpine 安装依赖，使用国内镜像源加速（如果卡住可以去掉 --repository 参数）
RUN apk add --no-cache ffmpeg curl tzdata

# 使用清华源安装 yt-dlp
RUN pip install --no-cache-dir -i https://pypi.tuna.tsinghua.edu.cn/simple yt-dlp

# 拷贝编译好的二进制文件
COPY --from=builder /app/summarizer /app/summarizer

# 拷贝配置（如果在 docker-compose 中挂载了，会被覆盖）
COPY bilibili_cookie.txt /app/
COPY deepseek_key.txt /app/

# 暴露 Web 端口
EXPOSE 8080

# 启动 Web 服务
CMD ["./summarizer", "-web"]
