/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package triple

import (
	"io"
	"net"
	"net/http"
	"sync"
)

import (
	dubboCommon "github.com/dubbogo/gost/dubbogo"
	logger "github.com/dubbogo/gost/dubbogo/logger"
	"golang.org/x/net/http2"
)

import (
	"github.com/dubbogo/triple/internal/tools"
	"github.com/dubbogo/triple/pkg/config"
)

// TripleServer is the object that can be started and listening remote request
type TripleServer struct {
	lst           net.Listener
	addr          string
	rpcServiceMap *sync.Map
	url           *dubboCommon.URL
	h2Controller  *H2Controller
	closeChain    chan struct{}

	// config
	opt *config.Option
}

// NewTripleServer can create Server with url and some user impl providers stored in @serviceMap
// @serviceMap should be sync.Map: "interfaceKey" -> Dubbo3GrpcService
func NewTripleServer(url *dubboCommon.URL, serviceMap *sync.Map, opt *config.Option) *TripleServer {
	opt = tools.AddDefaultOption(opt)
	return &TripleServer{
		addr:          url.Location,
		rpcServiceMap: serviceMap,
		url:           url,
		closeChain:    make(chan struct{}, 1),
		opt:           opt,
	}
}

// Stop
func (t *TripleServer) Stop() {
	if t.h2Controller != nil {
		t.h2Controller.Destroy()
	}
	t.closeChain <- struct{}{}
}

// Start can start a triple server
func (t *TripleServer) Start() {
	logger.Info("tripleServer Start at ", t.addr)
	lst, err := net.Listen("tcp", t.addr)
	if err != nil {
		panic(err)
	}
	t.lst = lst
	go t.run()
}

// run can start a loop to accept tcp conn
func (t *TripleServer) run() {
	go func() {
		select {
		case <-t.closeChain:
			return
		}
	}()
	for {
		conn, err := t.lst.Accept()
		if err != nil {
			return
		}
		go func() {
			defer func() {
				if e := recover(); e != nil {
					logger.Error(" handle raw conn panic = ", err)
				}
			}()
			if err := t.handleRawConn(conn); err != nil && err != io.EOF {
				logger.Error(" handle raw conn err = ", err)
			}
		}()
	}
}

// handleRawConn create a H2 Controller to deal with new conn
func (t *TripleServer) handleRawConn(conn net.Conn) error {
	srv := &http2.Server{}
	h2Controller, err := NewH2Controller(true, t.rpcServiceMap, t.url, t.opt)
	if err != nil {
		return err
	}
	t.h2Controller = h2Controller
	opts := &http2.ServeConnOpts{Handler: http.HandlerFunc(h2Controller.GetHandler())}
	srv.ServeConn(conn, opts)
	return nil
}
