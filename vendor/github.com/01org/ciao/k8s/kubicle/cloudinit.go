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

package main

const workloadTemplate = `description: "{{.Description}}"
vm_type: qemu
fw_type: legacy
defaults:
    vcpus: {{.VCPUs}}
    mem_mb: {{.RAMMiB}}
cloud_init: "{{.UserDataFile}}"
disks:
  - source:
       service: image
       id: "{{.ImageUUID}}"
    size: {{.DiskGiB}}
    ephemeral: true
    bootable: true
`

const udCommonTemplate = `{{- define "PROXIES" -}}
{{if len .HTTPSProxy }}https_proxy={{.HTTPSProxy}} {{end -}}
{{if len .HTTPProxy }}http_proxy={{.HTTPProxy}} {{end -}}
{{end -}}
---
#cloud-config
{{- if len (or $.HTTPProxy $.HTTPSProxy "")}}
write_files:
 - content: |
     [Service]
     Environment={{if len .HTTPProxy}}"HTTP_PROXY={{$.HTTPProxy}}" {{end}}{{if len .HTTPSProxy}}"HTTPS_PROXY={{.HTTPSProxy}}{{end}}"{{if len .NoProxy}} "NO_PROXY={{.NoProxy}}{{end}}"
   path: /etc/systemd/system/docker.service.d/http-proxy.conf
{{- end}}

apt:
{{- if len $.HTTPProxy }}
  proxy: "{{$.HTTPProxy}}"
{{- end}}
{{- if len $.HTTPSProxy }}
  https_proxy: "{{$.HTTPSProxy}}"
{{- end}}

users:
  - name: {{.User}}
    gecos: k8s Demo User
    groups: docker
    lock-passwd: true
    shell: /bin/bash
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
      - {{.PublicKey}}

runcmd:
 - systemctl restart networking`

const udMasterTemplate = `
 - echo "export KUBECONFIG=/home/{{.User}}/admin.conf" > /home/{{.User}}/.bash_aliases
{{- if len .HTTPSProxy }}
 - echo "export https_proxy={{.HTTPSProxy}}" >> /home/{{.User}}/.bash_aliases
{{- end}}
{{- if len .HTTPProxy }}
 - echo "export http_proxy={{.HTTPProxy}}" >> /home/{{.User}}/.bash_aliases
{{- end}}
 - echo "export no_proxy={{if len .NoProxy}}{{.NoProxy}},{{end}}` + "`hostname -i`" + `" >> /home/{{.User}}/.bash_aliases
 - chown {{.User}}:{{.User}} /home/{{.User}}/.bash_aliases
 - apt-get update && apt-get install -y apt-transport-https
 - {{template "PROXIES" .}}curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
 - echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" >/etc/apt/sources.list.d/kubernetes.list
 - apt-get update
 - DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y docker-engine
 - DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y kubelet kubeadm kubectl kubernetes-cni
 - {{template "PROXIES" .}}no_proxy=` + "`hostname -i`" + ` kubeadm init --pod-network-cidr 10.244.0.0/16 --token {{.Token}} {{if len .ExternalIP}}--apiserver-cert-extra-sans={{.ExternalIP}}{{end}}
 - cp /etc/kubernetes/admin.conf /home/{{.User}}/
 - chown {{.User}}:{{.User}} /home/{{.User}}/admin.conf
 - {{template "PROXIES" .}}wget https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel-rbac.yml
 - KUBECONFIG=/home/{{.User}}/admin.conf kubectl create -f kube-flannel-rbac.yml
 - {{template "PROXIES" .}}wget https://raw.githubusercontent.com/coreos/flannel/master/Documentation/kube-flannel.yml
 - KUBECONFIG=/home/{{.User}}/admin.conf kubectl create --namespace kube-system -f kube-flannel.yml
 - cat /home/{{.User}}/admin.conf | sed -E 's/\/\/[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+/\/\/{{.ExternalIP}}/' | curl -T - {{.PhoneHomeIP}}:9000
...
`

const udNodeTemplate = `
 - DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get update && apt-get install -y apt-transport-https
 - {{template "PROXIES" .}} curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
 - echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" >/etc/apt/sources.list.d/kubernetes.list
 - apt-get update
 - DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y docker-engine
 - DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true apt-get install -y kubelet kubeadm kubectl kubernetes-cni
 - kubeadm join --token {{.Token}} {{.MasterIP}}:6443
...
`
