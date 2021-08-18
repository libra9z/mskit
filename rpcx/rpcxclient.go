package rpcx

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/libra9z/mskit/trace"
	"github.com/smallnest/rpcx/client"
)

func RpcCallWithConsul(basepath, consuladdr, serviceName, methodName string, selectMode int, req *RpcRequest, ret *RpcResponse) error {

	ss := strings.Split(consuladdr, ";")

	d, err := client.NewConsulDiscovery(basepath, serviceName, ss, nil)
	if err != nil {
		return err
	}

	if selectMode < 0 {
		selectMode = int(client.RandomSelect)
	}

	c := client.NewXClient(serviceName, client.Failtry, client.SelectMode(selectMode), d, client.DefaultOption)
	defer c.Close()
	p := &client.OpenTracingPlugin{}
	pc := client.NewPluginContainer()
	pc.Add(p)
	c.SetPlugins(pc)

	serviceMethod := methodName
	err = c.Call(context.Background(), serviceMethod, req, ret)
	if err != nil {
		fmt.Printf("error for %s: %v \n", serviceMethod, err)
	} else {
		fmt.Printf("%s: call success.\n", serviceMethod)
	}

	return nil
}

func RpcxCall(ctx context.Context, tracer trace.Tracer,
	sdtype, sdaddr string,
	basepath, serviceName, service, methodName string,
	failMode client.FailMode, selectMode client.SelectMode,
	req *RpcRequest, vv ...interface{}) (ret *RpcResponse, err error) {

	var options []ClientOption

	options = append(options, BasePathOption(basepath))
	options = append(options, SdAddressOption(sdaddr))
	options = append(options, SdTypeOption(sdtype))
	options = append(options, FailModeOption(failMode))
	options = append(options, SelectModeOption(selectMode))
	options = append(options, MethodOption(methodName))
	options = append(options, ServiceOption(service))
	options = append(options, ServiceNameOption(serviceName))

	if len(vv) > 0 {
		t := reflect.ValueOf(vv[0])
		if t.Kind() == reflect.Map {
			options = append(options, ParamsOption(vv[0].(map[string]interface{})))
		}
	}

	resp := RpcResponse{}
	c := NewClient(&resp, options...)
	pc := c.GetClientPool()
	c.client = pc.Get().(client.XClient)

	r, err := c.Endpoint()(ctx, req)
	if r != nil {
		return r.(*RpcResponse), err
	}
	return nil, err
}
