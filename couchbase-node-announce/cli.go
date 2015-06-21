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

func addNodeToCluster(masterIP string, nodeIP string) error {
	endpointURL := fmt.Sprintf("http://%s:%v/controller/addNode", masterIP, 8091)
	log.Println(endpointURL)
	data := url.Values{
		"hostname": {nodeIP},
		"user":     {"Administrator"},
		"password": {"password"},
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
		if strings.Contains(string(body), "Prepare join failed. Node is already part of cluster.") {
			return nil
		}

		return errors.New("Invalid status code")
	}

	return err
}

func rebalanceNode(masterIP string, nodeIP string) error {
	otpNodeList, err := otpNodeList(masterIP)
	if err != nil {
		return err
	}

	otpNodes := strings.Join(otpNodeList, ",")

	endpointURL := fmt.Sprintf("http://%s:%v/controller/rebalance", masterIP, 8091)
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

func failoverClusterNode(masterIP string, nodeIP string) error {
	otpNodeList, err := otpNodeList(masterIP)
	if err != nil {
		return err
	}

	otpNodes := strings.Join(otpNodeList, ",")

	local, err := localOtpNode(masterIP, nodeIP)
	if err != nil {
		return err
	}

	endpointURL := fmt.Sprintf("http://%s:%v/controller/rebalance", masterIP, 8091)
	log.Println(endpointURL)
	data := url.Values{
		"ejectedNodes": {local},
		"knownNodes":   {otpNodes},
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

	if presp.StatusCode != 200 {
		log.Println(presp.Status)
		return errors.New("Invalid status code")
	}

	return err
}
