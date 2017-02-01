#!/bin/bash
#Defaults
image="clear-8260-ciao-networking.img"
certs_dir=$GOPATH/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/certs
cnci_agent=$GOPATH/bin/ciao-cnci-agent
cnci_sysd=$GOPATH/src/github.com/01org/ciao/networking/ciao-cnci-agent/scripts/ciao-cnci-agent.service
base_url="https://download.clearlinux.org/demos/ciao"
partition="2"
download=0

usage="$(basename "$0") [--image clear_cnci_image_name] [-certs certificate_directory] [-agent cnci_agent_binary] [-script cnci_systemd_script] \n\n A simple script to create a CNCI Image from a clear cloud image. \n Defaults for any unspecified option are as follows \n\n --agent $cnci_agent \n --certs $certs_dir \n --image $image \n --script $cnci_sysd\n\n"


while :
do
    case "$1" in
      -a | --agent)
	  cnci_agent="$2"
	  shift 2
	  ;;
      -c | --certs)
	  certs_dir="$2"
	  shift 2
	  ;;
      -d | --download)
	  download=1
	  shift 1
	  ;;
      -h | --help)
	  echo -e "$usage" >&2
	  exit 0
	  ;;
      -i | --image)
	  image="$2"
	  shift 2
	  ;;
      -p | --partition)
	  partition="$2"
	  shift 2
	  ;;
      -s | --script)
	  cnci_sysd="$2"
	  shift 2
	  ;;
      -u | --url)
	  base_url="$2"
	  shift 2
	  ;;
      *)
	  break
	  ;;
    esac
done

set -o nounset

if [ $download -eq 1 ]
then
	rm -f "$image"
	curl -O "$base_url"/"$image".xz
	unxz "$image".xz
fi

echo -e "\nMounting image: $image"
tmpdir=$(mktemp -d)

loop=`sudo losetup -f --show -P $image`
sudo udevadm settle
sudo mount ${loop}p$partition "$tmpdir"

# simplistic sanity check...most any linux image rootfs successfully mounted
# will have a /usr directory
if [ ! -e $tmpdir/usr ]; then
	echo "Unable to mount CNCI Image"
	return 1
fi

echo -e "Cleaning up any artifacts"
sudo rm -rf "$tmpdir"/var/lib/ciao

echo -e "Copying agent image"
sudo cp "$cnci_agent" "$tmpdir"/usr/sbin/

echo -e "Copying agent systemd service script"
sudo cp "$cnci_sysd" "$tmpdir"/usr/lib/systemd/system/

echo -e "Installing the service"
sudo mkdir -p "$tmpdir"/etc/systemd/system/default.target.wants
sudo rm -f "$tmpdir"/etc/systemd/system/default.target.wants/ciao-cnci-agent.service
sudo chroot "$tmpdir" /bin/bash -c "sudo ln -s /usr/lib/systemd/system/ciao-cnci-agent.service /etc/systemd/system/default.target.wants/"

echo -e "Copying CA certificates"
sudo mkdir -p "$tmpdir"/var/lib/ciao/
sudo cp "$certs_dir"/CAcert-* "$tmpdir"/var/lib/ciao/CAcert-server-localhost.pem

echo -e "Copying CNCI Agent certificate"
sudo cp "$certs_dir"/cert-CNCIAgent-* "$tmpdir"/var/lib/ciao/cert-client-localhost.pem

echo -e "Removing cloud-init traces"
sudo rm -rf "$tmpdir"/var/lib/cloud

#Umount
echo -e "Done! unmounting\n"
sudo umount "$tmpdir"
sudo losetup -d $loop
sudo rm -rf "$tmpdir"
exit 0
