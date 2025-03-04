// Copyright 2019-2022 The Inspektor Gadget authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"syscall"
	"testing"

	. "github.com/inspektor-gadget/inspektor-gadget/integration"
	toptcpTypes "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/top/tcp/types"
	eventtypes "github.com/inspektor-gadget/inspektor-gadget/pkg/types"
)

func newTopTCPCmd(ns string, cmd string, startAndStop bool, isDockerRuntime bool) *Command {
	validateOutputFn := func(t *testing.T, output string) {
		expectedEntry := &toptcpTypes.Stats{
			CommonData: BuildCommonDataK8s(ns, WithContainerImageName("docker.io/library/nginx:latest", isDockerRuntime)),
			Comm:       "curl",
			IPVersion:  syscall.AF_INET,
			SrcEndpoint: eventtypes.L4Endpoint{
				L3Endpoint: eventtypes.L3Endpoint{
					Addr:    "127.0.0.1",
					Version: 4,
					Kind:    eventtypes.EndpointKindRaw,
				},
			},
			DstEndpoint: eventtypes.L4Endpoint{
				L3Endpoint: eventtypes.L3Endpoint{
					Addr:    "127.0.0.1",
					Version: 4,
					Kind:    eventtypes.EndpointKindRaw,
				},
				Port: 80,
			},
		}

		normalize := func(e *toptcpTypes.Stats) {
			e.MountNsID = 0
			e.Pid = 0
			e.SrcEndpoint.Port = 0
			e.Sent = 0
			e.Received = 0

			e.K8s.Node = ""
			// TODO: Verify container runtime and container name
			e.Runtime.RuntimeName = ""
			e.Runtime.ContainerName = ""
			e.Runtime.ContainerID = ""
			e.Runtime.ContainerImageDigest = ""
		}

		ExpectEntriesInMultipleArrayToMatch(t, output, normalize, expectedEntry)
	}

	return &Command{
		Name:           "TopTCP",
		ValidateOutput: validateOutputFn,
		Cmd:            cmd,
		StartAndStop:   startAndStop,
	}
}

func TestTopTcp(t *testing.T) {
	t.Parallel()
	ns := GenerateTestNamespaceName("test-top-tcp")

	// TODO: Handle it once we support getting container image name from docker
	isDockerRuntime := IsDockerRuntime(t)

	commandsPreTest := []*Command{
		CreateTestNamespaceCommand(ns),
		PodCommand("test-pod", "nginx", ns, "[sh, -c]", "nginx && while true; do curl 127.0.0.1; sleep 0.1; done"),
		WaitUntilTestPodReadyCommand(ns),
	}
	RunTestSteps(commandsPreTest, t, WithCbBeforeCleanup(PrintLogsFn(ns)))

	t.Cleanup(func() {
		commandsPostTest := []*Command{
			DeleteTestNamespaceCommand(ns),
		}
		RunTestSteps(commandsPostTest, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
	})

	t.Run("StartAndStop", func(t *testing.T) {
		t.Parallel()

		cmd := fmt.Sprintf("$KUBECTL_GADGET top tcp -n %s -o json", ns)
		topTCPCmd := newTopTCPCmd(ns, cmd, true, isDockerRuntime)
		RunTestSteps([]*Command{topTCPCmd}, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
	})

	t.Run("Timeout", func(t *testing.T) {
		t.Parallel()

		cmd := fmt.Sprintf("$KUBECTL_GADGET top tcp -n %s -o json -m 100 --timeout %d", ns, topTimeoutInSeconds)
		topTCPCmd := newTopTCPCmd(ns, cmd, false, isDockerRuntime)
		RunTestSteps([]*Command{topTCPCmd}, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
	})

	t.Run("Interval=Timeout", func(t *testing.T) {
		t.Parallel()

		cmd := fmt.Sprintf("$KUBECTL_GADGET top tcp -n %s -o json -m 100 --timeout %d --interval %d", ns, topTimeoutInSeconds, topTimeoutInSeconds)
		topTCPCmd := newTopTCPCmd(ns, cmd, false, isDockerRuntime)
		RunTestSteps([]*Command{topTCPCmd}, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
	})
}
