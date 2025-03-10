package main

import (
	"fmt"
	"strconv"
	"flag"
	// "os/exec"
	"time"
	"github.com/gocolly/colly"
)

func main() {
	sti := time.Now()
	var pam int
	// errn := 0
	var sli []string
	
	flag.IntVar(&pam, "p", 50, "设置并发量")

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
    
	// 创建一个colly收集器
	c := colly.NewCollector(
		// 设置Colly的并发数
		colly.Async(true), // 启用异步请求
	)

	// 设置并发量
	c.Limit(&colly.LimitRule{
		DomainGlob:  "*.com",
		Parallelism: pam, // 调整为需要的并发量
	})

	// 设置抓取内容时的处理函数
	c.OnHTML(".user > .name > a", func(e *colly.HTMLElement) {
		if e.Text == "PurionPurion" {
			sli = append(sli, e.Request.URL.String())
		}
		// 用Break从回调中返回，这将阻止进一步的元素匹配
		// 因为colly并不原生支持选择第一个元素，所以一旦匹配到第一个元素，就通过Break中断处理
		e.Request.Abort()
	})

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
            } else{
		    ur := q.URL.String()
		    fmt.Println(err, "Error URL:", ur)
		    // exec.Command("cmd", "/c", "start", ur).Start()
		    // errn = errn + 1
	    }
	})

	// 遍历指定的id范围
	for id := num1; id <= num2; id++ {
		url := fmt.Sprintf("https://music.163.com/playlist?id=%d", id)
		// 访问URL
		err := c.Visit(url)
		if err != nil {
			fmt.Printf("Error visiting: %s - %v\n", url, err)
		}
	}
	c.Wait()
	fmt.Println(sli)
	fmt.Printf("pam:%d time:%s", pam, time.Since(sti))
}
