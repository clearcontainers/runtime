#!/bin/bash
ciao_host=$(hostname)
ciao_ip=$(hostname -i)
ciao_bridge=ciao_br
ciao_vlan_ip=198.51.100.1
ciao_vlan_subnet=${ciao_vlan_ip}/24
ciao_vlan_brdcast=198.51.100.255
ciao_bin="$HOME/local"
ciao_cert="$ciao_bin""/cert-Scheduler-""$ciao_host"".pem"
keystone_key="$ciao_bin"/keystone_key.pem
keystone_cert="$ciao_bin"/keystone_cert.pem
ui_key=ui_key.pem
ui_cert=ui_cert.pem
workload_sshkey="$ciao_bin"/testkey
ciao_pki_path=/etc/pki/ciao
export no_proxy=$no_proxy,$ciao_vlan_ip,$ciao_host

ciao_email="ciao-devel@lists.clearlinux.org"
ciao_org="Intel"
ciao_src="$GOPATH"/src/github.com/01org/ciao
ciao_gobin="$GOPATH"/bin
ciao_scripts="$GOPATH"/src/github.com/01org/ciao/testutil/singlevm
ciao_env="$ciao_bin/demo.sh"
ciao_dir=/var/lib/ciao
ciao_data_dir=${ciao_dir}/data
ciao_ctl_dir=${ciao_data_dir}/controller
ciao_cnci_image="clear-8260-ciao-networking.img"
ciao_cnci_url="https://download.clearlinux.org/demos/ciao"
fedora_cloud_image="Fedora-Cloud-Base-24-1.2.x86_64.qcow2"
fedora_cloud_url="https://download.fedoraproject.org/pub/fedora/linux/releases/24/CloudImages/x86_64/images/Fedora-Cloud-Base-24-1.2.x86_64.qcow2"
download=0
conf_file="$ciao_bin"/configuration.yaml
webui_conf_file="$ciao_bin"/webui_config.json
ciao_username=csr
ciao_password=hello
ciao_admin_username=admin
ciao_admin_password=giveciaoatry
ciao_demo_username=demo
ciao_demo_password=hello
compute_api_port=8774
storage_api_port=8776
keystone_public_port=5000
keystone_admin_port=35357
mysql_data_dir="${ciao_bin}"/mysql
ciao_identity_url="https://""$ciao_host"":""$keystone_public_port"
keystone_wait_time=60 # How long to wait for keystone to start
ciao_image_wait_time=60 # How long to wait for ciao_image to start

#Create a directory where all the certificates, binaries and other
#dependencies are placed
mkdir -p "$ciao_bin"

if [ ! -d  "$ciao_bin" ]
then
	echo "FATAL ERROR: Unable to create $ciao_bin"
	exit 1
fi

# Variables for ciao binaries
export CIAO_DEMO_PATH="$ciao_bin"
export CIAO_CONTROLLER="$ciao_host"
export CIAO_USERNAME="$ciao_username"
export CIAO_PASSWORD="$ciao_password"
export CIAO_ADMIN_USERNAME="$ciao_admin_username"
export CIAO_ADMIN_PASSWORD="$ciao_admin_password"
export CIAO_CA_CERT_FILE="$ciao_bin"/"CAcert-""$ciao_host"".pem"
export CIAO_IDENTITY="$ciao_identity_url"
export CIAO_SSH_KEY="$workload_sshkey"

# Save these vars for later use, too
> "$ciao_env" # Clean out previous data
set | grep ^CIAO_ | while read VAR; do
    echo export "$VAR" >> "$ciao_env"
done

# Variables for keystone
export OS_USER_DOMAIN_NAME=default
export OS_IMAGE_API_VERSION=2
export OS_PROJECT_NAME=admin
export OS_IDENTITY_API_VERSION=3
export OS_PASSWORD=${ciao_admin_password}
export OS_AUTH_URL=https://"$ciao_host":$keystone_admin_port/v3
export OS_USERNAME=${ciao_admin_username}
export OS_KEY=
export OS_CACERT="$keystone_cert"
export OS_PROJECT_DOMAIN_NAME=default

