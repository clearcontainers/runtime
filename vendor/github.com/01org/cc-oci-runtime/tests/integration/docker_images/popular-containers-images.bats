#!/usr/bin/env bats
# *-*- Mode: sh; sh-basic-offset: 8; indent-tabs-mode: nil -*-*

#  This file is part of cc-oci-runtime.
#
#  Copyright (C) 2016 Intel Corporation
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

# Tests for the most popular images from docker hub

SRC="${BATS_TEST_DIRNAME}/../../lib/"

setup() {
	source $SRC/test-common.bash
	clean_docker_ps
	runtime_docker
}

@test "create a directory in an aerospike container" {
	$DOCKER_EXE run --rm -i aerospike bash -c "mkdir /home/test | ls /home | grep test"
}

@test "hello world in an alpine container" {
	$DOCKER_EXE run --rm -i alpine sh -c "echo 'Hello, World'"
}

@test "check os version in an alpine container" {
	$DOCKER_EXE run --rm alpine cat /etc/alpine-release
}

@test "show the help menu in an arangodb container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/320)"
	$DOCKER_EXE run --rm -e ARANGO_ROOT_PASSWORD=secretword -e ARANGO_NO_AUTH=1 -p 8529:8529 arangodb/arangodb foxx-manager help
}

@test "search job_spec in the  help option of a bonita container" {
	$DOCKER_EXE run --rm -i -e "TENANT_LOGIN=testuser" -e "TENANT_PASSWORD=mysecretword" bonita bash -c "help | grep job"
}

@test "show the os version inside a buildpack deps container" {
	$DOCKER_EXE run --rm -i buildpack-deps cat /etc/os-release
}

@test "display the cqlsh version in a cassandra container" {
	$DOCKER_EXE run --rm -i cassandra cqlsh --version
}

@test "run a celery container" {
	$DOCKER_EXE run --rm -i celery bash -c "python -m timeit \"[i for i in range(1000)]"\"
}

@test "cat a file in a centos container" {
	$DOCKER_EXE run --rm -i centos bash -c "echo "Test" > testfile.txt | cat testfile.txt | grep Test"
}

@test "display options in a chronograf container" {
	$DOCKER_EXE run --rm -i chronograf -sample-config
}

@test "check os-core-update bundle in a clearlinux container" {
	$DOCKER_EXE run --rm -i clearlinux bash -c "ls /usr/share/clear/bundles | grep os-core-update"
}

@test "check hostname in a clearlinux container" {
	$DOCKER_EXE run --rm -i clearlinux bash -c "hostname"
}

@test "execute a message in a clojure container" {
	$DOCKER_EXE run --rm -i clojure bash -c "echo -e 'public class CC{public static void main(String[]a){System.out.println(\"ClearContainers\");}}' > CC.java && javac CC.java && java CC"
}

@test "run a couchbase container" {
	$DOCKER_EXE run --rm -i couchbase bash -c "couchbase-cli rebalance --help | grep rebalance"
}

@test "set environment variables in a couchdb container" {
	$DOCKER_EXE run --rm -i -e COUCHDB_USER=test -e COUCHDB_PASSWORD=secretword couchdb bash -c "timeout 10 couchdb -n"
}

@test "run a consul container" {
	$DOCKER_EXE run --rm -i consul sh -c "timeout -t 10 consul agent -dev -client 0.0.0.0 | grep 0.0.0.0"
}

@test "run a crate container" {
	$DOCKER_EXE run --rm -i -e CRATE_HEAP_SIZE=1g crate timeout -t 10 crate -v
}

@test "check the resolv.conf in a crux container" {
	$DOCKER_EXE run --rm -i crux bash -c "cat /etc/resolv.conf | grep nameserver"
}

@test "instance in a django container" {
	$DOCKER_EXE run --rm -i --user "$(id -u):$(id -g)" django bash -c "django-admin | grep sqlflush"
}

