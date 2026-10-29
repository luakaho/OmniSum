package main

import (
"crypto/md5"
"encoding/hex"
"encoding/json"
"fmt"
"net/http"
"net/url"
"os"
"os/exec"
"path/filepath"
"regexp"
"sort"
"strings"
"time"
)

var mixinKeyEncTab = []int{
46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35,
27, 43, 5, 49, 33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13,
37, 48, 7, 16, 24, 55, 40, 61, 26, 17, 0, 1, 60, 51, 30, 4,
22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11, 36, 20, 34, 44, 52,
}

func getMixinKey(imgKey, subKey string) string {
raw := imgKey + subKey
var sb strings.Builder
for _, idx := range mixinKeyEncTab { sb.WriteByte(raw[idx]) }
return sb.String()[:32]
}

func signWbi(params map[string]string, imgKey, subKey string) string {
mixinKey := getMixinKey(imgKey, subKey)
params["wts"] = fmt.Sprintf("%d", time.Now().Unix())
keys := make([]string, 0, len(params))
for k := range params { keys = append(keys, k) }
sort.Strings(keys)
var queryParts []string
for _, k := range keys {
v := params[k]
v = strings.NewReplacer("!", "", "'", "", "(", "", ")", "", "*", "").Replace(v)
queryParts = append(queryParts, k+"="+url.QueryEscape(v))
}
query := strings.Join(queryParts, "&")
h := md5.Sum([]byte(query + mixinKey))
return query + "&w_rid=" + hex.EncodeToString(h[:])
}

func getWbiKeys(cookieStr string) (string, string) {
cmd := exec.Command("curl", "-s", "--compressed", "-A", "Mozilla/5.0")
if cookieStr != "" { cmd.Args = append(cmd.Args, "-H", "Cookie: "+cookieStr) }
cmd.Args = append(cmd.Args, "https://api.bilibili.com/x/web-interface/nav")
out, _ := cmd.Output()
var res map[string]interface{}
json.Unmarshal(out, &res)
data, _ := res["data"].(map[string]interface{})
wbiImg, _ := data["wbi_img"].(map[string]interface{})
imgUrl, _ := wbiImg["img_url"].(string)
subUrl, _ := wbiImg["sub_url"].(string)
imgKey := "7cd084941338484aae1ad9425b84077c"
subKey := "4932caff0ff746eab6f01bf08b70ac45"
if strings.Contains(imgUrl, "/") && strings.Contains(subUrl, "/") {
imgKey = imgUrl[strings.LastIndex(imgUrl, "/")+1 : strings.LastIndex(imgUrl, ".")]
subKey = subUrl[strings.LastIndex(subUrl, "/")+1 : strings.LastIndex(subUrl, ".")]
}
return imgKey, subKey
}

func tryExtractSubtitleFromHTML(bvid, cookieStr string) string {
fmt.Printf("[%s] [-] 尝试从网页源码中提取字幕...\n", bvid)
htmlURL := "https://www.bilibili.com/video/" + bvid
cmd := exec.Command("curl", "-s", "--compressed", "-A", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)", "-H", "Cookie: "+cookieStr, htmlURL)
out, err := cmd.Output()
if err != nil { return "" }

html := string(out)
re := regexp.MustCompile(`"subtitle_url":"(//[^"]+|https?://[^"]+)"`)
matches := re.FindAllStringSubmatch(html, -1)

for _, match := range matches {
if len(match) > 1 {
subURL := match[1]
if strings.HasPrefix(subURL, "//") { subURL = "https:" + subURL }
subCmd := exec.Command("curl", "-s", "--compressed", "-A", "Mozilla/5.0", subURL)
subOut, _ := subCmd.Output()

var subJSON struct { Body []struct{ Content string `json:"content"` } `json:"body"` }
if err := json.Unmarshal(subOut, &subJSON); err == nil {
var sb strings.Builder
for _, line := range subJSON.Body { sb.WriteString(line.Content + "\n") }
if sb.Len() > 200 { return sb.String() }
}
}
}
return ""
}

