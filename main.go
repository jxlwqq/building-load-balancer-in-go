package main

import (
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type UpstreamServer struct {
	URL          *url.URL
	Alive        bool
	mux          sync.RWMutex
	ReverseProxy httputil.ReverseProxy // 反向代理
}

type ServerPool struct {
	upstreamServers []*UpstreamServer
	current         uint64
}

func (s *ServerPool) GetServerAmount() int {
	return len(s.upstreamServers)
}

func (s *ServerPool) GetNextIndex() int {
	amount := s.GetServerAmount()
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(amount))
}

func (s *ServerPool) GetNextSibling() (*UpstreamServer, error) {
	amount := s.GetServerAmount()
	nextIndex := s.GetNextIndex()
	loops := amount + nextIndex
	for i := nextIndex; i < loops; i++ {
		index := i % amount
		if s.upstreamServers[index].IsAlive() {
			if i != nextIndex {
				atomic.StoreUint64(&s.current, uint64(index))
			}
			return s.upstreamServers[index], nil
		}
	}
	return nil, errors.New("there is not alive upstream server")
}

func (b *UpstreamServer) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

func (b *UpstreamServer) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
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

var serverPool ServerPool

func main() {
	addr := ":8080"
	if err := http.ListenAndServe(addr, http.HandlerFunc(roundRobin)); err != nil {
		log.Fatal(err)
	}
}
