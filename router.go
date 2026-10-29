package main

import (
"fmt"
"net/url"
"strings"
)

type Platform string

const (
PlatformBilibili Platform = "bilibili"
PlatformDouyin   Platform = "douyin"
PlatformPodcast  Platform = "podcast"
PlatformZhihu    Platform = "zhihu"
PlatformGitHub   Platform = "github"
PlatformWeb      Platform = "web"
)

func detectPlatform(rawURL string) (Platform, error) {
u, err := url.Parse(strings.TrimSpace(rawURL))
if err != nil || u.Host == "" {
return "", fmt.Errorf("无法识别的链接: %s", rawURL)
}
host := strings.ToLower(u.Host)

switch {
case strings.Contains(host, "bilibili.com") || strings.Contains(host, "b23.tv"):
return PlatformBilibili, nil
case strings.Contains(host, "douyin.com") || strings.Contains(host, "iesdouyin.com"):
return PlatformDouyin, nil
case strings.Contains(host, "xiaoyuzhoufm.com"),
strings.Contains(host, "podcasts.apple.com"),
strings.Contains(host, "spotify.com"):
return PlatformPodcast, nil
case strings.Contains(host, "zhihu.com"):
return PlatformZhihu, nil
case strings.Contains(host, "github.com"):
return PlatformGitHub, nil
default:
return PlatformWeb, nil
}
}
