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

package main

const userDataTemplate = `
{{- define "ENV" -}}
{{if len .HTTPSProxy }}https_proxy={{.HTTPSProxy}} {{end -}}
{{if len .HTTPProxy }}http_proxy={{.HTTPProxy}} {{end -}}
{{- print "DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true " -}}
{{end}}
{{- define "CHECK" -}}
if [ $? -eq 0 ] ; then ret="OK" ; else ret="FAIL" ; fi ; curl -X PUT -d $ret 10.0.2.2:{{.HTTPServerPort -}}
{{end -}}
{{- define "OK" -}}
curl -X PUT -d "OK" 10.0.2.2:{{.HTTPServerPort -}}
{{end -}}
#cloud-config
mounts:
 - [hostgo, {{.GoPath}}, 9p, "x-systemd.automount,x-systemd.device-timeout=10,nofail,trans=virtio,version=9p2000.L", "0", "0"]
{{if len .UIPath }} - [hostui, {{.UIPath}}, 9p, "x-systemd.automount,x-systemd.device-timeout=10,nofail,trans=virtio,version=9p2000.L", "0", "0"]{{end}}
write_files:
{{- if len $.HTTPProxy }}
 - content: |
     [Service]
     Environment="HTTP_PROXY={{$.HTTPProxy}}"{{if len .HTTPSProxy}} "HTTPS_PROXY={{.HTTPSProxy}}{{end}}"{{if len .NoProxy}} "NO_PROXY={{.NoProxy}},singlevm{{end}}"
   path: /etc/systemd/system/docker.service.d/http-proxy.conf
{{- end}}
 - content: |
     #!/bin/sh
     printf "\n"
     printf "To run Single VM:\n"
     printf "\n"
     printf "cd {{.GoPath}}/src/github.com/01org/ciao/testutil/singlevm\n"
     printf "./setup.sh\n"
     printf "\n"
     printf "To start the web-ui:\n"
     printf "\n"
     printf "cd {{if len .UIPath}}{{.UIPath}}{{else}}/home/{{.User}}/ciao-webui{{end}}\n"
     printf "./deploy.sh production --config_file=/home/{{.User}}/local/webui_config.json\n"
     printf "\n"
     printf "Point your host's browser at https://localhost:3000"
   path: /etc/update-motd.d/10-ciao-help-text
   permissions: '0755'
 - content: |
     deb https://apt.dockerproject.org/repo ubuntu-xenial main
   path: /etc/apt/sources.list.d/docker.list

apt:
{{- if len $.HTTPProxy }}
  proxy: "{{$.HTTPProxy}}"
{{- end}}
{{- if len $.HTTPSProxy }}
  https_proxy: "{{$.HTTPSProxy}}"
{{- end}}
package_upgrade: true

runcmd:
 - echo "127.0.0.1 singlevm" >> /etc/hosts
 - mount hostgo
{{if len .UIPath }} - mount hostui{{end}}
 - chown {{.User}}:{{.User}} /home/{{.User}}
 - rm /etc/update-motd.d/10-help-text /etc/update-motd.d/51-cloudguest
 - rm /etc/update-motd.d/90-updates-available
 - rm /etc/legal
 - curl -X PUT -d "Booting VM" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "OK" .}}
{{if len $.HTTPProxy }}
 - echo "HTTP_PROXY=\"{{.HTTPProxy}}\"" >> /etc/environment
 - echo "http_proxy=\"{{.HTTPProxy}}\"" >> /etc/environment
{{end -}}
{{- if len $.HTTPSProxy }}
 - echo "HTTPS_PROXY=\"{{.HTTPSProxy}}\"" >> /etc/environment
 - echo "https_proxy=\"{{.HTTPSProxy}}\"" >> /etc/environment
{{end}}
{{- if or (len .HTTPSProxy) (len .HTTPProxy) }}
 - echo "no_proxy=\"{{if len .NoProxy}}{{.NoProxy}},{{end}}singlevm\""  >> /etc/environment
{{end}}

 - echo "GOPATH=\"{{.GoPath}}\"" >> /etc/environment
 - echo "PATH=\"$PATH:/usr/local/go/bin:{{$.GoPath}}/bin:/usr/local/nodejs/bin\""  >> /etc/environment

 - curl -X PUT -d "Downloading Go" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}wget https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz -O /tmp/go1.7.4.linux-amd64.tar.gz
 - {{template "CHECK" .}}
 - curl -X PUT -d "Unpacking Go" 10.0.2.2:{{.HTTPServerPort}}
 - tar -C /usr/local -xzf /tmp/go1.7.4.linux-amd64.tar.gz
 - {{template "CHECK" .}}
 - rm /tmp/go1.7.4.linux-amd64.tar.gz

 - groupadd docker
 - sudo gpasswd -a {{.User}} docker
 - curl -X PUT -d "Installing apt-transport-https and ca-certificates" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install apt-transport-https ca-certificates
 - {{template "CHECK" .}}

 - curl -X PUT -d "Add docker GPG key" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
 - {{template "CHECK" .}}

 - curl -X PUT -d "Retrieving updated list of packages" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get update
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Docker" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install docker-engine -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing GCC" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install gcc -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Make" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install make -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing QEMU" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install qemu-system-x86 -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing xorriso" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install xorriso -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing ceph-common" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install ceph-common -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Openstack client" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install python-openstackclient -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Updating NodeJS sources" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}curl -sL https://deb.nodesource.com/setup_7.x | sudo -E bash -
 - {{template "CHECK" .}}
 - curl -X PUT -d "Installing NodeJS" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install nodejs -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Auto removing unused components" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get auto-remove -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Building ciao" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} {{template "ENV" .}} GOPATH={{.GoPath}} /usr/local/go/bin/go get github.com/01org/ciao/...
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Go development utils" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} {{template "ENV" .}} GOPATH={{.GoPath}} /usr/local/go/bin/go get github.com/fzipp/gocyclo github.com/gordonklaus/ineffassign github.com/golang/lint/golint github.com/client9/misspell/cmd/misspell
 - {{template "CHECK" .}}

 - chown {{.User}}:{{.User}} -R {{.GoPath}}

 - curl -X PUT -d "Retrieving ciao-webui " 10.0.2.2:{{.HTTPServerPort}}
{{ if len .UIPath }}
 - cd {{.UIPath}}
 - git status || sudo -u {{.User}} {{template "ENV" .}} git clone https://github.com/01org/ciao-webui.git .
 - {{template "CHECK" .}}
{{else }}
 - cd /home/{{.User}}
 - sudo -u {{.User}} {{template "ENV" .}} git clone https://github.com/01org/ciao-webui.git
 - {{template "CHECK" .}}
{{end}}

{{if len .HTTPProxy}} - sudo -u {{.User}} npm config set proxy {{.HTTPProxy}}{{end}}
{{if len .NodeHTTPSProxy}} - sudo -u {{.User}} npm config set https-proxy {{.NodeHTTPSProxy}}{{end}}

 - mkdir -p /usr/local/nodejs
 - chown {{.User}}:{{.User}} /usr/local/nodejs
 - sudo -u {{.User}} npm config set prefix '/usr/local/nodejs'

 - curl -X PUT -d "Pulling ceph/demo" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}} docker pull ceph/demo
 - {{template "CHECK" .}}

 - curl -X PUT -d "Pulling clearlinux/keystone" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}} docker pull clearlinux/keystone:stable
 - {{template "CHECK" .}}

 - mkdir -p /home/{{.User}}/local

 - curl -X PUT -d "Downloading Fedora-Cloud-Base-24-1.2.x86_64.qcow2" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}wget https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2 -O /home/{{.User}}/local/Fedora-Cloud-Base-24-1.2.x86_64.qcow2
 - {{template "CHECK" .}}

 - curl -X PUT -d "Downloading CNCI image" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}wget https://download.clearlinux.org/demos/ciao/clear-8260-ciao-networking.img.xz -O /home/{{.User}}/local/clear-8260-ciao-networking.img.xz
 - {{template "CHECK" .}}

 - curl -X PUT -d "Downloading latest clear cloud image" 10.0.2.2:{{.HTTPServerPort}}
 - LATEST=$({{template "ENV" .}} curl -s https://download.clearlinux.org/latest) &&  {{template "ENV" .}} wget https://download.clearlinux.org/releases/"$LATEST"/clear/clear-"$LATEST"-cloud.img.xz -O /home/{{.User}}/local/clear-"$LATEST"-cloud.img.xz
 - {{template "CHECK" .}}

 - cd /home/{{.User}}/local && xz -T0 --decompress *.xz

 - chown {{.User}}:{{.User}} -R /home/{{.User}}/local
{{if len .GitUserName}}
 - curl -X PUT -d "Setting git user.name" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} git config --global user.name "{{.GitUserName}}"
 - {{template "CHECK" .}}
{{end}}
{{if len .GitEmail}}
 - curl -X PUT -d "Setting git user.email" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} git config --global user.email {{.GitEmail}}
 - {{template "CHECK" .}}
{{end}}
%s
 - curl -X PUT -d "FINISHED" 10.0.2.2:{{.HTTPServerPort}}

users:
  - name: {{.User}}
    gecos: CIAO Demo User
    lock-passwd: true
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - {{.PublicKey}}
`

