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

const (
	sleepDuration = 15 * time.Second
	promFilesDSDir = "/tmp/discomon"
	grafanaTplDir = "/grafana_tpl"
	jvmDashboard = "jvm"
	wfDashboard = "wildfly"
)

var (
	fetchedUrls = make(map[string][]string)
	dashboards = make(map[string]bool)
	// k8sUrl = ""
	// promUrl = "http://localhost:9090"
	// grafUrl = "http://localhost:3000"
	k8sSecret string
	k8sNamespace string
	k8sUrl = fmt.Sprintf("https://%s:443", os.Getenv("KUBERNETES_SERVICE_HOST"))
	grafUrl = fmt.Sprintf("http://%s:3000", os.Getenv("GRAFANA_SERVICE_HOST"))
	promUrl = fmt.Sprintf("http://%s:9090", os.Getenv("PROMETHEUS_SERVICE_HOST"))
)

func main() {
	fmt.Println("Kube url: " + k8sUrl)
	fmt.Println("Grafana url: " + grafUrl)
	fmt.Println("Prom url: " + promUrl)
	fmt.Println("Starting loop")
	initToken()
	k8sNamespace = findNamespace()
	initGrafana()
	for {
		fetchPods()
		time.Sleep(sleepDuration)
	}
}

func initToken() {
	file, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		fmt.Printf("Could not load secret token: %v", err)
		panic(err)
	}
	k8sSecret = string(file)
}

func findNamespace() string {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/pods", k8sUrl), nil)
	req.Header.Set("Authorization", "Bearer " + k8sSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Could not fetch all pods: %v", err)
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Could not read all pods: %v", err)
		panic(err)
	}
	jsonPods := PodsResponse{}
	json.Unmarshal(body, &jsonPods)

	hostname := os.Getenv("HOSTNAME")
	for _, pod := range jsonPods.Items {
		if pod.Metadata.Name == hostname {
			return pod.Metadata.Namespace
		}
	}
	panic("Could not find namespace")
}

func initGrafana() {
	// TODO: wait until grafana readiness is ok
	time.Sleep(30 * time.Second)
	// TODO: is ok when pod restarts?
	datasource := []byte(fmt.Sprintf(`{
		"name":"prometheus",
		"type":"prometheus",
		"url":"%s",
		"access":"proxy"
	}`, promUrl))
	resp, err := http.Post(grafUrl + "/api/datasources", "application/json", bytes.NewBuffer(datasource))
	if err != nil {
		fmt.Printf("Could not initialize datasource in Grafana: %v", err)
		return
	}
	defer resp.Body.Close()
}

func fetchPods() {
	fmt.Println("Fetching pods")
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/v1/namespaces/%s/pods", k8sUrl, k8sNamespace), nil)
	req.Header.Set("Authorization", "Bearer " + k8sSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Could not fetch pods: %v", err)
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Reading body failed: %v", err)
		return
	}
	jsonPods := PodsResponse{}
	json.Unmarshal(body, &jsonPods)
	processPods(jsonPods)
}

func processPods(jsonPods PodsResponse) {
	for _, pod := range jsonPods.Items {
		ip := pod.Status.PodIP
		for _, ctnr := range pod.Spec.Containers {
			for _, port := range ctnr.Ports {
				// TODO: check the readiness probe before scanning
				url := fmt.Sprintf("http://%s:%d/metrics", ip, port.ContainerPort)
				_, inCache := fetchedUrls[url]
				if !inCache {
					podDashboards := scan(url)
					toCache(url, podDashboards)
					if len(podDashboards) > 0 {
						addPromConfig(ip, port.ContainerPort)
						for dash := range podDashboards {
							_, alreadySet := dashboards[dash]
							if !alreadySet {
								addGrafanaDashboard(dash)
								dashboards[dash] = true
							}
						}
					}
				} else {
					fmt.Printf("Skipping %s, already in cache\n", url)
				}
			}
		}
	}
	// TODO: remove old pods from cache / prom config
}

func scan(url string) map[string]bool {
	// TODO: Use protocol buffer format?
	dashboards := make(map[string]bool)
	fmt.Printf("Scanning %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Scan failed: %v", err)
		return dashboards
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Reading body failed: %v", err)
		return dashboards
	}
	strBody := string(body)
	for _, line := range strings.Split(strBody, "\n") {
		if (strings.HasPrefix(line, "jvm")) {
			dashboards[jvmDashboard] = true
		} else if (strings.HasPrefix(line, "wildfly")) {
			dashboards[wfDashboard] = true
		}
	}
	return dashboards;
}

func toCache(url string, dashboards map[string]bool) {
	keys := make([]string, 0, len(dashboards))
	for k := range dashboards {
		keys = append(keys, k)
	}
	fetchedUrls[url] = keys
}

func addPromConfig(host string, port int) {
	filename := fmt.Sprintf("prom_%s_%d.json", host, port)
	fmt.Printf("Adding Prometheus config: %s\n", filename)
	fullpath := fmt.Sprintf("%s/%s", promFilesDSDir, filename)
	err := ioutil.WriteFile(fullpath, []byte(fmt.Sprintf(`
	[
		{
			"targets": [ "%s:%d" ]
		}
	]
	`, host, port)), 0644)
	if err != nil {
		fmt.Printf("Writing Prometheus config failed: %v", err)
	}
}

func addGrafanaDashboard(dashboard string) {
	fmt.Printf("Adding Grafana dashboard: %s\n", dashboard)
	file, err := ioutil.ReadFile(fmt.Sprintf("%s/%s.json", grafanaTplDir, dashboard))
	if err != nil {
		fmt.Printf("Could not load dashboard %s from file: %v", dashboard, err)
		return
	}
	resp, err := http.Post(grafUrl + "/api/dashboards/db", "application/json", bytes.NewBuffer(file))
	if err != nil {
		fmt.Printf("Could not load Grafana dashboard %s: %v", dashboard, err)
		return
	}
	defer resp.Body.Close()
}
