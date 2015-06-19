#!/bin/bash

set -m # Enable Job Control

control_c()
# run if user hits control-c
{
  echo -en "\n*** Exiting ***\n"
  exit 0
}

# trap keyboard interrupt (control-c)
trap control_c SIGINT

echo "Starting Couchbase"
/etc/init.d/couchbase-server start

untilsuccessful() {
  "$@"
  while [ $? -ne 0 ]
  do
    echo Retrying...
    sleep 1
    "$@"
  done
}

untilunsuccessful() {
  "$@"
  while [ $? -eq 0 ]
  do
    echo Heartbeat successful...
    sleep 60
    "$@"
  done

  exit $?
}

RAMSIZE=0
RAMSIZE=$(cat /proc/meminfo | grep MemFree | awk '{print $2}')
echo "Acceptable RAM SIZE" $(echo $RAM_SIZE)
echo "Configuring Couchbase cluster"
untilsuccessful /opt/couchbase/bin/couchbase-cli cluster-init -u Administrator -p password -c 127.0.0.1:8091 \
--cluster-init-username=Administrator --cluster-init-password=password \
--cluster-init-ramsize=$RAM_SIZE

echo "Cluster up"
#untilunsuccessful curl 127.0.0.1:8091
export PATH=$PATH:/opt/couchbase/bin/
couchbase-node-announce $@
