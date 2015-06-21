rm -rf /tmp/couchbase-array
mkdir -p /tmp/couchbase-array
go get github.com/andrewwebber/couchbase-array/couchbase-node-announce
CGO_ENABLED=0 GOOS=linux go build -a -o /tmp/couchbase-array/couchbase-node-announce.linux -a -tags netgo -ldflags '-w' github.com/andrewwebber/couchbase-array/couchbase-node-announce
cp src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/** /tmp/couchbase-array
docker build -t andrewwebber/couchbase-cloudarray /tmp/couchbase-array
