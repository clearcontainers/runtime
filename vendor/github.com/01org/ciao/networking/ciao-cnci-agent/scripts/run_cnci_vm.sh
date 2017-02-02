#!/bin/bash


#Create the cloud-init and Ciao specific ISO
xorriso -as mkisofs -R -V config-2 -o seed.iso seed/

if [ -z "$1" ]; then
	IMAGE="clear-8260-ciao-networking.img"
else
	IMAGE="$1"
fi

if [ -z "$2" ]; then
	PDEV="eth0"
else
	PDEV="$2"
fi

if [ -z "$3" ]; then
	MACVTAP="macvtap0"
else
	PDEV="$3"
fi


#Create your own macvtap device with a random mac
sudo ip link del "$MACVTAP"
sudo ip link add link "$PDEV" name "$MACVTAP" type macvtap mode bridge
sudo ip link set "$MACVTAP" address 02:00:DE:AD:02:01 up
sudo ip link show "$MACVTAP"

if [ ! -f "$IMAGE" ]; then
	>&2 echo "Can't find image file \"$IMAGE\""
	exit 1
fi
rm -f debug.log

tapindex=$(< /sys/class/net/"$MACVTAP"/ifindex)
tapdev=/dev/tap"$tapindex"

ifconfig "$MACVTAP" up

qemu-system-x86_64 \
	-enable-kvm \
	-bios OVMF.fd \
	-smp cpus=4,cores=2 -cpu host \
	-vga none -nographic \
	-drive file="$IMAGE",if=virtio,aio=threads \
	-net nic,model=virtio,macaddr=$(< /sys/class/net/"$MACVTAP"/address) -net tap,fd=3 3<>"$tapdev" \
	-drive file=seed.iso,if=virtio,media=cdrom \
	-debugcon file:debug.log -global isa-debugcon.iobase=0x402
