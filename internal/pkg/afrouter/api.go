/*
 * Copyright 2018-present Open Networking Foundation

 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

 * http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package afrouter

import (
	"errors"
	"fmt"
	"github.com/opencord/voltha-go/common/log"
	pb "github.com/opencord/voltha-protos/go/afrouter"
	common_pb "github.com/opencord/voltha-protos/go/common"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"net/url"
	"runtime"
	"strconv"
)

type ArouterApi struct {
	addr        string
	port        int
	apiListener net.Listener
	apiServer   *grpc.Server
	running     bool
	ar          *ArouterProxy
}

func newApi(config *ApiConfig, ar *ArouterProxy) (*ArouterApi, error) {
	var rtrn_err bool
	// Create a seperate server and listener for the API
	// Validate the ip address if one is provided
	if _, err := url.Parse(config.Addr); err != nil {
		log.Errorf("Invalid address '%s' provided for API server", config.Addr)
		rtrn_err = true
	}
	if rtrn_err {
		return nil, errors.New("Errors in API configuration")
	} else {
		var err error
		aa := &ArouterApi{addr: config.Addr, port: int(config.Port), ar: ar}
		// Create the listener for the API server
		if aa.apiListener, err =
			net.Listen("tcp", config.Addr+":"+
				strconv.Itoa(int(config.Port))); err != nil {
			log.Error(err)
			return nil, err
		}
		// Create the API server
		aa.apiServer = grpc.NewServer()
		pb.RegisterConfigurationServer(aa.apiServer, *aa)
		return aa, err
	}
}

func (aa *ArouterApi) getServer(srvr string) (*server, error) {
	if s, ok := aa.ar.servers[srvr]; !ok {
		err := fmt.Errorf("Server '%s' doesn't exist", srvr)
		return nil, err
	} else {
		return s, nil
	}
}

// nolint: unused
func (aa *ArouterApi) getRouter(s *server, clstr string) (Router, error) {
	for _, pkg := range s.routers {
		for _, r := range pkg {
			if c := r.FindBackendCluster(clstr); c != nil {
				return r, nil
			}
		}
	}
	err := fmt.Errorf("Cluster '%s' doesn't exist", clstr)
	return nil, err
}

func (aa *ArouterApi) getCluster(s *server, clstr string) (*cluster, error) {
	for _, pkg := range s.routers {
		for _, r := range pkg {
			if c := r.FindBackendCluster(clstr); c != nil {
				return c, nil
			}
		}
	}
	err := fmt.Errorf("Cluster '%s' doesn't exist", clstr)
	return nil, err
}

func (aa *ArouterApi) getBackend(c *cluster, bknd string) (*backend, error) {
	for _, b := range c.backends {
		if b.name == bknd {
			return b, nil
		}
	}
	err := fmt.Errorf("Backend '%s' doesn't exist in cluster %s",
		bknd, c.name)
	return nil, err
}

func (aa *ArouterApi) getConnection(b *backend, con string) (*connection, error) {
	if c, ok := b.connections[con]; !ok {
		err := fmt.Errorf("Connection '%s' doesn't exist", con)
		return nil, err
	} else {
		return c, nil
	}
}

func (aa *ArouterApi) updateConnection(in *pb.Conn, cn *connection, b *backend) error {
	return errors.New("updateConnection not implemented")
}

func (aa ArouterApi) SetAffinity(ctx context.Context, in *pb.Affinity) (*pb.Result, error) {
	log.Debugf("SetAffinity called! %v", in)
	//return &pb.Result{Success:true,Error:""},nil
	// Navigate down tot he connection and compare IP addresses and ports if they're
	// not the same then close the existing connection. If they are bothe the same
	// then return an error describing the situation.
	var err error

	aap := &aa

	_ = aap

	log.Debugf("Getting router %s and route %s", in.Router, in.Route)
	if r, ok := allRouters[in.Router+in.Route]; ok {
		switch rr := r.(type) {
		case AffinityRouter:
			log.Debug("Affinity router found")
			b := rr.FindBackendCluster(in.Cluster).getBackend(in.Backend)
			if b != nil {
				err := rr.setAffinity(in.Id, b)
				if err != nil {
					log.Debugf("Couldn't set affinity: %s", err.Error())
					return &pb.Result{Success: false, Error: err.Error()}, err
				}
			} else {
				log.Errorf("Requested backend '%s' not found", in.Backend)
			}
			_ = rr
		case MethodRouter:
			log.Debug("Method router found")
			_ = rr
		default:
			log.Debug("Some other router found")
			_ = rr
		}
	} else {
		err = errors.New("Couldn't get router type")
		log.Debugf("%v", err)
		return &pb.Result{Success: false, Error: err.Error()}, err
	}

	return &pb.Result{Success: true, Error: ""}, nil
}

func (aa ArouterApi) SetConnection(ctx context.Context, in *pb.Conn) (*pb.Result, error) {
	// Navigate down tot he connection and compare IP addresses and ports if they're
	// not the same then close the existing connection. If they are bothe the same
	// then return an error describing the situation.
	var s *server
	var c *cluster
	var b *backend
	var cn *connection
	var err error

	log.Debugf("SetConnection called! %v", in)

	aap := &aa
	if s, err = (aap).getServer(in.Server); err != nil {
		err := fmt.Errorf("Server '%s' doesn't exist", in.Server)
		log.Error(err)
		return &pb.Result{Success: false, Error: err.Error()}, err
	}
	// The cluster is usually accessed via tha router but since each
	// cluster is unique it's good enough to find the router that
	// has the cluster we're looking for rather than fully keying
	// the path
	if c, err = aap.getCluster(s, in.Cluster); err != nil {
		log.Error(err)
		return &pb.Result{Success: false, Error: err.Error()}, err
	}

	if b, err = aap.getBackend(c, in.Backend); err != nil {
		log.Error(err)
		return &pb.Result{Success: false, Error: err.Error()}, err
	}

	if cn, err = aap.getConnection(b, in.Connection); err != nil {
		log.Error(err)
		return &pb.Result{Success: false, Error: err.Error()}, err
	}

	if err = aap.updateConnection(in, cn, b); err != nil {
		log.Error(err)
		return &pb.Result{Success: false, Error: err.Error()}, err
	}

	return &pb.Result{Success: true, Error: ""}, nil
}

func (aa ArouterApi) GetGoroutineCount(ctx context.Context, in *pb.Empty) (*pb.Count, error) {
	return &pb.Count{Count: uint32(runtime.NumGoroutine())}, nil
}

func (aa ArouterApi) UpdateLogLevel(ctx context.Context, in *common_pb.Logging) (*pb.Empty, error) {
	intLevel := int(in.Level)

	if in.PackageName == "" {
		log.SetAllLogLevel(intLevel)
		log.SetDefaultLogLevel(intLevel)
	} else if in.PackageName == "default" {
		log.SetDefaultLogLevel(intLevel)
	} else {
		log.SetPackageLogLevel(in.PackageName, intLevel)
	}

	return &pb.Empty{}, nil
}

func (aa ArouterApi) GetLogLevels(ctx context.Context, in *common_pb.LoggingComponent) (*common_pb.Loggings, error) {
	logLevels := &common_pb.Loggings{}

	// do the per-package log levels
	for _, packageName := range log.GetPackageNames() {
		level, err := log.GetPackageLogLevel(packageName)
		if err != nil {
			return nil, err
		}
		logLevel := &common_pb.Logging{
			ComponentName: in.ComponentName,
			PackageName:   packageName,
			Level:         common_pb.LogLevel_LogLevel(level)}
		logLevels.Items = append(logLevels.Items, logLevel)
	}

	// now do the default log level
	logLevel := &common_pb.Logging{
		ComponentName: in.ComponentName,
		PackageName:   "default",
		Level:         common_pb.LogLevel_LogLevel(log.GetDefaultLogLevel())}
	logLevels.Items = append(logLevels.Items, logLevel)

	return logLevels, nil
}

func (aa *ArouterApi) serve() {
	// Start a serving thread
	go func() {
		aa.running = true
		if err := aa.apiServer.Serve(aa.apiListener); err != nil {
			aa.running = false
			log.Error(err)
			errChan <- err
		}
	}()
}
