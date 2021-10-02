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
	"strings"

	"github.com/iancoleman/strcase"
)

const (
	argok8sSecret     = "secret"
	argoSourceWebhook = "webhook"
	argoSyncLambda    = "lambda"
	argoNameSpace     = "argo-events"
)

type Label struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type argoManifests struct {
	Name         string `yaml:"name"`
	TemplateFile string `yaml:"templateFile"`
	OutputFile   string `yaml:"outputFile"`
	Type         string `yaml:"type"`
}

// secret data for building a secret template
type secret struct {
	Provider    Provider `json:"provider"`
	ID          string   `json:"id"`
	Environment string   `json:"environment"`
	Labels      []Label  `json:"labels"`
}

// lambda data for building a lambda function template
type lambda struct {
	Provider      Provider      `json:"provider"`
	Hook          WebHookConfig `json:"hook"`
	LambdaTrigger LambdaTrigger `json:"lambdaTrigger"`
	Labels        []Label       `json:"labels"`
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
	Name string `yaml:"name" json:"name"`

	// Environment of this sync configuration
	Environment string `yaml:"env" json:"environment"`

	// Version of this configuration
	Version string `yaml:"version" json:"version"`

	// Hook web hook to post events to
	Hook WebHookConfig `yaml:"hook" json:"hook"`

	// TODO: Move to environment
	Kubectx string `yaml:"kubectx" json:"kubectx"`

	// Dependencies for triggers to listen for
	//  Basically the name of the webhook
	Dependencies Dependencies `yaml:"dependencies" json:"dependencies"`

	// Triggers to fire for sync event
	Triggers Triggers `yaml:"triggers" json:"triggers"`

	// ManifestDirectory to save manifests in
	ManifestDirectory string `yaml:"manifests" json:"manifests"`

	// TemplateDirctory to load templates from
	TemplateDirctory string `yaml:"templates" json:"templates"`

	// PlogConfigID
	PlogConfigID string `yaml:"plogConfigID" json:"plogConfigID"`
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

	// ConfID configuration being synced
	ConfID string

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
		l.Value = strings.ToLower(fieldValue.String())
		labels = append(labels, l)
	}
	return labels
}

// GenerateManifests
func (sc *SyncConfiguration) GenerateManifests(cf *Customer, caller SyncInitiator) {
	var argoTemplates *template.Template
	var tplFiles []string
	labels := caller.GenerateLables()

	fmt.Println("template directory: ", sc.TemplateDirctory)
	for _, t := range argoSupportedEvents {
		tplFiles = append(tplFiles, filepath.Join(sc.TemplateDirctory, t.TemplateFile))
	}
	fmt.Println("template files: ", tplFiles)

	argoTemplates, err := template.New("").Funcs(stringFunctionMap()).ParseFiles(tplFiles...)
	if err != nil {
		fmt.Println("Argo template parsing failed: ", err)
	}

	// Break customers into different directories using a short UUID + Name
	mandir := filepath.Join(sc.ManifestDirectory + "/" + cf.ShortName() + "-" + cf.Name)

	mode := int(0755)
	os.MkdirAll(mandir, os.FileMode(mode))
	var defaultMode os.FileMode = 0660

	// Generate sources
	//  Note there is only webhooks right now
	//  The future holds more
	man, err := GetArgoManifest(argoSupportedEvents, argoSourceWebhook)
	if err != nil {
		fmt.Errorf("Failed to find webhook source template\n")
	}

	fn := filepath.Join(mandir, man.OutputFile)
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
		fn := filepath.Join(mandir, p.Name+man.OutputFile)
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
		err = argoTemplates.Funcs(stringFunctionMap()).ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &data)
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
		fn := filepath.Join(mandir, l.Name+man.OutputFile)
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

		err = argoTemplates.Funcs(stringFunctionMap()).ExecuteTemplate(bw, filepath.Join(sc.TemplateDirctory, man.TemplateFile), &data)
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

	kubecmd = append(kubecmd, "--namespace")
	kubecmd = append(kubecmd, argoNameSpace)
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
	conf := strings.Split(caller.ConfID, "-")[0]
	var name = "default"
	if cf.Name != "" {
		name = cf.Name
	}
	// TODO: fix this
	location, _ := filepath.Abs(filepath.Join("./manifest/" + conf + "-" + name))

	_, err = os.Stat(location)
	if os.IsNotExist(err) {
		return []byte("No deployment to delete: " + location), nil
	}

	if sc.Kubectx != "" {
		kubecmd = append(kubecmd, "--context")
		kubecmd = append(kubecmd, sc.Kubectx)
	}
	kubecmd = append(kubecmd, "--namespace")
	kubecmd = append(kubecmd, argoNameSpace)

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
	kubecmd = append(kubecmd, "--namespace")
	kubecmd = append(kubecmd, argoNameSpace)

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

// Template function maps
//
func stringFunctionMap() template.FuncMap {
	stringFuncMap := template.FuncMap{
		"ToUpper":    strings.ToUpper,
		"ToLower":    strings.ToLower,
		"ToCamel":    strcase.ToCamel,
		"ToSnake":    strcase.ToSnake,
		"Base64":     Encode64,
		"RFC1123":    RFC1123,
		"ValidValue": ValidK8sValue,
	}
	return stringFuncMap
}

// ValidK8sValue converts strings to rfc1123
// and add quotes if it an integer
func ValidK8sValue(inputStr string) string {
	return RFC1123(inputStr)
}

// RFC1123 & 1035 is sanity check for label names used in
// docker and k8s
// - contain at most 63 characters
// - contain only lowercase alphanumeric characters or '-'
// - start with an alphanumeric character
// - end with an alphanumeric character
func RFC1123(inputStr string) string {
	// Map to lowercase
	tmpStr := strings.ToLower(inputStr)

	// only allow rfc compliant characters to lowercase
	dash := rune('-')
	var na rune
	rfc := func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return r
		case r >= 'a' && r <= 'z':
			return r
		case r == dash:
			return r
		}
		return na
	}
	tmpStr = strings.Map(rfc, tmpStr)
	if len(tmpStr) > 63 {
		fmt.Println("first 63 chars:", tmpStr)
		return tmpStr[0:62]
	} else {
		return tmpStr
	}
}

// Encode64 for encoding secrets in manifests
func Encode64(inputStr string) string {
	return base64.StdEncoding.EncodeToString([]byte(inputStr))
}
