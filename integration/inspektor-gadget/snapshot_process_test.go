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
	"testing"

	snapshotprocessTypes "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/snapshot/process/types"

	. "github.com/inspektor-gadget/inspektor-gadget/integration"
)

func TestSnapshotProcess(t *testing.T) {
	ns := GenerateTestNamespaceName("test-snapshot-process")

	t.Parallel()

	// TODO: Handle it once we support getting container image name from docker
	isDockerRuntime := IsDockerRuntime(t)

	commandsPreTest := []*Command{
		CreateTestNamespaceCommand(ns),
		BusyboxPodCommand(ns, "nc -l -p 9090"),
		WaitUntilTestPodReadyCommand(ns),
	}
	RunTestSteps(commandsPreTest, t, WithCbBeforeCleanup(PrintLogsFn(ns)))

	t.Cleanup(func() {
		commandsPostTest := []*Command{
			DeleteTestNamespaceCommand(ns),
		}
		RunTestSteps(commandsPostTest, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
	})

	nodeName := GetPodNode(t, ns, "test-pod")

	commands := []*Command{
		{
			Name: "RunProcessCollectorGadget",
			Cmd:  fmt.Sprintf("$KUBECTL_GADGET snapshot process -n %s -o json --node %s", ns, nodeName),
			ValidateOutput: func(t *testing.T, output string) {
				expectedEntry := &snapshotprocessTypes.Event{
					Event:   BuildBaseEventK8s(ns, WithContainerImageName("docker.io/library/busybox:latest", isDockerRuntime)),
					Command: "nc",
				}
				expectedEntry.K8s.Node = nodeName

				normalize := func(e *snapshotprocessTypes.Event) {
					e.Pid = 0
					e.Tid = 0
					e.ParentPid = 0
					e.MountNsID = 0

					// TODO: Verify container runtime and container name
					e.Runtime.RuntimeName = ""
					e.Runtime.ContainerName = ""
					e.Runtime.ContainerID = ""
					e.Runtime.ContainerImageDigest = ""
				}

				ExpectEntriesInArrayToMatch(t, output, normalize, expectedEntry)
			},
		},
	}
	RunTestSteps(commands, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
}
