package main

import (
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/crypto/ssh"
)

var ip = flag.String("ip", "", "地址")
var port = flag.String("port", "22", "端口")
var user = flag.String("user", "root", "用户名")
var password = flag.String("password", "root", "密码")

var mutex sync.RWMutex
var scanedNum int
var wg sync.WaitGroup

func main() {
	flag.Parse()
	if *ip == "" {
		flag.Usage()
		return
	}

	var ips []string
	if strings.ContainsAny(*ip, "-") {
		arr := strings.Split(*ip, "-")
		end, err := strconv.Atoi(arr[1])
		arr = strings.Split(arr[0], ".")
		start, err := strconv.Atoi(arr[len(arr)-1])
		if err != nil {
			log.Fatalf("ip error: %s", err.Error())
			return
		}
		ipStart := strings.Join(arr[:len(arr)-1], ".")
		for i := start; i <= end; i++ {
			ips = append(ips, ipStart+"."+strconv.Itoa(i))
		}

		bar := progressbar.NewOptions(len(ips), progressbar.OptionSetRenderBlankState(true))
		go printProcess(bar)
		COROUTNUM := runtime.GOMAXPROCS(runtime.NumCPU())
		groupLength := len(ips) / COROUTNUM
		wg.Add(COROUTNUM)
		for i := 0; i < COROUTNUM; i++ {
			go doSSHs(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
		}
		go doSSHs(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
		wg.Wait()
		bar.Finish()
	} else {
		doSSH(*ip, *port, *user, *password)
	}
}

func printProcess(bar *progressbar.ProgressBar) {
	for {
		mutex.RLock()
		bar.Set(scanedNum)
		mutex.RUnlock()
	}
}

func doSSHs(ips []string, port, user, password string, bar *progressbar.ProgressBar) {
	for _, v := range ips {
		if doSSH(v, port, user, password) {
			bar.Clear()
		}
		mutex.Lock()
		scanedNum += 1
		mutex.Unlock()
	}
	wg.Done()
}

func doSSH(ip, port, user, password string) bool {
	_, err := ssh.Dial("tcp", ip+":"+port, &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.Password(password)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         3 * time.Second,
	})
	if err == nil {
		fmt.Println("\nSSH Success", ip, port, user, password)
		return true
	}
	return false
}
