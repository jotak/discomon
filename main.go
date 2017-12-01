package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
)

const (
	promUrl = "http://prometheus:9090"
	grafUrl = "http://grafana:3000"
	buildinDashboardsDir = "/dashboards"
	configDir = "/etc/discomon"
)

var (
	descriptors []Descriptor
	compiledPatterns = make(map[string]*regexp.Regexp)
	dashboards = make(map[string]bool)
	grafanaExternalUrl = "http://" + os.Getenv("GRAFANA_SERVICE_HOST") + ":" + os.Getenv("GRAFANA_SERVICE_PORT")
)

func main() {
	scanPeriod := 15 * time.Second
	strScanPeriod := os.Getenv("SCAN_PERIOD")
	if strScanPeriod != "" {
		i, err := strconv.Atoi(strScanPeriod)
		if err != nil {
			fmt.Println(err)
		} else {
			scanPeriod = time.Duration(i) * time.Second
		}
	}
	initPatterns()
	initGrafana()
	go initServer()
	log.Printf("Starting loop. Scan period set to %d seconds.\n", scanPeriod / time.Second)
	for {
		scanInventory()
		time.Sleep(scanPeriod)
	}
}

func initPatterns() {
	yml, err := ioutil.ReadFile(configDir + "/config.yml")
  if err != nil {
		log.Panicf("Could not read patterns file: %v\n", err)
	}
	var config Config
  err = yaml.Unmarshal(yml, &config)
  if err != nil {
		log.Panicf("Could not unmarshall config: %v\n", err)
	}
	descriptors = config.Descriptors
	for _, descriptor := range descriptors {
		compiledPatterns[descriptor.Name] = regexp.MustCompile(descriptor.Pattern)
	}
}

func initGrafana() {
	datasource := []byte(fmt.Sprintf(`{
		"name":"prometheus",
		"type":"prometheus",
		"url":"%s",
		"access":"proxy"
	}`, promUrl))
	resp, err := postToGrafana("/api/datasources", datasource)
	if err != nil {
		log.Panicf("Could not initialize datasource in Grafana: %v\n", err)
	}
	log.Printf("DB init response: %v\n", resp)
}

func addGrafanaDashboard(name string) {
	_, alreadySet := dashboards[name]
	if alreadySet {
		return
	}
	master.eventChan <- LogEvent(fmt.Sprintf("Found new dashboard to load: %s\n", name))
	file, err := loadDashboardFromFile(name)
	if err != nil {
		log.Printf("Could not load dashboard %s from file: %v\n", name, err)
		return
	}
	resp, err := postToGrafana("/api/dashboards/db", file)
	if err != nil {
		log.Printf("Could not send dashboard %s to Grafana: %v\n", name, err)
		return
	}
	log.Printf("Dashboard sent response: %s\n", resp)
	dashboards[name] = true
	master.eventChan <- DashChangedEvent()
}

func loadDashboardFromFile(dashboard string) ([]byte, error) {
	file, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", configDir, dashboard))
	if err == nil {
		return file, nil
	}
	file, err = ioutil.ReadFile(fmt.Sprintf("%s/%s.json", buildinDashboardsDir, dashboard))
	if err == nil {
		return file, nil
	}
	return nil, err
}

func postToGrafana(path string, data []byte) (string, error) {
	req, err := http.NewRequest("POST", grafUrl + path, bytes.NewBuffer(data))
	req.SetBasicAuth("admin", "admin")
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
