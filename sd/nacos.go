package sd

import (
	"bytes"
	"encoding/json"
	"fmt"
	_const "github.com/libra9z/mskit/v4/const"
	"github.com/libra9z/mskit/v4/grace"
	"github.com/libra9z/mskit/v4/log"
	"github.com/libra9z/utils"
	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"net"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
)

type nacosRegister struct {
	servers  string
	token    string
	prefix   string
	name     string
	callback ServiceCallback
	params   map[string]interface{}
	iclient  naming_client.INamingClient
	addr     string //listen on address and port
}

var _ Registar = (*nacosRegister)(nil)

func NewNacosRegistar(name string, prefix string, addr, nacos, token string, callback ServiceCallback, params map[string]interface{}) (Registar, error) {
	n := &nacosRegister{
		name:     name,
		prefix:   prefix,
		servers:  nacos,
		callback: callback,
		token:    token,
		params:   params,
		addr:     addr,
	}
	return n, nil
}

func (n *nacosRegister) Register(app *grace.MicroService, schema string, address string, params map[string]interface{}, callbacks ...ServiceCallback) {

	if n.name == "" {
		log.Mslog.Error("name empty")
	}
	if n.prefix == "" {
		log.Mslog.Error("prefix empty")
	}

	//nacos address split
	cs := strings.Split(n.servers, _const.ADDR_SPLIT_STRING)

	if len(cs) <= 0 {
		log.Mslog.Error("no nacos address config")
		return
	}

	n.params = params
	if len(callbacks) > 0 {
		n.callback = callbacks[0]
	}
	n.addr = address

	//nacos = cs[0]

	var interval, timeout string
	if n.params != nil {
		if n.params["interval"] != nil {
			interval = utils.ConvertToString(n.params["interval"])
		}
		if n.params["timeout"] != nil {
			timeout = utils.ConvertToString(n.params["timeout"])
		}
	}

	if interval == "" {
		interval = "30s"
	}

	if timeout == "" {
		timeout = "2s"
	}

	prefixes := strings.Split(n.prefix, ",")
	host, portstr, err := net.SplitHostPort(address)
	if err != nil {
		log.Mslog.Error(err)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Mslog.Error(err)
	}
	go func() {
		log.Mslog.Info("Listening on %s serving %s", address, n.prefix)
		if err := n.callback(app, n.params); err != nil {
			log.Mslog.Error(err)
		}
	}()

	var tags []string
	for _, p := range prefixes {
		tags = append(tags, "urlprefix-"+p)
	}

	clientConfig := GetClientConfig(n.params)
	serverConfigs := GetServerConfig(n.servers, n.params)

	clusterName := ""
	weight := 0.0
	if n.params != nil && n.params["clustername"] != nil {
		clusterName = utils.ConvertToString(n.params["clustername"])
	}
	if n.params != nil && n.params["weight"] != nil {
		weight = utils.Convert2Float64(n.params["weight"])
	}

	namingClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	n.iclient = namingClient
	serviceID := n.name + "-" + n.addr
	success, _ := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceID,
		Weight:      weight,
		ClusterName: clusterName,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})
	if !success {
		log.Mslog.Error("不能注册服务")
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

}

func (n *nacosRegister) Deregister() {
	serviceID := n.name + "-" + n.addr
	host, portstr, err := net.SplitHostPort(n.addr)
	if err != nil {
		log.Mslog.Error(err)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		log.Mslog.Error(err)
	}
	success, _ := n.iclient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceID,
		Ephemeral:   true,
	})

	log.Mslog.Info("Deregistered service %q in consul %v", n.name, success)
}

