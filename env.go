package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-yaml/yaml"
)

const envdir string = "environments/"

var defaultEnvironment = "environment"

type Environment struct {
	// EnvironmentName i,e. dev/test/staging/production
	LoadFrom             string `yaml:"loadFrom"`
	EnvironmentName      string `yaml:"environmentName"`
	EventBridgeConfigURL string `yaml:"eventBridgeConfigURL"`
	ConfigFile           string `yaml:"configFile"`
}

func (e *Environment) get() Environment {
	envname := defaultEnvironment

	newValue := os.Getenv("PR_BACKEND_END")
	if newValue != "" {
		envname = newValue
	}

	fn := envdir + envname + ".yaml"
	_, err := e.LoadFromDisk(fn)
	if err != nil {
		return *e
	}
	e.Patch()
	return *e
}

//Patch overload defaults from environment variables
func (e *Environment) Patch() {
	var newValue string

	newValue = os.Getenv("EB_CONFIG_URL")
	if newValue != "" {
		e.EventBridgeConfigURL = newValue
	}

	newValue = os.Getenv("EB_LOAD_FROM")
	if newValue != "" {
		e.LoadFrom = newValue
	}

	newValue = os.Getenv("EB_CONFIG_FILE")
	if newValue != "" {
		e.ConfigFile = newValue
	}
}

func (e *Environment) LoadFromDisk(file string) (Environment, error) {
	ne := Environment{}

	f, err := os.Open(file)
	if err != nil {
		fmt.Println("failed to open:", file, ", error:", err)
	}
	defer f.Close()

	byteValue, err := ioutil.ReadAll(f)
	if err != nil {
		fmt.Println("read failed for ", file)
		return ne, err
	}

	err = yaml.Unmarshal([]byte(byteValue), e)
	if err != nil {
		fmt.Println("Unmarshal faild", err)
		return ne, err
	}

	return ne, nil
}
