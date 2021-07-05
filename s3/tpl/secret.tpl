{{define "tpl/secret.tpl"}}
apiVersion: v1
kind: Secret
metadata:
  name: {{.Provider.Name}}
  lables:
  - customer: {{.ID}}
  - environment: {{.Environment}}
type: Opaque
data:
  accesskey: {{.Provider.Credentials}}
  secretkey: {{.Provider.Key}}
{{end}}
/*
 Takes a Provider type as input
*/
