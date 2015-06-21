./src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/build.sh
docker run -d --name couchbase3 -e ETCDCTL_PEERS=http://192.168.89.1:4001 andrewwebber/couchbase-cloudarray
docker run -d --name couchbase2 -e ETCDCTL_PEERS=http://192.168.89.1:4001 andrewwebber/couchbase-cloudarray
docker run -d --name couchbase1 -p 8091:8091 -e ETCDCTL_PEERS=http://192.168.89.1:4001 andrewwebber/couchbase-cloudarray
