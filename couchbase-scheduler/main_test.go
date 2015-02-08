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

var SchedulerDesiredStateJoinCluster = "join"
var SchedulerDesiredStateDelete = "delete"
var SchedulerStateDeleted = "join"

func TestClusterInitialization(t *testing.T) {
	path := "/TestClusterInitialization"
	_, err := CreateTestNodes(path, 2)
	if err != nil {
		t.Fatal(err)
	}

	announcements, err := GetClusterAnnouncements(path)
	if err != nil {
		t.Fatal(err)
	}

	currentStates, err := GetClusterStates(path)
	if err != nil {
		t.Fatal(err)
	}

	for key, value := range announcements {
		if state, ok := currentStates[key]; ok {
			if state.SessionID == value.SessionID {
				continue
			} else {
				state.DesiredState = SchedulerDesiredStateJoinCluster
			}
		} else {
			currentStates[key] = NodeState{value, false, "", SchedulerDesiredStateJoinCluster}
		}
	}

	for key := range currentStates {
		if _, ok := announcements[key]; ok {
			continue
		} else {
			delete(currentStates, key)
		}
	}

	log.Println(currentStates)

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

		values[node.Key] = state
	}

	log.Println("Returing")
	return values, nil
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
		values[node.Key] = NodeAnnouncement{node.Key, node.Value}
	}

	return values, nil
}

func CreateTestNodes(base string, count int) (map[string]NodeAnnouncement, error) {
	client := NewEtcdClient()
	values := make(map[string]NodeAnnouncement)
	for i := 0; i < count; i++ {
		ip := fmt.Sprintf("10.100.2.%v", i)
		path := fmt.Sprintf("%s/announcements/%s", base, ip)
		log.Println("Setting ", path)
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