@test "run drupal container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/321)"
	$DOCKER_EXE run --name mysqlcontainertest -e MYSQL_ROOT_PASSWORD=secretword -e MYSQL_USER=test -e MYSQL_DATABASE=testdb -d mysql
	if timeout 10 $DOCKER_EXE run --rm -i --name containertest --link mysqlcontainertest:mysql -p 8080:8080 drupal | grep "Unable to open logs"; then false; else true; fi
}

@test "run an instance in a docker container" {
	$DOCKER_EXE run --rm -i docker sh -c "docker inspect --format --help"
}

@test "run an instance in an elixir container" {
	$DOCKER_EXE run --rm -i elixir bash -c "pwd"
}

@test "run an erlang container" {
	$DOCKER_EXE run -d erlang erl -name test@erlang.local
}

@test "date in a fedora container" {
	$DOCKER_EXE run --rm -i fedora bash -c "date"
}

@test "search python version in a fedora/tools container" {
	$DOCKER_EXE run --rm -i fedora/tools bash -c "python --version"
}

@test "find the timestamp help in a gazebo container" {
	$DOCKER_EXE run --rm -i gazebo gz log --help
}

@test "run a gcc container" {
	$DOCKER_EXE run --rm -i gcc bash -c "echo -e '#include<stdio.h>\nint main (void)\n{printf(\"Hello\");return 0;}' > demo.c && gcc demo.c -o demo && ./demo"
}

@test "run an instance in a glassfish container" {
	$DOCKER_EXE run --rm -i glassfish bash -c "echo 'public class T{public static void main(String[]a){System.out.println(\"Test\");}}' > T.java && javac T.java && java T"
}

@test "run golang container" {
	$DOCKER_EXE run --rm -i golang bash -c "echo -e 'package main\nimport \"fmt\"\nfunc main() { fmt.Println(\"hello world\")}' > hello-world.go && go run hello-world.go && go build hello-world.go"
}

@test "go env in a golang container" {
	$DOCKER_EXE run --rm -i golang bash -c "go env | grep GOARCH"
}

@test "set memory size in haproxy container" {
	$DOCKER_EXE run --rm -i haproxy haproxy -m 2 -v
}

@test "run haskell container" {
	$DOCKER_EXE run --rm -i haskell cabal --numeric-version
}

@test "run hello world container" {
	$DOCKER_EXE run --rm -i hello-world | grep "Hello from Docker"
}

@test "run hello seattle container" {
	$DOCKER_EXE run --rm -i hello-seattle | grep "Hello from DockerCon"
}

@test "start apachectl in a httpd container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/321)"
	if $DOCKER_EXE run -i httpd apachectl -k start | grep "Unable to open logs"; then false; else true; fi
}

@test "run python command in a hylang container" {
	$DOCKER_EXE run --rm -i hylang bash -c "python -m timeit -s \"L=[]; M=range(1000)\" \"for m in M: L.append(m*2)\""
}

@test "display config information of a influxdb container" {
	$DOCKER_EXE run --rm -i influxdb influxd config
}

@test "start an instance in iojs container" {
	$DOCKER_EXE run --rm -i iojs bash -c "mkdir /root/test; ls /root | grep test"
}

@test "set nick in an irssi container" {
	$DOCKER_EXE run -d irssi irssi -n test
}

@test "display configuration parameters in a jetty container" {
	$DOCKER_EXE run --rm -i jetty bash -c "echo 'public class HW{public static void main(String[]a){System.out.println(\"HelloWorld\");}}' > HW.java; ls -l ./HW.java"
}

@test "additional parameters for jetty container" {
	$DOCKER_EXE run -d jetty --modules=jmx jetty.threadPool.maxThreads=500
}

@test "run joomla container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/321)"
	$DOCKER_EXE run --name mysqlcontainertest -e MYSQL_ROOT_PASSWORD=secretword -e MYSQL_USER=test -e MYSQL_DATABASE=testdb -d mysql
	if timeout 10 $DOCKER_EXE run -i --name testcontainer --link mysqlcontainertest:mysql -p 8080:8080 joomla | grep "Unable to open logs"; then false; else true; fi
}

