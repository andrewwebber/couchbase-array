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
var machineIdentiferFlag = flag.String("ip", "", "machine ip address")
var whatIfFlag = flag.Bool("t", false, "what if")
var cliBase = flag.String("cli", "/opt/couchbase/bin/couchbase-cli", "path to couchbase cli")

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

	log.Printf("Machine ID: %s\n", machineIdentifier)

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

			master, err := getMasterNode(currentStates)
			if err != nil {
				err := couchbasearray.AcquireLock(sessionID, *servicePathFlag+"/master", 5)
				if err == nil {
					go couchbasearray.StartScheduler(*servicePathFlag, 3)
					go func() {
						failoverSet := false
						for {
							lockErr := couchbasearray.AcquireLock(sessionID, *servicePathFlag+"/master", 5)
							if lockErr != nil {
								log.Fatal(lockErr)
							}

							if !failoverSet {
								if failOverErr := setAutoFailover(machineIdentifier, 31); err != nil {
									log.Println(failOverErr)
								} else {
									failoverSet = true
								}
							}

							time.Sleep(4 * time.Second)
						}
					}()
				}

				if err != nil && err != couchbasearray.ErrLockInUse {
					log.Fatal(err)
				}
			} else {
				if state, ok := currentStates[sessionID]; ok {
					if state.DesiredState != machineState.State {
						log.Printf("DesiredState: %s - Current State: %s", state.DesiredState, machineState.State)

						switch state.DesiredState {
						case couchbasearray.SchedulerStateClustered:
							log.Println("rebalancing")

							if master.IPAddress == machineIdentifier {
								log.Println("Already master no action required")
							} else {
								log.Printf("rebalancing with master node %s\n", master.IPAddress)
								if !*whatIfFlag {
									err = rebalanceNode(master.IPAddress, machineIdentifier)
								}
							}

							if err == nil {
								machineState.State = state.DesiredState
							} else {
								log.Println(err)
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
									if !*whatIfFlag {
										err = addNodeToCluster(master.IPAddress, machineIdentifier)
									}
								}

								if err == nil {
									machineState.State = state.DesiredState
								} else {
									log.Println(err)
								}
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

			couchbasearray.SetClusterAnnouncement(*servicePathFlag, machineState)

			time.Sleep(time.Duration(*ttlFlag) * time.Second)
		}
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)
	log.Println(<-ch)
	log.Println("Failing over")

	currentStates, err := couchbasearray.GetClusterStates(*servicePathFlag)
	if err != nil {
		log.Fatal(err)
		return
	}

	master, err := getMasterNode(currentStates)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = failoverClusterNode(master.IPAddress, machineIdentifier)
	if err != nil {
		log.Fatal(err)
	}
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
				break
			}
		}
	}

	if result != "" {
		return result, nil
	}

	return os.Hostname()
}
