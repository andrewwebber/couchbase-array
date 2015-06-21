# Elastic Couchbase Server Docker Container Array

## Features
- Automatically add and rebalance **cattle** couchbase nodes using etcd as a discovery service
- Gracefull failover/remove node on container shutdown
- Survive ETCD outages

## Concept

### Couchbase Node

The Couchbase Node is responsible for starting up an instance of Couchbase server. In addition it is responsible for registering it's self as available for clustering

- Each cluster node is disposable when replication at a bucket level is enabled
- Each cluster node runs a disposable docker container activated by a systemd unit
- Every time the systemd unit recylces
  1. Starts the couchbase docker container.
  2. Attempts to acquire a scheduler lock from etcd
  3. Writes a label registering it's self with the backend store for initialization
  4. Listens to state change requests from the master scheduler

### Scheduler

The scheduler is responsible for keeping the cluster balanced. Rebalancing the cluster as new cluster node are detected. Rebalancing when cluster nodes are no longer detected.

- The scheduler runs a loop monitoring the current state of the cluster
- The scheduler schedules actions upon detection of differences between node current state and the node desired state

  1. On acquiring a cluster wide master lock the scheduler is started
  2. The scheduler will elect a master node and set standard cluster settings like auto failure policy
  3. As nodes are detected desired actions are issued to nodes via etcd
  4. If the master goes down another cluster node aquires the master lock and begins the scheduler

## Building and testing

The project requires a golang project structure

1.  Build the Docker container

    ```bash
    ./src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/build.sh
    ```

2.  Start etcd (below using boot2docker IPAddress)

    ```bash
    go get github.com/coreos/etcd
    etcd --advertise-client-urls=http://192.168.89.1:4001,http://localhost:4001 --listen-client-urls=http://192.168.89.1:4001,http://localhost:4001
    ```

3.  Start as many couchbase containers as you want

    ```bash
    docker run -d --name couchbase1 -p 8091:8091 -e ETCDCTL_PEERS=http://192.168.89.1:4001 andrewwebber/couchbase-cloudarray
    docker run -d --name couchbase2 -e ETCDCTL_PEERS=http://192.168.89.1:4001 andrewwebber/couchbase-cloudarray
    docker run -d --name couchbase3 -e ETCDCTL_PEERS=http://192.168.89.1:4001 andrewwebber/couchbase-cloudarray
    ```

4.  Browser to a node [http://192.168.89.103:8091] and login with **Administrator** **password**

5.  Destroy and start containers at will


## Production setup

In production the docker arguments simply change to use **--net="host"**

Below is an example systemd service unit

```bash
[Service]
TimeoutStartSec=10m
ExecStartPre=-/usr/bin/docker kill couchbase
ExecStartPre=-/usr/bin/docker rm couchbase
ExecStart=/usr/bin/docker run --rm -it --name couchbase --net="host" -e ETCDCTL_PEERS=http://10.100.2.2:4001 --ulimit nofile=40960:40960 --ulimit core=100000000:100000000 --ulimit memlock=100000000:100000000  andrewwebber/couchbase-cloudarray
Restart=always
RestartSec=20
```
