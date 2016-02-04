#!/bin/bash

set -m # Enable Job Control

echo "Starting Couchbase"
cd /opt/couchbase
mkdir -p var/lib/couchbase var/lib/couchbase/config var/lib/couchbase/data \
    var/lib/couchbase/stats var/lib/couchbase/logs var/lib/moxi
chown -R couchbase:couchbase var
/etc/init.d/couchbase-server start

function clean_up {

	# Perform program exit housekeeping
	echo "# Perform program exit housekeeping $(echo $C_PID)"
  kill -SIGTERM $C_PID
  wait $C_PID
	exit
}

trap 'clean_up' SIGHUP SIGINT SIGTERM SIGKILL TERM

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
echo "Configuring Couchbase cluster with services --service=data,index,query"
untilsuccessful /opt/couchbase/bin/couchbase-cli cluster-init -u Administrator -p password -c 127.0.0.1:8091 \
--cluster-init-username=Administrator --cluster-init-password=password \
--cluster-init-ramsize=1000 --service=data,index,query

echo "Cluster up"
#untilunsuccessful curl 127.0.0.1:8091
export PATH=$PATH:/opt/couchbase/bin/
couchbase-node-announce $@ &
C_PID=$!
wait $C_PID
