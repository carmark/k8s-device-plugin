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
	"flag"
	"os"
	"syscall"

	"github.com/fsnotify/fsnotify"
	"k8s.io/klog"
	pluginapi "k8s.io/kubernetes/pkg/kubelet/apis/deviceplugin/v1beta1"
)

func main() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagResourceName := flag.String("resource-name", "nvidia.com/gpu", "Define the default resource name: nvidia.com/gpu.")
	flagResourceNum := flag.Int("resource-num", 8, "Define the default resource number: 8.")
	klog.InitFlags(flag.CommandLine)
	flag.Parse()
	defer klog.Flush()

	klog.Infof("Fetching devices.")

	klog.Infof("Starting FS watcher.")
	watcher, err := newFSWatcher(pluginapi.DevicePluginPath)
	if err != nil {
		klog.Infof("Failed to created FS watcher.")
		os.Exit(1)
	}
	defer watcher.Close()

	klog.Infof("Starting OS watcher.")
	sigs := newOSWatcher(syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	restart := true
	var devicePlugin *FakeDevicePlugin

L:
	for {
		if restart {
			if devicePlugin != nil {
				devicePlugin.Stop()
			}

			devicePlugin = NewFakeDevicePlugin(*flagResourceName, *flagResourceNum)
			if err := devicePlugin.Serve(); err != nil {
				klog.Infof("Could not contact Kubelet, retrying. Did you enable the device plugin feature gate?")
			} else {
				restart = false
			}
		}

		select {
		case event := <-watcher.Events:
			if event.Name == pluginapi.KubeletSocket && event.Op&fsnotify.Create == fsnotify.Create {
				klog.Infof("inotify: %s created, restarting.", pluginapi.KubeletSocket)
				restart = true
			}

		case err := <-watcher.Errors:
			klog.Infof("inotify: %v", err)

		case s := <-sigs:
			switch s {
			case syscall.SIGHUP:
				klog.Infof("Received SIGHUP, restarting.")
				restart = true
			default:
				klog.Infof("Received signal \"%v\", shutting down.", s)
				devicePlugin.Stop()
				break L
			}
		}
	}
}
