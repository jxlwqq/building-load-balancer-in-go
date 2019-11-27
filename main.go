package main

import (
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

const (
	Attempts int = iota
	Retry
)

type (
	UpstreamServer struct {
		URL          *url.URL
		Alive        bool
		mux          sync.RWMutex
		ReverseProxy httputil.ReverseProxy // 反向代理
	}

	ServersPool struct {
		upstreamServers []*UpstreamServer
		current         uint64
	}
)

func (sp *ServersPool) GetServerAmount() int {
	return len(sp.upstreamServers)
}

func (sp *ServersPool) GetNextIndex() int {
	amount := sp.GetServerAmount()
	return int(atomic.AddUint64(&sp.current, uint64(1)) % uint64(amount))
}

func (sp *ServersPool) GetNextSibling() (*UpstreamServer, error) {
	amount := sp.GetServerAmount()
	nextIndex := sp.GetNextIndex()
	loops := amount + nextIndex
	for i := nextIndex; i < loops; i++ {
		index := i % amount
		if sp.upstreamServers[index].IsAlive() {
			if i != nextIndex {
				atomic.StoreUint64(&sp.current, uint64(index))
			}
			return sp.upstreamServers[index], nil
		}
	}
	return nil, errors.New("there is not alive upstream server")
}

func (sp *ServersPool) CheckHealth() {
	for _, us := range sp.upstreamServers {
		status := "up"
		alive := checkUpstreamServerAlive(us.URL)
		us.SetAlive(alive)
		if !alive {
			status = "down"
		}
		log.Printf("%s status is [%s]", us.URL, status)
	}
}

func (us *UpstreamServer) IsAlive() (alive bool) {
	us.mux.RLock()
	alive = us.Alive
	us.mux.RUnlock()
	return
}

func (us *UpstreamServer) SetAlive(alive bool) {
	us.mux.Lock()
	us.Alive = alive
	us.mux.Unlock()
}

// 轮询
func roundRobin(w http.ResponseWriter, r *http.Request) {

	sibling, err := serverPool.GetNextSibling()
	if err != nil {
		http.Error(w, "there is not alive upstream server", http.StatusServiceUnavailable)
	}
	if sibling != nil {
		sibling.ReverseProxy.ServeHTTP(w, r)
		return
	}
}

func checkUpstreamServerAlive(u *url.URL) (alive bool) {
	timeout := time.Second * 2
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		log.Printf("%s is not alive", u)
		alive = false
	}
	_ = conn.Close()
	alive = true
	return
}

func getRetryFromRequestContext(r *http.Request) int {
	if retry, ok := r.Context().Value(Retry).(int); ok {
		return retry
	}
	return 0
}

func getAttemptsFromRequestContext(r *http.Request) int {
	if attempts, ok := r.Context().Value(Attempts).(int); ok {
		return attempts
	}
	return 1
}

var serverPool ServersPool

func main() {
	addr := ":8080"
	if err := http.ListenAndServe(addr, http.HandlerFunc(roundRobin)); err != nil {
		log.Fatal(err)
	}
}
