#!/bin/bash

ciao_bin="$HOME/local"
ciao_gobin="$GOPATH"/bin
event_counter=0

#Utility functions
function exitOnError {
	local exit_code="$1"
	local error_string="$2"

	if [ $1 -ne 0 ]
	then
		echo "FATAL ERROR exiting: " "$error_string" "$exit_code"
		exit 1
	fi
}

#Checks that no network artifacts are left behind
function checkForNetworkArtifacts() {

	#Verify that there are no ciao related artifacts left behind
	ciao_networks=`sudo docker network ls --filter driver=ciao -q | wc -l`

	if [ $ciao_networks -ne 0 ]
	then
		echo "FATAL ERROR: ciao docker networks not cleaned up"
		sudo docker network ls --filter driver=ciao
		exit 1
	fi


	#The only ciao interfaces left behind should be CNCI VNICs
	#Once we can delete tenants we should not even have them around
	cnci_vnics=`ip -d link | grep alias | grep cnci | wc -l`
	ciao_vnics=`ip -d link | grep alias | wc -l`

	if [ $cnci_vnics -ne $ciao_vnics ]
	then
		echo "FATAL ERROR: ciao network interfaces not cleaned up"
		ip -d link | grep alias
		exit 1
	fi
}

function rebootCNCI {
	ssh -T -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i "$CIAO_SSH_KEY" cloud-admin@"$ssh_ip" <<-EOF
	sudo reboot now
	EOF

	#Now wait for it to come back up
	ping -w 90 -c 3 $ssh_ip
	exitOnError $?  "Unable to ping CNCI after restart"

	#Dump the tables for visual verification
	ssh -T -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i "$CIAO_SSH_KEY" cloud-admin@"$ssh_ip" <<-EOF
	sudo iptables-save
	sudo ip l
	sudo ip a
	EOF

	echo "Rebooted the CNCI"
}

function checkExtIPConnectivity {
    #We checked the event before calling this, so the mapping should exist
    testip=`"$ciao_gobin"/ciao-cli external-ip list -f '{{with index . 0}}{{.ExternalIP}}{{end}}'`
    test_instance=`"$ciao_gobin"/ciao-cli instance list -f '{{with index . 0}}{{.ID}}{{end}}'`

    sudo ip route add 203.0.113.0/24 dev ciaovlan
    ping -w 90 -c 3 $testip
    ping_result=$?
    #Make sure we are able to reach the VM
    test_hostname=`ssh -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i "$CIAO_SSH_KEY" demouser@"$testip" hostname -s`
    sudo ip route del 203.0.113.0/24 dev ciaovlan

    exitOnError $ping_result "Unable to ping external IP"

    if [ "$test_hostname" == "$test_instance" ]
    then
	    echo "SSH connectivity using external IP verified"
    else
	    echo "FATAL ERROR: Unable to ssh via external IP"
	    exit 1
    fi
}

#There are too many failsafes in the CNCI. Hence just disable iptables utility to trigger failure
#This also ensures that the CNCI is always left in a consistent state (sans the permission)
# this function to be run on the CNCI
function triggerIPTablesFailure {
	ssh -T -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i "$CIAO_SSH_KEY" cloud-admin@"$ssh_ip" <<-EOF
	sudo chmod -x /usr/bin/iptables
	EOF
}

#Restore the iptables so that the cluster is usable
# this function to be run on the CNCI
function restoreIPTables {
	ssh -T -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i "$CIAO_SSH_KEY" cloud-admin@"$ssh_ip" <<-EOF
	sudo chmod +x /usr/bin/iptables
	EOF
}

function clearAllEvents() {
	#Clear out all prior events
	"$ciao_gobin"/ciao-cli event delete

	#Wait for the event count to drop to 0
	retry=0
	ciao_events=0

	until [ $retry -ge 6 ]
	do
		ciao_events=`"$ciao_gobin"/ciao-cli event list -f '{{len .}}'`

		if [ $ciao_events -eq 0 ]
		then
			break
		fi
		let retry=retry+1
		sleep 1
	done

	exitOnError $ciao_events "ciao events not deleted properly"
}

