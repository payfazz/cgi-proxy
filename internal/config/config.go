package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Entry .
type Entry struct {
	Path          string   `yaml:"path"`
	Cmd           []string `yaml:"cmd"`
	AllowParallel bool     `yaml:"allow_parallel"`
	AllowSubPath  bool     `yaml:"allow_sub_path"`
}

// Config .
type Config struct {
	AuthKeys []string `yaml:"static_key"`
	Entry    []Entry  `yaml:"entry"`
}

// Parse .
func Parse(file string) (*Config, error) {
	confBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	var conf Config

	if err := yaml.Unmarshal(confBytes, &conf); err != nil {
		return nil, err
	}

	return &conf, nil
}
