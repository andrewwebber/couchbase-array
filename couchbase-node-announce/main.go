package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
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
var heartBeatFlag = flag.Int("h", 3, "heart beat loop in seconds")
var ttlFlag = flag.Int("ttl", 30, "time to live in seconds")
var debugFlag = flag.Bool("v", false, "verbose")
var rebalanceOnExitFlag = flag.Bool("r", false, "rebalance on exit")
var machineIdentiferFlag = flag.String("ip", "", "machine ip address")
var whatIfFlag = flag.Bool("t", false, "what if")
var cliBase = flag.String("cli", "/opt/couchbase/bin/couchbase-cli", "path to couchbase cli")
var statefulSet = flag.String("statefulset", "", "use stateful")
var masterNodeAnnouncePathFlag = flag.String("m", "/services/couchbase", "announce etcd path for the master IP")

func main() {
	log.SetFlags(log.Llongfile)
	flag.Parse()
	log.Println("Couchbase Cluster Node")
	couchbasearray.TTL = uint64(*ttlFlag)
	log.Printf("TTL %v\n", couchbasearray.TTL)

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
	var isClusterMember bool

	go func() {
		for {
			announcments, err := couchbasearray.GetClusterAnnouncements(*servicePathFlag)
			if err != nil {
				log.Println(err)
				continue
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

			master, err := couchbasearray.GetMasterNode(currentStates)
			if err != nil {
				err := couchbasearray.AcquireLock(sessionID, *servicePathFlag+"/master", 5)
				if err == nil {
					stopScheduler := make(chan bool)
					go couchbasearray.StartScheduler(*servicePathFlag, *heartBeatFlag, stopScheduler, *masterNodeAnnouncePathFlag)
					go func() {
						failoverSet := false
						for {
							lockErr := couchbasearray.AcquireLock(sessionID, *servicePathFlag+"/master", 5)
							if lockErr != nil {
								log.Println(lockErr)
								stopScheduler <- true
								return
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
					log.Println(err)
					continue
				}
			} else {
				if state, ok := currentStates[sessionID]; ok {
					if state.DesiredState != machineState.State {
						log.Printf("DesiredState: %s - Current State: %s", state.DesiredState, machineState.State)

						switch state.DesiredState {
						case couchbasearray.SchedulerStateClustered:
							log.Println("rebalancing")

							if !alreadyClustered() {
								if master.IPAddress == machineIdentifier {
									log.Println("Already master no action required")
								} else {
									log.Printf("rebalancing with master node %s\n", master.IPAddress)
									if !*whatIfFlag {
										if isClusterMember {
											err = recoverNode(master.IPAddress, machineIdentifier)
										}
									}
								}
							}

							if err != nil {
								log.Println(err)
							} else {
								err = rebalanceNode(master.IPAddress, machineIdentifier)
							}

							if err == nil {
								machineState.State = state.DesiredState
							} else {
								log.Println(err)
							}

						case couchbasearray.SchedulerStateNew:
							log.Println("adding server to cluster")
							master, err := couchbasearray.GetMasterNode(currentStates)
							if err != nil {
								log.Println(err)
							} else {
								//if !alreadyClustered() {
								if master.IPAddress == machineIdentifier {
									log.Println("Already master no action required")
								} else {
									log.Printf("Adding to master node %s\n", master.IPAddress)
									if !*whatIfFlag {
										isClusterMember, err = addNodeToCluster(master.IPAddress, machineIdentifier)
										if err == nil {
											ioutil.WriteFile("/opt/couchbase/var/lib/couchbase/_clustered", []byte{}, os.ModePerm)
										}
									}
								}
								//}

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

			err = couchbasearray.SetClusterAnnouncement(*servicePathFlag, machineState)
			if err != nil {
				log.Println(err)
			}

			time.Sleep(time.Duration(*heartBeatFlag) * time.Second)
		}
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGKILL)
	log.Println(<-ch)
	log.Println("Failing over")
	log.Println("waiting for TTL drain")
	time.Sleep(time.Duration(*ttlFlag*2) * time.Second)

	currentStates, err := couchbasearray.GetClusterStates(*servicePathFlag)
	if err != nil {
		log.Fatal(err)
		return
	}

	master, err := couchbasearray.GetMasterNode(currentStates)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = failoverClusterNode(master.IPAddress, machineIdentifier)
	if err != nil {
		log.Fatal(err)
	}

	if *rebalanceOnExitFlag {
		time.Sleep(10 * time.Second)
		err = rebalanceNode(master.IPAddress, machineIdentifier)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func alreadyClustered() bool {
	// return false
	if _, err := os.Stat("/opt/couchbase/var/lib/couchbase/_clustered"); err == nil {
		log.Println("Already previously clustered")
		return true
	}

	return false
}

func getMachineIdentifier() (string, error) {
	if *statefulSet != "" {
		hostname, err := os.Hostname()
		if err != nil {
			return hostname, err
		}

		return fmt.Sprintf("%s%s", hostname, *statefulSet), nil
	}

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

type function func() error

func exponential(operation function, maxRetries int) error {
	var err error
	var sleepTime int
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}
		if i == 0 {
			sleepTime = 1
		} else {
			sleepTime = int(math.Exp2(float64(i)) * 100)
		}
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		log.Printf("Retry exponential: Attempt %d, sleep %d", i, sleepTime)
	}

	return err
}
