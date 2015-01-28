#!/bin/bash

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
    sleep 10
    "$@"
  done

  exit $?
}

RAMSIZE=0
RAMSIZE=$(cat /proc/meminfo | grep MemAvailable | awk '{print $2}')
let "SIZE = $RAMSIZE/1024"
echo "RAM SIZE" $(echo $RAMSIZE)
echo "Init RAM SIZE" $(echo $SIZE)
echo "Configuring Couchbase cluster"
untilsuccessful /opt/couchbase/bin/couchbase-cli cluster-init -u Administrator -p password -c 127.0.0.1:8091 \
--cluster-init-username=Administrator --cluster-init-password=password \
--cluster-init-ramsize=$SIZE

echo "Cluster up"
untilunsuccessful curl 127.0.0.1:8091
