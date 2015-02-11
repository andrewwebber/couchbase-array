CGO_ENABLED=0 GOOS=linux go build -o src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/couchbase-node-announce.linux -a -tags netgo -ldflags '-w' github.com/andrewwebber/couchbase-array/couchbase-node-announce
docker build -t andrewwebber/couchbase-cloudarray src/github.com/andrewwebber/couchbase-array/couchbase-node/Docker/
