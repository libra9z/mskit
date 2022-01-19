package rpcx

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"sync"

	_const "github.com/libra9z/mskit/v4/const"
	"github.com/libra9z/mskit/v4/sd"
	"github.com/libra9z/utils"
	etcd "github.com/rpcxio/rpcx-etcd/client"
	nacos "github.com/rpcxio/rpcx-nacos/client"
	"github.com/smallnest/rpcx/client"
	"github.com/smallnest/rpcx/share"

	"github.com/go-kit/kit/endpoint"
)

// Client wraps a RPCx connection and provides a method that implements
// endpoint.Endpoint.
type Client struct {
	client      client.XClient
	failMode    client.FailMode
	selectMode  client.SelectMode
	serviceName string
	service     string
	method      string
	sdType      string
	sdAddress   string
	basePath    string
	poolsize    int
	rpcxReply   reflect.Type
	before      []ClientRequestFunc
	after       []ClientResponseFunc
	finalizer   []ClientFinalizerFunc
	Params      map[string]interface{}
}

var ClientPool map[string]*XClientPool
var lock *sync.Mutex = &sync.Mutex{}

func init() {
	ClientPool = make(map[string]*XClientPool)
}

// NewClient constructs a usable Client for a single remote endpoint.
// Pass an zero-value protobuf message of the RPC response type as
// the rpcxReply argument.
func NewClient(
	rpcxReply interface{},
	options ...ClientOption,
) *Client {

	c := &Client{
		rpcxReply: reflect.TypeOf(
			reflect.Indirect(
				reflect.ValueOf(rpcxReply),
			).Interface(),
		),
		before: []ClientRequestFunc{},
		after:  []ClientResponseFunc{},
	}
	for _, option := range options {
		option(c)
	}

	if c.poolsize <= 0 {
		c.poolsize = 100
	}
	return c
}

// NewClient constructs a usable Client for a single remote endpoint.
// Pass an zero-value protobuf message of the RPC response type as
// the rpcxReply argument.
func NewClientPool(size int, sdtype, sdaddr, basepath, serviceName string, failMode client.FailMode, selectMode client.SelectMode, params map[string]interface{}) *XClientPool {

	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("error = %v\n", e)
			return
		}
	}()

	var cs client.ServiceDiscovery
	var err error
	switch sdtype {
	case "consul":
		ss := strings.Split(sdaddr, _const.ADDR_SPLIT_STRING)
		cs, err = client.NewConsulDiscovery(basepath, serviceName, ss, nil)
	case "redis":
		ss := strings.Split(sdaddr, _const.ADDR_SPLIT_STRING)
		cs, err = client.NewRedisDiscovery(basepath, serviceName, ss, nil)
	case "zookeeper":
		ss := strings.Split(sdaddr, _const.ADDR_SPLIT_STRING)
		cs, err = client.NewZookeeperDiscovery(basepath, serviceName, ss, nil)
	case "nacos":
		clientConfig := sd.GetClientConfig(params)
		sc := sd.GetServerConfig(sdaddr, params)
		clustername := utils.ConvertToString(params["cluster_name"])
		groupname := utils.ConvertToString(params["group_name"])
		cs, err = nacos.NewNacosDiscovery(serviceName, clustername, groupname,clientConfig, sc)
	case "etcd3":
		ss := strings.Split(sdaddr, _const.ADDR_SPLIT_STRING)
		cs, err = etcd.NewEtcdV3Discovery(basepath, serviceName, ss, true, nil)
	}
	if err != nil {
		fmt.Errorf("cannot discovery service: %v", err)
		return nil
	}
	xc := NewXClientPool(size, serviceName, failMode, selectMode, cs, client.DefaultOption)

	ClientPool[serviceName] = xc
	return xc
}

func GetRpcClientPool(serviceName string) *XClientPool {
	return ClientPool[serviceName]
}

// ClientOption sets an optional parameter for clients.
type ClientOption func(*Client)

