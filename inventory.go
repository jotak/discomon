package main

import (
	"bytes"
	"fmt"
	"hash"
	"io"
	"log"
	"strings"

	"crypto/md5"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

type Resource struct {
	Name string `json:"name"`
	Children []*Resource `json:"children"`
	Url string `json:"url"`
	Status Status `json:"status"`
}

type Status int
const (
	// Status (order matters, sorted by decreasing priority)
	DOWN Status = 0
	UNKNOWN Status = 1
	UP Status = 2
	EXPIRED Status = 3
	UNSET Status = 4
)

var (
	inventory = Resource{"_root_", []*Resource{}, "", UNSET}
	invHash = computeHash(&inventory)
)

func scanInventory() {
	// Fetch metric names
	metrics := fetchLabelValues("__name__")
	if metrics == nil {
		return
	}
	instances := make(map[string]*Resource)
	apps := make(map[string]*Resource)
	inventory.Children = []*Resource{}
	for _, desc := range descriptors {
		if metric := getMatchingMetric(metrics, desc.Name); metric != "" {
			metricDefs := fetchMetricDef(metric)
			for _, _labels := range metricDefs {
				labels := _labels.(map[string]interface{})
				instanceName := labels["instance"].(string)
				podName := labels["kubernetes_pod_name"].(string)
				status := getInstanceStatus(instanceName, podName)
				if status == EXPIRED {
					continue;
				}
				instanceIP := strings.Split(instanceName, ":")[0]
				instance, instanceExists := instances[instanceIP]
				dashUrl := fmt.Sprintf("%s/dashboard/db/%s?var-instance=%s&var-app=%s",
					grafanaExternalUrl,
					desc.Name,
					instanceName,
					labels["app"]);
				resource := Resource{desc.Name, []*Resource{}, dashUrl, status}
				if instanceExists {
					instance.Children = append(instance.Children, &resource)
				} else {
					instance = &Resource{instanceIP, []*Resource{&resource}, "", UNSET}
					instances[instanceIP] = instance
					appName := labels["app"].(string)
					app, appExists := apps[appName]
					if appExists {
						app.Children = append(app.Children, instance)
					} else {
						app = &Resource{appName, []*Resource{instance}, "", UNSET}
						apps[appName] = app
						inventory.Children = append(inventory.Children, app)
					}
				}
			}
		}
	}
	propagateStatus(&inventory)
	newHash := computeHash(&inventory)
	if !bytes.Equal(invHash, newHash) {
		invch()
		invHash = newHash
	}
	scanch()
}

func getMatchingMetric(metrics []interface{}, name string) string {
	for _, metric_ := range metrics {
		metric := metric_.(string)
		if compiledPatterns[name].MatchString(metric) {
			addGrafanaDashboard(name)
			return metric
		}
	}
	return ""
}

func fetchLabelValues(label string) []interface{} {
	json_ := promGenericQuery("/api/v1/label/" + label + "/values")
	if json_ == nil {
		return nil
	}
	return json_.([]interface{})
}

func fetchMetricDef(metric string) []interface{} {
	json_ := promGenericQuery("/api/v1/series?match[]=" + metric)
	if json_ == nil {
		return nil
	}
	return json_.([]interface{})
}

func getInstanceStatus(instance, pod string) Status {
	json_ := promGenericQuery("/api/v1/query?query=up{instance=\"" + instance + "\",kubernetes_pod_name=\"" + pod + "\"}")
	if json_ == nil {
		return UNKNOWN
	}
	result := json_.(map[string]interface{})["result"].([]interface{})
	if len(result) == 0 {
		// This instance doesn't exist (anymore?)
		return EXPIRED
	}
	up := result[0].(map[string]interface{})["value"].([]interface{})[1].(string)
	if up == "1" {
		return UP
	}
	return DOWN
}

func promGenericQuery(relativePath string) interface{} {
	resp, err := http.Get(promUrl + relativePath)
	if err != nil {
		log.Printf("Could not fetch Prometheus: %v\n", err)
		logch("Could not fetch Prometheus (check logs)")
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading body failed: %v\n", err)
		logch("Reading body failed (check logs)")
		return nil
	}
	var f interface{}
	err = json.Unmarshal(body, &f)
	if err != nil {
		log.Printf("Unmarshal json failed: %v\n", err)
		logch("Unmarshal json failed (check logs)")
		return nil
	}
	status := f.(map[string]interface{})["status"]
	if status != "success" {
		log.Printf("Prometheus call didn't succeed: %v", f)
		logch("Prometheus call didn't succeed (check logs)")
		return nil
	}
	return f.(map[string]interface{})["data"]
}

func propagateStatus(r *Resource) Status {
	for _, child := range r.Children {
		childStatus := propagateStatus(child)
		if childStatus < r.Status {
			r.Status = childStatus
		}
	}
	return r.Status
}

func computeHash(r *Resource) []byte {
	h := md5.New()
	appendHash(r, h, 0)
	return h.Sum(nil)
}

func appendHash(parent *Resource, h hash.Hash, depth int) {
	for i, r := range parent.Children {
		io.WriteString(h, fmt.Sprintf("@%d@%d@%s@%d", depth, i, r.Name, r.Status))
		appendHash(r, h, depth+1)
	}
}