func generateNetscapeCookieFile() string {
data, err := os.ReadFile("bilibili_cookie.txt")
if err != nil { return "" }
rawCookie := strings.TrimSpace(string(data))

// 智能解析：支持填纯 SESSDATA 值，也支持粘贴一整串完整的 Cookie
sessdataVal := rawCookie
parts := strings.Split(rawCookie, ";")
for _, part := range parts {
part = strings.TrimSpace(part)
if strings.HasPrefix(part, "SESSDATA=") {
sessdataVal = strings.TrimPrefix(part, "SESSDATA=")
break
}
}

cookiePath := "/app/data/tmp/yt-dlp-cookies.txt"
content := "# Netscape HTTP Cookie File\n"
content += fmt.Sprintf(".bilibili.com\tTRUE\t/\tFALSE\t0\tSESSDATA\t%s\n", sessdataVal)
os.WriteFile(cookiePath, []byte(content), 0644)
return cookiePath
}

func getBilibiliContent(bvid string) (string, string, string, error) {
cachePath := filepath.Join("/app/data/tmp", "subs_"+bvid+".txt")
if data, err := os.ReadFile(cachePath); err == nil && len(data) > 200 {
fmt.Printf("⚡ 发现本地缓存字幕，直接读取\n")
return string(data), "拉取字幕", "", nil
}

cid, err := getCid(bvid)
if err != nil { return "", "", "", fmt.Errorf("获取 CID 失败: %v", err) }

cookieStr := ""
if data, err := os.ReadFile("bilibili_cookie.txt"); err == nil {
cookieStr = strings.TrimSpace(string(data))
}
cookieFile := generateNetscapeCookieFile()

imgKey, subKey := getWbiKeys(cookieStr)
for attempt := 1; attempt <= 2; attempt++ {
fmt.Printf("[%s] [-] 正在通过 WBI API 提取字幕 (尝试 %d/2)...\n", bvid, attempt)
params := map[string]string{"bvid": bvid, "cid": cid}
query := signWbi(params, imgKey, subKey)
apiURL := "https://api.bilibili.com/x/player/wbi/v2?" + query

cmd := exec.Command("curl", "-s", "--compressed", "-A", "Mozilla/5.0")
if cookieStr != "" { cmd.Args = append(cmd.Args, "-H", "Cookie: "+cookieStr) }
cmd.Args = append(cmd.Args, apiURL)
out, _ := cmd.Output()

var res map[string]interface{}
json.Unmarshal(out, &res)
data, _ := res["data"].(map[string]interface{})
subtitleInfo, _ := data["subtitle"].(map[string]interface{})
subtitles, _ := subtitleInfo["subtitles"].([]interface{})

var validSubtitleURLs []string
for _, sub := range subtitles {
subMap, _ := sub.(map[string]interface{})
lan, _ := subMap["lan"].(string)
url, _ := subMap["subtitle_url"].(string)
aiStatus, _ := subMap["ai_status"].(float64)

if aiStatus == 2 { continue }
if (lan == "zh-CN" || lan == "ai-zh") && url != "" {
if strings.HasPrefix(url, "//") { url = "https:" + url }
validSubtitleURLs = append(validSubtitleURLs, url)
}
}

for _, subURL := range validSubtitleURLs {
subCmd := exec.Command("curl", "-s", "--compressed", "-A", "Mozilla/5.0", subURL)
subOut, err := subCmd.Output()
if err == nil {
var subJSON struct { Body []struct{ Content string `json:"content"` } `json:"body"` }
if err := json.Unmarshal(subOut, &subJSON); err == nil {
var sb strings.Builder
for _, line := range subJSON.Body { sb.WriteString(line.Content + "\n") }
if sb.Len() > 200 {
fmt.Printf("[%s] ✅ WBI API 成功截胡有效字幕，字数: %d\n", bvid, sb.Len())
os.MkdirAll("/app/data/tmp", 0755)
os.WriteFile(cachePath, []byte(sb.String()), 0644)
return sb.String(), "拉取字幕", "", nil
}
}
}
}
time.Sleep(1 * time.Second)
}

htmlText := tryExtractSubtitleFromHTML(bvid, cookieStr)
if htmlText != "" {
fmt.Printf("[%s] ✅ HTML 源码提取成功，字数: %d\n", bvid, len(htmlText))
os.MkdirAll("/app/data/tmp", 0755)
os.WriteFile(cachePath, []byte(htmlText), 0644)
return htmlText, "拉取字幕", "", nil
} else {
fmt.Printf("[%s]     ⚠️ HTML 源码中也未找到字幕数据\n", bvid)
}

fmt.Printf("[%s] [-] 尝试使用 yt-dlp 直接提取字幕...\n", bvid)
subFilePattern := "/app/data/tmp/" + bvid + "*.vtt"
os.Remove(subFilePattern)
os.Remove("/app/data/tmp/" + bvid + "*.srt")

cmdYtdlpSub := exec.Command("yt-dlp", "--skip-download", "--write-subs", "--write-auto-subs", "--sub-langs", "zh-CN,zh,ai-zh", "--cookies", cookieFile, "-o", "/app/data/tmp/"+bvid, "https://www.bilibili.com/video/"+bvid)
outSub, _ := cmdYtdlpSub.CombinedOutput()
fmt.Printf("[%s]     -> yt-dlp 提取字幕输出: %s\n", bvid, string(outSub))
files, _ := filepath.Glob(subFilePattern)
filesSrt, _ := filepath.Glob("/app/data/tmp/" + bvid + "*.srt")
files = append(files, filesSrt...)

if len(files) > 0 {
fmt.Printf("✅ yt-dlp 成功下载字幕文件: %s\n", files[0])
content, _ := os.ReadFile(files[0])
text := string(content)
if len(text) > 200 {
os.MkdirAll("/app/data/tmp", 0755)
os.WriteFile(cachePath, []byte(text), 0644)
return text, "拉取字幕", "", nil
}
} else {
fmt.Printf("    ⚠️ yt-dlp 也没抓到字幕\n")
}

fmt.Printf("[%s] ⚠️ 所有网页手段均失败，触发 NPU 终极兜底方案...\n", bvid)
audioPath := "/app/data/tmp/" + bvid + ".mp3"
os.Remove(audioPath)
os.Remove("/app/data/tmp/" + bvid + ".webm")
os.Remove("/app/data/tmp/" + bvid + ".m4a")
os.Remove("/app/data/tmp/" + bvid + ".mp3.part")
os.Remove("/app/data/tmp/" + bvid + ".m4a.part")
os.Remove("/app/data/tmp/" + bvid + ".webm.part")

audioURL := "https://www.bilibili.com/video/" + bvid
cmdAudio := exec.Command("yt-dlp", "-f", "bestaudio", "--no-warnings", "-x", "--audio-format", "mp3", "--cookies", cookieFile, "-o", audioPath, audioURL)
outAudio, err := cmdAudio.CombinedOutput()
if err != nil { 
fmt.Printf("[%s] ❌ yt-dlp 下载音频失败: %v, 输出: %s\n", bvid, err, string(outAudio))
return "", "", "", fmt.Errorf("yt-dlp 下载音频失败: %v", err) 
}
fmt.Printf("[%s] ✅ yt-dlp 音频下载成功\n", bvid)
fmt.Printf("[%s] ✅ 音频下载完成，准备交由 WebUI 流水线转码\n", bvid)
return "", "下载音频", audioPath, nil
}

