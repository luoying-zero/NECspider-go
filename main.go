package main

import (
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"bytes"
	"context"
	"github.com/gocolly/colly/v2"
	"golang.org/x/sync/semaphore"
	"github.com/schollz/progressbar/v3"
)

func main() {
	var pam int
	// errn := 0
	var sli []int
	author := []byte{0x22, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x22, 0x3a, 0x36, 0x32, 0x36, 0x39, 0x36, 0x32, 0x38, 0x39, 0x2c}

	flag.IntVar(&pam, "p", 200, "设置并发量")

	// 解析标志参数
	flag.Parse()

	// 处理位置参数（非标志参数）
	if len(flag.Args()) < 2 {
		fmt.Println("缺少范围")
		return
	}

	num1, err1 := strconv.Atoi(flag.Args()[0])
	num2, err2 := strconv.Atoi(flag.Args()[1])
	if err1 != nil || err2 != nil {
		fmt.Println("范围必须为整数")
		return
	}

	ctx := context.TODO()
	sem := semaphore.NewWeighted(int64(pam))
	
	bar := progressbar.Default(int64(num2 - num1 + 1))
	
	// 创建一个colly收集器
	c := colly.NewCollector(
		// 设置Colly的并发数
		colly.Async(true), // 启用异步请求
	)

	// 设置并发量
	// c.Limit(&colly.LimitRule{
		// DomainGlob:  "*.com",
		// Parallelism: pam, // 调整为需要的并发量
	// })

	c.OnResponse(func(res *colly.Response) {
		if bytes.Contains(res.Body, author) {
			plid , _ := res.Ctx.GetAny("plid").(int)
			sli = append(sli, plid)
		}
		sem.Release(1)
		bar.Add(1)
	})

	// 设置抓取内容时的处理函数
	//c.OnHTML("#content-operation > a.u-btni.u-btni-share", func(e *colly.HTMLElement) {
	//author, _ := e.DOM.Attr("data-res-author")
	//if author == "PurionPurion" {
	//sli = append(sli, e.Request.URL.String())
	//}
	// 用Break从回调中返回，这将阻止进一步的元素匹配
	// 因为colly并不原生支持选择第一个元素，所以一旦匹配到第一个元素，就通过Break中断处理
	//e.Request.Abort()
	//})

	// 错误处理
	c.OnError(func(r *colly.Response, err error) {
		q := r.Request
		retriesLeft := 5
		if x, ok := q.Ctx.GetAny("retriesLeft").(int); ok {
			retriesLeft = x
		}
		if retriesLeft > 0 {
			q.Ctx.Put("retriesLeft", retriesLeft-1)
			q.Retry()
		} else {
			plid , _ := r.Ctx.GetAny("plid").(int)
			fmt.Println(err, "Error plid:", plid)
			sem.Release(1)
			bar.Add(1)
			// exec.Command("cmd", "/c", "start", ur).Start()
			// errn = errn + 1
		}
	})

	// 遍历指定的id范围
	for id := num1; id <= num2; id++ {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Printf("Failed to acquire semaphore: %v", err)
			break
		}
		//url := fmt.Sprintf("http://music.163.com/playlist?id=%d", id)
		// 访问URL
		ctx := colly.NewContext()
		ctx.Put("plid", id)
		c.Request("POST", "http://music.163.com/api/v6/playlist/detail", strings.NewReader("id=" + strconv.Itoa(id)), ctx, http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}})
	}
	c.Wait()
	fmt.Println(sli)
}
