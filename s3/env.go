<<<<<<< HEAD
package s3
=======
package main
>>>>>>> 09db737181dbe3c78da8cef9158c13bca5f1e318

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-yaml/yaml"
)

const envdir string = "environments/"

var defaultEnvironment = "dev"

type Environment struct {
	// EnvironmentName i,e. dev/test/staging/production
	EnvironmentName      string `yaml:"environmentName"`
	EventBridgeConfigURL string `yaml:"eventBridgeConfigURL"`
}

func (e *Environment) get(envname, version string) Environment {

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
