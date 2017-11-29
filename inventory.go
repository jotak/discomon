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
}

var (
	inventory = Resource{"Inventory", []*Resource{}, ""}
	invHash = computeHash(&inventory)
)

func scanInventory() {
	// Fetch metric names
	metrics := fetchLabelValues("__name__")
	if metrics == nil {
		return
	}
	findPatterns(metrics)
}

func findPatterns(metrics []string) {
	instances := make(map[string]*Resource)
	apps := make(map[string]*Resource)
	inventory.Children = []*Resource{}
	for _, desc := range descriptors {
		if metric := getMatchingMetric(metrics, desc.Name); metric != "" {
			metricDefs := fetchMetricDef(metric)
			for _, labels := range metricDefs {
				log.Printf("Found [app=%s, instance=%s] for descriptor %s\n", labels["app"], labels["instance"], desc.Name)
				instanceName := strings.Split(labels["instance"], ":")[0]
				instance, exists := instances[instanceName]
				resource := Resource{desc.Name, []*Resource{}, grafanaExternalUrl + "/dashboard/db/" + desc.Name}
				if exists {
					log.Printf("BEFORE: %v", instance)
					instance.Children = append(instance.Children, &resource)
					log.Printf("AFTER: %v", instance)
					log.Printf("AFTER/from map: %v", instances[instanceName])
					} else {
					instance = &Resource{instanceName, []*Resource{&resource}, ""}
					instances[instanceName] = instance
					appName := labels["app"]
					app, appExists := apps[appName]
					if appExists {
						app.Children = append(app.Children, instance)
					} else {
						app = &Resource{appName, []*Resource{instance}, ""}
						apps[appName] = app
						inventory.Children = append(inventory.Children, app)
					}
				}
			}
		}
	}
	newHash := computeHash(&inventory)
	if !bytes.Equal(invHash, newHash) {
		log.Println("Different hash")
		invch()
		invHash = newHash
	} else {
		log.Println("Same hash")
	}
}

func getMatchingMetric(metrics []string, name string) string {
	for _, metric := range metrics {
		if compiledPatterns[name].MatchString(metric) {
			log.Printf("Matched [name=%s, metric=%s]\n", name, metric)
			addGrafanaDashboard(name)
			return metric
		}
	}
	return ""
}

func fetchLabelValues(label string) []string {
	resp, err := http.Get(promUrl + "/api/v1/label/" + label + "/values")
	if err != nil {
		log.Printf("Could not fetch labels: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading body failed: %v\n", err)
		return nil
	}
	jsonResp := PromLabelResponse{}
	json.Unmarshal(body, &jsonResp)
	return jsonResp.Data
}

func fetchMetricDef(metric string) []map[string]string {
	resp, err := http.Get(promUrl + "/api/v1/series?match[]=" + metric)
	if err != nil {
		log.Printf("Could not fetch metric: %v\n", err)
		return nil
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Reading body failed: %v\n", err)
		return nil
	}
	jsonResp := PromMetricResponse{}
	json.Unmarshal(body, &jsonResp)
	return jsonResp.Data
}

func computeHash(r *Resource) []byte {
	h := md5.New()
	appendHash(r, h, 0)
	return h.Sum(nil)
}

func appendHash(parent *Resource, h hash.Hash, depth int) {
	for i, r := range parent.Children {
		io.WriteString(h, fmt.Sprintf("@%d@%d@%s", depth, i, r.Name))
		log.Printf("@%d@%d@%s", depth, i, r.Name)
		appendHash(r, h, depth+1)
	}
}
