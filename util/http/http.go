package http

import (
	"io/ioutil"
	"net"
	"net/http"
	"pictorial/log"
	"pictorial/mysql"
	"strings"
)

const MethodGet = "GET"
const MethodPost = "POST"
const MethodDelete = "DELETE"

func Get(url string) ([]byte, error) {
	req, err := http.NewRequest(MethodGet, url, nil)
	if err != nil {
		return nil, nil
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func Do(req *http.Request) ([]byte, error) {
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

type Auth struct {
	Username string
	Password string
}

func NewRequestDo(url, tp string, auth *Auth, kv map[string]string, pl string) ([]byte, error) {
	var req *http.Request
	var err error
	if pl != "" {
		req, err = http.NewRequest(tp, url, strings.NewReader(pl))
	} else {
		req, err = http.NewRequest(tp, url, nil)
	}
	if err != nil {
		return nil, err
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	for k, v := range kv {
		req.Header.Set(k, v)
	}
	log.Logger.Debug(url)
	return Do(req)
}

const Header = "http://"

func ClearHttpHeader(v string) string {
	return strings.TrimPrefix(v, Header)
}

func GetIpList() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var iplist []string
	for _, i := range interfaces {
		if i.Flags&net.FlagLoopback == 0 && i.Flags&net.FlagUp != 0 {
			addrs, err := i.Addrs()
			if err != nil {
				log.Logger.Warn("Failed to get addresses for interface", i.Name, ":", err)
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
					ip := ipnet.IP
					iplist = append(iplist, ip.String())
				}
			}
		}
	}
	return iplist, nil
}

func MatchIp() (bool, error) {
	ipList, err := GetIpList()
	if err != nil {
		return false, err
	}
	log.Logger.Debug(ipList)
	for _, ip := range ipList {
		if ip == mysql.M.Host {
			return true, nil
		}
	}
	return false, nil
}
