package main

import ("bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"io/ioutil"
	"net/http"
	"encoding/json"
)

type PromResponse struct {
	status string
	Data []string	`json:"data"`
}

const (
	sleepDuration = 15 * time.Second
//	grafanaTplDir = "/work/discomon/grafana_tpl"
	grafanaTplDir = "/grafana_tpl"
	jvmDashboard = "jvm"
	wfDashboard = "wildfly"
)

var (
	dashboards = make(map[string]bool)
	// promUrl = "http://localhost:9090"
	// grafUrl = "http://localhost:3000"
	grafUrl = fmt.Sprintf("http://%s:3000", os.Getenv("GRAFANA_SERVICE_HOST"))
	promUrl = fmt.Sprintf("http://%s:9090", os.Getenv("PROMETHEUS_SERVICE_HOST"))
)

func main() {
	fmt.Println("Grafana url: " + grafUrl)
	fmt.Println("Prom url: " + promUrl)
	initGrafana()
	fmt.Println("Starting loop")
	for {
		fetchMetrics()
		time.Sleep(sleepDuration)
	}
}

func initGrafana() {
	// TODO: wait until grafana readiness is ok
	fmt.Println("Waiting Grafana to be ready...")
	time.Sleep(30 * time.Second)
	// TODO: is ok when pod restarts?
	datasource := []byte(fmt.Sprintf(`{
		"name":"prometheus",
		"type":"prometheus",
		"url":"%s",
		"access":"proxy"
	}`, promUrl))
	resp, err := postToGrafana("/api/datasources", datasource)
	if err != nil {
		fmt.Printf("Could not initialize datasource in Grafana: %v\n", err)
		panic(err)
	}
	// TODO: check for success, else panic
	fmt.Printf("DB init response: %v\n", resp)
}

func fetchMetrics() {
	fmt.Println("Fetching metric names")
	resp, err := http.Get(promUrl + "/api/v1/label/__name__/values")
	if err != nil {
		fmt.Printf("Could not fetch metric names: %v\n", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Reading body failed: %v\n", err)
		return
	}
	jsonResp := PromResponse{}
	json.Unmarshal(body, &jsonResp)
	processMetrics(jsonResp.Data)
}

func processMetrics(metrics []string) {
	newDashboards := make(map[string]bool)
	for _, metric := range metrics {
		if (strings.HasPrefix(metric, "jvm")) {
			newDashboards[jvmDashboard] = true
		} else if (strings.HasPrefix(metric, "wildfly")) {
			newDashboards[wfDashboard] = true
		}
	}
	// TODO: remove old entries that are in 'dashboards' and not in 'newDashboards'?
	for dash := range newDashboards {
		_, alreadySet := dashboards[dash]
		if !alreadySet {
			fmt.Printf("Found new dashboard to load: %s\n", dash)
			addGrafanaDashboard(dash)
			dashboards[dash] = true
		}
	}
}

func addGrafanaDashboard(dashboard string) {
	file, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", grafanaTplDir, dashboard))
	if err != nil {
		fmt.Printf("Could not load dashboard %s from file: %v\n", dashboard, err)
		return
	}
	resp, err := postToGrafana("/api/dashboards/db", file)
	if err != nil {
		fmt.Printf("Could not send dashboard %s to Grafana: %v\n", dashboard, err)
		return
	}
	fmt.Printf("Dashboard sent response: %s\n", resp)
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
