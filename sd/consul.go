package sd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/libra9z/log4go"
	_const "github.com/libra9z/mskit/v4/const"
	"github.com/libra9z/mskit/v4/grace"
	mslog "github.com/libra9z/mskit/v4/log"
	consulsd "github.com/libra9z/sd/consul"
	"github.com/libra9z/utils"
	"io/ioutil"
	"net"
	"os"
	"os/signal"
	"reflect"
	"strconv"
	"strings"
)

/**

{
  "service": {
    "id": "redis",
    "name": "redis",
    "tags": ["primary"],
    "address": "",
    "meta": {
      "meta": "for my service"
    },
    "tagged_addresses": {
      "lan": {
        "address": "192.168.0.55",
        "port": 8000,
      },
      "wan": {
        "address": "198.18.0.23",
        "port": 80
      }
    },
    "port": 8000,
    "enable_tag_override": false,
    "checks": [
      {
        "args": ["/usr/local/bin/check_redis.py"],
        "interval": "10s"
      }
    ],
    "kind": "connect-proxy",
    "proxy_destination": "redis", // Deprecated
    "proxy": {
      "destination_service_name": "redis",
      "destination_service_id": "redis1",
      "local_service_address": "127.0.0.1",
      "local_service_port": 9090,
      "config": {},
      "upstreams": [],
      "mesh_gateway": {
        "mode": "local"
      },
      "expose": {
        "checks": true,
        "paths": [
          {
            "path": "/healthz",
            "local_path_port": 8080,
            "listener_port": 21500,
            "protocol": "http2"
          }
       ]
      }
    },
    "connect": {
      "native": false,
      "sidecar_service": {}
      "proxy": {  // Deprecated
        "command": [],
        "config": {}
      }
    },
    "weights": {
      "passing": 5,
      "warning": 1
    },
    "token": "233b604b-b92e-48c8-a253-5f11514e4b50",
    "namespace": "foo"
  }
}

*/

var logger = mslog.Mslog

var _ Registar = (*consulRegister)(nil)

type consulRegister struct {
	servers  string
	token    string
	prefix   string
	name     string
	callback ServiceCallback
	params   map[string]interface{}
	reg      *consulsd.Registrar
	addr     string //listen on address and port
}

func NewConsulRegistar(name string, prefix string, addr, consul, token string, callback ServiceCallback, params map[string]interface{}) (Registar, error) {
	c := &consulRegister{
		name:     name,
		prefix:   prefix,
		servers:  consul,
		callback: callback,
		token:    token,
		params:   params,
		addr:     addr,
	}

	return c, nil
}
func (c *consulRegister) Register(app *grace.MicroService, schema string, address string, params map[string]interface{}, callbacks ...ServiceCallback) {

	if c.name == "" {
		mslog.Mslog.Critical("name empty")
	}
	if c.prefix == "" {
		mslog.Mslog.Critical("prefix empty")
	}

	//consul address split
	cs := strings.Split(c.servers, _const.ADDR_SPLIT_STRING)

	if len(cs) <= 0 {
		mslog.Mslog.Critical("no consul address config")
		return
	}
	c.params = params
	if len(callbacks) > 0 {
		c.callback = callbacks[0]
	}
	c.servers = cs[0]
	c.addr = address

	var interval, timeout string
	if c.params != nil {
		if c.params["interval"] != nil {
			interval = utils.ConvertToString(c.params["interval"])
		}
		if c.params["timeout"] != nil {
			timeout = utils.ConvertToString(c.params["timeout"])
		}
	}

	if interval == "" {
		interval = "30s"
	}

	if timeout == "" {
		timeout = "2s"
	}

	prefixes := strings.Split(c.prefix, ",")
	host, portstr, err := net.SplitHostPort(c.addr)
	if err != nil {
		mslog.Mslog.Critical(err)
	}
	port, err := strconv.Atoi(portstr)
	if err != nil {
		mslog.Mslog.Critical(err)
	}
	go func() {

		mslog.Mslog.Info("Listening on %s serving %s", c.addr, c.prefix)
		if err := c.callback(app, c.params); err != nil {
			mslog.Mslog.Critical(err)
		}
	}()

	var tags []string
	for _, p := range prefixes {
		tags = append(tags, "urlprefix-"+p)
	}

	serviceID := c.name + "-" + c.addr
	service := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    c.name,
		Port:    port,
		Address: host,
		Tags:    tags,
		Check: &api.AgentServiceCheck{
			CheckID:  "check-" + serviceID,
			HTTP:     schema + "://" + c.addr + c.prefix + "/health",
			Interval: interval,
			Timeout:  timeout,
		},
	}

	client := getConsulClient(c.servers, schema)

	c.reg = consulsd.NewRegistrar(client, service, log4go.NewDefaultLogger(log4go.FINEST))
	c.reg.Register()
}

