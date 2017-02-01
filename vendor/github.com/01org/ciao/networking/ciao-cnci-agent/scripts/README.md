# CNCI Image Creation Tools #

## Overview ##

Helper scripts to provision and test CNCI Images

## CNCI Image Provisioning ##

The CNCI Image creation scripts helps you create a CNCI Image from
a clear linux cloud image. Clear cloud images for the CNCI can be obtained from

https://download.clearlinux.org/demos/ciao/

The scripts are used to provision the image with the CNCI Agent and
the certificates it needs to connect to the ciao-scheduler.

1. Place the appropriate certificates under the certs directory

```
	├── certs
	│   ├── CAcert-*.pem
	│   ├── cert-CNCIAgent-*.pem
```

2. Ensure that you have built and installed the cnci agent
```
	cd $GOPATH/src/github.com/01org/ciao/networking/ciao-cnci-agent
   	go install
3. Download the appropriate version image and run the modification script:
```
	cd scripts
	curl -O https://download.clearlinux.org/demos/ciao/clear-${VERSION}-ciao-networking.img.xz
	xz --decompress clear-${VERSION}-ciao-networking.img.xz
	./generate_cnci_cloud_image.sh --image clear-${VERSION}-ciao-networking.img
```

This will yield a provisioned image. This can be used as a CNCI VM.

## CNCI Verification (Optional)##

A simple script to launch the CNCI VM using QEMU and a sample cloud-init
configuration. The cloud-init is setup to check if the CNCI Agent can
be successfully launched within this VM

0. Customize the cloud-init files

```
	├── ciao
	│   └──	ciao.yaml
	├── seed
	│   └── openstack
	│       └── latest
	├── meta_data.json
	│           └── user_data
```

1. Launch the VM (it will cloud-init the image)

```
 sudo ./run_cnci_vm.sh
```
2. Log into the VM using the cloud-init provisioned user/password (default demouser/ciao)
3. Verify the successful launch of the CNCI using
   systemctl status ciao-cnci-agent

An output of the form shown below indicates a successful provisioning of
the agent.

```
demouser@cncihostname ~ $ systemctl status ciao-cnci-agent -l
● ciao-cnci-agent.service - Ciao CNCI Agent
   Loaded: loaded (/usr/lib/systemd/system/ciao-cnci-agent.service; enabled; vendor preset: disabled)
   Active: active (running) since Thu 2016-04-07 20:34:40 UTC; 27s ago
 Main PID: 229 (ciao-cnci-agent)
   CGroup: /system.slice/ciao-cnci-agent.service
           └─229 /usr/sbin/ciao-cnci-agent -server auto -v 3
```

Note: This boot will result in the cloud-init of the image. Hence the original
image generated prior to the verification should be used as the CNCI image.