# Save these vars for later use, too
set | grep ^OS_ | while read VAR; do
    echo export "$VAR" >> "$ciao_env"
done

echo "Subnet =" $ciao_vlan_subnet

# Copy the cleanup scripts
cp "$ciao_scripts"/cleanup.sh "$ciao_bin"

cleanup()
{
    echo "Performing cleanup"
    "$ciao_bin"/cleanup.sh
}

# Ctrl-C Trapper
trap ctrl_c INT

function ctrl_c() {
    echo "Trapped CTRL-C, performing cleanup"
    cleanup
    exit 1
}

usage="$(basename "$0") [--download] The script will download dependencies if needed. Specifying --download will force download the dependencies even if they are cached locally"

while :
do
    case "$1" in
      -c | --cnciurl)
          ciao_cnci_url="$2"
	  shift 2
	  ;;
      -d | --download)
          download=1
          shift 1
          ;;
      -i | --cnciimage)
          ciao_cnci_image="$2"
	  shift 2
	  ;;
      -h | --help)
          echo -e "$usage" >&2
          exit 0
          ;;
      *)
          break
          ;;
    esac
done

set -o nounset

echo "Generating configuration file $conf_file"
(
cat <<-EOF
configure:
  scheduler:
    storage_uri: /etc/ciao/configuration.yaml
  storage:
    ceph_id: ciao
  controller:
    compute_ca: $keystone_cert
    compute_cert: $keystone_key
    identity_user: ${ciao_username}
    identity_password: ${ciao_password}
  image_service:
    type: glance
    url: https://${ciao_host}
  launcher:
    compute_net: [${ciao_vlan_subnet}]
    mgmt_net: [${ciao_vlan_subnet}]
    disk_limit: false
    mem_limit: false
  identity_service:
    type: keystone
    url: https://${ciao_host}:${keystone_admin_port}
EOF
) > $conf_file

echo "Generating webui configuration file $webui_conf_file"
(
cat <<-EOF
{
    "production": {
        "controller": {
            "host": "${ciao_host}",
            "port": "${compute_api_port}",
            "protocol": "https"
        },
        "storage":{
            "host": "${ciao_host}",
            "port": "${storage_api_port}",
            "protocol": "https"
        },
        "keystone": {
            "host": "${ciao_host}",
            "port": "${keystone_admin_port}",
            "protocol": "https",
            "uri": "/v3/auth/tokens"
        },
        "ui": {
            "protocol": "https",
            "certificates": {
                "key": "${ciao_pki_path}/${ui_key}",
                "cert": "${ciao_pki_path}/${ui_cert}",
                "passphrase": "",
                "trusted": []
            }
        }
    },
    "development": {
        "controller": {
            "host": "${ciao_host}",
            "port": "${compute_api_port}",
            "protocol": "https"
        },
        "storage":{
            "host": "${ciao_host}",
            "port": "${storage_api_port}",
            "protocol": "https"
        },
        "keystone": {
            "host": "${ciao_host}",
            "port": "${keystone_admin_port}",
            "protocol": "https",
            "uri": "/v3/auth/tokens"
        },
        "ui": {
            "protocol": "https",
            "certificates": {
                "key": "${ciao_pki_path}/${ui_key}",
                "cert": "${ciao_pki_path}/${ui_cert}",
                "passphrase": "",
                "trusted": []
            }
        }
    }
}
EOF
) > $webui_conf_file


sudo mkdir -p ${ciao_dir}/images
if [ ! -d ${ciao_dir}/images ]
then
	echo "FATAL ERROR: Unable to create $ciao_dir/images"
	exit 1

fi

