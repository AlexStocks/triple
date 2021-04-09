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
	"context"
	dubboConstant "github.com/apache/dubbo-go/common/constant"
	"github.com/dubbogo/triple/internal/codec"
	"github.com/dubbogo/triple/internal/tools"
	"github.com/dubbogo/triple/pkg/common"
	"github.com/dubbogo/triple/pkg/config"
	"reflect"
	"sync"
)

import (
	dubboCommon "github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	perrors "github.com/pkg/errors"
	"google.golang.org/grpc"
)

// TripleClient client endpoint for client end
type TripleClient struct {
	h2Controller *H2Controller
	addr         string
	StubInvoker  reflect.Value
	url          *dubboCommon.URL

	//once is used when destroy
	once sync.Once

	// triple config
	opt *config.Option

	// serializer is triple serializer to do codec
	serializer common.Dubbo3Serializer
}

// NewTripleClient create triple client with given @url,
// it's return tripleClient , contains invoker, and contain triple conn
// @url is the invocation url when dubbo client invoct. Now, triple only use Location and Protocol field of url.
// @impl must have method: GetDubboStub(cc *dubbo3.TripleConn) interface{}, to be capable with grpc
// @opt is used to init http2 controller, if it's nil, use the default config
func NewTripleClient(url *dubboCommon.URL, impl interface{}, opt *config.Option) (*TripleClient, error) {
	opt = tools.AddDefaultOption(opt)

	tripleClient := &TripleClient{
		url: url,
		opt: opt,
	}
	// start triple client connection,
	if err := tripleClient.connect(url); err != nil {
		return nil, err
	}

	// put dubbo3 network logic to tripleConn, creat pb stub invoker
	if opt.SerializerType == common.PBSerializerName {
		tripleClient.StubInvoker = reflect.ValueOf(getInvoker(impl, newTripleConn(tripleClient)))
	}

	return tripleClient, nil
}

// Invoke call remote using stub
func (t *TripleClient) Invoke(methodName string, in []reflect.Value) []reflect.Value {
	rsp := make([]reflect.Value, 0, 2)
	switch t.opt.SerializerType {
	case common.PBSerializerName:
		method := t.StubInvoker.MethodByName(methodName)
		// call function in pb.go
		return method.Call(in)
	case common.TripleHessianWrapperSerializerName:
		out := codec.HessianUnmarshalStruct{}
		ctx := in[0].Interface().(context.Context)
		interfaceKey := ctx.Value(dubboConstant.DubboCtxKey(dubboConstant.INTERFACE_KEY)).(string)
		err := t.Request(ctx, "/"+interfaceKey+"/"+methodName, in[1].Interface(), &out)
		rsp = append(rsp, reflect.ValueOf(out.Val))
		if err != nil {
			return append(rsp, reflect.ValueOf(err))
		}
		return append(rsp, reflect.Value{})
	}
	logger.Errorf("Invalid triple client serializerType = %s", t.opt.SerializerType)
	rsp = append(rsp, reflect.Value{})
	return append(rsp, reflect.ValueOf(perrors.Errorf("Invalid triple client serializerType = %s", t.opt.SerializerType)))
}

// Connect called when new TripleClient, which start a tcp conn with target addr
func (t *TripleClient) connect(url *dubboCommon.URL) error {
	logger.Info("want to connect to url = ", url.Location)
	t.addr = url.Location
	var err error
	t.h2Controller, err = NewH2Controller(false, nil, url, t.opt)
	if err != nil {
		logger.Errorf("dubbo client new http2 controller error = %v", err)
		return err
	}
	t.h2Controller.address = url.Location
	return nil
}

// Request call h2Controller to send unary rpc req to server
// @path is /interfaceKey/functionName e.g. /com.apache.dubbo.sample.basic.IGreeter/BigUnaryTest
// @arg is request body
func (t *TripleClient) Request(ctx context.Context, path string, arg, reply interface{}) error {
	if t.h2Controller == nil {
		if err := t.connect(t.url); err != nil {
			logger.Errorf("dubbo client connect to url error = %v", err)
			return err
		}
	}
	if err := t.h2Controller.UnaryInvoke(ctx, path, arg, reply); err != nil {
		return err
	}
	return nil
}

// StreamRequest call h2Controller to send streaming request to sever, to start link.
// @path is /interfaceKey/functionName e.g. /com.apache.dubbo.sample.basic.IGreeter/BigStreamTest
func (t *TripleClient) StreamRequest(ctx context.Context, path string) (grpc.ClientStream, error) {
	if t.h2Controller == nil {
		if err := t.connect(t.url); err != nil {
			logger.Errorf("dubbo client connect to url error = %v", err)
			return nil, err
		}
	}
	return t.h2Controller.StreamInvoke(ctx, path)
}

// Close destroy http controller and return
func (t *TripleClient) Close() {
	logger.Debug("Triple Client Is closing")
	t.h2Controller.Destroy()
}

// IsAvailable returns if ht
func (t *TripleClient) IsAvailable() bool {
	if t.h2Controller == nil {
		return false
	}
	return t.h2Controller.IsAvailable()
}
