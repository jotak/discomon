package main

// {"items": [{"spec": {"containers": [{"ports": [{"containerPort": 8080}]}]}, "status": {"podIP": "172.17.0.5"}}]}

type Status struct {
	PodIP string `json:"podIP"`
}

type ContainerPort struct {
	ContainerPort int `json:"containerPort"`
}

type Container struct {
	Ports []ContainerPort `json:"ports"`
}

type Spec struct {
	Containers []Container `json:"containers"`
}

type Metadata struct {
	Name string `json:"name"`
	Namespace string `json:"namespace"`
}

type Item struct {
	Metadata Metadata `json:"metadata"`
	Spec Spec `json:"spec"`
	Status Status `json:"status"`
}

type PodsResponse struct {
	Items []Item `json:"items"`
}
