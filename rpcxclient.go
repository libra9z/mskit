package mskit

import (
	"context"
	"github.com/smallnest/rpcx/client"
	"fmt"
	"strings"
)


func RpcCallWithConsul(basepath,consuladdr,serviceName,methodName string,selectMode int,req *RpcRequest,ret *RpcResponse) error {


	ss := strings.Split(consuladdr,";")

	d := client.NewConsulDiscovery(basepath, serviceName, ss, nil)

	if selectMode < 0  {
		selectMode = int(client.RandomSelect)
	}

	client := client.NewXClient(serviceName, client.Failtry, client.SelectMode(selectMode), d, client.DefaultOption)
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