package main

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

var query = "service.yml"
var matchCount = 0
var workerCount = 1
var MaxWorker = 12
var findChan = make(chan struct{})
var workDone = make(chan struct{})
var searchRequest = make(chan string)

func main() {
	start := time.Now()

	go queryPath("/", true)
	waitWorker()

	fmt.Println(matchCount, " match")
	fmt.Println("spend ", time.Since(start))
}

func queryPath(path string, master bool) {
	itemList, err := os.ReadDir(path)
	if err == nil {
		for _, item := range itemList {
			if item.IsDir() {
				if workerCount < runtime.NumCPU() {
					searchRequest <- path + item.Name() + "/"
				} else {
					queryPath(path+item.Name()+"/", false)
				}
			} else {
				if item.Name() == query {
					fmt.Println(path + item.Name())
					findChan <- struct{}{}
				}
			}
		}
	}
	if master {
		workDone <- struct{}{}
	}
}

func waitWorker() {
	for {
		select {
		case <-findChan:
			matchCount++
		case <-workDone:
			workerCount--
			if workerCount == 0 {
				return
			}
		case path := <-searchRequest:
			workerCount++
			go queryPath(path, true)
		}
	}
}
