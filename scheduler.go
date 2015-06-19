package couchbasearray

import (
	"log"
	"time"
)

func StartScheduler(servicePath string, timeoutInSeconds int) {
	for {
		currentStates, err := Schedule(servicePath)
		if err != nil {
			log.Fatal(err)
		}

		err = SaveClusterStates(servicePath, currentStates)
		if err != nil {
			log.Fatal(err)
		}

		time.Sleep(time.Duration(timeoutInSeconds) * time.Second)
	}
}
