#!/bin/bash

set -m # Enable Job Control

echo "Starting Couchbase"
cd /opt/couchbase
mkdir -p var/lib/couchbase var/lib/couchbase/config var/lib/couchbase/data \
    var/lib/couchbase/stats var/lib/couchbase/logs var/lib/moxi
chown -R couchbase:couchbase var

if [ -v "$KUBERNETES" ]; then
  echo "Setting hostname"
  echo "$(hostname).couchbase-master-service" > /opt/couchbase/var/lib/couchbase/ip
  echo "$(hostname).couchbase-master-service" > /opt/couchbase/var/lib/couchbase/ip_start
fi

# /etc/init.d/couchbase-server start
/entrypoint.sh couchbase-server &

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
  status=$?
  while [ $status -ne 0 -a $status -ne 2 ]
  do
    echo Retrying... $status
    sleep 1
    "$@"
    status=$?
  done

  echo Success $status
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

untilsuccessful curl 127.0.0.1:8091

if [ ! -f /opt/couchbase/var/lib/couchbase/_init ]; then
  RAMSIZE=0
  RAMSIZE=$(cat /proc/meminfo | grep MemFree | awk '{print $2}')
  echo "Initiated with RAM SIZE $RAM_SIZE"
  echo "Configuring Couchbase cluster with services --service=data,index,query"
  /opt/couchbase/bin/couchbase-cli cluster-init --cluster-username $COUCHBASE_ADMIN --cluster-password $COUCHBASE_PASSWORD -c 127.0.0.1:8091 --cluster-ramsize=$RAM_SIZE --cluster-index-ramsize=512 #--service=data,index,query
  touch /opt/couchbase/var/lib/couchbase/_init
fi

echo "Cluster up"
export PATH=$PATH:/opt/couchbase/bin/
couchbase-node-announce $@ &
C_PID=$!
wait $C_PID
