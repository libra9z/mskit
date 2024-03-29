package grace

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/libra9z/httprouter"
	"github.com/libra9z/mskit/v4/endpoint"
	"github.com/libra9z/mskit/v4/log"
	"github.com/libra9z/mskit/v4/rest"
	"github.com/libra9z/mskit/v4/trace"
)

// App defines msrest application with a new PatternServeMux.
type MicroService struct {
	Router *httprouter.Router
	Server *http.Server
	logger log.Logger
	tracer trace.Tracer

	GraceListener    net.Listener
	SignalHooks      map[int]map[os.Signal][]func()
	tlsInnerListener *graceListener
	wg               sync.WaitGroup
	sigChan          chan os.Signal
	isChild          bool
	state            uint8
	Network          string
	Meta             map[string]interface{}
}

/**
* params 为可变参数
* 第一个参数为ip host
* 第二个参数为ip port
* 第三个参数为ServerTimeOut
* 第四个参数为协议是否为Tcp4 or tcp6， 字符串，取值：tcp,tcp4,tcp6
* 第五个参数为协议是否为certFile， 字符串，取值：certFile
* 第六个参数为协议是否为keyFile， 字符串，取值：keyFile
* 第七个参数为协议是否为trustFile， 字符串，取值：trustFile
 */
func (srv *MicroService) Serve(params ...string) (err error) {

	if len(params) < 2 {
		if srv.Server.Addr == "" {
			fmt.Printf("err: no host port parameters set.\n")
			return
		}
	} else {
		srv.Server.Addr = params[0] + ":" + params[1]
	}

	if len(params) > 2 {
		ServerTimeOut, _ := strconv.ParseInt(params[2], 10, 64)
		srv.Server.ReadTimeout = time.Duration(ServerTimeOut) * time.Second
		srv.Server.WriteTimeout = time.Duration(ServerTimeOut) * time.Second
	}
	if len(params) > 3 {
		srv.Network = params[3]
	} else {
		srv.Network = "tcp"
	}

	if srv.GraceListener == nil {
		l, err := srv.getListener(srv.Server.Addr)
		if err != nil {
			srv.logger.Error("error = %v", err)
			return err
		}

		srv.GraceListener = newGraceListener(l, srv)
	}
	srv.state = StateRunning
	if srv.Server.Handler == nil {
		srv.Server.Handler = srv.Router
	}
	err = srv.Server.Serve(srv.GraceListener)
	srv.logger.Info("Waiting for connections to finish...: %v", syscall.Getpid())
	srv.wg.Wait()
	srv.state = StateTerminate
	return
}

// ListenAndServe listens on the TCP network address srv.Addr and then calls Serve
// to handle requests on incoming connections. If srv.Addr is blank, ":http" is
// used.
func (srv *MicroService) ListenAndServe(params ...string) (err error) {

	if srv.Server.Addr == "" {
		if len(params) < 2 {
			fmt.Printf("err: no host port parameters set.\n")
			return
		}
		srv.Server.Addr = params[0] + ":" + params[1]
	}

	go srv.handleSignals()

	l, err := srv.getListener(srv.Server.Addr)
	if err != nil {
		srv.logger.Error("error=%v", err)
		return err
	}

	srv.GraceListener = newGraceListener(l, srv)

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			srv.logger.Error("error=%v", err)
			return err
		}
		err = process.Signal(syscall.SIGTERM)
		if err != nil {
			return err
		}
	}
	srv.logger.Info("address=%s,pid=%d", srv.Server.Addr, os.Getpid())
	return srv.Serve(params...)
}

// ListenAndServeTLS listens on the TCP network address srv.Addr and then calls
// Serve to handle requests on incoming TLS connections.
//
// Filenames containing a certificate and matching private key for the server must
// be provided. If the certificate is signed by a certificate authority, the
// certFile should be the concatenation of the server's certificate followed by the
// CA's certificate.

