# Couchbase-array

[![Join the chat at https://gitter.im/andrewwebber/couchbase-array](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/andrewwebber/couchbase-array?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

## Concept

### Couchbase Node

The Couchbase Node is responsible for starting up an instance of Couchbase server. In addition it is responsible for registering it's self as available for clustering

- Each cluster node is disposable when replication at a bucket level is enabled
- Each cluster node runs a disposable docker container activated by a global  Fleet Systemd Unit on CoreOS
- Every time the systemd unit recylces
  1. Removes an existing labels associated within it from a backend store (e.g. etcd)
  2. Starts the couchbase docker container.
  3. Writes a label registering it's self with the backend store for initialization

### Scheduler

The scheduler is responsible for keeping the cluster balanced. Rebalancing the cluster as new cluster node are detected. Rebalancing when cluster nodes are no longer detected.

- The scheduler runs a loop monitoring\capturing the current state of the cluster
- The scheduler schedules actions upon detection of differences between current state and the previous state in the hope of realizing a cluster desired state

### Enforcer

The enforcer is responsible for transitioning cluster nodes to become initialized and rebalancing the cluster
- The enforcer connects to new nodes issuing node initialization requests
- The enforcer connects to active nodes and issues rebalance requests


## Approach

In line with the Unix philosophy a number of existing components will be attempted to be reused to achieve this goal.

- Couchbase Node (Dockerfile)
- Scheduler (Confd + Util)
- Enforcer (Couchbase-init)
