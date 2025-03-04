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

	traceexecTypes "github.com/inspektor-gadget/inspektor-gadget/pkg/gadgets/trace/exec/types"

	. "github.com/inspektor-gadget/inspektor-gadget/integration"
)

func TestTraceExec(t *testing.T) {
	ns := GenerateTestNamespaceName("test-trace-exec")

	t.Parallel()

	// TODO: Handle it once we support getting container image name from docker
	isDockerRuntime := IsDockerRuntime(t)

	cmd := "cp /bin/date /date ; setuidgid 1000:1111 sh -c 'while true; do /date ; /bin/sleep 0.1; done'"
	shArgs := []string{"/bin/sh", "-c", cmd}
	dateArgs := []string{"/date"}
	sleepArgs := []string{"/bin/sleep", "0.1"}

	traceExecCmd := &Command{
		Name:         "StartTraceExecGadget",
		Cmd:          fmt.Sprintf("$KUBECTL_GADGET trace exec -n %s -o json --cwd", ns),
		StartAndStop: true,
		ValidateOutput: func(t *testing.T, output string) {
			expectedEntries := []*traceexecTypes.Event{
				{
					Event: BuildBaseEventK8s(ns, WithContainerImageName("docker.io/library/busybox:latest", isDockerRuntime)),
					Comm:  "sh",
					Args:  shArgs,
					Cwd:   "/",
				},
				{
					Event:      BuildBaseEventK8s(ns, WithContainerImageName("docker.io/library/busybox:latest", isDockerRuntime)),
					Comm:       "date",
					Args:       dateArgs,
					Uid:        1000,
					Gid:        1111,
					Cwd:        "/",
					UpperLayer: true,
				},
				{
					Event: BuildBaseEventK8s(ns, WithContainerImageName("docker.io/library/busybox:latest", isDockerRuntime)),
					Comm:  "sleep",
					Args:  sleepArgs,
					Uid:   1000,
					Gid:   1111,
					Cwd:   "/",
				},
			}

			normalize := func(e *traceexecTypes.Event) {
				e.Timestamp = 0
				e.Pid = 0
				e.Ppid = 0
				e.LoginUid = 0
				e.SessionId = 0
				e.Retval = 0
				e.MountNsID = 0

				e.K8s.Node = ""
				// TODO: Verify container runtime and container name
				e.Runtime.RuntimeName = ""
				e.Runtime.ContainerName = ""
				e.Runtime.ContainerID = ""
				e.Runtime.ContainerImageDigest = ""
			}

			ExpectEntriesToMatch(t, output, normalize, expectedEntries...)
		},
	}

	commands := []*Command{
		CreateTestNamespaceCommand(ns),
		traceExecCmd,
		// Give time to kubectl-gadget to start the tracer
		SleepForSecondsCommand(3),
		BusyboxPodCommand(ns, cmd),
		WaitUntilTestPodReadyCommand(ns),
		DeleteTestNamespaceCommand(ns),
	}

	RunTestSteps(commands, t, WithCbBeforeCleanup(PrintLogsFn(ns)))
}
