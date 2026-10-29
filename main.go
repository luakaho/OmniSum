package main

import (
"fmt"
"io"
"net/http"
"os"
"path/filepath"
"strings"
"time"
)

func cleanOldTempFiles() {
// ... (保持原有的清理逻辑不变)
threshold := time.Now().Add(-24 * time.Hour)
cleanedCount := 0
chunkDir := "/app/data/tmp/chunks"
if files, err := os.ReadDir(chunkDir); err == nil {
for _, f := range files {
if fInfo, err := f.Info(); err == nil && fInfo.ModTime().Before(threshold) {
os.Remove(filepath.Join(chunkDir, f.Name()))
cleanedCount++
}
}
}
files, _ := os.ReadDir("/app/data/tmp")
for _, f := range files {
name := f.Name()
if strings.HasPrefix(name, "subs_") || strings.HasPrefix(name, "podcast_") || 
   strings.HasSuffix(name, ".mp3") || strings.HasSuffix(name, ".m4a") || 
   strings.HasSuffix(name, ".webm") || strings.HasSuffix(name, ".srt") || 
   strings.HasSuffix(name, ".vtt") {
if fInfo, err := f.Info(); err == nil && fInfo.ModTime().Before(threshold) {
os.Remove(filepath.Join("/app/data/tmp", name))
cleanedCount++
}
}
}
if cleanedCount > 0 {
fmt.Printf("?? 已自动清理 %d 个昨天及之前的临时文件\n", cleanedCount)
}
}

func resolveBilibiliShortURL(shortURL string) (string, error) {
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Get(shortURL)
if err != nil { return "", err }
defer resp.Body.Close()
io.Copy(io.Discard, resp.Body)
finalURL := resp.Request.URL.String()
if idx := strings.Index(finalURL, "BV"); idx != -1 {
bvid := finalURL[idx:]
if len(bvid) > 12 { bvid = bvid[:12] }
return bvid, nil
}
return "", fmt.Errorf("未找到 BVID")
}

func main() {
cleanOldTempFiles()

// 如果没带参数，或者带了 "-web"，启动 Web 服务
if len(os.Args) < 2 || os.Args[1] == "-web" {
port := os.Getenv("PORT")
if port == "" { port = "8080" }
startWebServer(port)
return
}

// --- 以下为原有的 CLI 逻辑 ---
target := os.Args[1]
platform, err := detectPlatform(target)
if err != nil {
fmt.Println("❌", err)
return
}
fmt.Printf("?? 目标平台: %s\n", platform)

var content, danmaku, comments, title, author, audioPath string

switch platform {
case PlatformBilibili:
bvid := ""
if strings.Contains(target, "b23.tv") {
fmt.Println("[-] 检测到 b23.tv 短链，正在解析跳转...")
bvid, err = resolveBilibiliShortURL(target)
if err != nil {
fmt.Println("❌ 短链解析失败:", err)
return
}
fmt.Printf("✅ 解析到 BVID: %s\n", bvid)
} else {
if idx := strings.Index(target, "BV"); idx != -1 {
bvid = target[idx:]
if len(bvid) > 12 { bvid = bvid[:12] }
} else {
bvid = target
}
fmt.Printf("✅ 解析到 BVID: %s\n", bvid)
}

fmt.Println("========== 1. 抓取 B站 基础信息 ==========")
info, err := getBilibiliInfo(bvid)
if err != nil {
fmt.Println("⚠️ 获取视频信息失败（继续）:", err)
} else {
title = info["title"]
author = info["up"]
fmt.Printf("✅ 视频标题: %s\n", title)
fmt.Printf("✅ UP主: %s\n", author)
}

fmt.Println("========== 2. 抓取 B站 内容 ==========")
content, _, audioPath, err = getBilibiliContent(bvid)
if err != nil {
fmt.Println("❌ 内容提取失败:", err)
return
}
// 【核心修复】CLI 模式下的音频 NPU 兜底
if strings.TrimSpace(content) == "" && audioPath != "" {
fmt.Println("⚠️ 无字幕，正在启动 NPU 转写:", audioPath)
content, err = callNPU(audioPath)
if err != nil {
fmt.Println("❌ NPU 转写失败:", err)
return
}
}
fmt.Printf("✅ 内容长度: %d 字\n", len(content))

fmt.Println("========== 3. 抓取弹幕 ==========")
danmaku, _ = getBilibiliDanmaku(bvid)
fmt.Printf("✅ 弹幕条数: %d\n", strings.Count(danmaku, "\n")+1)

fmt.Println("========== 4. 抓取评论 ==========")
comments, _ = getBilibiliComments(bvid)
fmt.Printf("✅ 评论长度: %d\n", len(comments))

default:
fmt.Printf("❌ 平台 %s 暂未实现抓取逻辑\n", platform)
return
}

fmt.Println("========== 正在呼叫 DeepSeek 总结 ==========")
header := ""
if title != "" {
header = fmt.Sprintf("【视频标题】%s\n【UP主】%s\n\n", title, author)
}

report, err := summarizeBilibili(header+content, danmaku, comments)
if err != nil {
fmt.Println("❌ 总结失败:", err)
return
}

fmt.Println("\n========== 最终报告 ==========\n")
fmt.Println(report)
outFile := "report_" + string(platform) + ".md"
os.WriteFile(outFile, []byte(report), 0644)
fmt.Println("\n✅ 已保存:", outFile)
}
