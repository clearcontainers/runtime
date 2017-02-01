#!/usr/bin/env bats
# *-*- Mode: sh; sh-basic-offset: 8; indent-tabs-mode: nil -*-*

#  This file is part of cc-oci-runtime.
#
#  Copyright (C) 2017 Intel Corporation
#
#  This program is free software; you can redistribute it and/or
#  modify it under the terms of the GNU General Public License
#  as published by the Free Software Foundation; either version 2
#  of the License, or (at your option) any later version.
#
#  This program is distributed in the hope that it will be useful,
#  but WITHOUT ANY WARRANTY; without even the implied warranty of
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
#  GNU General Public License for more details.
#
#  You should have received a copy of the GNU General Public License
#  along with this program; if not, write to the Free Software
#  Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

# Swarm testing

SRC="${BATS_TEST_DIRNAME}/../../lib/"
# maximum number of replicas that will be launch
number_of_replicas=4
url=http://127.0.0.1:8080/hostname
# number of attemps to obtain the hostname
# of the replicas using curl
number_of_attemps=5
# saves the hostname of the replicas
declare -a REPLICAS
# saves the process id of the replicas
declare -a REPLICAS_UP
# saves the name of the replicas
declare -a NAMES_IPS_REPLICAS
# saves the ip of the replicas
declare -a IPS_REPLICAS

#This function will verify that swarm is not running or it finishes
function clean_swarm_status() {
	for j in `seq 0 $number_of_attemps`; do
		if $DOCKER_EXE node ls; then
			$DOCKER_EXE swarm leave --force
			# docker swarm leave is not inmediately so it requires time to finish
			sleep 5
		else
			break
		fi
        done
}


