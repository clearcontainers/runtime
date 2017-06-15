
function createWorkloads() {
	mkdir -p ${ciao_bin}/workload_examples
	if [ ! -d ${ciao_bin}/workload_examples ]
	then
		echo "FATAL ERROR: Unable to create ${ciao_bin}/workload_examples"
		exit 1

	fi

	sudo chmod 755 "$ciao_bin"/workload_examples

	# create a VM test workload with ssh capability
	echo "Generating VM cloud-init"
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
	) > "$ciao_bin"/workload_examples/vm-test.yaml

	if [ $all_images -eq 1 ]
	then
	    # Get the image ID for the fedora cloud image
	    id=$(ciao-cli image list -f='{{$x := filter . "Name" "Fedora Cloud Base 24-1.2"}}{{if gt (len $x) 0}}{{(index $x 0).ID}}{{end}}')

	    # add 3 vm test workloads
	    echo "Generating Fedora VM test workload"
	    (
		cat <<-EOF
		description: "Fedora test VM"
		vm_type: qemu
		fw_type: legacy
		defaults:
		    vcpus: 2
		    mem_mb: 128
		    disk_mb: 80
		cloud_init: "vm-test.yaml"
		disks:
		  - source:
		       service: image
		       id: "$id"
		    ephemeral: true
		    bootable: true
		EOF
	    ) > "$ciao_bin"/workload_examples/fedora_vm.yaml

	    # get the clear image id
	    clear_id=$(ciao-cli image list -f='{{$x := filter . "Name" "Clear Linux '"${LATEST}"'"}}{{if gt (len $x) 0}}{{(index $x 0).ID}}{{end}}')

	    # create a clear VM workload definition
	    echo "Creating Clear test workload"
	    (
		cat <<-EOF
		description: "Clear Linux test VM"
		vm_type: qemu
		fw_type: efi
		defaults:
		    vcpus: 2
		    mem_mb: 128
		    disk_mb: 80
		cloud_init: "vm-test.yaml"
		disks:
		  - source:
		       service: image
		       id: "$clear_id"
		    ephemeral: true
		    bootable: true
		EOF
	    ) > "$ciao_bin"/workload_examples/clear_vm.yaml

	fi

	# get the clear image id
	ubuntu_id=$(ciao-cli image list -f='{{select (head (filter . "Name" "Ubuntu Server 16.04")) "ID"}}')

	echo "Generating Ubuntu VM test workload"
	(
	cat <<-EOF
	description: "Ubuntu test VM"
	vm_type: qemu
	fw_type: legacy
	defaults:
	    vcpus: 2
	    mem_mb: 256
	    disk_mb: 80
	cloud_init: "vm-test.yaml"
	disks:
	  - source:
	       service: image
	       id: "$ubuntu_id"
	    ephemeral: true
	    bootable: true
	EOF
	) > "$ciao_bin"/workload_examples/ubuntu_vm.yaml

	# create a container test cloud init
	echo "Creating Container cloud init"
	(
	cat <<-EOF
	---
	#cloud-config
	runcmd:
	    - [ /bin/bash, -c, "while true; do sleep 60; done" ]
	...
	EOF
	) > "$ciao_bin"/workload_examples/container-test.yaml

	# create a Debian container workload definition
	echo "Creating Debian Container test workload"
	(
	cat <<-EOF
	description: "Debian latest test container"
	vm_type: docker
	image_name: "debian:latest"
	defaults:
	  vcpus: 2
	  mem_mb: 128
	  disk_mb: 80
	cloud_init: "container-test.yaml"
	EOF
	) > "$ciao_bin"/workload_examples/debian_latest.yaml

	# create an Ubuntu container workload definition
	echo "Creating Ubuntu Container test workload"
	(
	cat <<-EOF
	description: "Ubuntu latest test container"
	vm_type: docker
	image_name: "ubuntu:latest"
	defaults:
	  vcpus: 2
	  mem_mb: 128
	  disk_mb: 80
	cloud_init: "container-test.yaml"
	EOF
	) > "$ciao_bin"/workload_examples/ubuntu_latest.yaml

	# store the new workloads into ciao
	pushd "$ciao_bin"/workload_examples
	if [ $all_images -eq 1 ]
	then
	    "$ciao_gobin"/ciao-cli workload create -yaml fedora_vm.yaml
	    "$ciao_gobin"/ciao-cli workload create -yaml clear_vm.yaml
	fi
	"$ciao_gobin"/ciao-cli workload create -yaml ubuntu_latest.yaml
	"$ciao_gobin"/ciao-cli workload create -yaml debian_latest.yaml
	"$ciao_gobin"/ciao-cli workload create -yaml ubuntu_vm.yaml
	popd
}

echo ""
echo "Creating public test workloads"
echo "---------------------------------------------------------------------------------------"
createWorkloads
