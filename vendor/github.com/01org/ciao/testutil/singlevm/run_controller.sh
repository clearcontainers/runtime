#!/bin/bash
ciao_host=$(hostname)
ciao_dir=/var/lib/ciao
ciao_data_dir=${ciao_dir}/data
ciao_ctl_dir=${ciao_data_dir}/controller

#Create controller dirs
echo "Making ciao workloads dir: ${ciao_ctl_dir}/workloads"
sudo mkdir -p ${ciao_ctl_dir}/workloads
if [ ! -d ${ciao_ctl_dir}/workloads ]
then
	echo "FATAL ERROR: Unable to create ${ciao_ctl_dir}/workloads}"
	exit 1
fi

sudo rm ciao-controller.db-shm ciao-controller.db-wal ciao-controller.db /tmp/ciao-controller-stats.db

sudo "$GOPATH"/bin/ciao-controller \
    --cacert="$CIAO_DEMO_PATH"/CAcert-"$ciao_host".pem \
    --cert="$CIAO_DEMO_PATH"/cert-Controller-"$ciao_host".pem --v 3 &