function checkEventStatus {
	local event_index="$1"
	local event_code="$2"
	local retry=0
	local ciao_events=0
	local total_events=0
	local code=""

	#We only need to wait for as many events as the index
	total_events=$((event_index + 1))

	until [ $retry -ge 6 ]
	do
		ciao_events=`"$ciao_gobin"/ciao-cli event list -f '{{len .}}'`

		if [ $ciao_events -eq $total_events ]
		then
			break
		fi

		let retry=retry+1
		sleep 1
	done

	if [ $ciao_events -ne $total_events ]
	then
		echo "FATAL ERROR: ciao event not reported. Events seen =" $ciao_events
		"$ciao_gobin"/ciao-cli event list
		exit 1
	fi

	code=$("$ciao_gobin"/ciao-cli event list -f "{{(index . ${event_index}).Message}}" | cut -d ' ' -f 1)

	if [ "$event_code" != "$code" ]
	then
		echo "FATAL ERROR: Unknown event $code. Looking for $event_code"
		"$ciao_gobin"/ciao-cli event list
		exit 1
	fi

	"$ciao_gobin"/ciao-cli event list
}

function createExternalIPPool() {
	# first create a new external IP pool and add a subnet to it.
	# this is an admin only operation, so make sure our env variables
	# are set accordingly. Since user admin might belong to more than one
	# tenant, make sure to specify that we are logging in as part of the
	# "admin" tenant/project.
	ciao_user=$CIAO_USERNAME
	ciao_passwd=$CIAO_PASSWORD
	export CIAO_USERNAME=$CIAO_ADMIN_USERNAME
	export CIAO_PASSWORD=$CIAO_ADMIN_PASSWORD
	"$ciao_gobin"/ciao-cli -tenant-name admin pool create -name test
	"$ciao_gobin"/ciao-cli -tenant-name admin pool add -subnet 203.0.113.0/24 -name test
	export CIAO_USERNAME=$ciao_user
	export CIAO_PASSWORD=$ciao_passwd
}

function deleteExternalIPPool() {
	#Cleanup the pool
	export CIAO_USERNAME=$CIAO_ADMIN_USERNAME
	export CIAO_PASSWORD=$CIAO_ADMIN_PASSWORD

	"$ciao_gobin"/ciao-cli -tenant-name admin pool delete -name test
	exitOnError $?  "Unable to delete pool"

	export CIAO_USERNAME=$ciao_user
	export CIAO_PASSWORD=$ciao_passwd
}

# Read cluster env variables
. $ciao_bin/demo.sh

vm_wlid=$("$ciao_gobin"/ciao-cli workload list -f='{{if gt (len .) 0}}{{(index . 0).ID}}{{end}}')
exitOnError $?  "Unable to list workloads"

"$ciao_gobin"/ciao-cli instance add --workload=$vm_wlid --instances=2
exitOnError $?  "Unable to launch VMs"

"$ciao_gobin"/ciao-cli instance list
exitOnError $? "Unable to list instances"

#Launch containers
#Pre-cache the image to reduce the start latency
sudo docker pull debian
debian_wlid=$("$ciao_gobin"/ciao-cli workload list -f='{{$x := filter . "Name" "Debian latest test container"}}{{if gt (len $x) 0}}{{(index $x 0).ID}}{{end}}')
echo "Starting workload $debian_wlid"
"$ciao_gobin"/ciao-cli instance add --workload=$debian_wlid --instances=1
exitOnError $? "Unable to launch containers"

sleep 5

"$ciao_gobin"/ciao-cli instance list
exitOnError $? "Unable to list instances"

container_1=`sudo docker ps -q -l`
container_1_ip=`sudo docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' $container_1`

"$ciao_gobin"/ciao-cli instance add --workload=$debian_wlid --instances=1
exitOnError $?  "Unable to launch containers"
sleep 5

