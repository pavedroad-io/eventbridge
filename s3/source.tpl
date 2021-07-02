/*
Template data is from a SyncConfiguration
*/
{{define "source.tpl}}
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: {{.Hook.Name}}
spec:
  service:
    ports:
      - port: {{.Hook.Port}}
        targetPort: {{.Hook.Port}}
  webhook:
    eventbridge:
      port: "{{.Hook.Port}}"
      endpoint: {{.Hook.Name}}
      method: POST
{{end}}
