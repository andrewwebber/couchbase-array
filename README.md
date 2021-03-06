# Elastic Couchbase Server Docker Container Array

## Features
- Automatically add and rebalance **cattle** couchbase nodes using etcd as a discovery service
- Gracefull failover/remove node on container shutdown
- Survive ETCD outages

[![Join the chat at https://gitter.im/andrewwebber/couchbase-array](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewwebber/couchbase-array?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Concept

### Couchbase Node

The Couchbase Node is responsible for starting up an instance of Couchbase server. In addition it is responsible for registering it's self as available for clustering

- Each cluster node is disposable when replication at a bucket level is enabled
- Each cluster node runs a disposable docker container activated by a systemd unit
- Every time the systemd unit recycles
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

## Gracefull faillover and Delta Rebalancing
- As a container shuts down it will try issue a gracefull failover
  + The container will block and wait until the gracefull failover has completed
  + The container will then issue an asynchronous rebalance before exiting
  + Here is it important that the container is given enough time to gracefully shutdown

    ```bash
    docker stop --time=120 couchbase
    ```

- As a container starts it will try to add its self to the cluster
  + If it is already a member of the cluster it will issue a 'setRecoveryType' to delta
  + In any case it will finally trigger a rebalance

Currently the program sets auto failover to be 31 seconds.

## Building and testing

The project requires a golang project structure

1.  Build the Docker container

    ```bash
    ./src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/build.sh
    ```

2.  Start etcd (below using boot2docker IPAddress)

    ```bash
    go get github.com/coreos/etcd
    etcd --advertise-client-urls=http://172.16.237.1:4001,http://localhost:4001 --listen-client-urls=http://172.16.237.1:4001,http://localhost:4001
    ```

3.  Start as many couchbase containers as you want

    ```bash
    docker run -d --name couchbase1 -p 8091:8091 -e ETCDCTL_PEERS=http://172.16.237.1:4001 andrewwebber/couchbase-cloudarray
    docker run -d --name couchbase2 -e ETCDCTL_PEERS=http://172.16.237.1:4001 andrewwebber/couchbase-cloudarray
    docker run -d --name couchbase3 -e ETCDCTL_PEERS=http://172.16.237.1:4001 andrewwebber/couchbase-cloudarray
    ```

4.  Browser to a node [http://172.16.237.103:8091] and login with **Administrator** **password**

5.  Destroy and start containers at will


## Production setup

In production the docker arguments simply change to use **--net="host"**
- When a container starts it will rebalance of a master node
- When a container stop it will gracefully failover from the cluster and issue a rebalance on exit

Below is an example systemd service unit

```bash
[Service]
TimeoutSec=0
ExecStartPre=-/usr/bin/mkdir /home/core/couchbase
ExecStartPre=/usr/bin/chown 999:999 /home/core/couchbase
ExecStartPre=-/usr/bin/docker kill couchbase
ExecStartPre=-/usr/bin/docker rm -f couchbase
ExecStart=/usr/bin/docker run --name couchbase --net="host" -v /home/core/couchbase:/opt/couchbase/var -e ETCDCTL_PEERS=http://192.168.89.215:4001 --ulimit nofile=40960:40960 --ulimit core=100000000:100000000 --ulimit memlock=100000000:100000000 andrewwebber/couchbase-array
ExecStop=/usr/bin/docker kill --signal=SIGTERM couchbase
Restart=always
RestartSec=20
```

## TODO

The direction of the project will be get as many arguments as possible from etcd including:
- Username and password
- Auto failover timeout in seconds
- Whether to issue a rebalance automatically upon graceful faillover
- Whether to issue a rebalance automatically upon new node detection
- Email alerts