func (n *nacosRegister) RegisterFromMemory(app *grace.MicroService, schema string, buf *bytes.Buffer, exparams map[string]interface{}, callbacks ...ServiceCallback) {

	if buf == nil {
		log.Mslog.Error("内存中没有默认配置。")
		return
	}
	var data map[string]interface{}
	var params map[string]interface{}

	body := buf.Bytes()

	err := json.Unmarshal(body, &data)

	if err != nil {
		log.Mslog.Error("json:" + err.Error())
		return
	}

	//nacos address split
	cs := strings.Split(n.servers, _const.ADDR_SPLIT_STRING)

	if len(cs) <= 0 {
		log.Mslog.Error("no consul address config")
		return
	}

	var p interface{}
	key := schema + "_"
	if data[key+"service"] != nil {
		p = data[key+"service"].(interface{})
	} else if data[key+"services"] != nil {
		p = data[key+"services"].(interface{})
	}

	cps := make(map[string]interface{})

	if data["TLSConfig"] != nil {
		vs := data["TLSConfig"].(map[string]interface{})
		cps["certfile"] = vs["certfile"]
		cps["keyfile"] = vs["keyfile"]
		cps["trustfile"] = vs["trustfile"]
	}

	var de bool = false
	if data["docker_enable"] != nil {
		cps["docker_enable"] = data["docker_enable"]
		de = data["docker_enable"].(bool)
	}
	switch schema {
	case "http", "https":
		t := reflect.ValueOf(p)
		switch t.Kind() {
		case reflect.Slice:
			ps := p.([]interface{})
			if len(ps) != len(callbacks) {
				log.Mslog.Error("服务数量与回调函数数量不匹配。")
				return
			}
			for i, vs := range ps {
				v := vs.(map[string]interface{})
				go nacosRegisterService(app, schema, n.servers, n.token, v, callbacks[i], cps)
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, os.Kill)
			<-quit

			//select {}
		case reflect.Map:
			if len(callbacks) < 1 {
				log.Mslog.Error("没有指定回调函数。")
				return
			}
			params = p.(map[string]interface{})
			nacosRegisterService(app, schema, n.servers, n.token, params, callbacks[0], cps)
		}
	case "rpcx":
		if data["rpcx"] != nil {
			vs := data["rpcx"].([]interface{})
			for i, vv := range vs {
				v := vv.(map[string]interface{})
				m := make(map[string]interface{})
				if v["address"] != nil {
					m["host"] = utils.ConvertToString(v["address"])
				}
				if v["port"] != nil {
					m["port"] = utils.ConvertToString(v["port"])
				}
				if v["consul_address"] != nil {
					m["consul_address"] = v["consul_address"]
				} else {
					if v["sd_address"] != nil {
						m["consul_address"] = v["sd_address"]
						m["sd_address"] = v["sd_address"]
						m["sd_type"] = v["sd_type"]
					}
				}
				if de {
					m["docker_enable"] = de
				}
				go callbacks[i](app, m)
			}
		}
	case "tcp":
		t := reflect.ValueOf(p)
		switch t.Kind() {
		case reflect.Slice:
			ps := p.([]interface{})
			if len(ps) != len(callbacks) {
				log.Mslog.Error("服务数量与回调函数数量不匹配。")
				return
			}
			for i, vs := range ps {
				v := vs.(map[string]interface{})
				go nacosRegisterService(app, schema, n.servers, n.token, v, callbacks[i], cps)
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, os.Kill)
			<-quit

			//select {}
		case reflect.Map:
			if len(callbacks) < 1 {
				log.Mslog.Error("没有指定回调函数。")
				return
			}
			params = p.(map[string]interface{})
			nacosRegisterService(app, schema, n.servers, n.token, params, callbacks[0], cps)
		}
	default:
		log.Mslog.Error("没有配置参数。")
		panic("没有配置参数")
	}

}

func (n *nacosRegister) RegisterWithConf(app *grace.MicroService, schema string, fname string, callbacks ...ServiceCallback) {
	if fname == "" {
		log.Mslog.Error("没有指定配置文件。\n")
		return
	}

	body := readFile(fname)

	buf := bytes.NewBuffer(body)

	n.RegisterFromMemory(app, schema, buf, nil, callbacks...)

}

func (n *nacosRegister) RegisterFile(app *grace.MicroService, schema string, fname string, callbacks ...ServiceCallback) {
	if fname == "" {
		log.Mslog.Error("没有指定配置文件。\n")
		return
	}

	body := readFile(fname)

	buf := bytes.NewBuffer(body)

	n.RegisterFromMemory(app, schema, buf, nil, callbacks...)

}

