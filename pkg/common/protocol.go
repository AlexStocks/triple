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

package common

import (
	"context"
	"fmt"
)

import (
	common "github.com/dubbogo/gost/dubbogo"
	logger "github.com/dubbogo/gost/dubbogo/logger"
	netTriple "github.com/dubbogo/net/http2/triple"

	perrors "github.com/pkg/errors"
)

type ProtocolHeaderHandlerFactory func(url *common.URL, ctx context.Context) netTriple.ProtocolHeaderHandler

var protocolHeaderHandlerFactoryMap = make(map[string]ProtocolHeaderHandlerFactory)

func GetProtocolHeaderHandler(protocol string, url *common.URL, ctx context.Context) (netTriple.ProtocolHeaderHandler, error) {
	if f, ok := protocolHeaderHandlerFactoryMap[protocol]; ok {
		return f(url, ctx), nil
	}
	logger.Error("Protocol ", protocol, " header undefined!")
	return nil, perrors.New(fmt.Sprintf("Protocol %s header undefined!", protocol))
}

func SetProtocolHeaderHandler(protocol string, factory ProtocolHeaderHandlerFactory) {
	protocolHeaderHandlerFactoryMap[protocol] = factory
}

// PackageHandler is to handle http framedata and raw data
type PackageHandler interface {
	Frame2PkgData(frameData []byte) ([]byte, uint32)
	Pkg2FrameData(pkgData []byte) []byte
}

type PackageHandlerFactory func() PackageHandler

var packageHandlerFactoryMap = make(map[string]PackageHandlerFactory, 8)

func GetPackagerHandler(protocol string) (PackageHandler, error) {
	if f, ok := packageHandlerFactoryMap[protocol]; ok {
		return f(), nil
	}
	logger.Error("Protocol ", protocol, " package handler undefined!")
	return nil, perrors.New(fmt.Sprintf("Protocol %s package handler undefined!", protocol))
}

func SetPackageHandler(protocol string, f PackageHandlerFactory) {
	packageHandlerFactoryMap[protocol] = f
}

// Dubbo3Serializer
type Dubbo3Serializer interface {
	MarshalRequest(interface{}) ([]byte, error)
	UnmarshalRequest(data []byte, v interface{}) error
	MarshalResponse(interface{}) ([]byte, error)
	UnmarshalResponse(data []byte, v interface{}) error
}

type SerializerFactory func() Dubbo3Serializer

var dubbo3SerializerMap = make(map[string]SerializerFactory)

func GetDubbo3Serializer(serialization TripleSerializerName) (Dubbo3Serializer, error) {
	if f, ok := dubbo3SerializerMap[string(serialization)]; ok {
		return f(), nil
	}
	logger.Error("Serilization ", serialization, " factory undefined!")
	return nil, perrors.New(fmt.Sprintf("Serilization %sfactory undefined!", serialization))
}

func SetDubbo3Serializer(serialization TripleSerializerName, f SerializerFactory) {
	dubbo3SerializerMap[string(serialization)] = f
}
