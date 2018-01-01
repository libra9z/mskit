package mskit

import (
	"context"
	"github.com/smallnest/rpcx/client"
	"fmt"
)


func RpcCallWithConsul(basepath,consuladdr,serviceName,methodName string,selectMode int,req *RpcRequest,ret *RpcResponse) error {


	d := client.NewConsulDiscovery(basepath, serviceName, []string{consuladdr}, nil)
	client := client.NewXClient(serviceName, client.Failtry, client.RandomSelect, d, client.DefaultOption)
	defer client.Close()

	serviceMethod :=  methodName
	err := client.Call(context.Background(), serviceMethod, req, ret)
	if err != nil {
		fmt.Printf("error for %s: %v \n", serviceMethod, err)
	} else {
		fmt.Printf("%s: call success.\n", serviceMethod )
	}


	return nil
}