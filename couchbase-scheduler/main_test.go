package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"code.google.com/p/go-uuid/uuid"

	"github.com/coreos/go-etcd/etcd"
)

var SchedulerStateNew = "new"
var SchedulerStateClustered = "clustered"
var SchedulerStateDeleted = "deleted"

func TestClusterScenarios(t *testing.T) {
	path := "/TestClusterInitialization"
	if err := ClearClusterStates(path); err != nil {
		t.Fatal(err)
	}
	if err := ClearAnnouncments(path); err != nil {
		t.Fatal(err)
	}
	//
	//	First cluster boostrap
	//
	_, err := CreateTestNodes(path, 2)
	if err != nil {
		t.Fatal(err)
	}

	currentStates, err := Schedule(path)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Current States")
	log.Println(currentStates)

	log.Println(currentStates)
	for _, state := range currentStates {
		if state.DesiredState != SchedulerStateClustered {
			t.Fatal("Expected states should be 'clustered'")
		}

		if state.State != SchedulerStateNew {
			t.Fatal("Expected states should be 'new'")
		}
	}
	//
	// Set status to clustered
	//
	for key, state := range currentStates {
		state.State = state.DesiredState
		currentStates[key] = state
	}

	SaveClusterStates(path, currentStates)

	currentStates, err = Schedule(path)
	if err != nil {
		t.Fatal(err)
	}

	log.Println("Current States")
	log.Println(currentStates)

	log.Println(currentStates)
	for _, state := range currentStates {
		if state.DesiredState != SchedulerStateClustered {
			t.Fatal("Expected states should be 'clustered'")
		}

		if state.State != SchedulerStateClustered {
			t.Fatal("Expected states should be 'clustered'")
		}
	}

	//
	//	Simulate machine reboot
	//
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
	for key, _ := range currentStates {
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

func TestGetClusterAnnouncements(t *testing.T) {
	path := "/TestGetClusterAnnouncements"
	testNodes, err := CreateTestNodes(path, 2)
	if err != nil {
		t.Fatal(err)
	}

	nodes, err := GetClusterAnnouncements(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(nodes) != len(testNodes) {
		t.Fatal("Difference in result lengths")
	}
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

func CreateTestNodes(base string, count int) (map[string]NodeAnnouncement, error) {
	client := NewEtcdClient()
	values := make(map[string]NodeAnnouncement)
	for i := 0; i < count; i++ {
		ip := fmt.Sprintf("10.100.2.%v", i)
		path := fmt.Sprintf("%s/announcements/%s", base, ip)
		id := uuid.New()
		values[ip] = NodeAnnouncement{ip, id}
		if _, err := client.Set(path, id, 0); err != nil {
			return nil, err
		}
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
