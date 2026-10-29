package main

import (
"fmt"
)

func main() {
bvid := "BV1Wpnb6kEVu"
fmt.Printf("测试获取 BVID: %s 的基础信息...\n", bvid)
info, err := getBilibiliInfo(bvid)
if err != nil {
fmt.Println("❌ 获取失败，报错信息:", err)
return
}
fmt.Println("✅ 获取成功！返回的数据如下:")
for k, v := range info {
fmt.Printf("%s: %s\n", k, v)
}
}
