package main

type Config struct {
	Descriptors []Descriptor `yaml:"descriptors"`
}

type Descriptor struct {
	Pattern string `yaml:"pattern"`
	Name string `yaml:"name"`
	Category string `yaml:"category"`
}