func getBilibiliDanmaku(bvid string) (string, error) {
cid, err := getCid(bvid)
if err != nil { return "", err }
apiURL := fmt.Sprintf("https://comment.bilibili.com/%s.xml", cid)
cmd := exec.Command("curl", "-s", "--compressed", "-A", "Mozilla/5.0", apiURL)
out, err := cmd.Output()
if err != nil { return "", err }
re := regexp.MustCompile(`<d p="[^"]+">([^<]*)</d>`)
matches := re.FindAllStringSubmatch(string(out), -1)
var danmakuList []string
for _, m := range matches {
if len(m) > 1 && strings.TrimSpace(m[1]) != "" { danmakuList = append(danmakuList, m[1]) }
}
return strings.Join(danmakuList, "\n"), nil
}

func getBilibiliComments(bvid string) (string, error) {
aid, err := getAid(bvid)
if err != nil { return "", err }
cookieStr := ""
if data, err := os.ReadFile("bilibili_cookie.txt"); err == nil { cookieStr = strings.TrimSpace(string(data)) }
apiURL := fmt.Sprintf("https://api.bilibili.com/x/v2/reply?type=1&oid=%s&pn=1&sort=2", aid)
cmdArgs := []string{"-s", "--compressed", "-A", "Mozilla/5.0"}
if cookieStr != "" { cmdArgs = append(cmdArgs, "-H", "Cookie: "+cookieStr) }
cmdArgs = append(cmdArgs, apiURL)
cmd := exec.Command("curl", cmdArgs...)
out, err := cmd.Output()
if err != nil { return "", err }
var result map[string]interface{}
json.Unmarshal(out, &result)
data, _ := result["data"].(map[string]interface{})
replies, _ := data["replies"].([]interface{})
var sb strings.Builder
for i, r := range replies {
if i >= 10 { break }
replyMap, _ := r.(map[string]interface{})
content, _ := replyMap["content"].(map[string]interface{})
message, _ := content["message"].(string)
rpid := int64(replyMap["rpid"].(float64))
rcount := int(replyMap["rcount"].(float64))
sb.WriteString(fmt.Sprintf("- [一级评论] %s\n", message))
if rcount > 0 && i < 5 {
subAPI := fmt.Sprintf("https://api.bilibili.com/x/v2/reply/reply?type=1&oid=%s&root=%d&pn=1&ps=3", aid, rpid)
subCmdArgs := []string{"-s", "--compressed", "-A", "Mozilla/5.0"}
if cookieStr != "" { subCmdArgs = append(subCmdArgs, "-H", "Cookie: "+cookieStr) }
subCmdArgs = append(subCmdArgs, subAPI)
subCmd := exec.Command("curl", subCmdArgs...)
subOut, _ := subCmd.Output()
var subResult map[string]interface{}
json.Unmarshal(subOut, &subResult)
subData, _ := subResult["data"].(map[string]interface{})
subReplies, _ := subData["replies"].([]interface{})
for _, sr := range subReplies {
srMap, _ := sr.(map[string]interface{})
subMsg, _ := srMap["content"].(map[string]interface{})["message"].(string)
sb.WriteString(fmt.Sprintf("  └─ [楼中楼] %s\n", subMsg))
}
time.Sleep(1 * time.Second)
}
}
return sb.String(), nil
}

