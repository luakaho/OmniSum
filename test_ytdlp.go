package main

import (
"fmt"
"os"
"os/exec"
"path/filepath"
"strings"
)

func main() {
bvid := "BV1QHNb6EEx8"

data, err := os.ReadFile("bilibili_cookie.txt")
if err != nil {
fmt.Println("❌ 读取 bilibili_cookie.txt 失败，请确认文件存在。")
return
}
rawCookie := strings.TrimSpace(string(data))
cookieFile := "/tmp/test-cookies.txt"
content := "# Netscape HTTP Cookie File\n"
parts := strings.Split(rawCookie, ";")
for _, part := range parts {
kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
if len(kv) == 2 {
content += fmt.Sprintf(".bilibili.com\tTRUE\t/\tFALSE\t0\t%s\t%s\n", kv[0], kv[1])
}
}
os.WriteFile(cookieFile, []byte(content), 0644)

fmt.Println("[-] 正在用完整 Cookie 测试 yt-dlp 提取字幕...")
subFilePattern := "/tmp/" + bvid + "*.vtt"
os.Remove(subFilePattern)

cmd := exec.Command("yt-dlp", "--skip-download", "--write-subs", "--write-auto-subs", "--sub-langs", "zh-CN,zh,ai-zh", "--cookies", cookieFile, "-o", "/tmp/"+bvid, "https://www.bilibili.com/video/"+bvid)
outSub, _ := cmd.CombinedOutput()

files, _ := filepath.Glob(subFilePattern)
if len(files) > 0 {
fmt.Printf("\n✅ 奇迹出现！yt-dlp 成功下载到了字幕文件: %s\n", files[0])
content, _ := os.ReadFile(files[0])
preview := string(content)
if len(preview) > 200 { preview = preview[:200] }
fmt.Println("字幕前 200 字预览：\n", preview)
} else {
fmt.Printf("\n❌ 还是失败了，yt-dlp 完整日志：\n%s\n", string(outSub))
}
}
