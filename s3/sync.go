package s3

import (
	"bufio"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
)

const (
	argok8sSecret     = "secret"
	argoSourceWebhook = "webhook"
	argoSyncLambda    = "lambda"
)

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
}

// lambda data for building a lambda function template
type lambda struct {
	Provider      Provider
	Hook          WebHookConfig
	LambdaTrigger LambdaTrigger
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

// GenerateManifests
func (sc *SyncConfiguration) GenerateManifests(cf *Customer) {
	var argoTemplates *template.Template
	var tplFiles []string
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
	err = argoTemplates.ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &sc.Hook)
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
		bw := bufio.NewWriter(file)
		data := secret{
			Provider:    p,
			ID:          cf.ID.String(),
			Environment: cf.Configuration.Environment,
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
