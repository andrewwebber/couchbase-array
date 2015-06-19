package main

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pborman/uuid"

	"flag"

	couchbasearray "github.com/andrewwebber/couchbase-array"
)

var servicePathFlag = flag.String("s", "/services/couchbase-array", "etcd directory")
var ttlFlag = flag.Int("ttl", 3, "time to live in seconds")
var debugFlag = flag.Bool("v", false, "verbose")
var processState = flag.Bool("p", true, "process state requests")
var machineIdentiferFlag = flag.String("ip", "", "machine ip address")
var whatIfFlag = flag.Bool("t", false, "what if")

func main() {
	flag.Parse()

	machineIdentifier := *machineIdentiferFlag
	if machineIdentifier == "" {
		var err error
		machineIdentifier, err = getMachineIdentifier()
		if err != nil {
			log.Fatal(err)
		}
	}

	sessionID := uuid.New()

	go func() {
		for {
			announcments, err := couchbasearray.GetClusterAnnouncements(*servicePathFlag)
			if err != nil {
				panic(err)
			}

			machineState, ok := announcments[sessionID]
			if !ok {
				machineState = couchbasearray.NodeState{
					IPAddress:    machineIdentifier,
					SessionID:    sessionID,
					Master:       false,
					State:        "",
					DesiredState: ""}
			}

			currentStates, err := couchbasearray.GetClusterStates(*servicePathFlag)

			if err == nil {
				if state, ok := currentStates[sessionID]; ok {
					if state.DesiredState != machineState.State {
						log.Printf("DesiredState: %s - Current State: %s", state.DesiredState, machineState.State)
						if *processState {
							switch state.DesiredState {
							case couchbasearray.SchedulerStateClustered:
								log.Println("rebalancing")
								master, err := getMasterNode(currentStates)
								if err != nil {
									log.Println(err)
								} else {
									if master.IPAddress == machineIdentifier {
										log.Println("Already master no action required")
									} else {
										log.Printf("rebalancing with master node %s\n", master.IPAddress)
									}

									machineState.State = state.DesiredState
								}

							case couchbasearray.SchedulerStateNew:
								log.Println("adding server to cluster")
								master, err := getMasterNode(currentStates)
								if err != nil {
									log.Println(err)
								} else {
									if master.IPAddress == machineIdentifier {
										log.Println("Already master no action required")
									} else {
										log.Printf("Adding to master node %s\n", master.IPAddress)
									}

									machineState.State = state.DesiredState
								}

							default:
								log.Println(state.DesiredState)
								log.Fatal("unknown state")
							}
						}
					} else {
						log.Println("Running")
					}
				}
			}

			couchbasearray.SetClusterAnnouncement(*servicePathFlag, machineState)

			time.Sleep(time.Duration(*ttlFlag) * time.Second)
		}
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	log.Println(<-ch)
	log.Println("Failover")
}

func getMasterNode(nodes map[string]couchbasearray.NodeState) (couchbasearray.NodeState, error) {
	var master couchbasearray.NodeState
	for _, masterState := range nodes {
		if masterState.Master {
			return masterState, nil
		}
	}

	return master, errors.New("Not found")
}

func getMachineIdentifier() (string, error) {

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Fatalln(err)
	}

	var result string
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				result = ipnet.IP.String()
				log.Println(ipnet.Network())
				log.Printf("Found IP %s\n", result)
			}
		}
	}

	if result != "" {
		return result, nil
	}

	return os.Hostname()
}
