package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

var SchedulerStateNew = "new"
var SchedulerStateClustered = "clustered"
var SchedulerStateDeleted = "deleted"

func main() {
	for {
		currentStates, err := Schedule("/services/couchbase-array")
		if err != nil {
			panic(err)
		}

		log.Println("Current States")
		log.Println(currentStates)
	}
}

func Schedule(path string) (map[string]NodeState, error) {
	announcements, err := GetClusterAnnouncements(path)
	if err != nil {
		return nil, err
	}

	currentStates, err := GetClusterStates(path)
	if err != nil {
		return nil, err
	}

	return ScheduleCore(announcements, currentStates), nil
}

func ScheduleCore(announcements map[string]NodeAnnouncement, currentStates map[string]NodeState) map[string]NodeState {
	for key := range currentStates {
		log.Println("Current state key ", key)
	}

	for key, value := range announcements {
		if state, ok := currentStates[key]; ok {
			if state.SessionID == value.SessionID {
				continue
			} else {
				log.Println("Resetting node")
				state.DesiredState = SchedulerStateClustered
				state.State = SchedulerStateNew
				currentStates[key] = state
			}
		} else {
			log.Println("Unabled to find state for node ", key)
			currentStates[key] = NodeState{value, false, SchedulerStateNew, SchedulerStateClustered}
		}
	}

	for key := range currentStates {
		if _, ok := announcements[key]; ok {
			continue
		} else {
			delete(currentStates, key)
		}
	}

	return currentStates
}

func GetClusterStates(base string) (map[string]NodeState, error) {
	values := make(map[string]NodeState)
	client := NewEtcdClient()
	key := fmt.Sprintf("%s/states/", base)
	response, err := client.Get(key, false, false)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			return values, nil
		}
		return nil, err
	}

	for _, node := range response.Node.Nodes {
		var state NodeState
		err = json.Unmarshal([]byte(node.Value), &state)
		if err != nil {
			return nil, err
		}

		sections := strings.Split(node.Key, "/")
		nodeKey := sections[len(sections)-1]
		values[nodeKey] = state
		log.Println("Loaded state ", state)
	}

	return values, nil
}

func SaveClusterStates(base string, states map[string]NodeState) error {
	client := NewEtcdClient()
	for _, stateValue := range states {
		bytes, err := json.Marshal(stateValue)
		key := fmt.Sprintf("%s/states/%s", base, stateValue.IPAddress)
		log.Println("Saving State ", stateValue)
		_, err = client.Set(key, string(bytes), 0)
		if err != nil {
			return err
		}
	}

	return nil
}

func ClearClusterStates(base string) error {
	client := NewEtcdClient()
	key := fmt.Sprintf("%s/states/", base)
	_, err := client.Delete(key, true)
	return err
}

func ClearAnnouncments(base string) error {
	client := NewEtcdClient()
	key := fmt.Sprintf("%s/announcements/", base)
	_, err := client.Delete(key, true)
	return err
}

func GetClusterAnnouncements(path string) (map[string]NodeAnnouncement, error) {
	values := make(map[string]NodeAnnouncement)
	client := NewEtcdClient()
	key := fmt.Sprintf("%s/announcements/", path)
	response, err := client.Get(key, false, false)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			return values, nil
		}
		return nil, err
	}

	for _, node := range response.Node.Nodes {
		sections := strings.Split(node.Key, "/")
		nodeKey := sections[len(sections)-1]
		values[nodeKey] = NodeAnnouncement{nodeKey, node.Value}
	}

	return values, nil
}

type Node struct {
	Address string
	Joined  bool
}

type NodeAnnouncement struct {
	IPAddress string
	SessionID string
}

type NodeState struct {
	NodeAnnouncement
	Master       bool
	State        string
	DesiredState string
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
