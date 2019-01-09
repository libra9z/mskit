//

package rpcx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/smallnest/rpcx/server"
	"github.com/smallnest/rpcx/serverplugin"
	"strings"
	"time"
)

const (
	JSONRPC_ERR_METHOD_NOT_FOUND = 32601
)

type Method func(context.Context,int64, int64, string, interface{}) (interface{}, error)
type RpcServer struct {
	Server *server.Server

	Network     string
	ServiceAddr string

	Methods map[string]Method
}

var defautlServer *RpcServer

type RpcRequest struct {
	Appid  int64
	SiteId int64
	Id     int64 //修改某一条记录时的记录标识
	Token  string
	Req    string
}

type RpcResponse struct {
	Ret string
}

type RpcService interface {
	Services(ctx context.Context, req *RpcRequest, ret *RpcResponse) error
}

type RpcServiceName interface {
	SetServiceName(string)
	GetServiceName() string
}

/*
	参数network的定义如下：
	kcp：
	reuseport：
	quic
	default   tcp
*/
func InitRpcServerWithConsul(network, serviceAddr string, consulAddr string, basepath string) {

	defautlServer = NewRpcServerWithConsul(network, serviceAddr, consulAddr, basepath)
	if defautlServer == nil {
		fmt.Printf("cannot initial rpc server.\n")
	}
}

func RpcRegisterService(servName RpcServiceName, service RpcService, metadata string) {
	if defautlServer != nil && service != nil {
		defautlServer.RegisterService(servName, service, metadata)
	}
}

func RpcRegisterDefaultService(servName RpcServiceName, service RpcService, meta string) {
	if defautlServer != nil {
		defautlServer.RegisterDefaultService(servName, service, meta)
	} else {
		fmt.Printf("register default services failed.\n")
	}

}

func RpcRegisterDefaultMethod(methodName string, m Method) {

	if defautlServer != nil {
		defautlServer.RegisterMethod(methodName, m)
	} else {
		fmt.Printf("register default method failed.\n")
	}

}

func RpcGetMethodByName(name string) Method {

	if defautlServer != nil {
		return defautlServer.GetMethodByName(name)
	}

	return nil
}

func RpcServe() {

	if defautlServer != nil {
		defautlServer.Serve()
	} else {
		fmt.Printf("cannot start Rpcx server,default server is nil.\n")
	}
}

func NewRpcServerWithConsul(network, serviceAddr string, consulAddr string, basepath string) *RpcServer {

	s := new(RpcServer)

	s.Server = server.NewServer()

	if network == "" {
		network = "tcp"
	}

	fmt.Println("开始向consul注册服务...")

	cs := strings.Split(consulAddr, ",")

	s.Network = network
	s.ServiceAddr = serviceAddr
	s.Methods = make(map[string]Method)

	p := &serverplugin.ConsulRegisterPlugin{
		ServiceAddress: network + "@" + serviceAddr,
		ConsulServers:  cs,
		BasePath:       basepath,
		Metrics:        metrics.NewRegistry(),
		UpdateInterval: time.Minute,
	}

	err := p.Start()
	if err != nil {
		fmt.Errorf("不能注册服务：%v\n", err)
	}
	s.Server.Plugins.Add(p)

	return s
}

func (s *RpcServer) RegisterService(servName RpcServiceName, service RpcService, metadata string) {
	if service != nil {
		err := s.Server.RegisterName(servName.GetServiceName(), service, metadata)
		//s.Server.Register(service,metadata)
		if err != nil {
			fmt.Printf("不能注册服务:%v\n", err)
		}
	}
}

func (s *RpcServer) RegisterDefaultService(servName RpcServiceName, service RpcService, meta string) {

	if service != nil {
		fmt.Printf("注册服务：%s\n", servName.GetServiceName())
		err := s.Server.RegisterName(servName.GetServiceName(), service, meta)
		//err := s.Server.Register(service,meta)
		if err != nil {
			fmt.Errorf("不能注册服务:%v\n", err)
		}
	} else {
		fmt.Errorf("不能注册服务，service为nil")
	}
}

func (s *RpcServer) Serve() error {

	fmt.Printf("rpcx server running on : %s\n", s.ServiceAddr)
	err := s.Server.Serve(s.Network, s.ServiceAddr)

	if err != nil {
		fmt.Printf("cannot run rpcx server: %v\n", err)
		return err
	}
	return nil
}

func (s *RpcServer) RegisterMethod(methodName string, m Method) {

	if methodName == "" {
		return
	}
	s.Methods[methodName] = m
}

func (s *RpcServer) GetMethodByName(name string) Method {

	if name == "" {
		return nil
	}

	if m, ok := s.Methods[name]; ok {
		return m
	}

	return nil
}

type JSONRpc struct{}

func (jr *JSONRpc) Services(ctx context.Context, req *RpcRequest, ret *RpcResponse) error {

	var err error
	if req == nil || ret == nil {
		err = errors.New("input parameter is nil")
		return err
	}

	if req.Req == "" {
		return errors.New("json-rpc request is empty.")
	}

	var vs map[string]interface{}
	err = json.Unmarshal([]byte(req.Req), &vs)

	if err != nil {
		return err
	}

	if vs["jsonrpc"] != nil {
		v := vs["jsonrpc"].(string)
		if v != "2.0" {
			return errors.New("unsupport json-rpc version.")
		}
	}

	var result interface{}
	em := make(map[string]interface{})
	if vs["method"] != nil {
		method := vs["method"].(string)
		fmt.Printf("call method : %s\n", method)
		if method != "" {
			function := RpcGetMethodByName(method)
			if function != nil {
				result, err = function(ctx,req.Appid, req.SiteId, req.Token, vs["params"])
				fmt.Printf("result=%v\n", result)
			} else {
				fmt.Errorf("没有找对对应的方法。")
			}
		} else {
			em["code"] = JSONRPC_ERR_METHOD_NOT_FOUND
			em["message"] = "该方法不存在或者无效"
			err = errors.New("method not found")
		}
	}

	var rm map[string]interface{}

	rm = make(map[string]interface{})

	rm["jsonrpc"] = "2.0"
	rm["result"] = result
	if err != nil {
		rm["error"] = em
	}
	if vs["id"] != nil {
		rm["id"] = vs["id"]
	}

	r, err := json.Marshal(&rm)

	if err != nil {
		return errors.New("cannot marshal return json.")
	}

	ret.Ret = string(r)

	return nil
}
