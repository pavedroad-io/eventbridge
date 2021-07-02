/*
Template data is derived from a Sync Configuration
We need the Hook, the Provider and the LambdaTrigger
*/
{{define "lambda.tpl}}
apiVersion: argoproj.io/v1alpha1
kind: Sensor
metadata:
  name: eventbridgelambda
spec:
  dependencies:
    - name: {{.Hook.Name}}
      eventSourceName: {{.Hook.Name}}
      eventName: {{.Hook.Name}}
  triggers:
    - template:
        name: {{.LambdaTrigger.Name}}
        awsLambda:
          functionName: {{.LambdaTrigger.FunctionName}}
          accessKey:
            name: {{.Provider.Credentials}}
            key: {{.Provider.Key}}
          secretKey:
            name: {{.Provider.Credentials}}
            key: {{.Provider.Key}}
          region: {{.LambdaTrigger.Region}}
          payload:
            - src:
                dependencyName: {{.Hook.Name}}
                dataKey: body
              dest: data
{{end}}