func (srv *MicroService) ListenAndServeTLS(certFile, keyFile string, params ...string) (err error) {

	if srv.Server.Addr == "" {
		if len(params) < 2 {
			fmt.Printf("err: no host port parameters set.\n")
			return
		}
		srv.Server.Addr = params[0] + ":" + params[1]
	}

	if srv.Server.TLSConfig == nil {
		srv.Server.TLSConfig = &tls.Config{}
	}
	if srv.Server.TLSConfig.NextProtos == nil {
		srv.Server.TLSConfig.NextProtos = []string{"http/1.1"}
	}

	srv.Server.TLSConfig.Certificates = make([]tls.Certificate, 1)
	srv.Server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return
	}

	go srv.handleSignals()

	l, err := srv.getListener(srv.Server.Addr)
	if err != nil {
		srv.logger.Error("error=%v", err)
		return err
	}

	srv.tlsInnerListener = newGraceListener(l, srv)
	srv.GraceListener = tls.NewListener(srv.tlsInnerListener, srv.Server.TLSConfig)

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			srv.logger.Error("error=%v", err)
			return err
		}
		err = process.Signal(syscall.SIGTERM)
		if err != nil {
			return err
		}
	}
	srv.logger.Info("address=%s,pid=%d", srv.Server.Addr, os.Getpid())
	return srv.Serve(params...)
}

// ListenAndServeMutualTLS listens on the TCP network address srv.Addr and then calls
// Serve to handle requests on incoming mutual TLS connections.
func (srv *MicroService) ListenAndServeMutualTLS(certFile, keyFile, trustFile string, params ...string) (err error) {
	if srv.Server.Addr == "" {
		if len(params) < 2 {
			fmt.Printf("err: no host port parameters set.\n")
			return
		}
		srv.Server.Addr = params[0] + ":" + params[1]
	}

	if srv.Server.TLSConfig == nil {
		srv.Server.TLSConfig = &tls.Config{}
	}
	if srv.Server.TLSConfig.NextProtos == nil {
		srv.Server.TLSConfig.NextProtos = []string{"http/1.1"}
	}

	srv.Server.TLSConfig.Certificates = make([]tls.Certificate, 1)
	srv.Server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return
	}
	srv.Server.TLSConfig.ClientAuth = tls.RequireAndVerifyClientCert
	pool := x509.NewCertPool()
	data, err := ioutil.ReadFile(trustFile)
	if err != nil {
		srv.logger.Error("error=%v", err)
		return err
	}
	pool.AppendCertsFromPEM(data)
	srv.Server.TLSConfig.ClientCAs = pool
	go srv.handleSignals()

	l, err := srv.getListener(srv.Server.Addr)
	if err != nil {
		srv.logger.Error("error=%v", err)
		return err
	}

	srv.tlsInnerListener = newGraceListener(l, srv)
	srv.GraceListener = tls.NewListener(srv.tlsInnerListener, srv.Server.TLSConfig)

	if srv.isChild {
		process, err := os.FindProcess(os.Getppid())
		if err != nil {
			srv.logger.Error("error=%v", err)
			return err
		}
		err = process.Kill()
		if err != nil {
			return err
		}
	}
	srv.logger.Info("address=%s,pid=%d", srv.Server.Addr, os.Getpid())
	return srv.Serve(params...)
}

// getListener either opens a new socket to listen on, or takes the acceptor socket
// it got passed when restarted.
func (srv *MicroService) getListener(laddr string) (l net.Listener, err error) {
	if srv.isChild {
		var ptrOffset uint
		if len(socketPtrOffsetMap) > 0 {
			ptrOffset = socketPtrOffsetMap[laddr]
			srv.logger.Info("laddr=%s,ptr offset=%d", laddr, socketPtrOffsetMap[laddr])
		}

		f := os.NewFile(uintptr(3+ptrOffset), "")
		l, err = net.FileListener(f)
		if err != nil {
			err = fmt.Errorf("net.FileListener error: %v", err)
			return
		}
	} else {
		l, err = net.Listen(srv.Network, laddr)
		if err != nil {
			err = fmt.Errorf("net.Listen error: %v", err)
			return
		}
	}
	return
}

