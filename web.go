package main

import (
_ "embed"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"os/exec"
"path/filepath"
"strings"
"sync"
"time"
)

//go:embed index.html
var indexHTML string

type WebTask struct {
ID           string `json:"id"`
Bvid         string `json:"bvid"`
URL          string `json:"url"`
Status       string `json:"status"`
Msg          string `json:"msg"`
Title        string `json:"title"`
Author       string `json:"author"`
Cover        string `json:"cover"`
Result       string `json:"result"`
SourceType   string `json:"sourceType"`
Step1Status  string `json:"step1Status"`
Step1Time    string `json:"step1Time"`
Step2Status  string `json:"step2Status"`
Step2Time    string `json:"step2Time"`
Step3Status  string `json:"step3Status"`
Step3Time    string `json:"step3Time"`
Step4Status  string `json:"step4Status"`
Step4Time    string `json:"step4Time"`
CostTime     string `json:"costTime"`
}

var (
tasks         = make(map[string]WebTask)
taskMux       sync.RWMutex
historyMux    sync.Mutex
platformFiles = map[string]string{"bilibili": "bilibili_cookie.txt", "douyin": "douyin_cookie.txt", "zhihu": "zhihu_cookie.txt"}
)

func loadHistory() []WebTask {
historyMux.Lock()
defer historyMux.Unlock()
data, err := os.ReadFile("/app/data/history.json")
if err != nil { return []WebTask{} }
var h []WebTask
json.Unmarshal(data, &h)
return h
}

func saveHistoryFile(h []WebTask) {
historyMux.Lock()
defer historyMux.Unlock()
out, _ := json.Marshal(h)
os.WriteFile("/app/data/history.json", out, 0644)
}

func startWebServer(port string) {
http.HandleFunc("/api/proxy", func(w http.ResponseWriter, r *http.Request) {
imgURL := r.URL.Query().Get("url")
if imgURL == "" { return }
req, _ := http.NewRequest("GET", imgURL, nil)
req.Header.Set("Referer", "https://www.bilibili.com/")
req.Header.Set("User-Agent", "Mozilla/5.0")
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Do(req)
if err != nil { return }
defer resp.Body.Close()
w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
w.Header().Set("Cache-Control", "public, max-age=86400")
io.Copy(w, resp.Body)
})

http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "text/html; charset=utf-8")
w.Write([]byte(indexHTML))
})

http.HandleFunc("/api/cookie", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
p := r.URL.Query().Get("platform")
if r.Method == http.MethodGet {
data, _ := os.ReadFile(platformFiles[p])
json.NewEncoder(w).Encode(map[string]string{"cookie": string(data)})
return
}
var req struct{ Cookie string `json:"cookie"` }
json.NewDecoder(r.Body).Decode(&req)
os.WriteFile(platformFiles[p], []byte(strings.TrimSpace(req.Cookie)), 0644)
w.Write([]byte(`{"status":"ok"}`))
})

http.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(loadHistory())
})

http.HandleFunc("/api/tasks", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
taskMux.RLock()
var list []WebTask
for _, t := range tasks { list = append(list, t) }
taskMux.RUnlock()
if list == nil { list = []WebTask{} }
json.NewEncoder(w).Encode(list)
})

http.HandleFunc("/api/delete", func(w http.ResponseWriter, r *http.Request) {
w.Header().Set("Content-Type", "application/json")
var req struct {
ID         string `json:"id"`
DeleteFile bool   `json:"deleteFile"`
URL        string `json:"url"`
}
json.NewDecoder(r.Body).Decode(&req)
var bvid string
taskMux.RLock()
if t, ok := tasks[req.ID]; ok { bvid = t.Bvid }
taskMux.RUnlock()
if bvid == "" && req.URL != "" {
if idx := strings.Index(req.URL, "BV"); idx != -1 { 
bvid = req.URL[idx:]; if len(bvid) > 12 { bvid = bvid[:12] }
}
}
taskMux.Lock()
delete(tasks, req.ID)
taskMux.Unlock()
h := loadHistory()
var newH []WebTask
for _, item := range h { if item.ID != req.ID { newH = append(newH, item) } }
saveHistoryFile(newH)
if bvid != "" {
_ = exec.Command("pkill", "-f", bvid).Run()
_ = exec.Command("pkill", "-f", "chunk_").Run()
}
if req.DeleteFile && bvid != "" {
files, _ := filepath.Glob("/app/data/tmp/" + bvid + "*")
for _, f := range files { os.Remove(f) }
os.Remove("/app/data/tmp/subs_" + bvid + ".txt")
chunks, _ := filepath.Glob("/app/data/tmp/chunks/*.wav")
for _, c := range chunks { os.Remove(c) }
}
w.Write([]byte(`{"status":"ok"}`))
})

http.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
taskMux.RLock()
t := tasks[r.URL.Query().Get("id")]
taskMux.RUnlock()
w.Header().Set("Content-Type", "application/json")
json.NewEncoder(w).Encode(t)
})