@test "start jruby container" {
	$DOCKER_EXE run --rm -i jruby bash -c "jruby -e \"puts 'Containers'\""
}

@test "display message in a julia container" {
	$DOCKER_EXE run --rm -i julia bash -c "julia -e 'println(\"this is a test\")'"
}

@test "run kapacitor container" {
	$DOCKER_EXE run --rm -i kapacitor bash -c "kapacitord config > kapacitor.conf | ls -l kapacitor.conf"
}

@test "display information kibana container" {
	$DOCKER_EXE run --rm -i kibana kibana plugin --install kibana/shield/latest
}

@test "check kong configuration file is valid" {
	$DOCKER_EXE run --rm -i kong bash -c "cp /etc/kong/kong.conf.default /usr/local/kong/kong.conf; kong check /usr/local/kong/kong.conf | grep valid"
}

@test "check kernel version in a mageia container" {
	$DOCKER_EXE run --rm -i mageia bash -c "uname -a | grep Linux"
}

@test "start an instance of a mariadb container" {
	$DOCKER_EXE run --rm -i -e MYSQL_ROOT_PASSWORD=secretword  mariadb bash -c "cat /etc/mysql/conf.d/mariadb.cnf | grep character"
}

@test "check memory maven container" {
	$DOCKER_EXE run --rm -i maven bash -c "mvn package | grep "Final Memory""
}

@test "run memcached container" {
	$DOCKER_EXE run --rm -i memcached bash -c "perl -e 'print \"memcachedtest\n\"'"
}

@test "start mongo container" {
	timeout 10 $DOCKER_EXE run -i mongo --auth | grep "MongoDB starting"
}

@test "start nats server" {
	timeout 10 $DOCKER_EXE run --rm -i nats | grep "nats-server"
}

@test "run nats streaming container" {
	timeout 10 $DOCKER_EXE run --rm -i nats-streaming | grep "Server is ready"
}

@test "create a file in a neo4j container" {
	$DOCKER_EXE run --rm -i neo4j bash -c "echo "running" > test; cat /var/lib/neo4j/test"
}

@test "configuration file neurodebian" {
	$DOCKER_EXE run --rm neurodebian cat /etc/apt/sources.list.d/neurodebian.sources.list
}

@test "run nginx container" {
	$DOCKER_EXE run --rm nginx dpkg-query -l openssl
}

@test "search in a node container" {
	$DOCKER_EXE run --rm -i node node --v8-options
}

@test "search in a nuxeo container" {
	$DOCKER_EXE run --rm -i nuxeo apt-cache search python
}

@test "run an odoo container" {
	$DOCKER_EXE run --rm -i odoo bash -c "python -m timeit -s \"M=range(1000)\" \"L=[m*2 for m in M]\""
}

@test "create files in an oraclelinux container" {
	$DOCKER_EXE run --rm -i oraclelinux bash -c 'for NUM in `seq 1 1 10`; do touch $NUM-file.txt && ls -l /$NUM-file.txt; done'
}

@test "run hello world in java in an openjdk container" {
	$DOCKER_EXE run --rm -i openjdk bash -c "echo 'public class HW{public static void main(String[]a){System.out.println(\"HelloWorld\");}}' > HW.java && javac HW.java && java HW"
}

@test "run an opensuse container" {
	$DOCKER_EXE run --rm -i opensuse bash -c "echo "testing" > test.txt | cat /test.txt | grep testing"
}

@test "start orientdb server" {
	$DOCKER_EXE run -e ORIENTDB_ROOT_PASSWORD=mysecretword -d orientdb timeout -t 10 /orientdb/bin/server.sh
}

@test "start instance in percona container" {
	if $DOCKER_EXE run -i percona perl -e 'print "Clear Containers\n"' | grep LANG; then false; else true; fi
}

@test "run php container" {
	$DOCKER_EXE run --rm -i php bash -c "php -r 'print(\"cc oci runtime\");'"
}

@test "check build number of a photon container" {
	$DOCKER_EXE run --rm -i photon cat /etc/photon-release
}