// handleSignals listens for os Signals and calls any hooked in function that the
// user had registered with the signal.
func (srv *MicroService) handleSignals() {
	var sig os.Signal

	signal.Notify(
		srv.sigChan,
		hookableSignals...,
	)

	pid := syscall.Getpid()
	for {
		sig = <-srv.sigChan
		srv.signalHooks(PreSignal, sig)
		switch sig {
		case syscall.SIGHUP:
			fmt.Println("Received SIGHUP. forking.", pid)
			err := srv.fork()
			if err != nil {
				srv.logger.Error("error=%v", err)
			}
		case syscall.SIGINT:
			fmt.Println("Received SIGINT.", pid)
			srv.shutdown()
		case syscall.SIGTERM:
			fmt.Println("Received SIGTERM.", pid)
			srv.shutdown()
		default:
			fmt.Printf("Received %v: nothing i care about...\n", sig)
		}
		srv.signalHooks(PostSignal, sig)
	}
}

func (srv *MicroService) signalHooks(ppFlag int, sig os.Signal) {
	if _, notSet := srv.SignalHooks[ppFlag][sig]; !notSet {
		return
	}
	for _, f := range srv.SignalHooks[ppFlag][sig] {
		f()
	}
}

// shutdown closes the listener so that no new connections are accepted. it also
// starts a goroutine that will serverTimeout (stop all running requests) the server
// after DefaultTimeout.
func (srv *MicroService) shutdown() {
	if srv.state != StateRunning {
		return
	}

	srv.state = StateShuttingDown
	if DefaultTimeout >= 0 {
		go srv.serverTimeout(DefaultTimeout)
	}
	err := srv.GraceListener.Close()
	if err != nil {
		fmt.Printf("pid=%v,Listener.Close() error: %v\n", syscall.Getpid(), err)
	} else {
		fmt.Printf("pid=%v,address=%v,Listener closed.", syscall.Getpid(), srv.GraceListener.Addr())
	}
}

// serverTimeout forces the server to shutdown in a given timeout - whether it
// finished outstanding requests or not. if Read/WriteTimeout are not set or the
// max header size is very big a connection could hang
func (srv *MicroService) serverTimeout(d time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			srv.logger.Error("error=%v", r)
		}
	}()
	if srv.state != StateShuttingDown {
		return
	}
	time.Sleep(d)
	fmt.Println("[STOP - Hammer Time] Forcefully shutting down parent")
	for {
		if srv.state == StateTerminate {
			break
		}
		srv.wg.Done()
	}
}

func (srv *MicroService) fork() (err error) {
	regLock.Lock()
	defer regLock.Unlock()
	if runningServersForked {
		return
	}
	runningServersForked = true

	var files = make([]*os.File, len(runningServers))
	var orderArgs = make([]string, len(runningServers))
	for _, srvPtr := range runningServers {
		switch srvPtr.GraceListener.(type) {
		case *graceListener:
			files[socketPtrOffsetMap[srvPtr.Server.Addr]] = srvPtr.GraceListener.(*graceListener).File()
		default:
			files[socketPtrOffsetMap[srvPtr.Server.Addr]] = srvPtr.tlsInnerListener.File()
		}
		orderArgs[socketPtrOffsetMap[srvPtr.Server.Addr]] = srvPtr.Server.Addr
	}

	fmt.Println(files)
	path := os.Args[0]
	var args []string
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if arg == "-graceful" {
				break
			}
			args = append(args, arg)
		}
	}
	args = append(args, "-graceful")
	if len(runningServers) > 1 {
		args = append(args, fmt.Sprintf(`-socketorder=%s`, strings.Join(orderArgs, ",")))
		fmt.Println(args)
	}
	cmd := exec.Command(path, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = files
	err = cmd.Start()
	if err != nil {
		fmt.Errorf("Restart: Failed to launch, error: %v\n", err)
	}

	return
}

