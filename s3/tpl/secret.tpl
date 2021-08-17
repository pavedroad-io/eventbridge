{{define "tpl/secret.tpl"}}
apiVersion: v1
kind: Secret
metadata:
  name: {{.Provider.Name}}
  lables:
  {{- range .Labels }}
    - {{.}}
  {{- end}}
type: Opaque
data:
  accesskey: {{.Provider.Credentials}}
  secretkey: {{.Provider.Key}}
{{end}}
/*
 Takes a Provider type as input
*/
