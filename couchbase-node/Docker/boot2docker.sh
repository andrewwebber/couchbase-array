./src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/build.sh
# docker run -d --name couchbase3 -e ETCDCTL_PEERS=http://172.17.0.1:4001 andrewwebber/couchbase-array
# docker run -d --name couchbase2 -e ETCDCTL_PEERS=http://172.17.0.1:4001 andrewwebber/couchbase-array
etcd &
docker run --rm -it --name couchbase1 --net=host -e ETCDCTL_PEERS=http://localhost:4001 andrewwebber/couchbase-array:5.0.1
