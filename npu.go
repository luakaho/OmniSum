package main

import (
"bytes"
"encoding/json"
"fmt"
"io"
"mime/multipart"
"net/http"
"os"
"os/exec"
"path/filepath"
"sort"
"strings"
	"time"
)

func callNPU(audioPath string) (string, error) {
fmt.Println("[-] 正在用 ffmpeg 切分音频 (120秒/块，转为 WAV 格式)...")
chunkDir := "/app/data/tmp/chunks_" + strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
os.RemoveAll(chunkDir)
os.MkdirAll(chunkDir, 0755)

// 核心修改：输出格式改为 pcm_s16le (标准 WAV)，采样率 16000，单声道
cmd := exec.Command("ffmpeg", "-y", "-i", audioPath, "-f", "segment", "-segment_time", "120", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", filepath.Join(chunkDir, "chunk_%04d.wav"))
if err := cmd.Run(); err != nil {
return "", fmt.Errorf("ffmpeg 切分失败: %v", err)
}

// 匹配 .wav 文件
files, _ := filepath.Glob(filepath.Join(chunkDir, "chunk_*.wav"))
sort.Strings(files)

var fullText strings.Builder
fmt.Printf("[-] 共切分为 %d 块，开始发送给 NPU...\n", len(files))

for i, file := range files {
fmt.Printf("    -> [NPU] 正在处理第 %d/%d 块: %s\n", i+1, len(files), filepath.Base(file))
tChunk := time.Now()
text, err := sendChunkToNPU(file)
if err == nil {
    fmt.Printf("       ✅ [NPU] 第 %d 块转写成功 (耗时 %.1fs, 字数 %d)\n", i+1, time.Since(tChunk).Seconds(), len(text))
} else {
fmt.Printf("       ⚠️ 第 %d 块处理失败: %v\n", i+1, err)
continue
}
fullText.WriteString(text + "\n")
}

return fullText.String(), nil
}

func sendChunkToNPU(filePath string) (string, error) {
file, err := os.Open(filePath)
if err != nil { return "", err }
defer file.Close()

body := &bytes.Buffer{}
writer := multipart.NewWriter(body)
part, _ := writer.CreateFormFile("file", filePath)
io.Copy(part, file)

// 补齐必填字段
writer.WriteField("model", "SenseVoiceSmall")

writer.Close()

npuHost := os.Getenv("NPU_ENDPOINT")
	if npuHost == "" { npuHost = "http://127.0.0.1:8099" }
	req, _ := http.NewRequest("POST", npuHost+"/v1/audio/transcriptions", body)
req.Header.Set("Authorization", "Bearer local")
req.Header.Set("Content-Type", writer.FormDataContentType())

client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
if err != nil { return "", err }
defer resp.Body.Close()

if resp.StatusCode != 200 {
errBody, _ := io.ReadAll(resp.Body)
return "", fmt.Errorf("NPU返回状态码 %d, 报错详情: %s", resp.StatusCode, string(errBody))
}

var result struct {
Text string `json:"text"`
}
json.NewDecoder(resp.Body).Decode(&result)
return result.Text, nil
}