http.HandleFunc("/api/start", func(w http.ResponseWriter, r *http.Request) {
var req struct{ Url string `json:"url"` }
json.NewDecoder(r.Body).Decode(&req)
taskID := fmt.Sprintf("%d", time.Now().UnixNano())
startTime := time.Now()
bvid := req.Url
if idx := strings.Index(req.Url, "BV"); idx != -1 { bvid = req.Url[idx:]; if len(bvid) > 12 { bvid = bvid[:12] } } else { bvid = "" }
if !strings.Contains(req.Url, "http") && strings.HasPrefix(req.Url, "BV") {
req.Url = "https://www.bilibili.com/video/" + req.Url
}
taskMux.Lock()
tasks[taskID] = WebTask{ID: taskID, Bvid: bvid, URL: req.Url, Status: "running", Msg: "正在初始化...", SourceType: "正在分析", Step1Status: "running", Step2Status: "pending", Step3Status: "pending", Step4Status: "pending"}
taskMux.Unlock()

// 同步获取信息
title, author, cover := "", "", ""
for i := 0; i < 3; i++ {
info, err := getBilibiliInfo(bvid)
if err == nil && info != nil && info["title"] != "" {
title = info["title"]
author = info["up"]
cover = info["cover"]
break
}
time.Sleep(1 * time.Second)
}

taskMux.Lock()
if t, ok := tasks[taskID]; ok {
t.Title = title
t.Author = author
t.Cover = cover
tasks[taskID] = t
}
taskMux.Unlock()

w.Header().Set("Content-Type", "application/json")
res := map[string]string{
"task_id": taskID,
"title": title,
"author": author,
"cover": cover,
}
json.NewEncoder(w).Encode(res)

go func() {
update := func(updates func(t *WebTask)) {
taskMux.Lock()
t, exists := tasks[taskID]
if !exists { taskMux.Unlock(); return }
updates(&t)
t.CostTime = fmt.Sprintf("%.0fs", time.Since(startTime).Seconds())
tasks[taskID] = t
taskMux.Unlock()
}
platform, _ := detectPlatform(req.Url)
if platform == PlatformBilibili {
// 1. 拉取信息阶段 (拿弹幕/评论)
step1Start := time.Now()
danmaku, _ := getBilibiliDanmaku(bvid)
comments, _ := getBilibiliComments(bvid)
update(func(t *WebTask) { 
t.Step1Status = "done"
t.Step1Time = fmt.Sprintf("%.1fs", time.Since(step1Start).Seconds())
t.Step2Status = "running"
})

// 2. 视频转码/拉字幕阶段
step2Start := time.Now()
content, sourceType, audioPath, err := getBilibiliContent(bvid)
if err != nil {
update(func(t *WebTask) { t.Status = "error"; t.Msg = err.Error(); t.Step2Status = "error" })
return
}

if sourceType == "拉取字幕" {
update(func(t *WebTask) {
t.SourceType = "拉取字幕"
t.Step2Status = "done"
t.Step2Time = "0s"
t.Step3Status = "done"
t.Step3Time = "0s"
t.Step4Status = "running"
})
} else {
update(func(t *WebTask) {
t.SourceType = "下载音频"
t.Step2Status = "done"
t.Step2Time = fmt.Sprintf("%.1fs", time.Since(step2Start).Seconds())
t.Step3Status = "running"
})

// 3. 音频识别 (NPU)
step3Start := time.Now()
text, err := callNPU(audioPath)
if err != nil {
update(func(t *WebTask) { t.Status = "error"; t.Msg = err.Error(); t.Step3Status = "error" })
return
}
content = text
update(func(t *WebTask) {
t.Step3Status = "done"
t.Step3Time = fmt.Sprintf("%.1fs", time.Since(step3Start).Seconds())
t.Step4Status = "running"
})
}

// 4. AI 总结阶段
step4Start := time.Now()
report, err := summarizeBilibili(fmt.Sprintf("【标题】%s\n【UP】%s\n\n", title, author)+content, danmaku, comments)
if err != nil {
update(func(t *WebTask) { t.Status = "error"; t.Msg = err.Error(); t.Step4Status = "error" })
return
}

update(func(t *WebTask) { 
t.Status = "done"
t.Result = report
t.Step4Status = "done"
t.Step4Time = fmt.Sprintf("%.1fs", time.Since(step4Start).Seconds())
})

taskMux.RLock()
finalTask := tasks[taskID]
taskMux.RUnlock()
h := loadHistory()
newH := []WebTask{finalTask}
for _, item := range h { if item.Bvid != finalTask.Bvid { newH = append(newH, item) } }
if len(newH) > 50 { newH = newH[:50] }
saveHistoryFile(newH)
} else {
update(func(t *WebTask) { t.Status = "error"; t.Msg = "暂不支持" })
}
}()
})

fmt.Printf("🚀 WebUI: http://0.0.0.0:%s\n", port)
http.ListenAndServe(":"+port, nil)
}