// ClientBefore sets the RequestFuncs that are applied to the outgoing RPCx
// request before it's invoked.
func ClientBefore(before ...ClientRequestFunc) ClientOption {
	return func(c *Client) { c.before = append(c.before, before...) }
}

// ClientAfter sets the ClientResponseFuncs that are applied to the incoming
// RPCx response prior to it being decoded. This is useful for obtaining
// response metadata and adding onto the context prior to decoding.
func ClientAfter(after ...ClientResponseFunc) ClientOption {
	return func(c *Client) { c.after = append(c.after, after...) }
}

// ClientFinalizer is executed at the end of every RPCx request.
// By default, no finalizer is registered.
func ClientFinalizer(f ...ClientFinalizerFunc) ClientOption {
	return func(s *Client) { s.finalizer = append(s.finalizer, f...) }
}

// Endpoint returns a usable endpoint that will invoke the RPCx specified by the
// client.
func (c Client) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		if c.finalizer != nil {
			defer func() {
				for _, f := range c.finalizer {
					f(ctx, err)
				}
			}()
		}

		md := make(map[string]string)
		for _, f := range c.before {
			ctx = f(ctx, &md)
		}

		req := request.(*RpcRequest)
		rpcxReply := reflect.New(c.rpcxReply).Interface()

		ctx = context.WithValue(ctx, share.ReqMetaDataKey, map[string]string{"method": c.method})
		ctx = context.WithValue(ctx, share.ResMetaDataKey, make(map[string]string))

		if err = c.client.Call(ctx, c.service, req, rpcxReply); err != nil {
			return nil, err
		}

		var header, trailer map[string]string
		for _, f := range c.after {
			ctx = f(ctx, header, trailer)
		}

		return rpcxReply, nil
	}
}

func (c *Client) Close() error {
	pc := ClientPool[c.serviceName]
	pc.Close()

	return nil
}

func (c *Client) GetClientPool() *XClientPool {

	if pc, ok := ClientPool[c.serviceName]; ok {
		return pc
	} else {
		lock.Lock()
		defer lock.Unlock()
		if ClientPool[c.serviceName] == nil {
			ClientPool[c.serviceName] = NewClientPool(c.poolsize, c.sdType, c.sdAddress, c.basePath, c.serviceName, c.failMode, c.selectMode, c.Params)
		}
	}

	return ClientPool[c.serviceName]
}

// ClientFinalizerFunc can be used to perform work at the end of a client RPCx
// request, after the response is returned. The principal
// intended use is for error logging. Additional response parameters are
// provided in the context under keys with the ContextKeyResponse prefix.
// Note: err may be nil. There maybe also no additional response parameters depending on
// when an error occurs.
type ClientFinalizerFunc func(ctx context.Context, err error)

func BasePathOption(basepath string) ClientOption {
	return func(c *Client) { c.basePath = basepath }
}
func SdTypeOption(sdtype string) ClientOption {
	return func(c *Client) { c.sdType = sdtype }
}
func SdAddressOption(sdaddress string) ClientOption {
	return func(c *Client) { c.sdAddress = sdaddress }
}
func FailModeOption(failmode client.FailMode) ClientOption {
	return func(c *Client) { c.failMode = failmode }
}
func SelectModeOption(sel client.SelectMode) ClientOption {
	return func(c *Client) { c.selectMode = sel }
}
func MethodOption(method string) ClientOption {
	return func(c *Client) { c.method = method }
}
func ServiceOption(service string) ClientOption {
	return func(c *Client) { c.service = service }
}
func ServiceNameOption(svrname string) ClientOption {
	return func(c *Client) { c.serviceName = svrname }
}

func PoolSizeOption(poolsize int) ClientOption {
	return func(c *Client) { c.poolsize = poolsize }
}

func ParamsOption(params map[string]interface{}) ClientOption {
	return func(c *Client) { c.Params = params }
}
