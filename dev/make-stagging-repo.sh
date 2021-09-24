
#!/bin/bash
aws ecr create-repository --repository-name io.pavedroad.stagging/eventbridge --region us-west-1 > ../docs/staggging-repo-eventbridge.json
