# Couchbase-array

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

## Building

The project requires a golang project structure

```bash
./src/github.com/andrewwebber/couchbase-cloudarray/couchbase-node/Docker/build.sh
```