setup() {
	source $SRC/test-common.bash
	clean_docker_ps
	runtime_docker
	clean_swarm_status
	interfaces=($(readlink /sys/class/net/* | grep -i pci | xargs basename -a))
	swarm_interface_arg=""
	for i in ${interfaces[@]}; do
		if [ "`cat /sys/class/net/${i}/operstate`" == "up" ]; then
			swarm_interface_arg="--advertise-addr ${i}"
			break;
		fi
	done
	$DOCKER_EXE swarm init ${swarm_interface_arg}
	$DOCKER_EXE service create --name testswarm --replicas $number_of_replicas --publish 8080:80 nginx /bin/bash -c "hostname > /usr/share/nginx/html/hostname; nginx -g \"daemon off;\"" 2> /dev/null
	while [ `$DOCKER_EXE ps --filter status=running --filter ancestor=nginx:latest -q | wc -l` -lt $number_of_replicas ]; do
		sleep 1
	done
}

@test "ping among replicas on their overlay ip" {
	# this test takes time as it performs pings in all the replicas
	name_network=`$DOCKER_EXE network ls --filter driver=overlay -q`
	NAMES_REPLICAS=(`$DOCKER_EXE network inspect $name_network --format='{{ range .Containers}} {{println .Name}} {{end}}' | head -n -2`)
	IPS_REPLICAS=(`$DOCKER_EXE network inspect $name_network --format='{{ range .Containers}} {{ println .IPv4Address}}{{end}}' | head -n -2 | cut -d'/' -f1`)
	for i in `seq 0 $((number_of_replicas-1))`; do
		for j in `seq 0 $((number_of_replicas-1))`; do
			if [ "$i" != "$j" ] && [ -n "${NAMES_REPLICAS[$i]}" ]; then
				# here we are performing a ping with the list
				# of the overlay ips obtained from the replicas
				$DOCKER_EXE exec ${NAMES_REPLICAS[$i]} bash -c "ping -c $number_of_attemps ${IPS_REPLICAS[$j]}"
			fi
		done
	done
}

@test "check that the replicas' names are different" {
	skip "this test is not working properly (see https://github.com/01org/cc-oci-runtime/issues/578)" 
	# this will help to obtain the hostname of 
	# the replicas from the curl
        unset http_proxy
        for j in `seq 0 $number_of_attemps`; do
	        for i in `seq 0 $((number_of_replicas-1))`; do
		        # this will help that bat does not exit incorrectly 
			# when curl fails in one of the attemps
			set +e
		        REPLICAS[$i]="$(curl $url 2> /dev/null)"
	                set -e
               done
		non_empty_elements="$(echo ${REPLICAS[@]} | egrep -o "[[:space:]]+" | wc -l)"
               if [ "$non_empty_elements" == "$((number_of_replicas-1))" ]; then
                	break
                fi
		# this will give enough time between attemps
                sleep 5
        done
	for i in `seq 0 $((number_of_replicas-1))`; do
                if [ -z "${REPLICAS[$i]}" ]; then
                    false
                fi
		for j in `seq $((i+1)) $((number_of_replicas-1))`; do
			if [ "${REPLICAS[$i]}" == "${REPLICAS[$j]}" ]; then
				false
			fi
		done
	done 
}

@test "check that replicas has two interfaces" {
	REPLICAS_UP=(`$DOCKER_EXE ps -a -q`)
	for i in ${REPLICAS_UP[@]}; do
		# here we are checking that each replica has two interfaces 
		# and they should be always eth0 and eth1
		$DOCKER_EXE exec $i bash -c "ip route show | grep -E eth0 && ip route show | grep -E eth1"
	done
}

@test "check service ip among the replicas" {
	service_name=`$DOCKER_EXE service ls --filter name=testswarm -q`
	ip_service=`$DOCKER_EXE service inspect $service_name --format='{{range .Endpoint.VirtualIPs}}{{.Addr}}{{end}}' | cut -d'/' -f1`
	REPLICAS_UP=(`$DOCKER_EXE ps -a -q`)
	for i in ${REPLICAS_UP[@]}; do
		# here we are checking that all the 
		# replicas have the service ip
		$DOCKER_EXE exec $i bash -c "ip a | grep $ip_service"
	done
}

@test "ping among replicas on their gateway ip" {
	REPLICAS_UP=(`$DOCKER_EXE ps -a | tail -n +2 | awk '{ print $1 }'`)
	IPS_REPLICAS=(`$DOCKER_EXE network inspect docker_gwbridge --format='{{range .Containers}} {{ println .IPv4Address}}{{end}}' | head -n -2 | cut -d'/' -f1`)
	for i in ${REPLICAS_UP[@]}; do
		for j in ${IPS_REPLICAS[@]}; do
			# here we are checking that the replica can not perform
			# a ping among the gateway ips
			run $DOCKER_EXE exec $i bash -c "ip a | grep $j; if [ $? -ne 0 ]; then ping -c $number_of_attemps $j; else exit 1; fi"
			[ $status -ne 0 ]
		done
	done
}

@test "quick ping among replicas on their overlay ip" {
	# this test makes a ping in each replica
	name_network=`$DOCKER_EXE network ls --filter driver=overlay -q`
	NAMES_REPLICAS=(`$DOCKER_EXE network inspect $name_network --format='{{ range .Containers}} {{println .Name}} {{end}}' | head -n -2`)
	IPS_REPLICAS=(`$DOCKER_EXE network inspect $name_network --format='{{ range .Containers}} {{ println .IPv4Address}}{{end}}' | head -n -2 | cut -d'/' -f1`)
	for i in `seq 0 $((number_of_replicas-1))`; do
		for j in `seq 0 $((number_of_replicas-1))`; do
			if [ "$i" != "$j" ] && [ -n "${NAMES_REPLICAS[$i]}" ]; then
				# here we are performing a ping with the list
				# of the overlay ips obtained from the replicas
				$DOCKER_EXE exec ${NAMES_REPLICAS[$i]} bash -c "ping -c $number_of_attemps ${IPS_REPLICAS[$j]}"
				break
			fi
		done
	done
}

teardown () {
	$DOCKER_EXE service remove testswarm
	clean_swarm_status
}