// RegisterSignalHook registers a function to be run PreSignal or PostSignal for a given signal.
func (srv *MicroService) RegisterSignalHook(ppFlag int, sig os.Signal, f func()) (err error) {
	if ppFlag != PreSignal && ppFlag != PostSignal {
		err = fmt.Errorf("Invalid ppFlag argument. Must be either grace.PreSignal or grace.PostSignal")
		return
	}
	for _, s := range hookableSignals {
		if s == sig {
			srv.SignalHooks[ppFlag][sig] = append(srv.SignalHooks[ppFlag][sig], f)
			return
		}
	}
	err = fmt.Errorf("Signal '%v' is not supported", sig)
	return
}

func (srv *MicroService) NewRestEndpoint(svc rest.RestService) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		if request == nil {
			return nil, errors.New("no request available")
		}

		req := request.(*rest.Mcontext)

		var ret interface{}
		var err error
		switch req.Method {
		case "GET":
			ret, err = svc.Get(ctx, req)
		case "POST":
			ret, err = svc.Post(ctx, req)
		case "PUT":
			ret, err = svc.Put(ctx, req)
		case "DELETE":
			ret, err = svc.Delete(ctx, req)
		case "HEAD":
			ret, err = svc.Head(ctx, req)
		case "PATCH":
			ret, err = svc.Patch(ctx, req)
		case "OPTIONS":
			ret, err = svc.Options(ctx, req)
		case "TRACE":
			ret, err = svc.Trace(ctx, req)
		case "CONNECT":
		}

		if err != nil {
			return svc.GetErrorResponse(), nil
		}
		return ret, nil
	}
}

func (srv *MicroService) NewEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		if request == nil {
			return nil, errors.New("no request available")
		}

		req := request.(*rest.Mcontext)

		return req, nil
	}
}

func (srv *MicroService) SetLogger(logger log.Logger) {
	srv.logger = logger
}

func (srv *MicroService) GetLogger() log.Logger {
	return srv.logger
}

func (srv *MicroService) SetTracer(tracer trace.Tracer) {
	srv.tracer = tracer
}

func (srv *MicroService) GetTracer() trace.Tracer {
	return srv.tracer
}

func (srv *MicroService) RegisterSwaggerDoc(path string, handler http.HandlerFunc) {
	srv.Router.HandlerFunc("GET", path, handler)
}

func (srv *MicroService) NewHttpHandler(withTracer bool, path string, r rest.RestService, middlewares ...rest.RestMiddleware) *rest.Engine {

	r.SetRouter(srv.Router)

	svc := srv.NewRestEndpoint(r)

	for i := 0; i < len(middlewares); i++ {
		svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
	}

	var options []rest.ServerOption

	if srv.tracer != nil {
		if withTracer {
			options = append(options, []rest.ServerOption{
				rest.ServerErrorHandler(rest.NewLogErrorHandler(srv.logger)),
				srv.tracer.HTTPServerTrace(path),
			}...)
		} else {
			options = append(options, []rest.ServerOption{
				rest.ServerErrorHandler(rest.NewLogErrorHandler(srv.logger)),
			}...)
		}
	} else {
		options = append(options, []rest.ServerOption{
			rest.ServerErrorHandler(rest.NewLogErrorHandler(srv.logger)),
		}...)
	}

	var before []rest.RequestFunc

	for _, f := range r.Before() {
		before = append(before, rest.RequestFunc(f))
	}

	options = append(options, rest.ServerBefore(before...))

	var after []rest.ServerResponseFunc
	for _, f := range r.After() {
		after = append(after, rest.ServerResponseFunc(f))
	}
	options = append(options, rest.ServerAfter(after...))

	handler := rest.NewEngine(
		svc,
		r.DecodeRequest,
		r.EncodeResponse,
		options...,
	)
	return handler
}

