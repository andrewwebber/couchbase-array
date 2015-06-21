package couchbasearray

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/coreos/go-etcd/etcd"
)

var SchedulerStateEmpty = ""
var SchedulerStateNew = "new"
var SchedulerStateRelax = "relax"
var SchedulerStateClustered = "clustered"
var SchedulerStateDeleted = "deleted"
var TTL uint64 = 5

var client *etcd.Client

func init() {
	client = NewEtcdClient()
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

	currentStates = ScheduleCore(announcements, currentStates)
	return SelectMaster(currentStates), nil
}

func ScheduleCore(announcements map[string]NodeState, currentStates map[string]NodeState) map[string]NodeState {
	for key, announcement := range announcements {
		if state, ok := currentStates[key]; ok {
			if state.SessionID == announcement.SessionID {
				if state.DesiredState == SchedulerStateNew && announcement.State == SchedulerStateNew {
					state.DesiredState = SchedulerStateClustered
					currentStates[key] = state
				}

				if state.DesiredState == SchedulerStateClustered && announcement.State == SchedulerStateClustered {
					state.State = SchedulerStateClustered
					state.DesiredState = SchedulerStateClustered
					currentStates[key] = state
				}
			} else {
				log.Println("Resetting node")
				state.DesiredState = SchedulerStateNew
				state.State = SchedulerStateNew
				state.SessionID = announcement.SessionID
				currentStates[key] = state
			}
		} else {
			log.Println("Unabled to find state for node ", key)
			ttl := time.Now().UnixNano()
			currentStates[key] = NodeState{announcement.IPAddress, announcement.SessionID, false, SchedulerStateNew, SchedulerStateNew, ttl}
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

func SelectMaster(currentStates map[string]NodeState) map[string]NodeState {
	if len(currentStates) == 0 {
		return currentStates
	}

	ttl := time.Now().UnixNano()
	var oldMasterKey string
	var lastKey string
	for key, state := range currentStates {
		if state.Master {
			if ttl > state.TTL {
				oldMasterKey = key
				log.Print("Master TTL reached")
			} else {
				return currentStates
			}
		} else {
			lastKey = key
		}
	}

	state := currentStates[lastKey]
	state.Master = true
	currentStates[lastKey] = state

	if oldMasterKey != "" {
		state = currentStates[oldMasterKey]
		state.Master = false
		currentStates[oldMasterKey] = state
	}
	return currentStates
}

func GetClusterStates(base string) (map[string]NodeState, error) {
	values := make(map[string]NodeState)
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
	}

	return values, nil
}

func SaveClusterStates(base string, states map[string]NodeState) error {
	for _, stateValue := range states {
		bytes, err := json.Marshal(stateValue)
		key := fmt.Sprintf("%s/states/%s", base, stateValue.SessionID)
		_, err = client.Set(key, string(bytes), TTL)
		if err != nil {
			return err
		}
	}

	return nil
}

func ClearClusterStates(base string) error {
	key := fmt.Sprintf("%s/states/", base)
	_, err := client.Delete(key, true)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			return nil
		}
	}

	return err
}

func ClearAnnouncments(base string) error {
	key := fmt.Sprintf("%s/announcements/", base)
	_, err := client.Delete(key, true)
	if err != nil {
		if strings.Contains(err.Error(), "Key not found") {
			return nil
		}
	}

	return err
}

func GetClusterAnnouncements(path string) (map[string]NodeState, error) {
	values := make(map[string]NodeState)
	key := fmt.Sprintf("%s/announcements/", path)
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
	}

	return values, nil
}

func SetClusterAnnouncement(base string, state NodeState) error {
	path := fmt.Sprintf("%s/announcements/%s", base, state.SessionID)
	bytes, err := json.Marshal(state)
	if err != nil {
		return err
	}
	if _, err := client.Set(path, string(bytes), TTL); err != nil {
		return err
	}

	return nil
}

type NodeState struct {
	IPAddress    string `json:"ipAddress"`
	SessionID    string `json:"sessionID"`
	Master       bool   `json:"master"`
	State        string `json:"state"`
	DesiredState string `json:"desiredState"`
	TTL          int64  `json:"ttl"`
}

func (n NodeState) String() string {
	return fmt.Sprintf("IP:%s, ID:%s, IsMaster:%v, State:%s, DesiredState:%s",
		n.IPAddress,
		n.SessionID,
		n.Master,
		n.State,
		n.DesiredState)
}

var etcdClient *etcd.Client

func NewEtcdClient() (client *etcd.Client) {
	if etcdClient != nil {
		return etcdClient
	}

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
