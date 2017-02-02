//
// Copyright (c) 2016 Intel Corporation
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

// noopAgent a.k.a. NO-OP Agent is an empty Agent implementation, for testing and
// mocking purposes.
type noopAgent struct {
}

// init initializes the Noop agent, i.e. it does nothing.
func (n *noopAgent) init(pod *Pod, config interface{}) error {
	return nil
}

// start is the Noop agent starting implementation. It does nothing.
func (n *noopAgent) startAgent() error {
	return nil
}

// exec is the Noop agent command execution implementation. It does nothing.
func (n *noopAgent) exec(pod Pod, container Container, cmd Cmd) error {
	return nil
}

// startPod is the Noop agent Pod starting implementation. It does nothing.
func (n *noopAgent) startPod(config PodConfig) error {
	return nil
}

// stopPod is the Noop agent Pod stopping implementation. It does nothing.
func (n *noopAgent) stopPod(pod Pod) error {
	return nil
}

// stop is the Noop agent stopping implementation. It does nothing.
func (n *noopAgent) stopAgent() error {
	return nil
}

// startContainer is the Noop agent Container starting implementation. It does nothing.
func (n *noopAgent) startContainer(pod Pod, contConfig ContainerConfig) error {
	return nil
}

// stopContainer is the Noop agent Container stopping implementation. It does nothing.
func (n *noopAgent) stopContainer(pod Pod, container Container) error {
	return nil
}
