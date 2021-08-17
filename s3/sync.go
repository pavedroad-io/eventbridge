package s3

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
)

const (
	argok8sSecret     = "secret"
	argoSourceWebhook = "webhook"
	argoSyncLambda    = "lambda"
)

type Label struct {
	Key   string
	Value string
}

type argoManifests struct {
	Name         string `yaml:"name"`
	TemplateFile string `yaml:"templateFile"`
	OutputFile   string `yaml:"outputFile"`
	Type         string `yaml:"type"`
}

// secret data for building a secret template
type secret struct {
	Provider    Provider
	ID          string
	Environment string
	Labels      []Label
}

// lambda data for building a lambda function template
type lambda struct {
	Provider      Provider
	Hook          WebHookConfig
	LambdaTrigger LambdaTrigger
	Labels        []Label
}

type argoKinds struct {
	Kind string
}

var argoSupportedEvents []argoManifests = []argoManifests{
	{Name: "kuberntes secret",
		TemplateFile: "secret.tpl",
		OutputFile:   "secret.yaml",
		Type:         argok8sSecret},
	{Name: "Webhook source",
		TemplateFile: "webhook.tpl",
		OutputFile:   "webhook.yaml",
		Type:         argoSourceWebhook},
	{Name: "Lambda trigger",
		TemplateFile: "lambda.tpl",
		OutputFile:   "lambda.yaml",
		Type:         argoSyncLambda},
}

// Argo kinds to  manage
var argoSupportedKinds []argoKinds = []argoKinds{
	{Kind: "Secret"},
	{Kind: "EventSource"},
	{Kind: "Sensor"},
}

// GetManifest returns an argoManifest for a given type
func GetArgoManifest(ml []argoManifests, mtype string) (man argoManifests, err error) {

	for _, v := range argoSupportedEvents {
		if v.Type == mtype {
			return v, nil
		}
	}

	return man, fmt.Errorf("Manifest for %v not found\n", mtype)
}

type SyncConfiguration struct {
	// Name of this sync configuration
	Name string `yaml:"name"`

	// Environment of this sync configuration
	Environment string `yaml:"env"`

	// Version of this configuration
	Version string `yaml:"version"`

	// Hook web hook to post events to
	Hook WebHookConfig `yaml:"hook"`

	// TODO: Move to environment
	Kubectx string `yaml:"kubectx"`

	// Dependencies for triggers to listen for
	//  Basically the name of the webhook
	Dependencies Dependencies `yaml:"dependencies"`

	// Triggers to fire for sync event
	Triggers Triggers `yaml:"triggers"`

	// ManifestDirectory to save manifests in
	ManifestDirectory string `yaml:"manifests"`

	// TemplateDirctory to load templates from
	TemplateDirctory string `yaml:"templates"`
}

// SyncInitiator
//  Is used to label generated structures and authenticate
//  synchronization requests if required
type SyncInitiator struct {
	// Mandatory
	// CustomerID
	CustomerID string

	// UserID requesting sync
	UserID string

	// Optional
	// AuthorizationToken
	AuthorizationToken string

	// ReferenceID
	ReferenceID string
}

func (si *SyncInitiator) GenerateLables() []Label {
	var labels []Label

	v := reflect.ValueOf(*si)

	for i := 0; i < v.NumField(); i++ {
		l := Label{}
		fieldValue := v.Field(i)
		fieldType := v.Type().Field(i)
		fieldName := fieldType.Name
		//		&#34;
		l.Key = string(fieldName)
		l.Value = fieldValue.String()
		labels = append(labels, l)
	}
	return labels
}

