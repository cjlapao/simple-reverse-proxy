package config

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"regexp"
)

var globalConfig *Config

type Config struct {
	ProxyPort string `json:"port"`
	Hosts     []Host `json:"hosts"`
}

func NewConfig() (*Config, error) {
	if globalConfig == nil {
		globalConfig = &Config{}
		// Define the command line flag
		path := flag.String("path", "config.json", "path to the configuration file")
		// Parse the command line arguments
		flag.Parse()
		var configPath string
		if *path == "" {
			configPath = "config.json"
		} else {
			configPath = *path
		}

		log.Println("Reading config from", configPath)

		err := globalConfig.ReadConfig(configPath)
		if err != nil {
			return nil, err
		}
	}

	return globalConfig, nil
}

type Host struct {
	Host   string     `json:"host"`
	Port   string     `json:"port"`
	Routes []Route    `json:"routes,omitempty"`
	Tcp    *TcpTarget `json:"tcp,omitempty"`
}

type TcpTarget struct {
	Target string `json:"target"`
}

type Route struct {
	Path       string         `json:"path"`
	Target     string         `json:"target"`
	TargetPort string         `json:"target_port"`
	Pattern    *regexp.Regexp `json:"-"`
}

func (c *Config) ReadConfig(filename string) error {
	// Open the configuration file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode the JSON configuration
	var config Config
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return err
	}

	c.Hosts = config.Hosts
	c.ProxyPort = config.ProxyPort
	for i, host := range c.Hosts {
		for j, route := range host.Routes {
			c.Hosts[i].Routes[j].Pattern = regexp.MustCompile(route.Path)
		}
	}

	return nil
}
