package main

import (
	bradfitz_memcache "github.com/bradfitz/gomemcache/memcache"
	pangudashu_memcache "github.com/pangudashu/memcache"
	"log"
	"net/http"
	"time"
)

var bradfitz_mc *bradfitz_memcache.Client
var pangudashu_mc *pangudashu_memcache.Memcache

func newPangudashuMemcache() {
	s1 := &pangudashu_memcache.Server{Address: "127.0.0.1:12000", InitConn: 10, Weight: 50, IdleTime: time.Second * 5}

	pangudashu_mc, _ = pangudashu_memcache.NewMemcache([]*pangudashu_memcache.Server{s1})

	pangudashu_mc.SetTimeout(time.Second, time.Second, time.Second)
	pangudashu_mc.SetRemoveBadServer(true)
	log.Println(pangudashu_mc.Set("qp_1", "存储value支持golang基本数据类型：string、[]byte、int、int8、int16、int32、int64、bool、uint8、uint16、uint32、uint64、float32、float64、map、结构体，不需要单独转为string存储,Replace、Increment/Decrement、Delete、Append/Prepend命令支持cas原子操作"))
}

func newBradfitzMemcache() {
	bradfitz_mc = bradfitz_memcache.New("127.0.0.1:12000")

	item := &bradfitz_memcache.Item{
		Key:   "qp_2",
		Value: []byte("存储value支持golang基本数据类型：string、[]byte、int、int8、int16、int32、int64、bool、uint8、uint16、uint32、uint64、float32、float64、map、结构体，不需要单独转为string存储,Replace、Increment/Decrement、Delete、Append/Prepend命令支持cas原子操作"),
	}
	log.Println(bradfitz_mc.Set(item))
}

func main() {
	newPangudashuMemcache()
	newBradfitzMemcache()

	http.HandleFunc("/pangudashu_foo", pangudashuHandler)
	http.HandleFunc("/bradfitz_foo", bradfitzHandler)

	log.Fatal(http.ListenAndServe(":9955", nil))
}

func pangudashuHandler(w http.ResponseWriter, r *http.Request) {
	v, _, _ := pangudashu_mc.Get("qp_1")
	w.Write([]byte(v.(string)))
}

func bradfitzHandler(w http.ResponseWriter, r *http.Request) {
	v, _ := bradfitz_mc.Get("qp_2")
	w.Write(v.Value)
}
