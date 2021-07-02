package s3

type SyncConfiguration struct {
	// Name of this sync configuration
	Name string `yaml:"name"`

	// Hook web hook to post events to
	Hook WebHookConfig `yaml:"hook"`

	// TODO: Move to environment
	Kubectx string `yaml:"kubectx"`

	// Dependencies for triggers to listen for
	//  Basically the name of the webhook
	Dependencies Dependencies `yaml:"dependencies"`
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
	Provider Provider `yaml:"provider"`

	// Region to execute function in
	//   TODO: Pull from provider if available
	Region string `yaml:"region"`
}

type Dependencies struct {
	Name            string `yaml:"name"`
	EventSourceName string `yaml:"eventSourceName"`
	EventName       string `yaml:"eventName"`
}
