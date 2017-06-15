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

if [ $all_images -eq 1 ]
then
    cd "$ciao_bin"
    if [ $download -eq 1 ]
    then
	LATEST=$(curl https://download.clearlinux.org/latest)
    else
	# replace this will a function that looks in ~local cache
	# for the last version of clear to use.
	LATEST="12620"
    fi

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
fi

#Ubuntu, needed for kubicle and BAT tests
cd "$ciao_bin"
if [ $download -eq 1 ] || [ ! -f $ubuntu_cloud_image ]
then
    rm -f $ubuntu_cloud_image
    curl -L -O $ubuntu_cloud_url
fi

if [ ! -f $ubuntu_cloud_image ]
then
	echo "FATAL ERROR: unable to download Ubuntu cloud Image"
	exit 1
fi

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
        --name "ciao CNCI image" --id 4e16e743-265a-4bf2-9fd1-57ada0b28904 \
	--visibility internal
fi

if [ $all_images -eq 1 ]
then
    if [ -f clear-"${LATEST}"-cloud.img ]; then
	"$ciao_gobin"/ciao-cli \
		     image add --file clear-"${LATEST}"-cloud.img \
		     --name "Clear Linux ${LATEST}" --id df3768da-31f5-4ba6-82f0-127a1a705169 \
		     --visibility public
    fi

    if [ -f $fedora_cloud_image ]; then
	"$ciao_gobin"/ciao-cli \
		     image add --file $fedora_cloud_image \
		     --name "Fedora Cloud Base 24-1.2" --id 73a86d7e-93c0-480e-9c41-ab42f69b7799 \
		     --visibility public
    fi
fi

if [ -f $ubuntu_cloud_image ]; then
    "$ciao_gobin"/ciao-cli \
        image add --file $ubuntu_cloud_image \
        --name "Ubuntu Server 16.04" \
	--visibility public
fi