func (c *consulRegister) RegisterFromMemory(app *grace.MicroService, schema string, reader *bytes.Buffer, exparams map[string]interface{}, callbacks ...ServiceCallback) {

	if reader == nil {
		mslog.Mslog.Critical("内存中没有默认配置。")
		return
	}
	body := reader.Bytes()

	var data map[string]interface{}
	var params map[string]interface{}

	err := json.Unmarshal(body, &data)

	if err != nil {
		mslog.Mslog.Critical("json:" + err.Error())
		return
	}

	//consul address split
	cs := strings.Split(c.servers, _const.ADDR_SPLIT_STRING)

	if len(cs) <= 0 {
		mslog.Mslog.Critical("no consul address config")
		return
	}

	c.servers = cs[0]

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
				mslog.Mslog.Critical("服务数量与回调函数数量不匹配。")
				return
			}
			for i, vs := range ps {
				v := vs.(map[string]interface{})
				go registerService(app, schema, c.servers, c.token, v, callbacks[i], cps)
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, os.Kill)
			<-quit

			//select {}
		case reflect.Map:
			if len(callbacks) < 1 {
				mslog.Mslog.Critical("没有指定回调函数。")
				return
			}
			params = p.(map[string]interface{})
			registerService(app, schema, c.servers, c.token, params, callbacks[0], cps)
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
				mslog.Mslog.Critical("服务数量与回调函数数量不匹配。")
				return
			}
			for i, vs := range ps {
				v := vs.(map[string]interface{})
				go registerService(app, schema, c.servers, c.token, v, callbacks[i], cps)
			}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt, os.Kill)
			<-quit

			//select {}
		case reflect.Map:
			if len(callbacks) < 1 {
				mslog.Mslog.Critical("没有指定回调函数。")
				return
			}
			params = p.(map[string]interface{})
			registerService(app, schema, c.servers, c.token, params, callbacks[0], cps)
		}
	default:
		mslog.Mslog.Critical("没有配置参数。")
		panic("没有配置参数")
	}
}

func (c *consulRegister) Deregister() {
	if c.reg != nil {
		c.reg.Deregister()
	}

}

func (c *consulRegister) RegisterWithConf(app *grace.MicroService, schema, fname string, callbacks ...ServiceCallback) {

	if fname == "" {
		mslog.Mslog.Critical("没有指定配置文件。\n")
		return
	}

	body := readFile(fname)

	buf := bytes.NewBuffer(body)

	c.RegisterFromMemory(app, schema, buf, nil, callbacks...)

}

func (c *consulRegister) RegisterFile(app *grace.MicroService, schema, fname string, callbacks ...ServiceCallback) {

	if fname == "" {
		mslog.Mslog.Critical("没有指定配置文件。\n")
		return
	}

	body := readFile(fname)

	buf := bytes.NewBuffer(body)

	c.RegisterFromMemory(app, schema, buf, nil, callbacks...)

}

func getConsulClient(addr, schema string) consulsd.Client {
	var client consulsd.Client
	{
		consulConfig := api.DefaultConfig()
		if addr != "" {
			consulConfig.Address = addr
		}
		if schema != "" {
			consulConfig.Scheme = schema
		} else {
			consulConfig.Scheme = "http"
		}
		consulClient, err := api.NewClient(consulConfig)
		if err != nil {
			mslog.Mslog.Critical(err)
			os.Exit(1)
		}
		client = consulsd.NewClient(consulClient)
	}

	return client
}

func readFile(path string) []byte {
	f, err := ioutil.ReadFile(path)
	if err != nil {
		mslog.Mslog.Critical(err)
	}
	return f
}