func GetClientConfig(params map[string]interface{}) constant.ClientConfig {
	var clientConfig constant.ClientConfig
	if params != nil {
		if params["timeout"] != nil {
			clientConfig.TimeoutMs = uint64(utils.Convert2Int64(params["timeout"]))
		} else {
			clientConfig.TimeoutMs = 10 * 1000
		}
		if params["listeninterval"] != nil {
			clientConfig.ListenInterval = uint64(utils.Convert2Int64(params["listeninterval"]))
		} else {
			clientConfig.ListenInterval = 30 * 1000
		}

		if params["beatinterval"] != nil {
			clientConfig.BeatInterval = utils.Convert2Int64(params["timeout"])
		} else {
			clientConfig.BeatInterval = 5 * 1000
		}
		if params["logdir"] != nil {
			clientConfig.LogDir = utils.ConvertToString(params["logdir"])
		} else {
			clientConfig.LogDir = "/nacos/logs"
		}
		if params["cachedir"] != nil {
			clientConfig.CacheDir = utils.ConvertToString(params["cachedir"])
		} else {
			clientConfig.CacheDir = "/nacos/cache"
		}

	} else {
		clientConfig = constant.ClientConfig{
			TimeoutMs:      10 * 1000,
			ListenInterval: 30 * 1000,
			BeatInterval:   5 * 1000,
			LogDir:         "/nacos/logs",
			CacheDir:       "/nacos/cache",
		}
	}

	return clientConfig
}

func GetServerConfig(nacos string, params map[string]interface{}) []constant.ServerConfig {
	var serverConfig []constant.ServerConfig

	ss := strings.Split(nacos, _const.ADDR_SPLIT_STRING)

	ContextPath := ""
	if params != nil && params["contextpath"] != nil {
		ContextPath = utils.ConvertToString(params["contextpath"])
	} else {
		ContextPath = "/nacos"
	}

	for _, v := range ss {
		var c constant.ServerConfig
		s2 := strings.Split(v, ":")
		c.IpAddr = s2[0]
		if len(s2) > 1 {
			c.Port = uint64(utils.Convert2Int(s2[1]))
		}
		c.ContextPath = ContextPath
		serverConfig = append(serverConfig, c)
	}

	return serverConfig
}

func nacosRegisterService(app *grace.MicroService, schema, nacos, token string, params map[string]interface{}, callback ServiceCallback, datas map[string]interface{}) {
	var name, prefix, host, addr string
	var tags []string

	var de bool = false
	if datas["docker_enable"] != nil {
		de = datas["docker_enable"].(bool)
	}

	if params["name"] != nil {
		name = utils.ConvertToString(params["name"])
	}
	if params["tags"] != nil {
		p := params["tags"].([]interface{})
		for _, v := range p {
			tags = append(tags, utils.ConvertToString(v))
		}
	}

	if params["address"] != nil {
		host = utils.Hostname2IPv4(utils.ConvertToString(params["address"]))
		datas["host"] = host
	}

	var port int
	if params["port"] != nil {
		port = utils.Convert2Int(params["port"])
		datas["port"] = port
	}

	if port == 0 {
		log.Mslog.Error("没有指定端口号。")
		return
	}

	prefix = strings.Join(tags, ",")

	go func(po int) {
		if de {
			datas["host"] = ""

		} else {
			datas["host"] = host
		}
		sp := fmt.Sprintf("Listening on %v:%d serving %s\n", datas["host"], po, prefix)
		log.Mslog.Info("%s", sp)
		if err := callback(app, datas); err != nil {
			log.Mslog.Error(err)
		}
	}(port)

	addr = host + ":" + utils.ConvertToString(port)

	var serviceID string

	if params["id"] != nil {
		serviceID = utils.ConvertToString(params["id"])
	}

	if serviceID == "" {
		serviceID = name + "-" + addr
	}

	clientConfig := GetClientConfig(params)
	serverConfigs := GetServerConfig(nacos, params)

	clusterName := ""
	weight := 0.0
	if params != nil && params["clustername"] != nil {
		clusterName = utils.ConvertToString(params["clustername"])
	}
	if params != nil && params["weight"] != nil {
		weight = utils.Convert2Float64(params["weight"])
	}

	namingClient, err := clients.CreateNamingClient(map[string]interface{}{
		"serverConfigs": serverConfigs,
		"clientConfig":  clientConfig,
	})
	if err != nil {
		log.Mslog.Error("不能获取nacos的nameClient：", err)
		return
	}
	grpname := ""
	if params != nil && params["group_name"] != nil {
		grpname = utils.ConvertToString(params["group_name"])
	}

	success, err := namingClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceID,
		Weight:      weight,
		ClusterName: clusterName,
		GroupName:   grpname,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
	})
	if !success {
		fmt.Printf("不能注册服务: %v\n", err.Error())
		return
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	success, _ = namingClient.DeregisterInstance(vo.DeregisterInstanceParam{
		Ip:          host,
		Port:        uint64(port),
		ServiceName: serviceID,
		Ephemeral:   true,
	})

	log.Mslog.Info("Deregistered service %q in consul", name)

}
