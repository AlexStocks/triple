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

package config

import "github.com/dubbogo/triple/pkg/common"

type Option struct {
	Timeout        uint32
	BufferSize     uint32
	SerializerType common.TripleSerializerName
}

// SetEmptyFieldDefaultConfig set empty field to default config
func (o *Option) SetEmptyFieldDefaultConfig() {
	if o.Timeout == uint32(0) {
		o.Timeout = uint32(common.DefaultTimeout)
	}

	if o.BufferSize == uint32(0) {
		o.BufferSize = uint32(common.DefaultHttp2ControllerReadBufferSize)
	}

	if o.SerializerType == "" {
		o.SerializerType = common.PBSerializerName
	}
}

type OptionFunction func(o *Option) *Option

// NewTripleOption return Triple Option with given config defined by @fs
func NewTripleOption(fs ...OptionFunction) *Option {
	opt := &Option{}
	for _, v := range fs {
		opt = v(opt)
	}
	return opt
}

// WithClientTimeout return OptionFunction with timeout of @timeout
func WithClientTimeout(timeout uint32) OptionFunction {
	return func(o *Option) *Option {
		o.Timeout = timeout
		return o
	}
}

// WithBufferSize return OptionFunction with buffer read size of @size
func WithBufferSize(size uint32) OptionFunction {
	return func(o *Option) *Option {
		o.BufferSize = size
		return o
	}
}

// WithSerializerType return OptionFunction with target @serializerType, now we support "protobuf" and "hessian2"
func WithSerializerType(serializerType common.TripleSerializerName) OptionFunction {
	return func(o *Option) *Option {
		o.SerializerType = serializerType
		return o
	}
}
