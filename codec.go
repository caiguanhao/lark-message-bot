package main

import (
	"encoding/json"
	"io"
	"net/rpc"
	"strings"
)

type (
	Codec struct {
		codec   rpc.ServerCodec
		request *rpc.Request
		isError bool
	}

	JSONRPCRequest struct {
		reader     io.Reader
		readWriter io.ReadWriter
		done       chan bool
	}
)

func (c *Codec) ReadRequestHeader(r *rpc.Request) error {
	c.request = r
	return c.codec.ReadRequestHeader(r)
}

func (c *Codec) ReadRequestBody(x interface{}) error {
	err := c.codec.ReadRequestBody(x)
	b, _ := json.Marshal(x)
	log.Debug("->", c.request.ServiceMethod, "-", strings.TrimSpace(string(b)))
	return err
}

func (c *Codec) WriteResponse(r *rpc.Response, x interface{}) error {
	if r.Error == "" {
		b, _ := json.Marshal(x)
		log.Debug("<-", r.ServiceMethod, "-", strings.TrimSpace(string(b)))
	} else {
		log.Debug("<-", r.ServiceMethod, "-", r.Error)
	}
	return c.codec.WriteResponse(r, x)
}

func (c *Codec) Close() error {
	return c.codec.Close()
}

func (r *JSONRPCRequest) Read(p []byte) (n int, err error) {
	return r.reader.Read(p)
}

func (r *JSONRPCRequest) Write(p []byte) (n int, err error) {
	return r.readWriter.Write(p)
}

func (r *JSONRPCRequest) Close() error {
	r.done <- true
	return nil
}
