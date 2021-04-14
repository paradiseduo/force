package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	ftp "github.com/jlaffaye/ftp"
	_ "github.com/lib/pq"
	elastic "github.com/olivere/elastic/v7"
	"github.com/schollz/progressbar/v3"
	mongo "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"golang.org/x/crypto/ssh"
)

var ip = flag.String("ip", "", "地址")
var port = flag.String("port", "22", "端口")
var user = flag.String("user", "", "用户名")
var password = flag.String("password", "", "密码")
var mode = flag.String("mode", "ssh", "爆破选项: ssh/ftp/mysql/postgres/mongo/es")
var timeout = flag.Int("-timeout", 3, "超时时间，默认3秒")

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
	} else if strings.ContainsAny(*ip, ",") {
		ips = strings.Split(*ip, ",")
	} else {
		ips = append(ips, *ip)
	}

	bar := progressbar.NewOptions(len(ips), progressbar.OptionSetRenderBlankState(true))
	go printProcess(bar)
	COROUTNUM := runtime.GOMAXPROCS(runtime.NumCPU())
	if len(ips) < COROUTNUM {
		COROUTNUM = len(ips)
	}
	groupLength := len(ips) / COROUTNUM
	wg.Add(COROUTNUM)
	switch *mode {
	case "ssh":
		if len(ips) > 1 {
			for i := 0; i < COROUTNUM; i++ {
				go doSSHs(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
			}
			go doSSHs(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
			wg.Wait()
		} else {
			doSSHs(ips, *port, *user, *password, bar)
		}
	case "mysql":
		if len(ips) > 1 {
			for i := 0; i < COROUTNUM; i++ {
				go doMySQLs(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
			}
			go doMySQLs(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
			wg.Wait()
		} else {
			doMySQLs(ips, *port, *user, *password, bar)
		}
	case "postgres":
		if len(ips) > 1 {
			for i := 0; i < COROUTNUM; i++ {
				go doPostgress(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
			}
			go doPostgress(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
			wg.Wait()
		} else {
			doPostgress(ips, *port, *user, *password, bar)
		}
	case "mongo":
		if len(ips) > 1 {
			for i := 0; i < COROUTNUM; i++ {
				go doMongos(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
			}
			go doMongos(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
			wg.Wait()
		} else {
			doMongos(ips, *port, *user, *password, bar)
		}
	case "ftp":
		if len(ips) > 1 {
			for i := 0; i < COROUTNUM; i++ {
				go doFTPs(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
			}
			go doFTPs(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
			wg.Wait()
		} else {
			doFTPs(ips, *port, *user, *password, bar)
		}
	case "es":
		if len(ips) > 1 {
			for i := 0; i < COROUTNUM; i++ {
				go doElasticSearchs(ips[i*groupLength:((i+1)*groupLength)], *port, *user, *password, bar)
			}
			go doElasticSearchs(ips[COROUTNUM*groupLength:], *port, *user, *password, bar)
			wg.Wait()
		} else {
			doElasticSearchs(ips, *port, *user, *password, bar)
		}
	default:
		flag.Usage()
	}
	bar.Finish()
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
		Timeout:         time.Duration(*timeout) * time.Second,
	})
	if err == nil {
		fmt.Println("\nSSH Success", ip, port, user, password)
		return true
	}
	return false
}

func doMySQLs(ips []string, port, user, password string, bar *progressbar.ProgressBar) {
	for _, v := range ips {
		if doMySQL(v, port, user, password) {
			bar.Clear()
		}
		mutex.Lock()
		scanedNum += 1
		mutex.Unlock()
	}
	wg.Done()
}

func doMySQL(ip, port, user, password string) bool {
	sss := user + ":" + password + "@tcp(" + ip + ":" + port + ")/mysql?charset=utf8&timeout=" + strconv.Itoa(*timeout) + "s"
	db, err := sql.Open("mysql", sss)
	if err == nil {
		if er := db.Ping(); er == nil {
			defer db.Close()
			fmt.Println("\nMySQL Success", ip, port, user, password)
			return true
		}
	}
	return false
}

func doPostgress(ips []string, port, user, password string, bar *progressbar.ProgressBar) {
	for _, v := range ips {
		if doPostgres(v, port, user, password) {
			bar.Clear()
		}
		mutex.Lock()
		scanedNum += 1
		mutex.Unlock()
	}
	wg.Done()
}

func doPostgres(ip, port, user, password string) bool {
	dataSourceName := "postgres://" + user + ":" + password + "@" + ip + ":" + port + "/postgres?sslmode=disable&connect_timeout=" + strconv.Itoa(*timeout)
	db, err := sql.Open("postgres", dataSourceName)
	if err == nil {
		if er := db.Ping(); er == nil {
			defer db.Close()
			fmt.Println("\nPostgres Success", ip, port, user, password)
			return true
		}
	}
	return false
}

func doMongos(ips []string, port, user, password string, bar *progressbar.ProgressBar) {
	for _, v := range ips {
		if doMongo(v, port, user, password) {
			bar.Clear()
		}
		mutex.Lock()
		scanedNum += 1
		mutex.Unlock()
	}
	wg.Done()
}

func doMongo(ip, port, user, password string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	uri := "mongodb://" + user + ":" + password + "@" + ip + ":" + port
	if password == "" {
		uri = "mongodb://" + ip + ":" + port
		user = ""
	}

	opt := new(options.ClientOptions)
	du, _ := time.ParseDuration(strconv.Itoa(*timeout * 1000))
	opt = opt.SetConnectTimeout(du)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri), opt)

	if err == nil {
		e := client.Ping(ctx, readpref.Primary())
		if e == nil {
			defer client.Disconnect(ctx)
			fmt.Println("\nMongoDB Success", ip, port, user, password)
			return true
		}
	}
	return false
}

func doFTPs(ips []string, port, user, password string, bar *progressbar.ProgressBar) {
	for _, v := range ips {
		if doFTP(v, port, user, password) {
			bar.Clear()
		}
		mutex.Lock()
		scanedNum += 1
		mutex.Unlock()
	}
	wg.Done()
}

func doFTP(ip, port, user, password string) bool {
	client, err := ftp.Dial(ip+":"+port, ftp.DialWithTimeout(time.Duration(*timeout)*time.Second))
	if err == nil {
		if user == "" {
			user = "anonymous"
		}
		err = client.Login(user, password)
		if err == nil {
			defer client.Quit()
			fmt.Println("\nFTP Success", ip, port, user, password)
			return true
		}
	}
	return false
}

func doElasticSearchs(ips []string, port, user, password string, bar *progressbar.ProgressBar) {
	for _, v := range ips {
		if doElasticSearch(v, port, user, password) {
			bar.Clear()
		}
		mutex.Lock()
		scanedNum += 1
		mutex.Unlock()
	}
	wg.Done()
}

func doElasticSearch(ip, port, user, password string) bool {
	url := "http://" + ip + ":" + port
	client, err := elastic.NewClient(elastic.SetSniff(false),
		elastic.SetURL(url),
		elastic.SetBasicAuth(user, password),
	)
	if err == nil {
		ctx := context.Background()
		_, _, err = client.Ping(url).Do(ctx)
		if err == nil {
			fmt.Println("\nElasticSearch Success", ip, port, user, password)
			return true
		}
	}
	return false
}
