
#!/bin/bash
aws ecr create-repository --repository-name io.pavedroad.staging/eventbridge --region us-west-1 > ../docs/staging-repo-eventbridge.json
