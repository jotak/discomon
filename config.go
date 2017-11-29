package main

type Config struct {
	Descriptors []Descriptor `yaml:"descriptors"`
}

type Descriptor struct {
	Pattern string `yaml:"pattern"`
	Name string `yaml:"name"`
	Category string `yaml:"category"`
	Child Child `yaml:"child"`
}

type Child struct {
	Label string `yaml:"label"`
	Name string `yaml:"name"`
	FoundIn string `yaml:"found_in"`
}

