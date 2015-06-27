# CoreOS - Couchbase

## ETCD

- Setup a CoreOS etcd cluster. You can start of with one node using the sample etcd.config
```bash
sudo coreos-install -c etcd.config -C alpha -d /dev/sda && sudo shutdown -r now
```
