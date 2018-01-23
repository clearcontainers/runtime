//
// Copyright (c) 2017 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package virtcontainers

import (
	"context"
	"net"
	"testing"

	"github.com/containers/virtcontainers/pkg/mock"
	gpb "github.com/gogo/protobuf/types"
	pb "github.com/kata-containers/agent/protocols/grpc"
	"google.golang.org/grpc"
)

var testKataProxyURL = "unix:///tmp/kata-proxy-test.sock"

func proxyHandlerDiscard(c net.Conn) {
	buf := make([]byte, 1024)
	c.Read(buf)
}

func TestKataProxyRegister(t *testing.T) {
	proxy := mock.ProxyUnixMock{
		ClientHandler: proxyHandlerDiscard,
	}

	if err := proxy.Start(testKataProxyURL); err != nil {
		t.Fatal(err)
	}

	defer proxy.Stop()

	p := &kataProxy{
		proxyURL: testKataProxyURL,
	}

	_, proxyURL, err := p.register(Pod{})
	if err != nil {
		t.Fatal(err)
	}

	if proxyURL != testKataProxyURL {
		t.Fatalf("Got URL %q, expecting %q", proxyURL, testKataProxyURL)
	}
}

func TestKataProxyConnect(t *testing.T) {
	proxy := mock.ProxyUnixMock{
		ClientHandler: proxyHandlerDiscard,
	}

	if err := proxy.Start(testKataProxyURL); err != nil {
		t.Fatal(err)
	}

	defer proxy.Stop()

	p := &kataProxy{
		proxyURL: testKataProxyURL,
	}

	_, proxyURL, err := p.connect(
		Pod{
			state: State{
				URL: testKataProxyURL,
			},
		},
		false)
	if err != nil {
		t.Fatal(err)
	}

	if proxyURL != testKataProxyURL {
		t.Fatalf("Got URL %q, expecting %q", proxyURL, testKataProxyURL)
	}
}

func TestKataProxyDisconnect(t *testing.T) {
	proxy := mock.ProxyUnixMock{
		ClientHandler: proxyHandlerDiscard,
	}

	if err := proxy.Start(testKataProxyURL); err != nil {
		t.Fatal(err)
	}

	defer proxy.Stop()

	p := &kataProxy{
		proxyURL: testKataProxyURL,
	}

	_, proxyURL, err := p.connect(
		Pod{
			state: State{
				URL: testKataProxyURL,
			},
		},
		false)
	if err != nil {
		t.Fatal(err)
	}

	if proxyURL != testKataProxyURL {
		t.Fatalf("Got URL %q, expecting %q", proxyURL, testKataProxyURL)
	}

	if err := p.disconnect(); err != nil {
		t.Fatal(err)
	}
}

type gRPCProxy struct{}

var emptyResp = &gpb.Empty{}

func (p *gRPCProxy) CreateContainer(ctx context.Context, req *pb.CreateContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) StartContainer(ctx context.Context, req *pb.StartContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) ExecProcess(ctx context.Context, req *pb.ExecProcessRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) SignalProcess(ctx context.Context, req *pb.SignalProcessRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) WaitProcess(ctx context.Context, req *pb.WaitProcessRequest) (*pb.WaitProcessResponse, error) {
	return &pb.WaitProcessResponse{}, nil
}

func (p *gRPCProxy) RemoveContainer(ctx context.Context, req *pb.RemoveContainerRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) WriteStdin(ctx context.Context, req *pb.WriteStreamRequest) (*pb.WriteStreamResponse, error) {
	return &pb.WriteStreamResponse{}, nil
}

func (p *gRPCProxy) ReadStdout(ctx context.Context, req *pb.ReadStreamRequest) (*pb.ReadStreamResponse, error) {
	return &pb.ReadStreamResponse{}, nil
}

func (p *gRPCProxy) ReadStderr(ctx context.Context, req *pb.ReadStreamRequest) (*pb.ReadStreamResponse, error) {
	return &pb.ReadStreamResponse{}, nil
}

func (p *gRPCProxy) CloseStdin(ctx context.Context, req *pb.CloseStdinRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) TtyWinResize(ctx context.Context, req *pb.TtyWinResizeRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) CreateSandbox(ctx context.Context, req *pb.CreateSandboxRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) DestroySandbox(ctx context.Context, req *pb.DestroySandboxRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) AddInterface(ctx context.Context, req *pb.AddInterfaceRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) RemoveInterface(ctx context.Context, req *pb.RemoveInterfaceRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) UpdateInterface(ctx context.Context, req *pb.UpdateInterfaceRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) AddRoute(ctx context.Context, req *pb.RouteRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) RemoveRoute(ctx context.Context, req *pb.RouteRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func (p *gRPCProxy) OnlineCPUMem(ctx context.Context, req *pb.OnlineCPUMemRequest) (*gpb.Empty, error) {
	return emptyResp, nil
}

func gRPCRegister(s *grpc.Server, srv interface{}) {
	switch g := srv.(type) {
	case *gRPCProxy:
		pb.RegisterAgentServiceServer(s, g)
	}
}

func TestKataProxySendCmd(t *testing.T) {
	impl := &gRPCProxy{}

	proxy := mock.ProxyGRPCMock{
		GRPCImplementer: impl,
		GRPCRegister:    gRPCRegister,
	}

	if err := proxy.Start(testKataProxyURL); err != nil {
		t.Fatal(err)
	}

	defer proxy.Stop()

	p := &kataProxy{
		proxyURL: testKataProxyURL,
	}

	_, proxyURL, err := p.connect(
		Pod{
			state: State{
				URL: testKataProxyURL,
			},
		},
		false)
	if err != nil {
		t.Fatal(err)
	}

	if proxyURL != testKataProxyURL {
		t.Fatalf("Got URL %q, expecting %q", proxyURL, testKataProxyURL)
	}

	req := &pb.DestroySandboxRequest{}
	if _, err := p.sendCmd(req); err != nil {
		t.Fatal(err)
	}
}
