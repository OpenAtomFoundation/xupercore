package utils

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

const (
	// producer Concurrent
	ProducerConcurrent = 1000
	// Total generate number
	ProducerGenTotal = 100000
)

func TestFileIsExist(t *testing.T) {
	exist := FileIsExist("/home/rd/test/")
	fmt.Println(exist)
}

// 用来验证logId生成算法的冲突率，1000并发下生成十万个id
// 冲突概率小于十万分之一，使用在日志记录场景下足够了
func TestGenLogId(t *testing.T) {
	ch := make(chan string, ProducerConcurrent)
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go producer(wg, ch)

	total, repeat := consumer(wg, ch)

	fmt.Printf("Total:%d Repeat:%d\n", total, repeat)
}

func producer(wg *sync.WaitGroup, ch chan<- string) {
	fmt.Printf("start producer...\n")
	ctlCh := make(chan int, ProducerConcurrent)
	defer close(ctlCh)
	defer wg.Done()

	for i := 0; i < ProducerGenTotal; i++ {
		ctlCh <- 1
		go func() {
			ch <- GenLogId()
			<-ctlCh
		}()
	}

	fmt.Printf("producer done\n")
}

func consumer(wg *sync.WaitGroup, ch <-chan string) (int, int) {
	fmt.Printf("start consumer...\n")
	repeat := 0
	dmap := make(map[string]int)
	isExist := false

	for !isExist {
		select {
		case logid := <-ch:
			//fmt.Printf("log_id:%s\n", logid)
			if _, ok := dmap[logid]; ok {
				repeat++
			} else {
				dmap[logid] = 1
			}
		case <-wait(wg, ch):
			fmt.Printf("recv exit.\n")
			isExist = true
		}
	}

	fmt.Printf("consumer done\n")
	return len(dmap), repeat
}

func wait(wg *sync.WaitGroup, tch <-chan string) <-chan int {
	ch := make(chan int)

	go func() {
		// producer finish
		wg.Wait()

		// check consumer is finish?
		isFin := false
		for !isFin {
			select {
			case <-time.Tick(1 * time.Second):
				if len(tch) < 1 {
					isFin = true
				}
			}
		}

		// set exit signal
		ch <- 1
	}()

	return ch
}

func TestGetFuncCall(t *testing.T) {
	file, fc := GetFuncCall(1)
	fmt.Println(file, fc)
}

func TestGetCurFileDir(t *testing.T) {
	fmt.Println(GetCurFileDir())
}

func TestGetCurExecDir(t *testing.T) {
	fmt.Println(GetCurExecDir())
}
