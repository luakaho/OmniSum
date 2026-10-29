package main

import (
"fmt"
)

func main() {
bvid := "BV1Wpnb6kEVu" // 用你刚才测试的视频，或者你随便改一个
fmt.Printf("========== 纯评论抓取测试 (BV: %s) ==========\n", bvid)

comments, err := getBilibiliComments(bvid)
if err != nil {
fmt.Println("❌ 抓取失败:", err)
return
}

fmt.Printf("\n✅ 抓取成功，总长度: %d 字\n", len(comments))
fmt.Println("========== 评论原文 ==========")
fmt.Println(comments)
}
