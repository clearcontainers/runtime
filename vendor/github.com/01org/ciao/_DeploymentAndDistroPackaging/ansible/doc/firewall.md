Firewall Settings
=================

### Ciao Controller Nodes

| Port          | Protocol      | Description  |
| ------------- |:-------------:| -----:       |
| 8888          | TCP           | Ciao Scheduler port |
| 8774          | TCP           | Ciao Compute API port |
| 8776          | TCP           | Ciao Volume API port |
| 9292          | TCP           | Ciao Image API port  |
| 5000          | TCP           | Openstack Keystone public API port |
| 35357         | TCP           | Openstack Keystone admin API port  |
| 443           | TCP           | Ciao WebUI port |

### Ciao Network Nodes

| Port          | Protocol      | Description  |
| ------------- |:-------------:| -----:       |
| 9999          | TCP           | Ciao Network port |
| 67:68         | UDP           | DHCP/BOOTP |
| 123           | UDP           | NTP |
| 53            | TCP/UDP       | DNS |
| 5355          | TCP/UDP       | LLMNR |
| 323           | TCP/UDP       | RPKI-RTR |

### Ciao Compute Nodes

| Port          | Protocol      | Description  |
| ------------- |:-------------:| -----:       |
| 9999          | TCP           | Ciao Compute port |
| 67:68         | UDP           | DHCP/BOOTP |
| 123           | UDP           | NTP |
| 53            | TCP/UDP       | DNS |

* Allow INPUT/OUTPUT port 47(GRE).
```
iptables -I INPUT   1 -p 47 -j ACCEPT
iptables -I OUTPUT  1 -p 47 -j ACCEPT
```

* For Ceph Cluster firewall configuration, check it's official documentation [Here](http://docs.ceph.com/docs/giant/rados/configuration/network-config-ref/).

Firewalld Service Example
-------------------------
In this example, a new zone is created per Ciao node

### Controller Node

```
[ciao@controller ~]$ sudo firewall-cmd --info-zone=ciao
ciao (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0
  sources: 
  services: 
  ports: 8888/tcp 8774/tcp 8776/tcp 9292/tcp 5000/tcp 22/tcp 35357/tcp 53/tcp 443/tcp
  protocols: icmp
  masquerade: no
  forward-ports: 
  source-ports: 
  icmp-blocks: 
  rich rules:
```

### Network Node
```
[ciao@network ~]$ sudo firewall-cmd --info-zone=ciao
ciao (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0
  sources: 
  services: dns http https dhcp
  ports: 9999/tcp 22/tcp 323/udp 68/udp 5355/tcp 67/udp 53/udp 5355/udp 53/tcp
  protocols: gre igmp icmp ipv6
  masquerade:
  forward-ports: 
  source-ports: 
  icmp-blocks: 
  rich rules: 
```

### Compute Node
```
[ciao@compute ~]$ sudo firewall-cmd --info-zone=ciao
ciao (active)
  target: default
  icmp-block-inversion: no
  interfaces: eth0
  sources: 
  services: http https dhcp dns
  ports: 22/tcp 9999/tcp 312/udp 323/udp 68/udp
  protocols: gre igmp
  masquerade: no
  forward-ports: 
  source-ports: 
  icmp-blocks: 
  rich rules: 
```
