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
	"bytes"
	h2Triple "github.com/dubbogo/net/http2/triple"
)
import (
	dubboCommon "github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"google.golang.org/grpc"
)
import (
	"github.com/dubbogo/triple/internal/status"
	"github.com/dubbogo/triple/pkg/common"
)

////////////////////////////////Buffer and MsgType

// MsgType show the type of Message in buffer
type MsgType uint8

const (
	DataMsgType              = MsgType(1)
	ServerStreamCloseMsgType = MsgType(2)
)

// BufferMsg is the basic transfer unit in one stream
type BufferMsg struct {
	buffer  *bytes.Buffer
	msgType MsgType
	st      *status.Status
	err     error
}

// GetMsgType can get buffer's type
func (bm *BufferMsg) GetMsgType() MsgType {
	return bm.msgType
}

// MsgBuffer contain the chan of BufferMsg
type MsgBuffer struct {
	c   chan BufferMsg
	err error
}

func newRecvBuffer() *MsgBuffer {
	b := &MsgBuffer{
		c: make(chan BufferMsg, 0),
	}
	return b
}

func (b *MsgBuffer) put(r BufferMsg) {
	b.c <- r
}

func (b *MsgBuffer) get() <-chan BufferMsg {
	return b.c
}

/////////////////////////////////stream state

type streamState uint32

const (
	open      = streamState(0)
	halfClose = streamState(1)
	closed    = streamState(2)
)

/////////////////////////////////stream
// stream is not only a buffer stream
// but an abstruct stream in h2 defination
type stream interface {
	// channel usage
	putRecv(data []byte, msgType MsgType)
	putSend(data []byte, msgType MsgType)
	putRecvErr(err error)
	getSend() <-chan BufferMsg
	getRecv() <-chan BufferMsg
	putSplitedDataRecv(splitedData []byte, msgType MsgType, handler common.PackageHandler)
	close()
	closeWithError(err error)

	// state usage
	getState() streamState
	setState(ss streamState)
	onRecvEs() streamState
}

// baseStream is the basic  impl of stream interface, it impl for basic function of stream
type baseStream struct {
	recvBuf *MsgBuffer
	sendBuf *MsgBuffer
	url     *dubboCommon.URL
	header  h2Triple.ProtocolHeader
	service Dubbo3GrpcService
	// splitBuffer is used to cache splited data from network, if exceed
	splitBuffer BufferMsg
	// fromFrameHeaderDataSize is got from dataFrame's header, which is 5 bytes and contains the total data size
	// of this package
	// when fromFrameHeaderDataSize is zero, its means we should parse header first 5byte, and then read data
	fromFrameHeaderDataSize uint32

	facadeStream stream

	// On client-side it is the status error received from the server.
	// On server-side it is unused.
	status *status.Status

	state streamState
}

func (s *baseStream) WriteStatus(st *status.Status) {
	s.sendBuf.put(BufferMsg{
		st:      st,
		msgType: ServerStreamCloseMsgType,
	})
}

func (s *baseStream) putRecv(data []byte, msgType MsgType) {
	s.recvBuf.put(BufferMsg{
		buffer:  bytes.NewBuffer(data),
		msgType: msgType,
	})
}

func (s *baseStream) putRecvErr(err error) {
	//s.recvBuf.put(BufferMsg{
	//	err:     err,
	//	msgType: ServerStreamCloseMsgType,
	//})
}

// putSplitedDataRecv is called when receive from tripleNetwork, dealing with big package partial to create the whole pkg
// @msgType Must be data
func (s *baseStream) putSplitedDataRecv(splitedData []byte, msgType MsgType, frameHandler common.PackageHandler) {
	if msgType != DataMsgType {
		return
	}
	if s.fromFrameHeaderDataSize == 0 {
		// should parse data frame header first
		var totalSize uint32
		if splitedData, totalSize = frameHandler.Frame2PkgData(splitedData); totalSize == 0 {
			return
		} else {
			s.fromFrameHeaderDataSize = totalSize
		}
		s.splitBuffer.buffer.Reset()
	}
	s.splitBuffer.buffer.Write(splitedData)
	if s.splitBuffer.buffer.Len() > int(s.fromFrameHeaderDataSize) {
		panic("Receive Splited Data is bigger than wanted!!!")
		return
	}

	if s.splitBuffer.buffer.Len() == int(s.fromFrameHeaderDataSize) {
		s.putRecv(frameHandler.Pkg2FrameData(s.splitBuffer.buffer.Bytes()), msgType)
		s.splitBuffer.buffer.Reset()
		s.fromFrameHeaderDataSize = 0
	}
}

