package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func localOtpNode(liveNodeIP string, nodeIP string) (otpNode string, err error) {

	otpNodeList, err := otpNodeList(liveNodeIP)
	if err != nil {
		return "", err
	}

	for _, otpNode := range otpNodeList {
		if strings.Contains(otpNode, nodeIP) {
			return otpNode, nil
		}
	}

	return "", fmt.Errorf("No otpnode found with ip %v in %v", nodeIP, otpNodeList)
}

func otpNodeList(liveNodeIP string) ([]string, error) {

	otpNodeList := []string{}

	nodes, err := getClusterNodes(liveNodeIP)
	if err != nil {
		return otpNodeList, err
	}

	for _, node := range nodes {

		nodeMap, ok := node.(map[string]interface{})
		if !ok {
			return otpNodeList, fmt.Errorf("Node had unexpected data type")
		}

		otpNode := nodeMap["otpNode"] // ex: "ns_1@10.231.192.180"
		otpNodeStr, ok := otpNode.(string)
		log.Printf("OtpNodeList, otpNode: %v", otpNodeStr)

		if !ok {
			return otpNodeList, fmt.Errorf("No otpNode string found")
		}

		otpNodeList = append(otpNodeList, otpNodeStr)

	}

	return otpNodeList, nil
}

func getClusterNodes(liveNodeIP string) ([]interface{}, error) {
	endpointURL := fmt.Sprintf("http://%v:8091/pools/default", liveNodeIP)
	requestURL := fmt.Sprintf(endpointURL)
	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth("Administrator", "password")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	if resp.StatusCode > 202 {
		return nil, errors.New(resp.Status)
	}

	jsonMap := map[string]interface{}{}
	if err := json.Unmarshal(body, &jsonMap); err != nil {
		return nil, err
	}

	nodes := jsonMap["nodes"]

	nodeMaps, ok := nodes.([]interface{})
	if !ok {
		return nil, fmt.Errorf("Unexpected data type in nodes field")
	}

	return nodeMaps, nil
}

