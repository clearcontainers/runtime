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

import (
	"fmt"
	"io/ioutil"
	"strings"
	"syscall"
	"time"

	"golang.org/x/crypto/ssh"
)

// SshdConfig is a structure storing information needed for
// sshd agent initialization.
type SshdConfig struct {
	Username    string
	PrivKeyFile string
	Server      string
	Port        string
	Protocol    string

	Spawner SpawnerType
}

// sshd is an Agent interface implementation for the sshd agent.
type sshd struct {
	config SshdConfig
	client *ssh.Client

	spawner spawner
}

func (c SshdConfig) validate() bool {
	return true
}

func publicKeyAuth(file string) (ssh.AuthMethod, error) {
	if file == "" {
		return nil, ErrNeedFile
	}

	privateBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("Failed to load private key")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse private key")
	}

	return ssh.PublicKeys(private), nil
}

func execCmd(session *ssh.Session, cmd string) error {
	if session == nil {
		return fmt.Errorf("session cannot be empty")
	}

	stdout, err := session.CombinedOutput(cmd)

	if err != nil {
		return fmt.Errorf("Failed to run %s", cmd)
	}

	fmt.Printf("%s\n", stdout)

	return nil
}

// init is the agent initialization implementation for sshd.
func (s *sshd) init(pod *Pod, config interface{}) error {
	if pod == nil {
		return ErrNeedPod
	}

	if config == nil {
		return fmt.Errorf("config cannot be empty")
	}

	c := config.(SshdConfig)
	if c.validate() == false {
		return fmt.Errorf("Invalid configuration")
	}
	s.config = c

	s.spawner = newSpawner(c.Spawner)

	return nil
}

// start is the agent starting implementation for sshd.
func (s *sshd) start(pod *Pod) error {
	if pod == nil {
		return ErrNeedPod
	}

	if s.client != nil {
		session, err := s.client.NewSession()
		if err == nil {
			session.Close()
			return nil
		}
	}

	sshAuthMethod, err := publicKeyAuth(s.config.PrivKeyFile)
	if err != nil {
		return err
	}
	sshConfig := &ssh.ClientConfig{
		User: s.config.Username,
		Auth: []ssh.AuthMethod{
			sshAuthMethod,
		},
	}

	for i := 0; i < 1000; i++ {
		s.client, err = ssh.Dial(s.config.Protocol, s.config.Server+":"+s.config.Port, sshConfig)
		if err == nil {
			break
		}

		select {
		case <-time.After(100 * time.Millisecond):
			break
		}
	}

	if err != nil {
		return fmt.Errorf("Failed to dial: %s", err)
	}

	return nil
}

// stop is the agent stopping implementation for sshd.
func (s *sshd) stop(pod Pod) error {
	return nil
}

// exec is the agent command execution implementation for sshd.
func (s *sshd) exec(pod *Pod, c Container, cmd Cmd) (*Process, error) {
	if pod == nil {
		return nil, ErrNeedPod
	}

	session, err := s.client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session")
	}
	defer session.Close()

	if s.spawner != nil {
		cmd.Args, err = s.spawner.formatArgs(cmd.Args)
		if err != nil {
			return nil, err
		}
	}

	strCmd := strings.Join(cmd.Args, " ")

	return nil, execCmd(session, strCmd)
}

// startPod is the agent Pod starting implementation for sshd.
func (s *sshd) startPod(pod Pod) error {
	return nil
}

// stopPod is the agent Pod stopping implementation for sshd.
func (s *sshd) stopPod(pod Pod) error {
	return nil
}

// createContainer is the agent Container creation implementation for sshd.
func (s *sshd) createContainer(pod *Pod, c *Container) error {
	return nil
}

// startContainer is the agent Container starting implementation for sshd.
func (s *sshd) startContainer(pod Pod, c Container) error {
	return nil
}

// stopContainer is the agent Container stopping implementation for sshd.
func (s *sshd) stopContainer(pod Pod, c Container) error {
	return nil
}

// killContainer is the agent Container signaling implementation for sshd.
func (s *sshd) killContainer(pod Pod, c Container, signal syscall.Signal) error {
	return nil
}