sudo mkdir -p ${ciao_pki_path}
if [ ! -d ${ciao_pki_path} ]
then
	echo "FATAL ERROR: Unable to create ${ciao_pki_path}"
	exit 1
fi

sudo mkdir -p /etc/ciao/
if [ ! -d /etc/ciao ]
then
	echo "FATAL ERROR: Unable to create /etc/ciao"
	exit 1
fi
sudo install -m 0644 -t /etc/ciao $conf_file

#Stop any running agents and CNCIs
sudo killall ciao-scheduler
sudo killall ciao-controller
sudo killall ciao-launcher
sudo killall qemu-system-x86_64
sudo rm -rf ${ciao_dir}


cd "$ciao_bin"

#Cleanup any old artifacts
rm -f "$ciao_bin"/*.pem

#Build ciao
rm -f "$ciao_gobin"/ciao*
cd "$ciao_src"
go install -v --tags 'debug' ./...

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to build ciao"
	exit 1
fi

cd "$ciao_bin"

#Check if the build was a success
if [ ! -f "$ciao_gobin"/ciao-cli ]
then
	echo "FATAL ERROR: build failed"
	exit 1
fi



#Generate Certificates
"$GOPATH"/bin/ciao-cert -anchor -role scheduler -email="$ciao_email" \
    -organization="$ciao_org" -host="$ciao_host" -ip="$ciao_vlan_ip" -verify

"$GOPATH"/bin/ciao-cert -role cnciagent -anchor-cert "$ciao_cert" \
    -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" \
    -ip="$ciao_vlan_ip" -verify

"$GOPATH"/bin/ciao-cert -role controller -anchor-cert "$ciao_cert" \
    -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" \
    -ip="$ciao_vlan_ip" -verify

"$GOPATH"/bin/ciao-cert -role agent,netagent -anchor-cert "$ciao_cert" \
    -email="$ciao_email" -organization="$ciao_org" -host="$ciao_host" \
    -ip="$ciao_vlan_ip" -verify

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$keystone_key" -out "$keystone_cert" -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=$ciao_host"

#Copy the certs
sudo install -m 0644 -t "$ciao_pki_path" "$keystone_cert"
sudo install -m 0644 -t "$ciao_pki_path" \
    "$ciao_bin"/"cert-Controller-""$ciao_host"".pem"
sudo install -m 0644 -t "$ciao_pki_path" \
    "$ciao_bin"/"CAcert-""$ciao_host"".pem"

if [ ! -f ${ciao_pki_path}/${ui_cert} ] || [ ! -f ${ciao_pki_path}/${ui_key} ]; then
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
	    -keyout ${ciao_bin}/${ui_key} -out ${ciao_bin}/${ui_cert} \
	    -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=localhost"
    sudo install -m 0644 -t "$ciao_pki_path" ${ciao_bin}/${ui_cert} ${ciao_bin}/${ui_key}
fi

#Update system's trusted certificates
cacert_prog_ubuntu=$(type -p update-ca-certificates)
cacert_prog_fedora=$(type -p update-ca-trust)
if [ x"$cacert_prog_ubuntu" != x ] && [ -x "$cacert_prog_ubuntu" ]; then
    cacert_dir=/usr/local/share/ca-certificates
    sudo install -m 0644 -T "$keystone_cert" "$cacert_dir"/keystone.crt
    sudo install -m 0644 -T "$CIAO_CA_CERT_FILE" "$cacert_dir"/ciao.crt
    sudo "$cacert_prog_ubuntu"

    # Do it a second time with nothing new to make it clean out the old
    sudo "$cacert_prog_ubuntu" --fresh
elif [ x"$cacert_prog_fedora" != x ] && [ -x "$cacert_prog_fedora" ]; then
    cacert_dir=/etc/pki/ca-trust/source/anchors/
    if [ -d "$cacert_dir" ]; then
        sudo install -m 0644 -t "$cacert_dir" "$keystone_cert"
        sudo install -m 0644 -t "$cacert_dir" "$CIAO_CA_CERT_FILE"
        sudo "$cacert_prog_fedora" extract
    fi
else
    echo "Unable to add keystone's CA certificate to your system's trusted \
        store!"
    exit 1
fi

#Create controller dirs

sudo mkdir -p ${ciao_ctl_dir}/tables
if [ ! -d ${ciao_ctl_dir}/tables ]
then
	echo "FATAL ERROR: Unable to create ${ciao_ctl_dir}/tables"
	exit 1

fi

sudo mkdir -p ${ciao_ctl_dir}/workloads
if [ ! -d ${ciao_ctl_dir}/workloads ]
then
	echo "FATAL ERROR: Unable to create ${ciao_ctl_dir}/workloads"
	exit 1

fi

#Copy the configuration
cd "$ciao_bin"
sudo cp -a "$ciao_src"/ciao-controller/tables ${ciao_ctl_dir}
sudo cp -a "$ciao_src"/ciao-controller/workloads ${ciao_ctl_dir}

#Over ride the configuration with test specific defaults
sudo cp -f "$ciao_scripts"/workloads/* ${ciao_ctl_dir}/workloads
sudo cp -f "$ciao_scripts"/tables/* ${ciao_ctl_dir}/tables

#Over ride the cloud-init configuration
echo "Generating workload ssh key $workload_sshkey"
rm -f "$workload_sshkey" "$workload_sshkey".pub
ssh-keygen -f "$workload_sshkey" -t rsa -N ''
test_sshkey=$(< "$workload_sshkey".pub)
chmod 600 "$workload_sshkey".pub
#Note: Password is set to ciao
test_passwd='$6$rounds=4096$w9I3hR4g/hu$AnYjaC2DfznbPSG3vxsgtgAS4mJwWBkcR74Y/KHNB5OsfAlA4gpU5j6CHWMOkkt9j.9d7OYJXJ4icXHzKXTAO.'

workload_cloudinit=${ciao_ctl_dir}/workloads/test.yaml
sudo echo "Generating workload cloud-init file $workload_cloudinit"
(
cat <<-EOF
---
#cloud-config
users:
  - name: demouser
    gecos: CIAO Demo User
    lock-passwd: false
    passwd: ${test_passwd}
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh-authorized-keys:
    - ${test_sshkey}
...
EOF
) > $workload_cloudinit



#Copy the launch scripts
cp "$ciao_scripts"/run_scheduler.sh "$ciao_bin"
cp "$ciao_scripts"/run_controller.sh "$ciao_bin"
cp "$ciao_scripts"/run_launcher.sh "$ciao_bin"
cp "$ciao_scripts"/verify.sh "$ciao_bin"

#Download the firmware
cd "$ciao_bin"
if [ $download -eq 1 ] || [ ! -f OVMF.fd ]
then
	rm -f OVMF.fd
	curl -O https://download.clearlinux.org/image/OVMF.fd
fi

if [ ! -f OVMF.fd ]
then
	echo "FATAL ERROR: unable to download firmware"
	exit 1
fi

sudo cp -f OVMF.fd  /usr/share/qemu/OVMF.fd

#Generate the CNCI VM and seed the image and populate the image cache
cd "$ciao_bin"
rm -f "$ciao_cnci_image".qcow

if [ $download -eq 1 ] || [ ! -f "$ciao_cnci_image" ] 
then
	rm -f "$ciao_cnci_image"
	"$GOPATH"/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/generate_cnci_cloud_image.sh -c "$ciao_bin" -i "$ciao_cnci_image" -d -u "$ciao_cnci_url"
else
	"$GOPATH"/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/generate_cnci_cloud_image.sh -c "$ciao_bin" -i "$ciao_cnci_image"
fi

if [ $? -ne 0 ]
then
	echo "FATAL ERROR: Unable to mount CNCI Image"
	exit 1
fi

if [ ! -f "$ciao_cnci_image" ]
then
	echo "FATAL ERROR: unable to download CNCI Image"
	exit 1
fi

qemu-img convert -f raw -O qcow2 "$ciao_cnci_image" "$ciao_cnci_image".qcow

#Clear
cd "$ciao_bin"
LATEST=$(curl https://download.clearlinux.org/latest)

if [ $download -eq 1 ] || [ ! -f clear-"${LATEST}"-cloud.img ] 
then
	rm -f clear-"${LATEST}"-cloud.img.xz
	rm -f clear-"${LATEST}"-cloud.img
	curl -O https://download.clearlinux.org/releases/"$LATEST"/clear/clear-"$LATEST"-cloud.img.xz
	xz -T0 --decompress clear-"${LATEST}"-cloud.img.xz
fi


if [ ! -f clear-"${LATEST}"-cloud.img ]
then
	echo "FATAL ERROR: unable to download clear cloud Image"
	exit 1
fi

#Fedora, needed for BAT tests
cd "$ciao_bin"
if [ $download -eq 1 ] || [ ! -f $fedora_cloud_image ]
then
    rm -f $fedora_cloud_image
    curl -L -O $fedora_cloud_url
fi

if [ ! -f $fedora_cloud_image ]
then
	echo "FATAL ERROR: unable to download fedora cloud Image"
	exit 1
fi

# Set macvlan interface
if [ -x "$(command -v ip)" ]; then
    sudo ip link del "$ciao_bridge"
    sudo ip link add name "$ciao_bridge" type bridge
    sudo iptables -A FORWARD -p all -i "$ciao_bridge" -j ACCEPT
    sudo ip link add link "$ciao_bridge" name ciaovlan type macvlan mode bridge
    sudo ip addr add "$ciao_vlan_subnet" brd "$ciao_vlan_brdcast" dev ciaovlan
    sudo ip link set dev ciaovlan up
    sudo ip -d link show ciaovlan
    sudo ip link set dev "$ciao_bridge" up
    sudo ip -d link show "$ciao_bridge"
else
    echo 'ip command is not supported'
fi

# Set DHCP server with dnsmasq
sudo mkdir -p /var/lib/misc
if [ -x "$(command -v ip)" ]; then
    sudo dnsmasq -C $ciao_scripts/dnsmasq.conf.ciaovlan \
	 --pid-file=/tmp/dnsmasq.ciaovlan.pid
else
    echo 'dnsmasq command is not supported'
fi

# Generate the post-keystone script to issue singlevm-specific openstack
# commands
( cat <<-EOF
#!/bin/bash

# Create basic services, users, and projects/tenants
openstack service create --name ciao compute
openstack user create --password "$ciao_password" "$ciao_username"
openstack role add --project service --user "$ciao_username" admin
openstack user create --password "$ciao_demo_password" "$ciao_demo_username"
openstack project show demo
if [[ \$? == 1 ]]; then
    openstack project create --domain default demo
fi
openstack role add --project demo --user "$ciao_demo_username" user

# Create image service endpoints
openstack service create --name glance --description "Image Service" image
openstack endpoint create --region RegionOne image public   https://$ciao_host:9292
openstack endpoint create --region RegionOne image internal https://$ciao_host:9292
openstack endpoint create --region RegionOne image admin    https://$ciao_host:9292

# admin should only be admin of the admin project. This role was created by the
# keystone container's bootstrap.
openstack role remove --project service --user admin admin

# Create storage endpoints
openstack service create --name cinderv2 --description "Volume Service" volumev2
openstack endpoint create --region RegionOne volumev2 public   'https://$ciao_host:8776/v2/%(tenant_id)s'
openstack endpoint create --region RegionOne volumev2 internal 'https://$ciao_host:8776/v2/%(tenant_id)s'
openstack endpoint create --region RegionOne volumev2 admin    'https://$ciao_host:8776/v2/%(tenant_id)s'

EOF
) > "$ciao_bin"/post-keystone.sh
chmod 755 "$ciao_bin"/post-keystone.sh

## Install keystone
sudo docker run -d -it --name keystone \
    --add-host="$ciao_host":"$ciao_ip" \
    -p $keystone_public_port:5000 \
    -p $keystone_admin_port:35357 \
    -e IDENTITY_HOST="$ciao_host" -e KEYSTONE_ADMIN_PASSWORD="${OS_PASSWORD}" \
    -v "$ciao_bin"/post-keystone.sh:/usr/bin/post-keystone.sh \
    -v $mysql_data_dir:/var/lib/mysql \
    -v "$keystone_cert":/etc/nginx/ssl/keystone_cert.pem \
    -v "$keystone_key":/etc/nginx/ssl/keystone_key.pem clearlinux/keystone

echo -n "Waiting up to $keystone_wait_time seconds for keystone identity" \
    "service to become available"
try_until=$(($(date +%s) + $keystone_wait_time))
while : ; do
    while [ $(date +%s) -le $try_until ]; do
        # The keystone container tails the log at the end of its
        # initialization script
        if docker exec keystone pidof tail > /dev/null 2>&1; then
            echo READY
            break 2
        else
            echo -n .
            sleep 1
        fi
    done
    echo FAILED
    break
done

# Install ceph
# This runs *after* keystone so keystone will get port 5000 first
sudo docker run --name ceph-demo -d --net=host -v /etc/ceph:/etc/ceph -e MON_IP=$ciao_vlan_ip -e CEPH_PUBLIC_NETWORK=$ciao_vlan_subnet ceph/demo
sudo ceph auth get-or-create client.ciao -o /etc/ceph/ceph.client.ciao.keyring mon 'allow *' osd 'allow *' mds 'allow'

#Kick off the agents
cd "$ciao_bin"
"$ciao_bin"/run_scheduler.sh  &> /dev/null
"$ciao_bin"/run_launcher.sh   &> /dev/null
"$ciao_bin"/run_controller.sh &> /dev/null

. $ciao_env

echo -n "Waiting up to $ciao_image_wait_time seconds for the ciao image" \
    "service to become available "
try_until=$(($(date +%s) + $ciao_image_wait_time))
while : ; do
    while [ $(date +%s) -le $try_until ]; do
        if ciao-cli image list > /dev/null 2>&1; then
            echo " READY"
            break 2
        else
            echo -n .
            sleep 1
        fi
    done
    echo FAILED
    break
done

echo ""
echo "Uploading test images to image service"
echo "---------------------------------------------------------------------------------------"
if [ -f "$ciao_cnci_image".qcow ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file "$ciao_cnci_image".qcow \
        --name "ciao CNCI image" --id 4e16e743-265a-4bf2-9fd1-57ada0b28904
fi

if [ -f clear-"${LATEST}"-cloud.img ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file clear-"${LATEST}"-cloud.img \
        --name "Clear Linux ${LATEST}" --id df3768da-31f5-4ba6-82f0-127a1a705169
fi

if [ -f $fedora_cloud_image ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file $fedora_cloud_image \
        --name "Fedora Cloud Base 24-1.2" --id 73a86d7e-93c0-480e-9c41-ab42f69b7799
fi

echo "---------------------------------------------------------------------------------------"
echo ""
echo "Your ciao development environment has been initialised."
echo "To get started run:"
echo ""
echo ". ~/local/demo.sh"
echo ""
echo "Verify the cluster is working correctly by running"
echo ""
echo "~/local/verify.sh"
echo ""
echo "Use ciao-cli to manipulate and inspect the cluster, e.g., "
echo ""
echo "ciao-cli instance add --workload=ab68111c-03a6-11e6-87de-001320fb6e31 --instances=1"
echo ""
echo "When you're finished run the following command to cleanup"
echo ""
echo "~/local/cleanup.sh"
