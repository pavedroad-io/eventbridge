/*
Template data is from a SyncConfiguration
*/
{{define "tpl/webhook.tpl"}}
apiVersion: argoproj.io/v1alpha1
kind: EventSource
metadata:
  name: {{.HookData.Name}}
  label:
  {{- range .LabelData}}
    - {{.}}
  {{- end}}
spec:
  service:
    ports:
      - port: {{.HookData.Port}}
        targetPort: {{.HookData.Port}}
  webhook:
    eventbridge:
      port: "{{.HookData.Port}}"
      endpoint: {{.HookData.Name}}
      method: POST
{{end}}
