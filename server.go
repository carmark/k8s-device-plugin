/*
 * Copyright (c) 2019, FAKE-TEST.  All rights reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	"k8s.io/klog"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

const (
	envDisableHealthChecks = "DP_DISABLE_HEALTHCHECKS"
	allHealthChecks        = "xids"
)

// FakeDevicePlugin implements the Kubernetes device plugin API
type FakeDevicePlugin struct {
	name   string
	devs   []*pluginapi.Device
	socket string

	stop   chan interface{}
	health chan *pluginapi.Device

	server *grpc.Server
}

// NewFakeDevicePlugin returns an initialized NvidiaDevicePlugin
func NewFakeDevicePlugin(rName string, num int) *FakeDevicePlugin {
	devs := []*pluginapi.Device{}
	for i := 0; i < num; i++ {
		devs = append(devs, &pluginapi.Device{
			ID:     fmt.Sprintf("%s-%d", rName, i),
			Health: pluginapi.Healthy,
		})
	}
	return &FakeDevicePlugin{
		devs:   devs,
		name:   rName,
		socket: filepath.Join(pluginapi.DevicePluginPath, strings.ReplaceAll(rName, "/", "-")+"-fake.sock"),

		stop:   make(chan interface{}),
		health: make(chan *pluginapi.Device),
	}
}

func (m *FakeDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{}, nil
}

// dial establishes the gRPC communication with the registered device plugin.
func dial(unixSocketPath string, timeout time.Duration) (*grpc.ClientConn, error) {
	c, err := grpc.Dial(unixSocketPath, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(timeout),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}),
	)

	if err != nil {
		return nil, err
	}

	return c, nil
}

// Start starts the gRPC server of the device plugin
func (m *FakeDevicePlugin) Start() error {
	err := m.cleanup()
	if err != nil {
		klog.Error(err)
		return err
	}

	sock, err := net.Listen("unix", m.socket)
	if err != nil {
		klog.Error(err)
		return err
	}

	m.server = grpc.NewServer([]grpc.ServerOption{}...)
	pluginapi.RegisterDevicePluginServer(m.server, m)

	go func() {
		lastCrashTime := time.Now()
		restartCount := 0
		for {
			klog.Infof("Starting GRPC server")
			err := m.server.Serve(sock)
			if err != nil {
				klog.Infof("GRPC server crashed with error: %v", err)
			}
			// restart if it has not been too often
			// i.e. if server has crashed more than 5 times and it didn't last more than one hour each time
			if restartCount > 5 {
				// quit
				klog.Fatal("GRPC server has repeatedly crashed recently. Quitting")
			}
			timeSinceLastCrash := time.Since(lastCrashTime).Seconds()
			lastCrashTime = time.Now()
			if timeSinceLastCrash > 3600 {
				// it has been one hour since the last crash.. reset the count
				// to reflect on the frequency
				restartCount = 1
			} else {
				restartCount += 1
			}
		}
	}()

	// Wait for server to start by launching a blocking connection
	conn, err := dial(m.socket, 5*time.Second)
	if err != nil {
		klog.Error(err)
		return err
	}
	conn.Close()

	return nil
}

// Stop stops the gRPC server
func (m *FakeDevicePlugin) Stop() error {
	if m.server == nil {
		return nil
	}

	m.server.Stop()
	m.server = nil
	close(m.stop)

	return m.cleanup()
}

// Register registers the device plugin for the given resourceName with Kubelet.
func (m *FakeDevicePlugin) Register(kubeletEndpoint, resourceName string) error {
	conn, err := dial(kubeletEndpoint, 5*time.Second)
	if err != nil {
		klog.Error(err)
		return err
	}
	defer conn.Close()

	client := pluginapi.NewRegistrationClient(conn)
	reqt := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     path.Base(m.socket),
		ResourceName: resourceName,
	}

	_, err = client.Register(context.Background(), reqt)
	if err != nil {
		klog.Error(err)
		return err
	}
	return nil
}

// ListAndWatch lists devices and update that list according to the health status
func (m *FakeDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})

	for {
		select {
		case <-m.stop:
			return nil
		case d := <-m.health:
			// FIXME: there is no way to recover from the Unhealthy state.
			d.Health = pluginapi.Unhealthy
			klog.Infof("device marked unhealthy: %s", d.ID)
			s.Send(&pluginapi.ListAndWatchResponse{Devices: m.devs})
		}
	}
}

func (m *FakeDevicePlugin) unhealthy(dev *pluginapi.Device) {
	m.health <- dev
}

// Allocate which return list of devices.
func (m *FakeDevicePlugin) Allocate(ctx context.Context, reqs *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	responses := pluginapi.AllocateResponse{}
	for _, req := range reqs.ContainerRequests {
		response := pluginapi.ContainerAllocateResponse{
			Envs: map[string]string{
				"NVIDIA_VISIBLE_DEVICES": strings.Join(req.DevicesIDs, ","),
			},
		}
		responses.ContainerResponses = append(responses.ContainerResponses, &response)
	}

	return &responses, nil
}

func (m *FakeDevicePlugin) PreStartContainer(context.Context, *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	return &pluginapi.PreStartContainerResponse{}, nil
}

func (m *FakeDevicePlugin) cleanup() error {
	if err := os.Remove(m.socket); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (m *FakeDevicePlugin) healthcheck() {
}

// Serve starts the gRPC server and register the device plugin to Kubelet
func (m *FakeDevicePlugin) Serve() error {
	err := m.Start()
	if err != nil {
		klog.Infof("Could not start device plugin: %s", err)
		return err
	}
	klog.Infof("Starting to serve on %v", m.socket)

	err = m.Register(pluginapi.KubeletSocket, m.name)
	if err != nil {
		klog.Infof("Could not register device plugin: %s", err)
		m.Stop()
		return err
	}
	klog.Infof("Registered device plugin with Kubelet")

	return nil
}
