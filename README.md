# TiPoC
Automated test case tool for TiDB

## UI
![ui](https://github.com/7yyo/tipoc/blob/master/img/Screenshot%202023-07-06%20at%2023.57.09.png)

## configuration
```toml
[log]
level = "info"

[cluster]
name = "tidb-test"

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
Deployed on the tiup server where the tidb cluster is located, sudo permission is required
```shell
go build -o tipoc main/main.go
./tipoc -c config.toml
```

## todo
#### base test case
- [ ] more and more (currently, there are over 100)
#### high availability
- [x] kill
- [x] crash
- [x] disaster by label
- [x] reboot
- [x] disk
- [ ] network
#### data load
- [x] load data
- [x] import into
- [x] select info outfile
#### scalability
- [ ] scale out
- [x] scale in
#### online ddl
- [x] online add index
- [x] online modify column
- [x] add index performance
#### htap
- [ ] htap workload
#### auto install
- [x] sysbench
- [ ] benchmarkSQL
