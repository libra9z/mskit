package mskit

import (
	"context"
	"time"
	"fmt"
	"github.com/smallnest/rpcx"
	"github.com/smallnest/rpcx/clientselector"
)


func RpcCallWithConsul(consuladdr,serviceName,methodName string,selectMode int,req *RpcRequest,ret *RpcResponse) error {

	s := clientselector.NewConsulClientSelector(consuladdr, serviceName, 2*time.Minute, rpcx.RoundRobin, time.Minute)
	client := rpcx.NewClient(s)

	serviceMethod := serviceName+"." + methodName
	err := client.Call(context.Background(), serviceMethod, req, ret)
	if err != nil {
		fmt.Printf("error for %s: %v \n", serviceMethod, err)
	} else {
		fmt.Printf("%s: call success.\n", serviceMethod )
	}

	client.Close()

	return err
}