// GenerateManifests
func (sc *SyncConfiguration) GenerateManifests(cf *Customer, caller SyncInitiator) {
	var argoTemplates *template.Template
	var tplFiles []string
	labels := caller.GenerateLables()

	for _, t := range argoSupportedEvents {
		tplFiles = append(tplFiles, filepath.Join(sc.TemplateDirctory, t.TemplateFile))
	}

	argoTemplates, err := template.New("").ParseFiles(tplFiles...)
	if err != nil {
		fmt.Println("Argo template parsing failed: ", err)
	}

	// Break customers into different directories using a short UUID + Name
	sc.ManifestDirectory = filepath.Join(sc.ManifestDirectory + "/" + cf.ShortName() + "-" + cf.Name)

	mode := int(0755)
	os.MkdirAll(sc.ManifestDirectory, os.FileMode(mode))
	var defaultMode os.FileMode = 0660

	// Generate sources
	//  Note there is only webhooks right now
	//  The future holds more
	man, err := GetArgoManifest(argoSupportedEvents, argoSourceWebhook)
	if err != nil {
		fmt.Errorf("Failed to find webhook source template\n")
	}

	fn := filepath.Join(sc.ManifestDirectory, man.OutputFile)
	file, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultMode)
	if err != nil {
		log.Fatal(err, fn)
	}

	bw := bufio.NewWriter(file)
	tplData := struct {
		HookData  WebHookConfig
		LabelData []Label
	}{
		HookData:  sc.Hook,
		LabelData: labels,
	}
	// err = argoTemplates.ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &sc.Hook)
	err = argoTemplates.ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &tplData)
	if err != nil {
		fmt.Errorf("Failed to create webhook manifest %v\n", sc.Hook)
	}

	if err := bw.Flush(); err != nil {
		fmt.Errorf("Flush for file %s failed with error %v\n",
			file.Name(), err)
	}
	if err := file.Close(); err != nil {
		fmt.Errorf("Close for file %s failed with error %v\n",
			file.Name(), err)
	}

	// Generate a secrete for for each  provider configured
	//  then it doesn't matter which one a given trigger uses
	man, err = GetArgoManifest(argoSupportedEvents, argok8sSecret)
	if err != nil {
		fmt.Errorf("Failed to read template %v\n", err)
	}
	for _, p := range cf.Providers {
		fn := filepath.Join(sc.ManifestDirectory, p.Name+man.OutputFile)
		file, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultMode)
		if err != nil {
			log.Fatal(err, fn)
		}
		p.Credentials = base64.StdEncoding.EncodeToString([]byte(p.Credentials))
		p.Key = base64.StdEncoding.EncodeToString([]byte(p.Key))
		bw := bufio.NewWriter(file)
		data := secret{
			Provider:    p,
			ID:          cf.ID.String(),
			Environment: cf.Configuration.Environment,
			Labels:      labels,
		}
		err = argoTemplates.ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &data)
		if err != nil {
			fmt.Errorf("Failed to create webhook manifest %v\n", sc.Hook)
		}

		if err := bw.Flush(); err != nil {
			fmt.Errorf("Flush for file %s failed with error %v\n",
				file.Name(), err)
		}
		if err := file.Close(); err != nil {
			fmt.Errorf("Close for file %s failed with error %v\n",
				file.Name(), err)
		}

	}

	// Generate triggers
	man, err = GetArgoManifest(argoSupportedEvents, argoSyncLambda)
	if err != nil {
		fmt.Errorf("Failed to read template %v\n", err)
	}
	for _, l := range sc.Triggers.Lambda {
		fn := filepath.Join(sc.ManifestDirectory, l.Name+man.OutputFile)
		file, err := os.OpenFile(fn, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultMode)
		if err != nil {
			log.Fatal(err, fn)
		}
		bw := bufio.NewWriter(file)
		p, err := cf.Providers.Lookup(l.Provider)

		if err != nil {
			fmt.Errorf("Provider looked failed for %v with %v\n", l.Provider, err)
		}
		data := lambda{
			Provider:      p,
			Hook:          cf.Configuration.Hook,
			LambdaTrigger: l,
			Labels:        labels,
		}

		err = argoTemplates.ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &data)
		if err != nil {
			fmt.Errorf("Failed to create lambda manifest %v\n", sc.Hook)
		}

		if err := bw.Flush(); err != nil {
			fmt.Errorf("Flush for file %s failed with error %v\n",
				file.Name(), err)
		}
		if err := file.Close(); err != nil {
			fmt.Errorf("Close for file %s failed with error %v\n",
				file.Name(), err)
		}
	}
}

