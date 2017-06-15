# kubicle

kubicle is a command line tool for creating [Kubernetes](https://kubernetes.io/) clusters on top of an existing ciao cluster.  It automatically creates ciao workloads for the various k8s roles (master and worker), creates instances from these workloads which self-form into a k8s cluster, and extracts the configuration information needed to control the new cluster from the host machine.

## Creating a k8s cluster

Creating a cluster is easy.  Kubicle only needs one piece of information, the UUID of the image to use for the k8s nodes.  Currently, this UUID must refer to an Ubuntu server image, as the workloads created by kubicle for the k8s nodes assume Ubuntu.  If you're using SingleVM you should already have an Ubuntu image in your cluster's image service.  To determine its UUID use ciao-cli image list, e.g.,

```
$ ciao-cli image list
Image #1
	Name             [Ubuntu Server 16.04]
	Size             [2361393152 bytes]
	UUID             [bf36c771-d5cc-47c4-b965-78eaca505229]
	Status           [active]
	Visibility       [public]
	Tags             []
	CreatedAt        [2017-06-07 10:41:49.37279755 +0000 UTC]
```

Now we simply need to run the kubicle create command specifying the UUID (bf36c771-d5cc-47c4-b965-78eaca505229) of the above image, e.g.,

```
$ kubicle create bf36c771-d5cc-47c4-b965-78eaca505229
Creating master
Creating workers

k8s cluster successfully created
--------------------------------
Created master:
 - ef4bf81f-2911-440d-ada1-0e7430f9e250
Created 1 workers:
 - 8cffa6e4-b4c0-4556-8273-0102aa5fca07
```

By default kubicle creates a 2 node cluster, with one master and one worker.  You should be able to view these nodes using ciao-cli instance list, e.g.,

```
$ ciao-cli instance list
# UUID                                 Status Private IP SSH IP        SSH PORT
1 ef4bf81f-2911-440d-ada1-0e7430f9e250 active 172.16.0.2 198.51.100.94 33002
2 8cffa6e4-b4c0-4556-8273-0102aa5fca07 active 172.16.0.3 198.51.100.94 33003
```

To manipulate the k8s cluster we need to ssh into the master node, which has the kubectl tool installed.  To connect to the node you need to specify a private key.  By default, kubicle uses the key created by SingleVM in ~/local/testkey.  So to connect to the master node we would type

```
ssh 198.51.100.94 -p 33002 -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -i ~/local/testkey
```

Now try executing the kubectl command to see if everything is okay, e.g.,

```
kubectl get nodes
NAME                                   STATUS    AGE       VERSION
8cffa6e4-b4c0-4556-8273-0102aa5fca07   Ready     2m        v1.6.4
ef4bf81f-2911-440d-ada1-0e7430f9e250   Ready     3m        v1.6.4
```

One issue you need to be aware of is that when kubicle is run without specifying an external ip address, as we have done here, it returns as soon as the instances have been created.  However, just because the instances have been created doesn't mean the cluster is up and running.  Thus if you connect to the master node as soon as kubicle returns you may find that the kubectl command fails or doesn't even exist.  If this happens, disconnect, wait a couple of minutes and then try again.  A nicer way of solving this issue is to specify an external ip which we will do in the following section.

## External-ip

kubicle allows you to specify an external ip address when running the create command.  Doing so provides a way to access the k8s cluster externally.  This means that the kubectl command can be run on the host machine rather than from inside the master node.  It also allows us to access k8s services from outside the k8s cluster.

An external-ip can be specified via the --external-ip option.  If you're using SingleVM you can simply select an unused ip address in the CNCI's subnet.  How do we know what the CNCI's subnet is?  Well the easiest was is to derive it from the SSH IP address shown when performing a ciao-cli instance list command.  Looking at the instance list command above the CNCI IP address is 198.51.100.94, so we could chose 198.51.100.2 as an external-ip address. If you select an IP address from another subnet you may need to set up some routes.  Here's an example of using an external-ip address in SingleVM.

```
$ kubicle create --external-ip=198.51.100.2 bf36c771-d5cc-47c4-b965-78eaca505229
Creating master
Creating workers
Mapping external-ip
Instances launched.  Waiting for k8s cluster to start

k8s cluster successfully created
--------------------------------
Created master:
 - 417c19ff-9efd-4e5c-92d0-12b4e19c65ab
Created 1 workers:
 - 1d96f6ac-a857-4f26-8c05-6a70ad57ee11
Created external-ips:
- 198.51.100.2
Created pools:
- k8s-pool-46959cfe-f584-45f1-9218-50ea3549a0ee
To access k8s cluster:
- export KUBECONFIG=$GOPATH/src/github.com/01org/ciao/testutil/singlevm/admin.conf
- If you use proxies, set
  - export no_proxy=$no_proxy,198.51.100.2
```

The first thing you will notice is that the command takes a lot longer to complete.  This is because when you specify an external-ip address kubicle will wait until the k8s cluster is ready before completing, thereby avoiding the problem we encountered above where kubicle had completed successfully before our k8s cluster was up and running.

The second thing you should notice is that the output is slightly different.  In addition to the extra information about the ciao external-ip resources created by kubicle we also see some information for configuring the kubectl tool.  Before we can use the kubectl tool on our host machine we need to download it.  Assuming your host machine is Ubuntu 16.04, kubectl can be installed as follows

```
apt-get update && apt-get install -y apt-transport-https
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add -
echo "deb http://apt.kubernetes.io/ kubernetes-xenial main" >/etc/apt/sources.list.d/kubernetes.list
apt-get update
apt-get install kubectl
```

Now we can access of cluster from the outside.

```
$ export KUBECONFIG=$GOPATH/src/github.com/01org/ciao/testutil/singlevm/admin.conf
$ export no_proxy=$no_proxy,198.51.100.2
$ kubectl get nodes
NAME                                   STATUS    AGE       VERSION
1d96f6ac-a857-4f26-8c05-6a70ad57ee11   Ready     1m        v1.6.4
417c19ff-9efd-4e5c-92d0-12b4e19c65ab   Ready     2m        v1.6.4
```

## Creating and exposing a deployment

Now we've got our kubectl tool running let's create a deployment

```
$ kubectl run nginx --image=nginx --port=80 --replicas=2
deployment "nginx" created
$ kubectl get pods
NAME                    READY     STATUS              RESTARTS   AGE
nginx-158599303-0p5dj   0/1       ContainerCreating   0          6s
nginx-158599303-th0fc   0/1       ContainerCreating   0          6s
```

So far so good.  We've created a deployment of nginx with two pods.  Let's now expose that deployment via an external ip.  There is a slight oddity here.  We don't actually specify the external-ip we passed to the kubicle create command.  Instead we need to specify the ciao internal-ip address of the master node, which is associated with the external-ip address we passed to the create command.  The reason for this is due to the way ciao implements external-ip addresses.  The CNCI translates external-ip addresses into internal-ip addresses and for this reason our k8s services need to be exposed using the ciao internal addresses.  To find out which address to use, execute the ciao-cli external-ip list command, e.g.,

```
$ ciao-cli external-ip list
# ExternalIP   InternalIP
1 198.51.100.2 172.16.0.3
```

You can see from the above command that the internal ip address associated with the external ip address we specified earlier is 172.16.0.3.  So to expose the deployment we simply need to type. 

```
kubectl expose deployment nginx --external-ip=172.16.0.3
service "nginx" exposed
```

The nginx service should now be accessible outside the k8s cluster via the external-ip.  You can verify this using curl, e.g.,

```
$ curl 198.51.100.2
<!DOCTYPE html>
<html>
<head>
<title>Welcome to nginx!</title>
<style>
    body {
        width: 35em;
        margin: 0 auto;
        font-family: Tahoma, Verdana, Arial, sans-serif;
    }
</style>
</head>
<body>
<h1>Welcome to nginx!</h1>
<p>If you see this page, the nginx web server is successfully installed and
working. Further configuration is required.</p>

<p>For online documentation and support please refer to
<a href="http://nginx.org/">nginx.org</a>.<br/>
Commercial support is available at
<a href="http://nginx.com/">nginx.com</a>.</p>

<p><em>Thank you for using nginx.</em></p>
</body>
```

## Tearing down the cluster

Kubicle created k8s clusters can be deleted with the kubicle delete command.  This could of course be done manually, but it would be tedious to do so, particularly with large clusters.  Instead use kubicle delete which deletes all the ciao objects (instances, volumes, workloads, pools and external-ips) created to support the k8s cluster in the correct order.  For example, to delete the cluster created above simply type.

```
$ kubicle delete
External-ips deleted:
198.51.100.2

Pools Deleted:
k8s-pool-46959cfe-f584-45f1-9218-50ea3549a0ee

Workloads deleted:
ad353ef4-6ea0-435c-affd-70b2fab24b8d
46959cfe-f584-45f1-9218-50ea3549a0ee

Instances deleted:
1d96f6ac-a857-4f26-8c05-6a70ad57ee11
417c19ff-9efd-4e5c-92d0-12b4e19c65ab
```

The kubicle delete command blocks until the cluster has been deleted.

## Configuring the k8s cluster

kubicle has a set of sensible defaults which is why the create command can be run with a single parameter.  By default it will create one worker node, with 1 VCPU, 2048 MB of RAM and 10GB of disk, and one master node with 1 VCPU, 1024 MB of RAM and 10GB of disk.  Some of these defaults can be overridden by providing command line options.  For example, to create a cluster with 2 worker nodes each with 20GB of disk one would execute.

```
$ kubicle create --workers=2 --wdisk=20 bf36c771-d5cc-47c4-b965-78eaca505229
```

To see a complete list of options type

```
kubicle create --help
```

## Limitations

- Only one master node is supported.
- Only one external-ip address is supported.
- Currently, its only possible to create one k8s cluster per tenant.  An attempt to create a second cluster in the same tenant will result in an error.
