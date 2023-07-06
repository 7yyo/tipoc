# TiPoC
Automated test case tool for TiDB

## UI

## configuration
```toml[log]
level = "info"

[cluster]
name = "tidb-test"
plugin = ""

[mysql]
host = "10.2.103.202"
port = "5000"
user = "root"
password = ""

[ssh]
user = "tidb"
sshPort = "22"

[load]
cmd = "tiup bench tpcc -H 10.2.103.202 -P 5000 -D tpcc --warehouses 1 --threads 10 --ignore-error --time 5m run"
interval = 0
sleep = 2

[other]
dir = "/go/src/pictorial/other"



```

## how 

```shell
go build -o tipoc main/main.go
./tipoc -c config.toml
```