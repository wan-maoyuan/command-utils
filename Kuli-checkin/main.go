package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/robfig/cron"
)

const (
	CheckInURL = "https://cyooo.co/user/checkin"
	LoginURL   = "https://cyooo.co/auth/login"
	email      = ""
	passwd     = ""
)

type CheckStatus struct {
	Ret int    `json:"ret"`
	Msg string `json:"msg"`
}

func main() {
	c := cron.New()
	spec := "00 30 22 * * ?" // 每天晚上10：30执行任务
	c.AddFunc(spec, func() {
		cookie := getCookie()
		if checkIn(cookie) {
			fmt.Println("签到成功")
		} else {
			fmt.Println("签到失败")
		}
	})

	c.Start()
	select {}
}

func getCookie() string {
	var cookie = ""
	resp, err := http.PostForm(LoginURL, url.Values{
		"email":       {email},
		"passwd":      {passwd},
		"code":        {},
		"remember_me": {"on"},
	})
	if err != nil {
		fmt.Println(time.Now())
		panic("get cookie by email and passwd error" + err.Error())
	}
	defer resp.Body.Close()

	byteData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(time.Now())
		panic("read login request body error:" + err.Error())
	}

	result, err := byte2CheckStatus(byteData)
	if err != nil {
		fmt.Println(time.Now())
		panic("convert login message error:" + err.Error())
	}

	fmt.Println(time.Now(), result.Msg)
	if result.Msg == "登录成功" {
		KuLicookie := resp.Cookies()
		for _, i := range KuLicookie {
			cookie += i.String() + ";"
		}
		cookie = cookie[:len(cookie)-1]
	}
	return cookie
}

// 签到
func checkIn(cookie string) bool {
	var client = &http.Client{}

	req, err := http.NewRequest("POST", CheckInURL, nil)
	if err != nil {
		fmt.Println("create http request error" + err.Error())
		return false
	}

	req.Header.Add("cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("post" + CheckInURL + "error" + err.Error())
		return false
	}
	defer resp.Body.Close()

	byteData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("read response byte data error:" + err.Error())
		return false
	}

	stats, err := byte2CheckStatus(byteData)
	if err != nil {
		return false
	}
	fmt.Println(time.Now(), stats.Msg)
	return true
}

func byte2CheckStatus(data []byte) (CheckStatus, error) {
	var stat CheckStatus
	if err := json.Unmarshal(data, &stat); err != nil {
		return stat, err
	}

	if stat.Msg == "" {
		return stat, errors.New("msg is empty")
	}

	return stat, nil
}
