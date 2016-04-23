package mskit

import (
	kitlog "github.com/go-kit/kit/log"
	
	"net/http"
	"net"
	"os"
	"strconv"
	"time"
	"github.com/libra9z/httprouter"
)


var logger kitlog.Logger

// App defines msrest application with a new PatternServeMux.
type MicroService struct {
	Router 	*httprouter.Router
	Server	*http.Server
}

// NewApp returns a new msrest application.
func NewRestMicroService() *MicroService {
	router := httprouter.New()
	ms := &MicroService{Router: router,Server: &http.Server{}}
	return ms
}

var (
	// BeeApp is an application instance
	MsRest *MicroService
)

func init(){
	MsRest = NewRestMicroService()
}

// Run Rest MicroService.
/**
* params 为可变参数
* 第一个参数为ip host
* 第二个参数为ip port
* 第三个参数为ServerTimeOut
* 第四个参数为协议是否为Tcp4 or tcp6，bool值：true or false
 */
func (ms *MicroService) Serve(params ...string) {

	logger = kitlog.NewLogfmtLogger(os.Stderr)

	if len(params) < 2 {
		logger.Log("err: no host port parameters set.")
		return
	}

	addr := params[0] + ":" + params[1]

	ms.Server.Handler = ms.Router

	if len(params) > 2 {
		ServerTimeOut, _ := strconv.ParseInt(params[2], 10, 64)
		ms.Server.ReadTimeout = time.Duration(ServerTimeOut) * time.Second
		ms.Server.WriteTimeout = time.Duration(ServerTimeOut) * time.Second
	}

	var isListenTCP4 bool= false
	
	if len(params)>3 {
		isListenTCP4,_ = strconv.ParseBool(params[3])
	}
	
	// run normal mode
	ms.Server.Addr = addr

	go func() {
		ms.Server.Addr = addr
		logger.Log("http server Running on %s", ms.Server.Addr)
		if isListenTCP4 {
			ln, err := net.Listen("tcp4", ms.Server.Addr)
			if err != nil {
				logger.Log("ListenAndServe: ", err)
				time.Sleep(100 * time.Microsecond)
				return
			}
			if err = ms.Server.Serve(ln); err != nil {
				logger.Log("ListenAndServe: ", err)
				time.Sleep(100 * time.Microsecond)
				return
			}
		} else {
			if err := ms.Server.ListenAndServe(); err != nil {
				logger.Log("ListenAndServe: ", err)
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()
}


func Serve(params ...string) {
	if MsRest != nil {
		MsRest.Serve(params...)
	}
}