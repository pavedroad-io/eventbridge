{{define "tpl/secret.tpl"}}
apiVersion: v1
kind: Secret
metadata:
  name: {{.Provider.Name}}
  labels:
  {{- range .Labels }}
    {{.Key}}: "{{.Value}}"
  {{- end}}
type: Opaque
data:
  accesskey: {{.Provider.Credentials}}
  secretkey: {{.Provider.Key}}
{{end}}
/*
 Takes a Provider type as input
*/