func getBilibiliInfo(bvid string) (map[string]string, error) {
apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?bvid=%s", bvid)

cookieStr := ""
if data, err := os.ReadFile("bilibili_cookie.txt"); err == nil {
cookieStr = strings.TrimSpace(string(data))
}

// 改用 Go 原生 HTTP，自带证书库，无视 Alpine 缺少证书的问题
req, _ := http.NewRequest("GET", apiURL, nil)
req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64)")
if cookieStr != "" {
req.Header.Set("Cookie", cookieStr)
}

client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Do(req)
if err != nil { 
fmt.Printf("[%s] ❌ HTTP 请求信息失败: %v\n", bvid, err)
return nil, err 
}
defer resp.Body.Close()

var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)

code, _ := result["code"].(float64)
if code != 0 {
fmt.Printf("[%s] ❌ B站API返回错误码: %.0f\n", bvid, code)
return nil, fmt.Errorf("B站接口错误码: %.0f", code)
}

data, _ := result["data"].(map[string]interface{})
if data == nil { return nil, fmt.Errorf("无数据") }

title, _ := data["title"].(string)
if title == "" { return nil, fmt.Errorf("标题为空") }

stat, _ := data["stat"].(map[string]interface{})
owner, _ := data["owner"].(map[string]interface{})
pic, _ := data["pic"].(string)

if strings.HasPrefix(pic, "http://") { pic = strings.Replace(pic, "http://", "https://", 1) }

upName := ""
if owner != nil { upName = owner["name"].(string) }

return map[string]string{
"title": title, 
"up": upName, 
"view": fmt.Sprintf("%.0f", stat["view"].(float64)), 
"like": fmt.Sprintf("%.0f", stat["like"].(float64)), 
"cover": pic,
}, nil
}

func getCid(bvid string) (string, error) {
apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?bvid=%s", bvid)
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Get(apiURL)
if err != nil { return "", err }
defer resp.Body.Close()
var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)
data, _ := result["data"].(map[string]interface{})
return fmt.Sprintf("%.0f", data["cid"].(float64)), nil
}

func getAid(bvid string) (string, error) {
apiURL := fmt.Sprintf("https://api.bilibili.com/x/web-interface/view?bvid=%s", bvid)
client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Get(apiURL)
if err != nil { return "", err }
defer resp.Body.Close()
var result map[string]interface{}
json.NewDecoder(resp.Body).Decode(&result)
data, _ := result["data"].(map[string]interface{})
return fmt.Sprintf("%.0f", data["aid"].(float64)), nil
}

func checkBilibiliCookie() bool {
cookieStr := ""
if data, err := os.ReadFile("bilibili_cookie.txt"); err == nil { cookieStr = strings.TrimSpace(string(data)) }
if cookieStr == "" { return false }
cmd := exec.Command("curl", "-s", "-A", "Mozilla/5.0", "-H", "Cookie: "+cookieStr, "https://api.bilibili.com/x/web-interface/nav")
out, _ := cmd.Output()
var result map[string]interface{}
json.Unmarshal(out, &result)
return int(result["code"].(float64)) == 0
}
