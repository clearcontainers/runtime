#!/bin/bash -x

#Tests switching performance within a subnet
#
#Simple script to start iperf as a server on
#containers with even IPs and as a client on
#odd IPs connecting to the previous even IP
#This assumes the IPs of containers range from
#.2 to .254
#All traffic will remain within a set of 
#interconnected linux bridges
# 
#.254 will be used to test routing performance
#between subnets in the future


hostIP=$(hostname -i)
IFS=. read -r ip1 ip2 ip3 ip4 <<< $hostIP

if [ $(( $ip4 %2 )) -eq 0 ]; then
	echo "Starting Server on " $hostIP
	while :
	do
		iperf3 -s
	done
else
	while :
	do
		let ip4s=ip4-1
		serverIP=$ip1.$ip2.$ip3.$ip4s
		echo "Starting Client connecting to " $serverIP
		if [ $# -eq 0 ]; then
			iperf3 -c "$serverIP" -i 1 -t 20 -w 32M -P 4
			sleep 1
		else
			iperf3 -c "$serverIP" "$@"
		fi
	done
fi