const metaDataTemplate = `
{
  "uuid": "ddb4a2de-e5a5-4107-b302-e845cecd7613",
  "hostname": "singlevm"
}
`

/* TODO: For now create a new template for clear containers
 * in the future we need this to be a more dynamic avoid
 * duplication
 */
const ccUserDataTemplate = `
{{- define "ENV" -}}
{{if len .HTTPSProxy }}https_proxy={{.HTTPSProxy}} {{end -}}
{{if len .HTTPProxy }}http_proxy={{.HTTPProxy}} {{end -}}
{{print "DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true " -}}
{{end}}
{{- define "CHECK" -}}
if [ $? -eq 0 ] ; then ret="OK" ; else ret="FAIL" ; fi ; curl -X PUT -d $ret 10.0.2.2:{{.HTTPServerPort -}}
{{end -}}
{{- define "OK" -}}
curl -X PUT -d "OK" 10.0.2.2:{{.HTTPServerPort -}}
{{end -}}
#cloud-config
mounts:
 - [hostgo, {{.GoPath}}, 9p, "x-systemd.automount,x-systemd.device-timeout=10,nofail,trans=virtio,version=9p2000.L", "0", "0"]
write_files:
{{- if len $.HTTPProxy }}
 - content: |
     [Service]
     Environment="HTTP_PROXY={{$.HTTPProxy}}"{{if len .HTTPSProxy}} "HTTPS_PROXY={{.HTTPSProxy}}{{end}}"{{if len .NoProxy}} "NO_PROXY={{.NoProxy}},singlevm{{end}}"
   path: /etc/systemd/system/docker.service.d/http-proxy.conf
{{- end}}
 - content: |
     [Service]
     ExecStart=
     ExecStart=/usr/bin/dockerd -D --add-runtime cor=/usr/bin/cc-oci-runtime --default-runtime=cor
   path: /etc/systemd/system/docker.service.d/clr-containers.conf
 - content: |
     #!/bin/sh
     printf "\n"
     printf "\n"
     printf "Your go code is at {{.GoPath}}\n"
     printf "You can also edit your code on your host system"
     printf "\n"
     printf "\n"
   path: /etc/update-motd.d/10-ciao-help-text
   permissions: '0755'
 - content: |
     deb https://apt.dockerproject.org/repo ubuntu-xenial main
   path: /etc/apt/sources.list.d/docker.list

apt:
{{- if len $.HTTPProxy }}
  proxy: "{{$.HTTPProxy}}"
{{- end}}
{{- if len $.HTTPSProxy }}
  https_proxy: "{{$.HTTPSProxy}}"
{{- end}}
package_upgrade: true

runcmd:
 - echo "127.0.0.1 singlevm" >> /etc/hosts
 - mount hostgo
 - chown {{.User}}:{{.User}} /home/{{.User}}
 - rm /etc/update-motd.d/10-help-text /etc/update-motd.d/51-cloudguest
 - rm /etc/update-motd.d/90-updates-available
 - rm /etc/legal
 - curl -X PUT -d "Booting VM" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "OK" .}}
{{if len $.HTTPProxy }}
 - echo "HTTP_PROXY=\"{{.HTTPProxy}}\"" >> /etc/environment
 - echo "http_proxy=\"{{.HTTPProxy}}\"" >> /etc/environment
{{end -}}
{{- if len $.HTTPSProxy }}
 - echo "HTTPS_PROXY=\"{{.HTTPSProxy}}\"" >> /etc/environment
 - echo "https_proxy=\"{{.HTTPSProxy}}\"" >> /etc/environment
{{end}}
{{- if or (len .HTTPSProxy) (len .HTTPProxy) }}
 - echo "no_proxy=\"{{if len .NoProxy}}{{.NoProxy}},{{end}}singlevm\""  >> /etc/environment
{{end}}

 - echo "GOPATH=\"{{.GoPath}}\"" >> /etc/environment
 - echo "PATH=\"$PATH:/usr/local/go/bin:{{$.GoPath}}/bin:/usr/local/nodejs/bin\""  >> /etc/environment

 - curl -X PUT -d "Downloading Go" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}wget https://storage.googleapis.com/golang/go1.7.4.linux-amd64.tar.gz -O /tmp/go1.7.4.linux-amd64.tar.gz
 - {{template "CHECK" .}}
 - curl -X PUT -d "Unpacking Go" 10.0.2.2:{{.HTTPServerPort}}
 - tar -C /usr/local -xzf /tmp/go1.7.4.linux-amd64.tar.gz
 - {{template "CHECK" .}}
 - rm /tmp/go1.7.4.linux-amd64.tar.gz

 - groupadd docker
 - sudo gpasswd -a {{.User}} docker
 - curl -X PUT -d "Installing apt-transport-https and ca-certificates" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install apt-transport-https ca-certificates
 - {{template "CHECK" .}}

 - curl -X PUT -d "Add docker GPG key" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-key adv --keyserver hkp://p80.pool.sks-keyservers.net:80 --recv-keys 58118E89F3A912897C070ADBF76221572C52609D
 - {{template "CHECK" .}}

 - curl -X PUT -d "Add Clear Containers OBS Repository " 10.0.2.2:{{.HTTPServerPort}}
 - sudo sh -c "echo 'deb http://download.opensuse.org/repositories/home:/clearlinux:/preview:/clear-containers-2.1/xUbuntu_16.04/ /' >> /etc/apt/sources.list.d/cc-oci-runtime.list"
 - {{template "ENV" .}}curl -fsSL http://download.opensuse.org/repositories/home:clearlinux:preview:clear-containers-2.1/xUbuntu_16.04/Release.key | sudo apt-key add -
 - {{template "CHECK" .}}

 - curl -X PUT -d "Retrieving updated list of packages" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get update
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Clear Containers Runtime" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install cc-oci-runtime -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Docker" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install docker-engine -y
 - {{template "CHECK" .}}


 - curl -X PUT -d "Start Clear Containers Runtime" 10.0.2.2:{{.HTTPServerPort}}
 - sudo systemctl daemon-reload
 - sudo systemctl restart docker
 - sudo systemctl enable cc-proxy.socket
 - sudo systemctl start cc-proxy.socket
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing GCC" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install gcc -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Make" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install make -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing QEMU" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install qemu-system-x86 -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing xorriso" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get install xorriso -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Auto removing unused components" 10.0.2.2:{{.HTTPServerPort}}
 - {{template "ENV" .}}apt-get auto-remove -y
 - {{template "CHECK" .}}

 - curl -X PUT -d "Installing Go development utils" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} {{template "ENV" .}} GOPATH={{.GoPath}} /usr/local/go/bin/go get github.com/mattn/goveralls golang.org/x/tools/cmd/cover github.com/pierrre/gotestcover github.com/fzipp/gocyclo github.com/gordonklaus/ineffassign github.com/golang/lint/golint github.com/client9/misspell/cmd/misspell github.com/01org/ciao/test-cases github.com/opencontainers/runc/libcontainer/configs
 - {{template "CHECK" .}}

 - chown {{.User}}:{{.User}} -R {{.GoPath}}

{{if len .GitUserName}}
 - curl -X PUT -d "Setting git user.name" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} git config --global user.name "{{.GitUserName}}"
 - {{template "CHECK" .}}
{{end}}
{{if len .GitEmail}}
 - curl -X PUT -d "Setting git user.email" 10.0.2.2:{{.HTTPServerPort}}
 - sudo -u {{.User}} git config --global user.email {{.GitEmail}}
 - {{template "CHECK" .}}
{{end}}
%s
 - curl -X PUT -d "FINISHED" 10.0.2.2:{{.HTTPServerPort}}

users:
  - name: {{.User}}
    gecos: CIAO Demo User
    lock-passwd: true
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - {{.PublicKey}}
`
