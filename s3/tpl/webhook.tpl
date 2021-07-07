/*
Template data is from a SyncConfiguration
*/
{{define "tpl/webhook.tpl"}}
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: {{.Name}}
spec:
  service:
    ports:
      - port: {{.Port}}
        targetPort: {{.Port}}
  webhook:
    eventbridge:
      port: "{{.Port}}"
      endpoint: {{.Name}}
      method: POST
{{end}}
