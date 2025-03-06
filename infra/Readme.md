
# upload ssm config
aws ssm create-document --name "deployBackend" --content file://aws-ssm-deploy.json --document-type Command


# Run ssm 
aws ssm send-command --document-name "deployBackend" --targets '[{"Key":"tag:ServerType","Values":["Backend"]}]' --comment "Deploying app to all Backend instances"

aws ssm send-command --document-name "deployBackend" --targets '[{"Key":"tag:ServerType","Values":["Backend"]}]' --comment "Deploying app to all Backend instances"

aws ssm send-command --document-name "deployScyllaMonitoring" --targets '[{"Key":"tag:ServerType","Values":["scylla-monitoring"]}]' --comment "Deploying scylla monitoring"