func (s *baseStream) putSend(data []byte, msgType MsgType) {
	s.sendBuf.put(BufferMsg{
		buffer:  bytes.NewBuffer(data),
		msgType: msgType,
	})
}

// getRecv get channel of receiving message
// in server end, when unary call, msg from client is send to recvChan, and then it is read and push to processor to get response.
// in client end, when unary call, msg from server is send to recvChan, and then response in invoke method.
/*
client  ---> send chan ---> triple ---> recv Chan ---> processor
			sendBuf						recvBuf			   |
		|	clientStream |          | serverStream |       |
			recvBuf						sendBuf			   V
client <--- recv chan <--- triple <--- send chan <---  response
*/
func (s *baseStream) getRecv() <-chan BufferMsg {
	return s.recvBuf.get()
}

func (s *baseStream) getSend() <-chan BufferMsg {
	return s.sendBuf.get()
}

func (s *baseStream) getState() streamState {
	return s.state
}

func (s *baseStream) setState(ss streamState) {
	logger.Debug("setState, set to close")
	s.state = ss
}

func (s *baseStream) onRecvEs() streamState {
	if s.state == open {
		logger.Debug("onRecvEs, change to half Close")
		s.state = halfClose
	} else if s.state == halfClose {
		logger.Debug("onRecvEs, change to closed")
		s.state = closed
	}
	return s.state
}

func (s *baseStream) closeWithError(err error) {
	//s.putRecvErr(err)
	//if s.facadeStream != nil {
	//	s.facadeStream.close()
	//}
}

func newBaseStream(streamID uint32, service Dubbo3GrpcService) *baseStream {
	// stream and pkgHeader are the same level
	return &baseStream{
		recvBuf: newRecvBuffer(),
		sendBuf: newRecvBuffer(),
		service: service,
		state:   open,
		splitBuffer: BufferMsg{
			buffer: bytes.NewBuffer(make([]byte, 0)),
		},
	}
}

// serverStream is running in server end
type serverStream struct {
	baseStream
	processor processor
	header    h2Triple.ProtocolHeader
}

// todo close logic
func (ss *serverStream) close() {
	//	ss.processor.close()
	//	// if buffer not close, it will block client end, which is waiting for <- chan
	//	close(ss.sendBuf.c)
	//	close(ss.recvBuf.c)
}

func newServerStream(header h2Triple.ProtocolHeader, desc interface{}, url *dubboCommon.URL, service Dubbo3GrpcService) (*serverStream, error) {
	baseStream := newBaseStream(header.GetStreamID(), service)

	serverStream := &serverStream{
		baseStream: *baseStream,
		header:     header,
	}
	serverStream.baseStream.facadeStream = serverStream
	pkgHandler, err := common.GetPackagerHandler(url.Protocol)
	if err != nil {
		logger.Error("GetPkgHandler error with err = ", err)
		return nil, err
	}
	if methodDesc, ok := desc.(grpc.MethodDesc); ok {
		// pkgHandler and processor are the same level
		serverStream.processor, err = newUnaryProcessor(serverStream, pkgHandler, methodDesc)
	} else if streamDesc, ok := desc.(grpc.StreamDesc); ok {
		serverStream.processor, err = newStreamingProcessor(serverStream, pkgHandler, streamDesc)
	} else {
		logger.Error("grpc desc invalid:", desc)
		return nil, nil
	}

	serverStream.processor.runRPC()

	return serverStream, nil
}

func (s *serverStream) getService() Dubbo3GrpcService {
	return s.service
}

func (s *serverStream) getHeader() h2Triple.ProtocolHeader {
	return s.header
}

// clientStream is running in client end
type clientStream struct {
	baseStream
	closeChan chan struct{}
}

func newClientStream(streamID uint32) *clientStream {
	baseStream := newBaseStream(streamID, nil)
	newclientStream := &clientStream{
		baseStream: *baseStream,
		closeChan:  make(chan struct{}, 1),
	}
	newclientStream.baseStream.facadeStream = newclientStream
	return newclientStream
}

// todo close logic
func (cs *clientStream) close() {
	//cs.closeChan <- struct{}{}
	//// if buffer not close, it will block client end, which is waiting for <- chan
	//close(cs.sendBuf.c)
	//close(cs.recvBuf.c)
}
