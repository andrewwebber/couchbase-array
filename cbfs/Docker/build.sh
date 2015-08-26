rm -rf /tmp/cbfs
mkdir -p /tmp/cbfs
go get github.com/couchbaselabs/cbfs
CGO_ENABLED=0 GOOS=linux go build -o /tmp/cbfs/cbfs.linux -a -tags netgo -ldflags '-w' github.com/couchbaselabs/cbfs
cp src/github.com/andrewwebber/couchbase-array/cbfs/Docker/** /tmp/cbfs
sudo docker build -t andrewwebber/cbfs /tmp/cbfs
