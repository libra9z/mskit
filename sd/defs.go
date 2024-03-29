package sd

import (
	"bytes"
	"github.com/hashicorp/consul/api"
	"github.com/libra9z/mskit/v4/grace"
	"github.com/libra9z/utils"
)

const (
	SERVICE_SCHEME_HTTP  = "http"
	SERVICE_SCHEME_HTTPS = "https"
	SERVICE_SCHEME_RPCX  = "rpcx"
	SERVICE_SCHEME_TCP  = "tcp"
)

const(
	SD_TYPE_CONSUL 	= "consul"
	SD_TYPE_NACOS 	= "nacos"
)


type Registar interface {
	Register(app *grace.MicroService,schema string,address string,params map[string]interface{},callbacks ...ServiceCallback)
	RegisterFile(app *grace.MicroService,schema string,fname string,callbacks ...ServiceCallback)
	RegisterWithConf(app *grace.MicroService,schema string,fname string, callbacks ...ServiceCallback)
	RegisterFromMemory(app *grace.MicroService,schema string,buf *bytes.Buffer, exparams map[string]interface{},callbacks ...ServiceCallback)
	Deregister()
}

type ServiceCallback func(app *grace.MicroService, params map[string]interface{}) error

type ServiceOptions struct {
	ServiceId   string
	ServiceName string
	Tags        []string
	Host        string
	Scheme      string //scheme is : http,https,rpcx
	Port        int
	Sc          ServiceCallback
	Checks      []map[string]interface{}
}

func (so *ServiceOptions) GetConsulRegistration() *api.AgentServiceRegistration {
	service := &api.AgentServiceRegistration{
		ID:      so.ServiceId,
		Name:    so.ServiceName,
		Port:    so.Port,
		Address: so.Host,
		Tags:    so.Tags,
	}

	var checks []*api.AgentServiceCheck
	if len(so.Checks) > 0 {
		for _, p := range so.Checks {
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

	service.Checks = checks

	return service
}
