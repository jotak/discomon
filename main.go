package main

import ("bytes"
	"fmt"
	"regexp"
	"time"

	"gopkg.in/yaml.v2"
	"io/ioutil"
	"net/http"
	"encoding/json"
)

type PromResponse struct {
	status string
	Data []string	`json:"data"`
}

type RawPatterns struct {
	Patterns map[string][]string `yaml:"patterns"`
}

const (
	sleepDuration = 15 * time.Second
	promUrl = "http://prometheus:9090"
	grafUrl = "http://grafana:3000"
	dashboardsDir = "/dashboards"
	// promUrl = "http://localhost:9090"
	// grafUrl = "http://localhost:3000"
	// dashboardsDir = "/work/discomon/dashboards"
)

var (
	dashboards = make(map[string]bool)
	patterns = make(map[string][]*regexp.Regexp)
)

func main() {
	initPatterns()
	initGrafana()
	fmt.Println("Starting loop")
	for {
		fetchMetrics()
		time.Sleep(sleepDuration)
	}
}

func initPatterns() {
	yml, err := ioutil.ReadFile(dashboardsDir + "/patterns.yml")
  if err != nil {
		fmt.Printf("Could not read patterns file: %v\n", err)
    panic(err)
	}
	var rawPatterns RawPatterns
  err = yaml.Unmarshal(yml, &rawPatterns)
  if err != nil {
		fmt.Printf("Could not unmarshall patterns: %v\n", err)
    panic(err)
  }
	fmt.Printf("patterns: %v\n", rawPatterns)

	for dash, regs := range rawPatterns.Patterns {
		compiled := make([]*regexp.Regexp, len(regs))
		for i, r := range regs {
			compiled[i] = regexp.MustCompile(r)
		}
		patterns[dash] = compiled
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
	findPatterns(jsonResp.Data)
}

func findPatterns(metrics []string) {
	for _, metric := range metrics {
		for dash, regs := range patterns {
			_, alreadySet := dashboards[dash]
			if !alreadySet {
				for _, reg := range regs {
					if reg.MatchString(metric) {
						fmt.Printf("Found new dashboard to load: %s\n", dash)
						addGrafanaDashboard(dash)
						dashboards[dash] = true
						break
					}
				}
			}
		}
	}
}

func addGrafanaDashboard(dashboard string) {
	file, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", dashboardsDir, dashboard))
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
