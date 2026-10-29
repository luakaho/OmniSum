package main

import (
"fmt"
)

func main() {
bvid := "BV1hjHT6cERT"
fmt.Println("========== 纯字幕抓取测试 ==========")
// 只调用抓取函数，不调用大模型和音频切片
content, err := getBilibiliContent(bvid)
if err != nil {
fmt.Println("❌ 抓取失败:", err)
return
}

fmt.Printf("\n✅ 抓取成功，总长度: %d 字\n", len(content))
fmt.Println("========== 字幕内容预览 ==========")
// 如果太长，截取前 800 字
if len(content) > 800 {
fmt.Println(content[:800] + "\n\n...[内容截断]...")
} else {
fmt.Println(content)
}
}
