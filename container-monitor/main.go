package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type ConStatus struct {
	Read         string `json:"read"`
	Name         string `json:"name"`
	MemoryStatus struct {
		Usage    int `json:"usage"`
		MaxUsage int `json:"max_usage"`
		Limit    int `json:"limit"`
	} `json:"memory_stats"`
	CPUStatus struct {
		UsePercent int `json:"online_cpus"`
	} `json:"cpu_stats"`
}

type monitorTask struct {
	containerID  string // 容器ID
	dataSavePath string // 数据保存的文件路径
}

var (
	dockerClient  *client.Client // docker 监控客户端
	memoryDivisor = 1024 * 1024  // memory divite number, default convert M
	flag          = true         // 程序是否继续运行的标志
	taskList      []monitorTask  // 监控任务列表
)

func init() {
	pflag.String("id", "", "need monite container ID,split by space")
	pflag.String("memory_unit", "M", "memory usage unit, option: K,M,G ")

	pflag.Parse()

	viper.BindPFlags(pflag.CommandLine)
}

func main() {
	initDockerClient()
	initMemoryDivisor()
	initMonitorTask()

	for _, task := range taskList {
		go task.run()
	}

	signalMsg := make(chan os.Signal, 1)
	signal.Notify(signalMsg, syscall.SIGINT, syscall.SIGTERM)

	<-signalMsg
	flag = false
	time.Sleep(time.Second * 6)
	fmt.Println("stop .........")
}

func initMemoryDivisor() {
	unit := viper.GetString("memory_unit")
	switch unit {
	case "K":
		memoryDivisor = 1 << 10
	case "M":
		memoryDivisor = 1 << 20
	case "G":
		memoryDivisor = 1 << 30
	default:
		fmt.Println("input memory_unit is error, option: K, M, G")
	}
}

// 初始化 DockerClient
func initDockerClient() {
	var err error
	//docker客户端对象
	dockerClient, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		fmt.Println("docker client create error")
		panic(err)
	}
}

// 初始化监控任务列表
func initMonitorTask() {
	// dataDir := viper.Get("path").(string) // 路径
	dataDir, err := os.Getwd()
	if err != nil {
		fmt.Println("get workpath is error: ", err)
	}

	idString := viper.Get("id").(string) // 输入的 id 字符串
	ids := handleId(idString)

	// 填充监控任务队列
	for _, id := range ids {
		var task monitorTask
		task.containerID = id
		task.dataSavePath = dataDir + "/" + id + ".csv"
		taskList = append(taskList, task)
	}
}

// 处理输入的 id 字符串
func handleId(idString string) []string {
	var idList []string

	if idString == "" {
		panic("input id is empty")
	}

	ids := strings.Split(idString, " ")
	if len(ids) == 0 {
		panic("input id is empty")
	}

	for i := 0; i < len(ids); i++ {
		id := strings.Trim(ids[i], " ")
		if id != "" {
			idList = append(idList, id)
		}
	}

	if len(idList) == 0 {
		panic("input id is empty")
	}
	return idList
}

// 启动任务
func (task *monitorTask) run() {
	os.Remove(task.dataSavePath)
	csvFile, err := os.Create(task.dataSavePath)

	if err != nil {
		fmt.Println(task.dataSavePath + "file create error")
		panic(err)
	}
	defer csvFile.Close()

	// 写入 csv 表格标题
	_, err = csvFile.WriteString("time,memory,CPU\n")
	if err != nil {
		fmt.Println("csv title write error: ", err)
	}

	var timeIndex = 0
	for flag {
		info, err := getStatus(task.containerID)
		var msg = ""
		memory := info.MemoryStatus.Usage / memoryDivisor

		if err != nil {
			msg = strconv.Itoa(timeIndex) + "," + " ," + " " + "\n"
		} else {
			msg = strconv.Itoa(timeIndex) + "," + strconv.Itoa(memory) + "," + strconv.Itoa(info.CPUStatus.UsePercent) + "\n"
		}
		_, err = csvFile.WriteString(msg)
		if err != nil {
			fmt.Println("msg: "+msg+"write error: ", err)
		}

		timeIndex++
		time.Sleep(time.Second * 5)
	}
}

func getStatus(containerID string) (ConStatus, error) {
	//获取ctx
	ctx := context.Background()

	//通过cli的ContainerStats方法可以获取到 docker stats命令的详细信息，其实是一个容器监控的方法
	//这个方法返回了容器使用CPU 内存 网络 磁盘等诸多信息
	containerStats, err := dockerClient.ContainerStats(ctx, containerID, false)
	if err != nil {
		fmt.Println("get docker container info error", err)
		return ConStatus{}, err
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(containerStats.Body)
	return byte2ConStatus(buf.Bytes()), nil
}

func byte2ConStatus(stats []byte) ConStatus {
	var con ConStatus
	if err := json.Unmarshal([]byte(stats), &con); err != nil {
		fmt.Println("containerStats.Body byte[] convert ConStatus struct error", err)
	}
	return con
}

func (con ConStatus) String() string {
	return fmt.Sprintf(`
	Time: %s
	ContainerName: %s
	MemeryUsage: %d
	CPU_percent:%d
	`, con.Read, con.Name, con.MemoryStatus.MaxUsage, con.CPUStatus.UsePercent)
}
