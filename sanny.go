package main

import (
	"flag"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755) // 0755 是常见的目录权限
}

// ParseNginxDirectory 解析Nginx目录列表页面，返回所有文件的相对路径
func ParseDirectory(u *url.URL) ([]string, error) {

	// 确保URL以/结尾
	if !strings.HasSuffix(u.Path, "/") {
		u.Path += "/"
	}

	// 发起HTTP请求
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回错误状态码: %d", resp.StatusCode)
	}

	// 解析HTML
	return parseHTML(resp.Body)
}

// parseNginxIndexHTML 解析Nginx生成的目录索引HTML
func parseHTML(body io.Reader) ([]string, error) {
	var paths []string
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	// 递归查找所有<a>标签
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					href := attr.Val
					// 跳过父目录链接和空链接
					if href == "../" || href == "" || href == "./" {
						continue
					}
					// 跳过绝对URL和外部链接
					if strings.HasPrefix(href, "http://") ||
						strings.HasPrefix(href, "https://") ||
						strings.HasPrefix(href, "//") {
						continue
					}
					// 处理目录链接（以/结尾）
					//fmt.Println(href)
					if strings.HasSuffix(href, "/") {
						//href = strings.TrimSuffix(href, "/")
						continue
					}
					pattern := `^[\w\-\.]+\.[a-zA-Z0-9]{1,10}$`
					re, err := regexp.Compile(pattern)
					if err != nil {
						log.Fatal(err)
					}

					matched := re.MatchString(href)
					if matched {
						//fmt.Println("匹配成功")
						paths = append(paths, href)
					} else {
						//fmt.Println("匹配失败")
						continue
					}
					// 构建相对路径
					//fullPath := path.Join(basePath, href)

				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return paths, nil
}
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	// 这里我们检查是否是一个文件
	return !info.IsDir()
}

func downloadFile(url, outputDir string) error {

	// 从URL提取文件名
	filename := filepath.Base(url)
	if filename == "." || filename == "/" {
		filename = "downloaded_file"
	}

	// 创建本地文件
	outputPath := filepath.Join(outputDir, filename)
	if fileExists(outputPath) {
		fmt.Printf("文件 %s 已存在\n", outputPath)
		return nil
	}
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP请求失败: %s", resp.Status)
	}
	// 写入文件
	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	fmt.Printf("文件已下载到: %s\n", outputPath)
	return nil
}

func main() {
	targetURL := flag.String("url", "", "网络文件夹URL")
	output := flag.String("output", "./downloads", "本地保存目录")
	thread := flag.String("thread", "1", "线程数")
	flag.Parse()
	baseURL, err := url.Parse(*targetURL)
	if err != nil {
		fmt.Errorf("无效的URL: %v", err)
	}
	paths, err := ParseDirectory(baseURL)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Sanny found %d files\n", len(paths))
	maxGoroutines, err := strconv.Atoi(*thread)
	if err != nil {
		fmt.Println("转换错误:", err)
		return
	}
	if err := EnsureDir(*output); err != nil {
		panic(err)
	}
	fmt.Printf("max thead num is %d\n", maxGoroutines)
	fmt.Println("Downloading ...")
	fmt.Println("找到的文件路径:")
	semaphore := make(chan struct{}, maxGoroutines)
	var wg sync.WaitGroup
	for _, p := range paths {
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量
		fullURL := baseURL.ResolveReference(&url.URL{Path: p})
		go func(url, output string) {
			defer func() {
				<-semaphore // 释放信号量
				wg.Done()
			}()
			err := downloadFile(fullURL.String(), output)
			if err != nil {
				panic(err)
				return
			}
		}(fullURL.String(), *output)

	}
	wg.Wait()
}
