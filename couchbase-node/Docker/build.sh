CGO_ENABLED=0 GOOS=linux go build -o /tmp/couchbase-array/couchbase-node-announce.linux -a -tags netgo -ldflags '-w' github.com/andrewwebber/couchbase-array/couchbase-node-announce
mkdir -p /tmp/couchbase-array
cp src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/** /tmp/couchbase-array
docker build -t andrewwebber/couchbase-cloudarray /tmp/couchbase-array
