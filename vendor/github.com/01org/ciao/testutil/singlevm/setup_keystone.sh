openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
    -keyout "$keystone_key" -out "$keystone_cert" -subj "/C=US/ST=CA/L=Santa Clara/O=ciao/CN=$ciao_host"

#Copy the certs
sudo install -m 0644 -t "$ciao_pki_path" "$keystone_cert"

#Update system's trusted certificates
cacert_prog_ubuntu=$(type -p update-ca-certificates)
cacert_prog_fedora=$(type -p update-ca-trust)
if [ x"$cacert_prog_ubuntu" != x ] && [ -x "$cacert_prog_ubuntu" ]; then
    cacert_dir=/usr/local/share/ca-certificates
    sudo install -m 0644 -T "$keystone_cert" "$cacert_dir"/keystone.crt
    sudo "$cacert_prog_ubuntu"

    # Do it a second time with nothing new to make it clean out the old
    sudo "$cacert_prog_ubuntu" --fresh
elif [ x"$cacert_prog_fedora" != x ] && [ -x "$cacert_prog_fedora" ]; then
    cacert_dir_fedora=/etc/pki/ca-trust/source/anchors/
    cacert_dir_archlinux=/etc/ca-certificates/trust-source/anchors
    cacert_dir=""

    if [ -d "$cacert_dir_fedora" ]; then
	cacert_dir=$cacert_dir_fedora
    elif [ -d "$cacert_dir_archlinux" ]; then
	cacert_dir=$cacert_dir_archlinux
    fi
    
    if [ -d "$cacert_dir" ]; then
        sudo install -m 0644 -t "$cacert_dir" "$keystone_cert"
        sudo "$cacert_prog_fedora" extract
    else
	echo "Unable to add keystone's CA certificate to your system's trusted \
             store!"
	exit 1
    fi
else
    echo "Unable to add keystone's CA certificate to your system's trusted \
        store!"
    exit 1
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
    -v "$keystone_key":/etc/nginx/ssl/keystone_key.pem clearlinux/keystone:stable

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