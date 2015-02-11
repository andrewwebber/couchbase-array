package main

import (
	"flag"
	"log"
	"time"

	"github.com/andrewwebber/couchbase-array"
)

var SchedulerStateNew = "new"
var SchedulerStateClustered = "clustered"
var SchedulerStateDeleted = "deleted"

var servicePathFlag = flag.String("path", "/services/couchbase-array", "etcd directory")
var timeOutFlag = flag.Int64("t", 10, "timeout look in seconds")

func main() {
	for {
		currentStates, err := couchbasearray.Schedule(*servicePathFlag)
		if err != nil {
			panic(err)
		}

		log.Println("Current States")
		log.Println(currentStates)
		err = couchbasearray.SaveClusterStates(*servicePathFlag, currentStates)
		if err != nil {
			panic(err)
		}

		time.Sleep(time.Duration(*timeOutFlag) * time.Second)
	}
}
