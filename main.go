package main

import "fmt"
import "time"

const (
	promUrl = "http://localhost:9090"
	grafUrl = "http://localhost:3000"
	k8sUrl = ""
)

func main() {
	fmt.Println("Starting loop")
	for {
		fetchServices()
		time.Sleep(15 * time.Second)
	}
}

func fetchServices() {
	fmt.Println("Fetching services")
	// TODO: fetch k8sUrl/api/v1/namespaces/test/pods with header "Authorization: Bearer `cat /var/run/secrets/kubernetes.io/serviceaccount/token`"
	const json = `{"items": [{"spec": {"containers": [{"ports": [{"containerPort": 8080}]}]}, "status": {"podIP": "172.17.0.5"}}]}`
}
