#!/bin/bash

#Simple script to boot a clear VM and attach it to the CN VNIC
#Use the cncli command to create the VNIC
#Note: This cannot be used for CNCI VNICs
#For CNCI VNICs, create the CNCI VNIC using the cncli and
#then use the script ciao-cnci-agent/scripts/run_cn_vm.sh to launch the CNCI VM

if [ -z "$1" ]; then
        IMAGE=clear.img
else
        IMAGE="$1"
fi

if [ -z "$2" ]; then
        VNIC="ERROR"
else
        VNIC="$2"
fi

if [ -z "$3" ]; then
        MAC="DE:AD:DE:AD:DE:AD"
else
        MAC="$3"
fi

if [[ "$IMAGE" =~ .xz$ ]]; then
        >&2 echo "File \"$IMAGE\" is still xz compressed. Uncompress it first with \"unxz\""
        exit 1
fi

if [ ! -f "$IMAGE" ]; then
        >&2 echo "Can't find image file \"$IMAGE\""
        exit 1
fi
rm -f debug.log

qemu-system-x86_64 \
        -enable-kvm \
        -bios OVMF.fd \
        -smp cpus=4,cores=2 -cpu host \
        -vga none -nographic \
        -drive file="$IMAGE",if=virtio,aio=threads,format=qcow2 \
        -net nic,model=virtio,macaddr="$MAC" -net tap,ifname="$VNIC",script=no,downscript=no \
        -debugcon file:debug.log -global isa-debugcon.iobase=0x402
