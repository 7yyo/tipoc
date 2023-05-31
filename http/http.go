package http

import (
	"io/ioutil"
	"net/http"
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
	return Do(req)
}
