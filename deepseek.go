package main

import (
"bytes"
"encoding/json"
"fmt"
"io"
"net/http"
"os"
"strings"
"time"
)

const deepseekEndpoint = "https://api.deepseek.com/v1/chat/completions"

func getDeepSeekKey() (string, error) {
if data, err := os.ReadFile("deepseek_key.txt"); err == nil {
if k := strings.TrimSpace(string(data)); k != "" { return k, nil }
}
if k := os.Getenv("DEEPSEEK_API_KEY"); k != "" { return k, nil }
return "", fmt.Errorf("未找到 DeepSeek Key")
}

type chatMessage struct {
Role    string `json:"role"`
Content string `json:"content"`
}

func callDeepSeek(systemPrompt, userContent string) (string, error) {
key, err := getDeepSeekKey()
if err != nil { return "", err }
reqBody := map[string]interface{}{
"model": "deepseek-chat",
"messages": []chatMessage{
{Role: "system", Content: systemPrompt},
{Role: "user", Content: userContent},
},
"temperature": 0.4,
"stream":      false,
}
data, _ := json.Marshal(reqBody)
req, _ := http.NewRequest("POST", deepseekEndpoint, bytes.NewReader(data))
req.Header.Set("Authorization", "Bearer "+key)
req.Header.Set("Content-Type", "application/json")

client := &http.Client{Timeout: 180 * time.Second}
resp, err := client.Do(req)
if err != nil { return "", fmt.Errorf("请求 DeepSeek 失败: %w", err) }
defer resp.Body.Close()
body, _ := io.ReadAll(resp.Body)
if resp.StatusCode != 200 { return "", fmt.Errorf("DeepSeek 返回 %d: %s", resp.StatusCode, string(body)) }

var result struct {
Choices []struct {
Message struct {
Content string `json:"content"`
} `json:"message"`
} `json:"choices"`
}
if err := json.Unmarshal(body, &result); err != nil { return "", fmt.Errorf("解析失败: %w", err) }
if len(result.Choices) == 0 { return "", fmt.Errorf("DeepSeek 返回空") }
return result.Choices[0].Message.Content, nil
}

func summarizeBilibili(content, danmaku, comments string) (string, error) {
system := `你是一个拥有百万粉丝的 B站“课代表”兼视频解说员。你的任务是阅读用户提供的视频元数据、字幕文本、弹幕和评论，写出一篇让读者“看完报告就等于看完了整个视频，甚至比看视频还爽”的深度总结。

【写作要求】
1. 语言生动、网感强，拒绝干巴巴的公文汇报。遇到专有名词或英文，请保留英文并在其后附上中文翻译。
2. 充分利用所有素材：字幕是主干（讲了什么），弹幕是情绪（大家在哪笑了/吐槽了），评论/楼中楼是深度思考（大家怎么看）。
3. 结构必须包含以下四个模块：

### ?? 视频背景板
用 1-2 句话总结标题、UP主属性，并结合播放量和点赞数给出整体定调。

### ?? 核心内容全景复刻
（这是最重要的一环！）不要只写干巴巴的几条大纲。请详细还原视频的叙事脉络或干货，分点写出“UP主抛出了什么观点”、“用了什么精彩的案例或梗”。让读者感觉像看了一场微电影或一场精彩的演讲。

### ?? 弹幕与评论区“同屏”神反应
将弹幕和评论混合分析。提取弹幕里的高频词和玩梗，以及评论区（含楼中楼）的高赞神评。
必须体现出 B站独特的社区氛围，比如：大家在哪个环节破防了？哪里开始疯狂玩梗？评论区出现了什么“课代表补充”或者“神仙打架”？

### ?? 课代表终极洞察
结合视频内容和观众反应，提炼出 UP主 的潜台词，或者这个视频反映了当下什么社会/圈层情绪。

### ?? 一句话避坑/安利指南
用一句话告诉没看的人，这个视频值不值得看，适合什么人看。

【注意】不要编造未出现的内容，信息不足的模块可适度精简。`

// 核心截断逻辑：如果正文超过 3 万字，掐头去尾留中间
if len(content) > 30000 {
content = content[:15000] + "\n\n...[由于内容过长，中间部分已省略]...\n\n" + content[len(content)-15000:]
}

var sb strings.Builder
sb.WriteString("# 视频元数据与正文\n")
sb.WriteString(content)
if strings.TrimSpace(danmaku) != "" {
d := danmaku
if len(d) > 4000 { d = d[:4000] }
sb.WriteString("\n\n# 弹幕采样（体现实时情绪和造梗）\n")
sb.WriteString(d)
}
if strings.TrimSpace(comments) != "" {
sb.WriteString("\n\n# 评论区与楼中楼（体现深度探讨和神评）\n")
sb.WriteString(comments)
}
return callDeepSeek(system, sb.String())
}
