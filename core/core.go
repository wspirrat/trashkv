package core

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/sync/syncmap"
)

const (
	DatabasePath = "core.db"
)

type Database struct {
	Url     string
	Syncmap syncmap.Map
}

type Core interface {
	Store(key string, value interface{})
	Delete(key string)
	Load(key string) (interface{}, bool)
	Save()
	Sync()
}

// req = request
// request server save
type reqHTTPdataSave struct {
	Sender   *string
	Receiver *string
	Cache    *map[string]interface{}
}

var client *http.Client

// funcs
func Connect(url, privateKey string) (Core, error) {
	// dat is used for unmarshalling database from /connect
	// syncm is Syncmap passed in *Database
	// core is interface which is returned
	var dat map[string]interface{}
	var syncm syncmap.Map
	var core Core

	resp, err := http.Get(fmt.Sprintf("%s/?key=%s", url, privateKey))
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(resp.Body)

	if len(body) <= 2 {
		dat = map[string]interface{}{}
	} else {
		// unmarshal decrypted text
		if err := json.Unmarshal(body, &dat); err != nil {
			return nil, err
		}
	}

	// add all keys from dat to syncm
	for key, value := range dat {
		syncm.Store(key, value)
	}

	resDb := &Database{
		Url:     url,
		Syncmap: syncm,
	}

	core = resDb

	return core, nil
}

func (db *Database) Store(key string, value interface{}) {
	if !replace_key {
		_, exist := db.Syncmap.Load(key)
		if !exist {
			db.Syncmap.Store(key, value)
		}
	} else {
		db.Syncmap.Store(key, value)
		return
	}
}

func (db *Database) Delete(key string) {
	db.Syncmap.Delete(key)
}

// returns value
// and bool
// if object exist returns true
// else if object do not exist returns false
func (db *Database) Load(key string) (value interface{}, exist bool) {
	result, exist := db.Syncmap.Load(key)
	if exist {
		return result, true
	}

	return nil, false
}

// save function send request to server
// server compare and set var db *Database
// as database send in json request
func (db *Database) Save() {
	dataMap := make(map[string]interface{})
	db.Syncmap.Range(func(k interface{}, v interface{}) bool {
		dataMap[k.(string)] = v
		return true
	})

	request := &reqHTTPdataSave{
		Cache: &dataMap,
	}

	j, err := json.Marshal(&request)
	if err != nil {
		fmt.Println(err)
	}

	tr := &http2.Transport{
		AllowHTTP: true,
		DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		},
	}
	client = &http.Client{Transport: tr}

	client.Post(fmt.Sprintf("%s", db.Url), "application/json", bytes.NewBuffer(j))

}

func (db *Database) Sync() {
	tr := &http.Transport{
		MaxIdleConnsPerHost: 1024,
		TLSHandshakeTimeout: 1 * time.Second,
	}
	client = &http.Client{Transport: tr}
	client.PostForm(fmt.Sprintf("%s/tkv_v1/sync", db.Url), nil)
}

/*
func (db *Database) Access() syncmap.Map {
	return db.Syncmap
}
*/
