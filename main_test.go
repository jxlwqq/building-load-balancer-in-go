package main

import (
	"net/http/httputil"
	"sync"
	"testing"
)

func TestGetServerAmount(t *testing.T) {
	us1 := UpstreamServer{
		URL:          nil,
		Alive:        false,
		mux:          sync.RWMutex{},
		ReverseProxy: httputil.ReverseProxy{},
	}

	us2 := UpstreamServer{
		URL:          nil,
		Alive:        false,
		mux:          sync.RWMutex{},
		ReverseProxy: httputil.ReverseProxy{},
	}
	sp := ServersPool{
		upstreamServers: nil,
		current:         0,
	}

	sp.upstreamServers = append(sp.upstreamServers, &us1, &us2)

	if sp.GetServerAmount() != 2 {
		t.Errorf("length of server pool is not right.")
	}
}
