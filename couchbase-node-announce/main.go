package main

import (
	"log"
	"net"
	"os"
	"strings"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"flag"
	"fmt"

	"github.com/coreos/go-etcd/etcd"
)

var servicePathFlag = flag.String("path", "/services/couchbase-array/announcements", "etcd directory")
var ttlFlag = flag.Int("ttl", 30, "time to live in seconds")
var debugFlag = flag.Bool("v", false, "verbose")

func main() {
	flag.Parse()
	client := NewEtcdClient()

	machineIdentifier, err := getMachineIdentifier()
	if err != nil {
		panic(err)
	}

	directory := fmt.Sprintf("%s/%s", *servicePathFlag, machineIdentifier)
	sessionID := uuid.New()

	for {
		_, err = client.Set(directory, sessionID, uint64(*ttlFlag+10))
		if err != nil {
			panic(err)
		}

		if *debugFlag {
			log.Printf("Written to %s\n", directory)
		}

		time.Sleep(time.Duration(*ttlFlag) * time.Second)
	}
}

func NewEtcdClient() (client *etcd.Client) {
	var etcdClient *etcd.Client
	peersStr := os.Getenv("ETCDCTL_PEERS")
	if len(peersStr) > 0 {
		log.Println("Connecting to etcd peers : " + peersStr)
		peers := strings.Split(peersStr, ",")
		etcdClient = etcd.NewClient(peers)
	} else {
		etcdClient = etcd.NewClient(nil)
	}

	return etcdClient
}

func getMachineIdentifier() (string, error) {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return os.Hostname()
}
