package agent

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/d3sw/replicator/logging"
	"github.com/d3sw/replicator/replicator/structs"
	"github.com/ugorji/go/codec"
)

const (
	// ErrInvalidMethod is used if the HTTP method is not supported
	ErrInvalidMethod = "Invalid method"
)

var (
	// JSONHandle is the codec that handles to JSON encode structs.
	JSONHandle = &codec.JsonHandle{
		HTMLCharsAsIs: true,
	}
)

// CodedError returns an interface to the Replicator HTTP error code.
func CodedError(c int, s string) HTTPCodedError {
	return &codedError{s, c}
}

func (e *codedError) Error() string {
	return e.s
}

func (e *codedError) Code() int {
	return e.code
}

type codedError struct {
	s    string
	code int
}

// HTTPCodedError is used to provide the HTTP error code.
type HTTPCodedError interface {
	error
	Code() int
}

// HTTPServer is used to wrap an Agent and expose it over an HTTP interface
type HTTPServer struct {
	agent    *Command
	mux      *http.ServeMux
	listener net.Listener
	logger   *log.Logger
	Addr     string
}

// Listener can be used to get a new listener using a custom bind address. If
// the bind provided address is empty, the BindAddr is used instead.
func Listener(proto, addr string, port int) (net.Listener, error) {
	if 0 > port || port > 65535 {
		return nil, &net.OpError{
			Op:  "listen",
			Net: proto,
			Err: &net.AddrError{Err: "invalid port", Addr: fmt.Sprint(port)},
		}
	}
	return net.Listen(proto, net.JoinHostPort(addr, strconv.Itoa(port)))
}

// NewHTTPServer starts the HTTP API server for the Replicator agent.
func NewHTTPServer(agent *Command, config *structs.Config) (*HTTPServer, error) {

	// Start the listener
	lnAddr, err := net.ResolveTCPAddr("tcp", config.BindAddress+":"+config.HTTPPort)
	if err != nil {
		return nil, err
	}
	ln, err := Listener("tcp", lnAddr.IP.String(), lnAddr.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to start HTTP listener: %v", err)
	}

	// Create the mux
	mux := http.NewServeMux()

	// Create the server
	srv := &HTTPServer{
		agent:    agent,
		mux:      mux,
		listener: ln,
		Addr:     ln.Addr().String(),
	}
	srv.registerHandlers()

	// Handle requests with gzip compression
	gzip, err := gziphandler.GzipHandlerWithOpts(gziphandler.MinSize(0))
	if err != nil {
		return nil, err
	}

	go http.Serve(ln, gzip(mux))
	logging.Info("command/http: the API server has started and is listening at %s", srv.Addr)

	return srv, nil
}

// Shutdown is used to shutdown the HTTP server.
func (s *HTTPServer) Shutdown() {
	if s != nil {
		logging.Info("command/http: shutting down the HTTP server at %v", s.Addr)
		s.listener.Close()
	}
}

// registerHandlers is used to attach our handlers.
func (s *HTTPServer) registerHandlers() {
	s.mux.HandleFunc("/v1/status/leader", s.wrap(s.StatusLeaderRequest))
}

func (s *HTTPServer) wrap(handler func(resp http.ResponseWriter, req *http.Request) (interface{}, error)) func(resp http.ResponseWriter, req *http.Request) {
	f := func(resp http.ResponseWriter, req *http.Request) {

		// Invoke the handler
		reqURL := req.URL.String()
		start := time.Now()
		defer func() {
			logging.Debug("command/http: request %v %v (%v)", req.Method, reqURL, time.Now().Sub(start))
		}()
		obj, err := handler(resp, req)

		// Check for an error
	HAS_ERR:
		if err != nil {
			logging.Error("command/http: request %v, error: %v", reqURL, err)
			code := 500
			if http, ok := err.(HTTPCodedError); ok {
				code = http.Code()
			}
			resp.WriteHeader(code)
			resp.Write([]byte(err.Error()))
			return
		}

		// Write out the JSON object
		if obj != nil {
			var buf bytes.Buffer

			enc := codec.NewEncoder(&buf, JSONHandle)
			err = enc.Encode(obj)

			if err != nil {
				goto HAS_ERR
			}
			resp.Header().Set("Content-Type", "application/json")
			resp.Write(buf.Bytes())
		}
	}
	return f
}