"$ciao_gobin"/ciao-cli instance list
exitOnError $?  "Unable to list instances"

container_2=`sudo docker ps -q -l`

#Check SSH connectivity
"$ciao_gobin"/ciao-cli instance list

#The VM takes time to boot as you are running on two
#layers of virtualization. Hence wait a bit
retry=0
until [ $retry -ge 6 ]
do
	ssh_ip=""
	ssh_ip=$("$ciao_gobin"/ciao-cli instance list --workload=$vm_wlid -f='{{if gt (len .) 0}}{{(index . 0).SSHIP}}{{end}}')

	if [ "$ssh_ip" == "" ] 
	then
		echo "Waiting for instance to boot"
		let retry=retry+1
		sleep 30
		continue
	fi

	ssh_check=$(head -1 < /dev/tcp/"$ssh_ip"/33002)

	echo "Attempting to ssh to: $ssh_ip"

	if [[ "$ssh_check" == *SSH-2.0-OpenSSH* ]]
	then
		echo "SSH connectivity verified"
		break
	else
		let retry=retry+1
		echo "Retrying ssh connection $retry"
	fi
	sleep 30
done

if [ $retry -ge 6 ]
then
	echo "Unable check ssh connectivity into VM"
	exit 1
fi

#Check docker networking
echo "Checking Docker Networking"
sudo docker exec $container_2 /bin/ping -w 90 -c 3 $container_1_ip

exitOnError $?  "Unable to ping across containers"
echo "Container connectivity verified"

#Clear out all prior events
clearAllEvents

#Test External IP Assignment support
#Pick the first instance which is a VM, as we can even SSH into it
#We have already checked that the VM is up.
createExternalIPPool

testinstance=`"$ciao_gobin"/ciao-cli instance list -f '{{with index . 0}}{{.ID}}{{end}}'`
"$ciao_gobin"/ciao-cli external-ip map -instance $testinstance -pool test

#Wait for the CNCI to report successful map
checkEventStatus $event_counter "Mapped"

"$ciao_gobin"/ciao-cli event list
"$ciao_gobin"/ciao-cli external-ip list

checkExtIPConnectivity

#Check that the CNCI retains state after reboot
#If state has been restored, the Ext IP should be reachable
rebootCNCI
checkExtIPConnectivity

"$ciao_gobin"/ciao-cli external-ip unmap -address $testip

#Wait for the CNCI to report successful unmap
event_counter=$((event_counter+1))
checkEventStatus $event_counter "Unmapped"

"$ciao_gobin"/ciao-cli external-ip list

#Test for External IP Failures

#Map failure
triggerIPTablesFailure
"$ciao_gobin"/ciao-cli external-ip map -instance $testinstance -pool test
#Wait for the CNCI to report unsuccessful map
event_counter=$((event_counter+1))
checkEventStatus $event_counter "Failed"
restoreIPTables

#Unmap failure
"$ciao_gobin"/ciao-cli external-ip map -instance $testinstance -pool test
event_counter=$((event_counter+1))
checkEventStatus $event_counter "Mapped"

triggerIPTablesFailure 
"$ciao_gobin"/ciao-cli external-ip unmap -address $testip
event_counter=$((event_counter+1))
checkEventStatus $event_counter "Failed"
restoreIPTables

#Cleanup
"$ciao_gobin"/ciao-cli external-ip unmap -address $testip
event_counter=$((event_counter+1))
checkEventStatus $event_counter "Unmapped"

#Cleanup pools
deleteExternalIPPool

#Now delete all instances
"$ciao_gobin"/ciao-cli instance delete --all
exitOnError $?  "Unable to delete instances"

"$ciao_gobin"/ciao-cli instance list

#Wait for all the instance deletions to be reported back
event_counter=$((event_counter+4))
checkEventStatus $event_counter "Deleted"

#Verify that there are no ciao related artifacts left behind
checkForNetworkArtifacts

echo "###########################################"
echo "-----------All checks passed!--------------"
echo "###########################################"
