package couchbasearray

import (
	"errors"
	"log"
	"time"
)

// StartScheduler starts a scheduling loop
func StartScheduler(servicePath string, timeoutInSeconds int, stop <-chan bool) {
	for {
		currentStates, err := Schedule(servicePath)
		if err != nil {
			log.Println(err)
		}

		master, err := GetMasterNode(currentStates)
		if err == nil {
			ttl := time.Now().Add(time.Duration(timeoutInSeconds+3) * time.Second).UnixNano()
			master.TTL = ttl
			currentStates[master.SessionID] = master
		}

		if err == nil {
			err = SaveClusterStates(servicePath, currentStates)
			if err != nil {
				log.Println(err)
			}
		}

		select {
		case <-time.After(time.Duration(timeoutInSeconds) * time.Second):
		case <-stop:
			log.Println("Stopping scheduling")
		}
	}
}

// GetMasterNode gets the master node
func GetMasterNode(nodes map[string]NodeState) (NodeState, error) {
	var master NodeState
	for _, masterState := range nodes {
		if masterState.Master {
			return masterState, nil
		}
	}

	return master, errors.New("Not found")
}
