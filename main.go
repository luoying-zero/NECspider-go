package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/gocolly/colly/v2"
	"github.com/schollz/progressbar/v3"
	"golang.org/x/sync/semaphore"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	var pam int64
	field := []byte{0x22, 0x75, 0x73, 0x65, 0x72, 0x49, 0x64, 0x22, 0x3a}
	author := []byte{0x36, 0x32, 0x36, 0x39, 0x36, 0x32, 0x38, 0x39, 0x2c}
	flag.Int64Var(&pam, "p", 500, "设置并发量")

	// 解析标志参数
	flag.Parse()

	// 处理位置参数（非标志参数）
	if len(flag.Args()) < 2 {
		fmt.Fprintln(os.Stderr, "缺少范围")
		return
	}

	num1, err1 := strconv.ParseInt(flag.Arg(0), 0, 64)
	num2, err2 := strconv.ParseInt(flag.Arg(1), 0, 64)
	if err1 != nil || err2 != nil {
		fmt.Fprintln(os.Stderr, "范围必须为整数")
		return
	}

	dataChan := make(chan int64, 100)
	//var sli []int
	go func() {
		for num := range dataChan {
			//sli = append(sli, num)
			fmt.Printf("\"https://music.lliiiill.com/playlist/%d\",\n", num)
		}
	}()
	printChan := make(chan string, 100)
	go func() {
		for msg := range printChan {
			fmt.Fprintln(os.Stderr, msg)
		}
	}()

	ctx := context.TODO()
	sem := semaphore.NewWeighted(pam)

	bar := progressbar.DefaultSilent(num2 - num1 + 1)

	transport := &http.Transport{
		MaxIdleConns:        0,          // 全局最大空闲连接数
		MaxIdleConnsPerHost: 2^63-1,           // 每个主机的最大空闲连接数
		MaxConnsPerHost:    0,
		IdleConnTimeout:        72 * time.Second,
		TLSHandshakeTimeout:     20,
	}

	c := colly.NewCollector(
		colly.Async(true),
	)
	c.WithTransport(transport)

	c.OnResponse(func(res *colly.Response) {
	    sem.Release(1)
		if checkSequence(res.Body, field, author) {
			plid, _ := res.Ctx.GetAny("plid").(int64)
			dataChan <- plid
		}
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
			plid, _ := r.Ctx.GetAny("plid").(int64)
			printChan <- fmt.Sprintf("Error plid: %v %v", plid, err)
			sem.Release(1)
		}
	})

	// 遍历指定的id范围
	for id := num1; id <= num2; id++ {
		if err := sem.Acquire(ctx, 1); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to acquire semaphore: %v", err)
			break
		}
		bar.Add(1)
		ctx := colly.NewContext()
		ctx.Put("plid", id)
		c.Request(
			"POST", 
			"http://music.163.com/api/v6/playlist/detail", 
			strings.NewReader("id="+strconv.FormatInt(id, 10)), 
			ctx, 
			http.Header{
				"Content-Type": []string{"application/x-www-form-urlencoded"},
			},
		)
	}

	c.Wait()
	fmt.Fprintf(os.Stderr, "%+v", bar.State())
	time.Sleep(1 * time.Second)
	close(dataChan)
	close(printChan)

	// for _, id := range sli {
		// fmt.Printf("\"https://music.163.com/playlist?id=%d\",", id)
	//}
	// fmt.Printf("\n")
	// for _, id := range sli {
		// fmt.Printf("\"https://music.lliiiill.com/playlist/%d\",", id)
	// }
}

func checkSequence(s, sub1, sub2 []byte) bool {
	// 查找第一个子字节串的位置
	idx := bytes.Index(s, sub1)
	if idx == -1 {
		return false
	}
	// 截取第一个子字节串之后的部分
	remaining := s[idx+len(sub1):]
	// 判断剩余部分是否以第二个子字节串开头
	return bytes.HasPrefix(remaining, sub2)
}
