rm -rf /tmp/cbfs-client
mkdir -p /tmp/cbfs-client
go get github.com/couchbaselabs/cbfs/tools/cbfsclient
CGO_ENABLED=0 GOOS=linux go build -o /tmp/cbfs-client/cbfs-client.linux -a -tags netgo -ldflags '-w' github.com/couchbaselabs/cbfs/tools/cbfsclient
cp src/github.com/andrewwebber/couchbase-array/cbfs-client/Docker/** /tmp/cbfs-client
cp -r src/github.com/couchbaselabs/cbfs/monitor /tmp/cbfs-client/monitor
sudo docker build -t andrewwebber/cbfs-client /tmp/cbfs-client
