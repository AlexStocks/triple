package config

import (
	"github.com/dubbogo/triple/pkg/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewTripleOption(t *testing.T) {
	opt := NewTripleOption()
	assert.NotNil(t, opt)
}

func TestWithClientTimeout(t *testing.T) {
	opt := NewTripleOption(
		WithClientTimeout(120),
	)
	assert.NotNil(t, opt)
	assert.Equal(t, opt.Timeout, uint32(120))
}

func TestWithSerializerType(t *testing.T) {
	opt := NewTripleOption(
		WithSerializerType(common.HessianSerializerName),
	)
	assert.NotNil(t, opt)
	assert.Equal(t, opt.SerializerType, common.HessianSerializerName)

	opt = NewTripleOption(
		WithSerializerType(common.PBSerializerName),
	)
	assert.NotNil(t, opt)
	assert.Equal(t, opt.SerializerType, common.PBSerializerName)
}

func TestWithBufferSize(t *testing.T) {
	opt := NewTripleOption(
		WithBufferSize(100000),
	)
	assert.NotNil(t, opt)
	assert.Equal(t, opt.BufferSize, uint32(100000))
}

func TestOption_SetEmptyFieldDefaultConfig(t *testing.T) {
	opt := NewTripleOption(
		WithBufferSize(100000),
	)
	assert.NotNil(t, opt)
	assert.Equal(t, uint32(100000), opt.BufferSize)
	assert.Equal(t, uint32(0), opt.Timeout)
	opt.SetEmptyFieldDefaultConfig()
	assert.Equal(t, uint32(100000), opt.BufferSize)
	assert.Equal(t, uint32(common.DefaultTimeout), opt.Timeout)

	opt = NewTripleOption()
	assert.Equal(t, uint32(0), opt.BufferSize)
	assert.Equal(t, uint32(0), opt.Timeout)
	opt.SetEmptyFieldDefaultConfig()
	assert.Equal(t, uint32(common.DefaultHttp2ControllerReadBufferSize), opt.BufferSize)
	assert.Equal(t, uint32(common.DefaultTimeout), opt.Timeout)
}