@test "print a piwik container" {
	$DOCKER_EXE run --rm -i piwik bash -c "php -r 'print(\"Clear Containers\");'"
}

@test "execute a pypy container" {
	$DOCKER_EXE run --rm -i pypy bash -c "python -m timeit -s \"M=range(1000);f=lambda x:x*2\" \"L=[f(m) for m in M]"\"
}

@test "run python container" {
	$DOCKER_EXE run --rm -i python bash -c "python -m timeit -s \"M=range(1000);f=lambda x: x*2\" \"L=map(f,M)\""
}

@test "start a rabbitmq container" {
	$DOCKER_EXE run --hostname my-rabbit-container -e RABBITMQ_DEFAULT_USER=testuser -e RABBITMQ_DEFAULT_PASS=mysecretword -d rabbitmq rabbitmqctl reset
}

@test "print message in a r-base container" {
	$DOCKER_EXE run --rm r-base r -e 'print ( "Hello World!")'
}

@test "create rails application" {
	$DOCKER_EXE run --rm -i rails timeout 10 rails new commandsapp
}

@test "run rakudo star container" {
	if $DOCKER_EXE run -i rakudo-star perl -e 'print "Hello\n"' | grep "LANG"; then false; else true; fi
}

@test "start redis server with a certain port" {
	$DOCKER_EXE run --rm -i redis bash -c "timeout 10 redis-server --port 7777 | grep 7777"
}

@test "start redis server" {
	$DOCKER_EXE run --rm -i redis bash -c "timeout 5 redis-server --appendonly yes | grep server"
}

@test "search gcc in a ros container" {
	$DOCKER_EXE run --rm -i ros apt-cache search gcc
}

@test "print message in a ruby container" {
	$DOCKER_EXE run --rm -i ruby bash -c "ruby -e \"puts 'Clear Linux'\""
}

@test "generate key in a sentry container" {
	$DOCKER_EXE run --rm -i sentry config generate-secret-key
}

@test "start solr server" {
	$DOCKER_EXE run --rm -i solr timeout 10 solr -h test
}

@test "start swarm container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/222)"
	if $DOCKER_EXE run -i swarm create | grep "EXEC spawning"; then false; else true; fi
}

@test "generate a telegraf conf file" {
	$DOCKER_EXE run --rm -i telegraf bash -c "telegraf config > telegraf.conf; ls telegraf.conf"
}

@test "run a tomcat container" {
	$DOCKER_EXE run --rm -i tomcat bash -c "echo 'public class HW{public static void main(String[]a){System.out.println(\"HelloWorld\");}}' > HW.java | ls -l /usr/local/tomcat/HW.java"
}

@test "run tomee container" {
	$DOCKER_EXE run --rm -i tomee bash -c "echo -e 'public class CL{public static void main(String[]a){System.out.println(\"ClearLinux\");}}' > CL.java && javac CL.java && java CL"
}

@test "run an instance in a traefik container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/222)"
	if $DOCKER_EXE run -i traefik traefik --version | grep "EXEC spawning"; then false; else true; fi
}

@test "run an instance in an ubuntu debootstrap container" {
	$DOCKER_EXE run --rm -i ubuntu-debootstrap bash -c 'if [ -f /etc/bash.bashrc ]; then echo "/etc/bash.bashrc exists"; fi'
}

@test "search nano in an ubuntu upstart container" {
	$DOCKER_EXE run --rm -i ubuntu-upstart bash -c "apt-cache search nano"
}

@test "start server in a vault container" {
	$DOCKER_EXE run --rm -i -e 'VAULT_DEV_ROOT_TOKEN_ID=mytest' vault timeout -t 10 vault server -dev
}

@test "start wordpress container" {
	skip "this image is not working (see https://github.com/01org/cc-oci-runtime/issues/318)"
	if $DOCKER_EXE run -i wordpress perl -e 'print "test\n"' | grep LANG; then false; else true; fi
}

@test "start zookeeper container" {
	$DOCKER_EXE run --rm -i zookeeper zkServer.sh start
}
