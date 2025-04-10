/*
Copyright 2025 Flant JSC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"fmt"
	"os"

	connlib "github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/rpc"
)

type Probe struct{}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: rpcprobe <unix_socket_path>")
		fmt.Println("Example: rpcprobe /tmp/rpc.sock")
		os.Exit(1)
	}

	socketPath := os.Args[1]

	ctx := context.Background()
	csiConn, err := connlib.Connect(ctx, socketPath, nil)
	if err != nil {
		fmt.Printf("Failed to establish connection to CSI driver: %v", err)
		os.Exit(1)
	}
	defer csiConn.Close()

	fmt.Printf("Calling CSI driver to discover driver name\n")

   	methodCtx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	csiDriverName, err := rpc.GetDriverName(methodCtx, csiConn)
	if err != nil {
		fmt.Printf("Failed to get CSI driver name: %v", err)
		os.Exit(1)
	}
	fmt.Printf("CSI driver name %s", csiDriverName)
	fmt.Printf("CSI driver name %s\n", csiDriverName)
	os.Exit(0)
}
