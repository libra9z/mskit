package rpcx

import (
	"context"
	"fmt"
	"github.com/openzipkin/zipkin-go"
	"github.com/smallnest/rpcx/client"
	"strings"
)

func RpcCallWithConsul(basepath, consuladdr, serviceName, methodName string, selectMode int, req *RpcRequest, ret *RpcResponse) error {

	ss := strings.Split(consuladdr, ";")

	d := client.NewConsulDiscovery(basepath, serviceName, ss, nil)

	if selectMode < 0 {
		selectMode = int(client.RandomSelect)
	}

	client := client.NewXClient(serviceName, client.Failtry, client.SelectMode(selectMode), d, client.DefaultOption)
	defer client.Close()

	serviceMethod := methodName
	err := client.Call(context.Background(), serviceMethod, req, ret)
	if err != nil {
		fmt.Printf("error for %s: %v \n", serviceMethod, err)
	} else {
		fmt.Printf("%s: call success.\n", serviceMethod)
	}

	return nil
}

func RpcxCall(ctx context.Context,zktracer *zipkin.Tracer,
			sdtype,sdaddr string,
			basepath, serviceName, methodName string,
			failMode client.FailMode,selectMode client.SelectMode,
			req *RpcRequest) (ret *RpcResponse,err error) {


	var options []ClientOption

	options = append(options,BasePathOption(basepath))
	options = append(options,SdAddressOption(sdaddr))
	options = append(options,SdTypeOption(sdtype))
	options = append(options,FailModeOption(failMode))
	options = append(options,SelectModeOption(selectMode))
	options = append(options,MethodOption(methodName))
	options = append(options,ServiceNameOption(serviceName))

	options = append(options,RpcxClientTrace(zktracer))

	resp := RpcResponse{}
	c := NewClient(&resp,options...)
	defer c.Close()

	r,err := c.Endpoint()(ctx,req)
	if r != nil {
		resp  = r.(RpcResponse)
	}
	return &resp,err
}