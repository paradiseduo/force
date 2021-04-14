# force
使用golang编写的服务弱口令检测

## 支持协议

|序号|协议|是否支持|
|:---|:---:| :---: |
|1|ftp|✅|
|2|telnet|❌|
|3|ssh|✅|
|4|mysql|✅|
|5|smtp|❌|
|6|smb|❌|
|7|mssql|❌|
|8|postgres|✅|
|9|hive|❌|
|10|redis|❌|
|11|mangoDB|✅|
|12|rdp|❌|
|13|Elasticsearch|✅|


## 使用方式

```bash
> chmod +x force
> ./force 
Usage of ./force:
  --timeout int
    	超时时间，默认3秒 (default 3)
  -ip string
    	地址
  -mode string
    	爆破选项: ssh/ftp/mysql/postgres/mongo/es (default "ssh")
  -password string
    	密码
  -port string
    	端口 (default "22")
  -user string
    	用户名
```
