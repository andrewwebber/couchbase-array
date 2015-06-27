# CoreOS - Couchbase

- Download a CoreOS ISO and start a couple of VMs

## ETCD

- Setup a CoreOS etcd cluster. You can start of with one node using the sample etcd.config
```bash
wget https://raw.githubusercontent.com/andrewwebber/couchbase-array/master/coreos/etcd.config
# ... change IP address of server (nodes) in etcd.config before continuing
sudo coreos-install -c etcd.config -C alpha -d /dev/sda && sudo shutdown -r now
```

- Setup a Couchbase server node.
```bash
wget https://raw.githubusercontent.com/andrewwebber/couchbase-array/master/coreos/server-node.config
# ... change IP address for etcd peers in server-node.config before continuing
sudo coreos-install -c server-node.config -C alpha -d /dev/sda && sudo shutdown -r now
```
