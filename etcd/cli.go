package etcd

import (
	"context"
	"go.etcd.io/etcd/clientv3"
	"time"
)

func create(endpoint string) (*clientv3.Client, error) {
	return clientv3.New(clientv3.Config{
		Endpoints:   []string{endpoint},
		DialTimeout: 5 * time.Second,
	})
}

func Get(endpoint, k string) (*clientv3.GetResponse, error) {
	cli, err := create(endpoint)
	if err != nil {
		return nil, err
	}
	return cli.Get(context.TODO(), k)
}

func GetByPrefix(endpoint, k string) (*clientv3.GetResponse, error) {
	cli, err := create(endpoint)
	if err != nil {
		return nil, err
	}
	return cli.Get(context.TODO(), k, clientv3.WithPrefix())
}

func KVToString(b []byte) string {
	return string(b)
}