func registerService(app *grace.MicroService, schema, consul, token string, params map[string]interface{}, callback ServiceCallback, datas map[string]interface{}) {
	var name, prefix, host, addr string
	var tags []string

	var de bool = false
	if datas["docker_enable"] != nil {
		de = datas["docker_enable"].(bool)
	}

	//log.Print(params)

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
		mslog.Mslog.Critical("没有指定端口号。")
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
		mslog.Mslog.Info("%s", sp)

		if err := callback(app, datas); err != nil {
			mslog.Mslog.Critical(err)
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

	var checks []*api.AgentServiceCheck
	var check *api.AgentServiceCheck

	if params["checks"] != nil {
		vp := params["checks"].([]interface{})
		for _, v := range vp {
			p := v.(map[string]interface{})
			var c api.AgentServiceCheck
			if p["http"] != nil {
				c.HTTP = utils.ConvertToString(p["http"])
			}
			if p["interval"] != nil {
				c.Interval = utils.ConvertToString(p["interval"])
			}
			if p["timeout"] != nil {
				c.Timeout = utils.ConvertToString(p["timeout"])
			}
			if p["name"] != nil {
				c.Name = utils.ConvertToString(p["name"])
			}
			if p["id"] != nil {
				c.CheckID = utils.ConvertToString(p["id"])
			}
			if p["tcp"] != nil {
				c.TCP = utils.ConvertToString(p["tcp"])
			}
			if p["shell"] != nil {
				c.Shell = utils.ConvertToString(p["shell"])
			}
			if p["ttl"] != nil {
				c.TTL = utils.ConvertToString(p["ttl"])
			}
			if p["method"] != nil {
				c.Method = utils.ConvertToString(p["method"])
			}
			if p["status"] != nil {
				c.Status = utils.ConvertToString(p["status"])
			}

			if p["args"] != nil {
				vs := p["args"].([]interface{})
				for _, s := range vs {
					c.Args = append(c.Args, utils.ConvertToString(s))
				}
			}
			if p["notes"] != nil {
				c.Notes = utils.ConvertToString(p["notes"])
			}
			if p["grpc"] != nil {
				c.GRPC = utils.ConvertToString(p["grpc"])
			}

			if p["docker_container_id"] != nil {
				c.DockerContainerID = utils.ConvertToString(p["docker_container_id"])
			}

			if p["tls_skip_verify"] != nil {
				c.TLSSkipVerify = p["tls_skip_verify"].(bool)
			}
			if p["grpc_use_tls"] != nil {
				c.GRPCUseTLS = p["grpc_use_tls"].(bool)
			}

			if p["header"] != nil {
				vs := p["header"].(map[string]interface{})
				var h map[string][]string
				h = make(map[string][]string)

				for k, v := range vs {
					var ss []string
					s1 := v.([]interface{})
					for _, s := range s1 {
						ss = append(ss, utils.ConvertToString(s))
					}
					h[k] = ss
				}

				c.Header = h
			}

			checks = append(checks, &c)
		}
	}

	if checks == nil {
		check = &api.AgentServiceCheck{
			CheckID:  "check-" + serviceID,
			HTTP:     schema + "://" + addr + prefix + "/health",
			Interval: "30s",
			Timeout:  "3s",
		}
	}
	service := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    name,
		Port:    port,
		Address: host,
		Tags:    tags,
		Check:   check,
		Checks:  checks,
	}

	if params["meta"] != nil {
		vs := params["meta"].(map[string]interface{})

		var m map[string]string
		m = make(map[string]string)
		for k, v := range vs {
			m[k] = utils.ConvertToString(v)
		}
		service.Meta = m
	}

	if params["enable_tag_override"] != nil {
		service.EnableTagOverride = params["enable_tag_override"].(bool)
	}

	if params["kind"] != nil {
		service.Kind = api.ServiceKind(utils.ConvertToString(params["kind"]))
	}

	if params["connect"] != nil {
		var connect api.AgentServiceConnect

		p := params["connect"].(map[string]interface{})

		if p["native"] != nil {
			connect.Native = p["native"].(bool)
		}

		service.Connect = &connect
	}

	var caddr string
	if params["consul_address"] != nil {
		caddr = utils.ConvertToString(params["consul_address"])
	} else {
		if params["sd_address"] != nil {
			caddr = utils.ConvertToString(params["sd_address"])
		}
	}

	if caddr != "" {
		consul = caddr
	}
	c := getConsulClient(consul, schema)

	reg := consulsd.NewRegistrar(c, service, log4go.NewDefaultLogger(log4go.FINEST))

	reg.Register()

	mslog.Mslog.Info("Registered service %q in consul with tags %q", name, strings.Join(tags, ","))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit

	reg.Deregister()

	mslog.Mslog.Info("Deregistered service %q in consul", name)

}
