package mskit

import (
	"time"

	"github.com/smallnest/rpcx"
	"github.com/smallnest/rpcx/plugin"
	"context"
	"fmt"
	"encoding/json"
	"errors"

)

const (
	JSONRPC_ERR_METHOD_NOT_FOUND = 32601
)


type Method func(interface{})(interface{},error)
type RpcServer struct {
	Server 			*rpcx.Server

	Network			string
	ServiceAddr 	string

	Methods			map[string]Method
}

var defautlServer *RpcServer

type RpcRequest struct {
	Req		 		string
}

type RpcResponse struct {
	Ret 			string
}


type RpcService interface {
	Services(ctx context.Context,req *RpcRequest,ret *RpcResponse) error
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
func InitRpcServerWithConsul(network,serviceAddr string,consulAddr string) {

	defautlServer = NewRpcServerWithConsul(network,serviceAddr,consulAddr)
	if defautlServer == nil {
		fmt.Printf("cannot initial rpc server.\n")
	}
}

func RpcRegisterService(servName RpcServiceName,service RpcService,metadata string) {
	if defautlServer != nil && service != nil {
		defautlServer.RegisterService(servName,service,metadata)
	}
}

func RpcRegisterDefaultService(servName RpcServiceName,service RpcService) {
	if defautlServer != nil {
		defautlServer.RegisterDefaultService(servName,service)
	}else{
		fmt.Printf("register default services failed.\n")
	}


}

func RpcRegisterDefaultMethod(methodName string,m Method) {

	if defautlServer != nil {
		defautlServer.RegisterMethod(methodName,m)
	}else{
		fmt.Printf("register default method failed.\n")
	}


}

func RpcGetMethodByName(name string) Method {

	if defautlServer != nil {
		return defautlServer.GetMethodByName(name)
	}

	return nil
}

func RpcServe(){

	if defautlServer != nil {
		defautlServer.Serve()
	}else{
		fmt.Printf("cannot start Rpcx server,default server is nil.\n")
	}
}


func NewRpcServerWithConsul(network,serviceAddr string,consulAddr string) *RpcServer {

	s := new(RpcServer)

	s.Server = rpcx.NewServer()

	if network == "" {
		network = "tcp"
	}

	s.Network = network
	s.ServiceAddr = serviceAddr
	s.Methods = make(map[string]Method)

	p :=  &plugin.ConsulRegisterPlugin{
		ServiceAddress: network +"@" + serviceAddr,
		ConsulAddress:  consulAddr,
		UpdateInterval: time.Second,
	}

	p.Start()
	s.Server.PluginContainer.Add(p)

	return s
}

func ( s *RpcServer ) RegisterService(servName RpcServiceName,service RpcService,metadata string) {
	if service != nil {
		s.Server.RegisterName(servName.GetServiceName(),service,metadata)
	}
}

func ( s *RpcServer ) RegisterDefaultService(servName RpcServiceName,service RpcService) {

	if service != nil {
		s.Server.RegisterName(servName.GetServiceName(),service)
	}
}

func ( s *RpcServer )Serve() error {

	fmt.Printf("rpcx server running on : %s\n",s.ServiceAddr)
	err :=s.Server.Serve(s.Network,s.ServiceAddr)

	if err != nil {
		fmt.Printf("cannot run rpcx server: %v\n",err)
		return err
	}
	return nil
}

func ( s *RpcServer ) RegisterMethod(methodName string,m Method) {

	if methodName == "" {
		return
	}
	s.Methods[methodName] = m
}

func ( s *RpcServer ) GetMethodByName(name string) Method {

	if name == "" {
		return nil
	}

	if 	m ,ok := s.Methods[name];ok {
		return m
	}

	return nil
}


type DefaultJSONRpc struct {}

func ( jr *DefaultJSONRpc ) Services( ctx context.Context,req *RpcRequest,ret *RpcResponse ) error {

	var err error
	if req == nil || ret == nil {
		err = errors.New("input parameter is nil")
		return err
	}

	if req.Req == "" {
		return errors.New("json-rpc request is empty.")
	}

	var vs map[string]interface{}
	err = json.Unmarshal([]byte(req.Req),&vs)

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
		fmt.Printf("call method : %s\n",method)
		if method != "foobar" {
			if vs["params"] != nil {
				params := vs["params"].(interface{})
				if params != nil {
					function := RpcGetMethodByName(method)
					if function != nil {
						result ,err = function(params)
						fmt.Printf("result=%v\n",result)
					}
				}
			}
		}else{
			em["code"] = JSONRPC_ERR_METHOD_NOT_FOUND
			em["message"] = "该方法不存在或者无效"
			err = errors.New("method not found")
		}
	}


	var rm map[string]interface{}

	rm = make(map[string]interface{})

	rm["jsonrpc"] = "2.0"
	rm["result"] = result
	if err !=nil {
		rm["error"] = em
	}
	if vs["id"] != nil {
		rm["id"] = vs["id"]
	}

	r,err := json.Marshal(&rm)

	if err != nil {
		return errors.New("cannot marshal return json.")
	}

	ret.Ret = string(r)

	return nil
}