func setAutoFailover(masterIP string, timeoutInSeconds int) error {
	endpointURL := fmt.Sprintf("http://%v:%v/settings/autoFailover", masterIP, 8091)
	log.Println(endpointURL)
	data := url.Values{
		"enabled": {"true"},
		"timeout": {strconv.Itoa(timeoutInSeconds)}}

	preq, err := http.NewRequest("POST", endpointURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	preq.SetBasicAuth("Administrator", "password")

	preq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	pclient := &http.Client{}
	presp, err := pclient.Do(preq)
	if err != nil {
		return err
	}

	if presp.StatusCode != 200 {
		log.Println(presp.Status)
		return errors.New("Invalid status code")
	}

	return err
}

func addNodeToCluster(masterIP string, nodeIP string) (bool, error) {
	endpointURL := fmt.Sprintf("http://%s:%v/controller/addNode", masterIP, 8091)
	log.Println(endpointURL)
	data := url.Values{
		"hostname": {nodeIP},
		"user":     {"Administrator"},
		"password": {"password"},
		"services": {"kv,index,n1ql"},
	}

	preq, err := http.NewRequest("POST", endpointURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return false, err
	}

	preq.SetBasicAuth("Administrator", "password")

	preq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	pclient := &http.Client{}
	presp, err := pclient.Do(preq)
	if err != nil {
		return false, err
	}

	body, err := ioutil.ReadAll(presp.Body)
	if err != nil {
		return false, err
	}

	if presp.StatusCode != 200 {
		log.Println(presp.Status)
		log.Println(string(body))
		if strings.Contains(string(body), "Prepare join failed. Node is already part of cluster.") {
			return true, nil
		}

		return false, errors.New("Invalid status code")
	}

	return false, err
}

func recoverNode(masterIP string, nodeIP string) error {
	local, err := localOtpNode(masterIP, nodeIP)
	if err != nil {
		return err
	}

	endpointURL := fmt.Sprintf("http://%s:%v/controller/setRecoveryType", masterIP, 8091)
	log.Println(endpointURL)
	data := url.Values{
		"otpNode":      {local},
		"recoveryType": {"delta"},
	}

	preq, err := http.NewRequest("POST", endpointURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	preq.SetBasicAuth("Administrator", "password")

	preq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	pclient := &http.Client{}
	presp, err := pclient.Do(preq)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(presp.Body)
	if err != nil {
		return err
	}

	if presp.StatusCode != 200 {
		log.Println(presp.Status)
		log.Println(string(body))
		return errors.New("Invalid status code")
	}

	return err
}

func rebalanceNode(masterIP string, nodeIP string) error {
	pclient := &http.Client{}
	endpointURL := fmt.Sprintf("http://%s:%v/pools/default/rebalanceProgress", masterIP, 8091)
	log.Println(endpointURL)
	for {
		rebalanceRequest, err := http.NewRequest("GET", endpointURL, nil)
		rebalanceRequest.SetBasicAuth("Administrator", "password")
		rResp, err := pclient.Do(rebalanceRequest)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(rResp.Body)
		if err != nil {
			return err
		}

		if rResp.StatusCode != 200 {
			log.Println(rResp.Status)
			log.Println(string(body))
			log.Println("Invalid status code")
		}

		type rebalanceStatus struct {
			Status string `json:"status"`
		}

		var status rebalanceStatus
		if err = json.Unmarshal(body, &status); err != nil {
			return err
		}

		if status.Status != "running" {
			log.Printf("rebalance status: %s", status.Status)
			break
		}

		time.Sleep(1 * time.Second)
		log.Println(status.Status)
	}

	otpNodeList, err := otpNodeList(masterIP)
	if err != nil {
		return err
	}

	otpNodes := strings.Join(otpNodeList, ",")

	endpointURL = fmt.Sprintf("http://%s:%v/controller/rebalance", masterIP, 8091)
	log.Println(endpointURL)
	log.Println(endpointURL)
	data := url.Values{
		"ejectedNodes": {},
		"knownNodes":   {otpNodes},
	}

	preq, err := http.NewRequest("POST", endpointURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	preq.SetBasicAuth("Administrator", "password")

	preq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	presp, err := pclient.Do(preq)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(presp.Body)
	if err != nil {
		return err
	}

	if presp.StatusCode != 200 {
		log.Println(presp.Status)
		log.Println(string(body))
		return errors.New("Invalid status code")
	}

	endpointURL = fmt.Sprintf("http://%s:%v/pools/default/rebalanceProgress", masterIP, 8091)
	log.Println(endpointURL)

	for {
		rebalanceRequest, err := http.NewRequest("GET", endpointURL, nil)
		rebalanceRequest.SetBasicAuth("Administrator", "password")
		rResp, err := pclient.Do(rebalanceRequest)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(rResp.Body)
		if err != nil {
			return err
		}

		if rResp.StatusCode != 200 {
			log.Println(rResp.Status)
			log.Println(string(body))
			return errors.New("Invalid status code")
		}

		type rebalanceStatus struct {
			Status string `json:"status"`
		}

		var status rebalanceStatus
		if err = json.Unmarshal(body, &status); err != nil {
			return err
		}

		if status.Status != "running" {
			log.Printf("rebalance status: %s", status.Status)
			break
		}

		time.Sleep(1 * time.Second)
		log.Println(status.Status)
	}

	return err
}

func failoverClusterNode(masterIP string, nodeIP string) error {
	pclient := &http.Client{}
	endpointURL := fmt.Sprintf("http://%s:%v/pools/default/rebalanceProgress", masterIP, 8091)
	log.Println(endpointURL)

	for {
		rebalanceRequest, err := http.NewRequest("GET", endpointURL, nil)
		rebalanceRequest.SetBasicAuth("Administrator", "password")
		rResp, err := pclient.Do(rebalanceRequest)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(rResp.Body)
		if err != nil {
			return err
		}

		if rResp.StatusCode != 200 {
			log.Println(rResp.Status)
			log.Println(string(body))
			log.Println("Invalid status code")
		}

		type rebalanceStatus struct {
			Status string `json:"status"`
		}

		var status rebalanceStatus
		if err = json.Unmarshal(body, &status); err != nil {
			return err
		}

		if status.Status != "running" {
			log.Printf("rebalance status: %s", status.Status)
			break
		}

		time.Sleep(1 * time.Second)
		log.Println(status.Status)
	}

	local, err := localOtpNode(masterIP, nodeIP)
	if err != nil {
		return err
	}

	endpointURL = fmt.Sprintf("http://%s:%v/controller/startGracefulFailover", masterIP, 8091)
	log.Println(endpointURL)
	data := url.Values{
		"otpNode": {local},
	}

	preq, err := http.NewRequest("POST", endpointURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	preq.SetBasicAuth("Administrator", "password")

	preq.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	presp, err := pclient.Do(preq)
	if err != nil {
		return err
	}

	if presp.StatusCode != 200 {
		log.Println(presp.Status)
		return errors.New("Invalid status code")
	}

	endpointURL = fmt.Sprintf("http://%s:%v/pools/default/rebalanceProgress", masterIP, 8091)
	log.Println(endpointURL)

	for {
		rebalanceRequest, err := http.NewRequest("GET", endpointURL, nil)
		rebalanceRequest.SetBasicAuth("Administrator", "password")
		rResp, err := pclient.Do(rebalanceRequest)
		if err != nil {
			return err
		}

		body, err := ioutil.ReadAll(rResp.Body)
		if err != nil {
			return err
		}

		if rResp.StatusCode != 200 {
			log.Println(rResp.Status)
			log.Println(string(body))
			return errors.New("Invalid status code")
		}

		type rebalanceStatus struct {
			Status string `json:"status"`
		}

		var status rebalanceStatus
		if err = json.Unmarshal(body, &status); err != nil {
			return err
		}

		if status.Status != "running" {
			log.Printf("rebalance status: %s", status.Status)
			break
		}

		time.Sleep(1 * time.Second)
		log.Println(status.Status)
	}

	return err
}