// DeployManifests to using the Kubectx provided
func (sc *SyncConfiguration) DeployManifests(cf *Customer, caller SyncInitiator) (data []byte, err error) {
	var kubecmd = []string{}
	location, _ := filepath.Abs(filepath.Join(sc.ManifestDirectory + "/" + cf.ShortName() + "-" + cf.Name))

	if sc.Kubectx != "" {
		kubecmd = append(kubecmd, "--context")
		kubecmd = append(kubecmd, sc.Kubectx)
	}
	kubecmd = append(kubecmd, "apply")
	kubecmd = append(kubecmd, "-f")
	kubecmd = append(kubecmd, location)
	kubecmd = append(kubecmd, "-o")
	kubecmd = append(kubecmd, "json")

	data, err = sc.KubeExec(kubecmd...)

	if err != nil {
		return nil, err
	}
	return data, err
}

// DeleteDeployment to using the Kubectx provided
func (sc *SyncConfiguration) DeleteDeployment(cf *Customer, caller SyncInitiator) (data []byte, err error) {
	var kubecmd = []string{}
	location, _ := filepath.Abs(filepath.Join(sc.ManifestDirectory + "/" + cf.ShortName() + "-" + cf.Name))

	if sc.Kubectx != "" {
		kubecmd = append(kubecmd, "--context")
		kubecmd = append(kubecmd, sc.Kubectx)
	}
	kubecmd = append(kubecmd, "delete")
	kubecmd = append(kubecmd, "-f")
	kubecmd = append(kubecmd, location)
	kubecmd = append(kubecmd, "-o")
	kubecmd = append(kubecmd, "name")

	data, err = sc.KubeExec(kubecmd...)

	if err != nil {
		return nil, err
	}

	return data, err
}

func (sc *SyncConfiguration) KubeExec(options ...string) (data []byte, err error) {
	data, err = exec.Command("kubectl", options...).Output()
	if err != nil {
		fmt.Println("Error executing kubectl: ", err)
		return nil, err
	}
	return data, nil
}

// GetDeployments to using the Kubectx provided
func (sc *SyncConfiguration) GetDeployments(cf *Customer, caller SyncInitiator) (data []byte, err error) {
	var kubecmd = []string{}

	if sc.Kubectx != "" {
		kubecmd = append(kubecmd, "--context")
		kubecmd = append(kubecmd, sc.Kubectx)
	}
	kubecmd = append(kubecmd, "get")
	l := len(argoSupportedKinds)
	var resourceKinds string
	for i, v := range argoSupportedKinds {
		resourceKinds += v.Kind
		if i != l-1 {
			resourceKinds += ","
		}
	}

	kubecmd = append(kubecmd, resourceKinds)
	kubecmd = append(kubecmd, "-l")
	kubecmd = append(kubecmd, "CustomerID="+caller.CustomerID)
	kubecmd = append(kubecmd, "-o")
	kubecmd = append(kubecmd, "json")

	data, err = sc.KubeExec(kubecmd...)

	if err != nil {
		return nil, err
	}

	return data, err
}

type WebHookConfig struct {
	Name string `yaml:"name"`
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Triggers struct {
	Lambda []LambdaTrigger `yaml:"lambda"`
	//	Azure []
	//	Log []
	//	Kafka []
}

type LambdaTrigger struct {
	// Name for this trigger
	Name string `yaml:"name"`

	// FunctionName name of lambda function to call
	FunctionName string `yaml:"functionName"`

	// Provider credentials
	Provider string `yaml:"provider"`

	// Region to execute function in
	//   TODO: Pull from provider if available
	Region string `yaml:"region"`
}

type Dependencies struct {
	Name            string `yaml:"name"`
	EventSourceName string `yaml:"eventSourceName"`
	EventName       string `yaml:"eventName"`
}
