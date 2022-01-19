package rpcx

import (
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/protocol"
	"sync"
	"sync/atomic"
)
// XClientPool is a xclient pool with fixed size.
// It uses roundrobin algorithm to call its xclients.
// All xclients share the same configurations such as ServiceDiscovery and serverMessageChan.
type XClientPool struct {
	count    uint64
	index    uint64
	xclients []client.XClient
	mu       sync.RWMutex

	servicePath string
	failMode    client.FailMode
	selectMode  client.SelectMode
	discovery   client.ServiceDiscovery
	option      client.Option
	auth        string

	serverMessageChan chan<- *protocol.Message
}

// NewXClientPool creates a fixed size XClient pool.
func NewXClientPool(count int, servicePath string, failMode client.FailMode, selectMode client.SelectMode, discovery client.ServiceDiscovery, option client.Option) *XClientPool {
	pool := &XClientPool{
		count:       uint64(count),
		xclients:    make([]client.XClient, count),
		servicePath: servicePath,
		failMode:    failMode,
		selectMode:  selectMode,
		discovery:   discovery,
		option:      option,
	}

	for i := 0; i < count; i++ {
		xclient := client.NewXClient(servicePath, failMode, selectMode, discovery, option)

		//p := &client.OpenTracingPlugin{}
		pc := client.NewPluginContainer()
		//pc.Add(p)
		xclient.SetPlugins(pc)
		pool.xclients[i] = xclient
	}
	return pool
}

// NewBidirectionalXClientPool creates a BidirectionalXClient pool with fixed size.
func NewBidirectionalXClientPool(count int, servicePath string, failMode client.FailMode, selectMode client.SelectMode, discovery client.ServiceDiscovery, option client.Option, serverMessageChan chan<- *protocol.Message) *XClientPool {
	pool := &XClientPool{
		count:             uint64(count),
		xclients:          make([]client.XClient, count),
		servicePath:       servicePath,
		failMode:          failMode,
		selectMode:        selectMode,
		discovery:         discovery,
		option:            option,
		serverMessageChan: serverMessageChan,
	}

	for i := 0; i < count; i++ {
		xclient := client.NewBidirectionalXClient(servicePath, failMode, selectMode, discovery, option, serverMessageChan)
		//p := &client.OpenTracingPlugin{}
		pc := client.NewPluginContainer()
		//pc.Add(p)
		xclient.SetPlugins(pc)
		pool.xclients[i] = xclient
	}
	return pool
}

// Auth sets s token for Authentication.
func (c *XClientPool) Auth(auth string) {
	c.auth = auth
	c.mu.RLock()
	for _, v := range c.xclients {
		v.Auth(auth)
	}
	c.mu.RUnlock()
}

// Get returns a xclient.
// It does not remove this xclient from its cache so you don't need to put it back.
// Don't close this xclient because maybe other goroutines are using this xclient.
func (p *XClientPool) Get() client.XClient {
	i := atomic.AddUint64(&p.index, 1)
	picked := int(i % p.count)
	return p.xclients[picked]
}

// Close this pool.
// Please make sure it won't be used any more.
func (p *XClientPool) Close() {
	for _, c := range p.xclients {
		c.Close()
	}
	p.xclients = nil
}
