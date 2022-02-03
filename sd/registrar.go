package sd

import (
	"bytes"
	"fmt"
	"github.com/libra9z/mskit/v4/grace"
)

type serviceDiscovery struct {
	SdType    string
	SdAddress string
	SdToken   string
	r 				Registar
	prefix 			string
	name 			string
	callback 		ServiceCallback
	params 			map[string]interface{}
	addr 			string		//listen on address and port
}

type SdOption func(*serviceDiscovery)

var _ Registar = (*serviceDiscovery)(nil)

func WithSdType(sdtype string) SdOption {
	return func(s *serviceDiscovery) { s.SdType = sdtype }
}
func WithSdAddress(sdaddress string) SdOption {
	return func(s *serviceDiscovery) { s.SdAddress = sdaddress }
}
func WithSdToken(sdtoken string) SdOption {
	return func(s *serviceDiscovery) { s.SdToken = sdtoken }
}
func WithServiceName(name string) SdOption {
	return func(s *serviceDiscovery) { s.name = name }
}
func WithPrefix(prefix string) SdOption {
	return func(s *serviceDiscovery) { s.prefix = prefix }
}
func WithListenAddress(addr string) SdOption {
	return func(s *serviceDiscovery) { s.addr = addr }
}
func WithParams(params map[string]interface{}) SdOption {
	return func(s *serviceDiscovery) { s.params = params }
}
func WithCallback(callback ServiceCallback) SdOption {
	return func(s *serviceDiscovery) { s.callback = callback }
}


func NewRegistar(options ...SdOption) Registar {
	s := &serviceDiscovery{}
	for _, option := range options {
		option(s)
	}
	var err error
	switch s.SdType {
	case SD_TYPE_CONSUL:
		s.r,err = NewConsulRegistar(s.name,s.prefix,s.addr,s.SdAddress,s.SdToken,s.callback,s.params)
	case SD_TYPE_NACOS:
		s.r,err = NewNacosRegistar(s.name,s.prefix,s.addr,s.SdAddress,s.SdToken,s.callback,s.params)
	}
	if err != nil {
		logger.Log("error",fmt.Sprintf("不能注册服务-%s",err.Error()))
		return nil
	}
	return s
}
func (s *serviceDiscovery) Register(app *grace.MicroService,schema string) {
	s.r.Register(app,schema)
}

func (s *serviceDiscovery) RegisterWithConf(app *grace.MicroService,schema string,fname string, callbacks ...ServiceCallback) {
	s.r.RegisterWithConf(app,schema,fname,callbacks...)
}
func (s *serviceDiscovery) RegisterFile(app *grace.MicroService,schema string,fname string,callbacks ...ServiceCallback) {
	s.r.RegisterFile(app,schema,fname,callbacks...)
}

func (s *serviceDiscovery) RegisterFromMemory(app *grace.MicroService,schema string,reader *bytes.Buffer, exparams map[string]interface{},callbacks ...ServiceCallback) {
	s.r.RegisterFromMemory(app,schema,reader,exparams,callbacks...)
}

func (s *serviceDiscovery)Deregister() {
	s.r.Deregister()
}