func (srv *MicroService) RegisterServiceWithTracer(path string, rest rest.RestService, tracer trace.Tracer, logger log.Logger, middlewares ...rest.RestMiddleware) {

	srv.SetLogger(logger)
	srv.SetTracer(tracer)

	handler := srv.NewHttpHandler(true, path, rest, middlewares...)
	regRoute(srv.Router, path, handler)
}

func (srv *MicroService) RegisterRestService(path string, rest rest.RestService, middlewares ...rest.RestMiddleware) {

	handler := srv.NewHttpHandler(false, path, rest, middlewares...)
	regRoute(srv.Router, path, handler)
}

func (srv *MicroService) Handler(method, path string, ohandler http.Handler, middlewares ...rest.RestMiddleware) {

	//handler := srv.NewHttpHandler(false, path, rest, middlewares...)
	handler := ohandler
	srv.Router.Handler(method, path, handler)
}
func (srv *MicroService) HandlerFunc(method, path string, handlerFunc http.HandlerFunc, tracer trace.Tracer, logger log.Logger, middlewares ...rest.RestMiddleware) {

	srv.SetTracer(tracer)
	srv.SetLogger(logger)

	//handler := srv.NewHandlerFunc(false, path, rest, middlewares...)
	handler := srv.NewHandlerFunc(true, path, handlerFunc, middlewares...)

	srv.Router.Handler(method, path, handler)
	//srv.Router.HandlerFunc(method, path, handler)
}

func (srv *MicroService) NewHandlerFunc(withTracer bool, path string, handlerFunc http.HandlerFunc, middlewares ...rest.RestMiddleware) http.HandlerFunc {

	return func(w http.ResponseWriter, req *http.Request) {

		svc := srv.NewEndpoint()

		for i := 0; i < len(middlewares); i++ {
			svc = middlewares[i].GetMiddleware()(middlewares[i].Object)(svc)
		}

		var options []rest.ServerOption

		if srv.tracer != nil {
			if withTracer {
				options = append(options, []rest.ServerOption{
					rest.ServerErrorHandler(rest.NewLogErrorHandler(srv.logger)),
					srv.tracer.HTTPServerTrace(path),
				}...)
			} else {
				options = append(options, []rest.ServerOption{
					rest.ServerErrorHandler(rest.NewLogErrorHandler(srv.logger)),
				}...)
			}
		} else {
			options = append(options, []rest.ServerOption{
				rest.ServerErrorHandler(rest.NewLogErrorHandler(srv.logger)),
			}...)
		}

		//h:= func() rest.Middleware{
		//	return func(next endpoint.Endpoint) endpoint.Endpoint {
		//		return func(ctx context.Context, request interface{}) (interface{}, error) {
		//			if request == nil {
		//				return nil, errors.New("no request available")
		//			}
		//
		//			m := request.(*rest.Mcontext)
		//			handlerFunc(w, req)
		//			return next(ctx, m)
		//		}
		//	}
		//}()
		//svc = h(svc)
		//
		//handler := rest.NewEngine(
		//	svc,
		//	r.DecodeRequest,
		//	rest.EncodeResponse,
		//	options...,
		//)
		//
		//handler.ServeHTTP(w,req)
		handlerFunc(w, req)
	}
}

func regRoute(r *httprouter.Router, path string, handler http.Handler) {

	r.Handler("GET", path, handler)
	r.Handler("POST", path, handler)
	r.Handler("PUT", path, handler)
	r.Handler("PATCH", path, handler)
	r.Handler("DELETE", path, handler)
	r.Handler("HEAD", path, handler)
	r.Handler("OPTIONS", path, handler)
	r.Handler("TRACE", path, handler)
}

func (srv *MicroService) ServeFiles(path string, root http.FileSystem) {
	if srv.Router != nil {
		srv.Router.ServeFiles(path, root)
	} else {
		fmt.Printf("no rest service avaliable.\n")
	}
}
