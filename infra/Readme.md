
# upload ssm config
aws ssm create-document --name "deployBackend" --content file://aws-ssm-deploy.json --document-type Command

# steps to stress test
1. spin up infra
2. run upload-code.sh
3. run aws ssm send-command
4. ssh into the server & init db
5. build exec
6. launch server
7. change ip in local prometheus config
8. open up scylla grafana with public ip
9. change public ip of backend in test agent
10. run test-agent

# Run ssm 
aws ssm send-command --document-name "deployBackend" --targets '[{"Key":"tag:ServerType","Values":["backend"]}]' --comment "Deploying app to all Backend instances"

aws ssm send-command --document-name "deployBackend" --targets '[{"Key":"tag:ServerType","Values":["Backend"]}]' --comment "Deploying app to all Backend instances"

aws ssm send-command --document-name "deployScyllaMonitoring" --targets '[{"Key":"tag:ServerType","Values":["scylla-monitoring"]}]' --comment "Deploying scylla monitoring"

# Debug info

config for scylla-monitoring ec2, TODO: make the scylla_server config file dynamic so that it reflects infra setup (right now it's backed into AMI snapshot)
```
# #!/bin/bash
# sudo apt-get update -y
# sudo apt-get install -y docker.io git
# sudo systemctl start docker
# sudo systemctl enable docker

# sudo git clone https://github.com/scylladb/scylla-monitoring.git /opt/scylla-monitoring
# cd /opt/scylla-monitoring

# sudo mkdir -p prometheus
# sudo cat > prometheus/scylla_servers.yml <<EOL
#   - targets:
#         - 10.0.2.10
#         - 10.0.2.11
#     labels:
#         cluster: chat-cluster
# EOL

# sudo ./start-all.sh -v 6.2
# sudo ufw allow 3000/tcp
# sudo ufw allow 9090/tcp
# sudo ufw --force enable
# EOF
```


replace scylla_image_builder or something like that, to debug issues with scylla not finding any disks other than root
```
# #!/usr/bin/env bash
# x="$(readlink -f "$0")"
# b="$(basename "$x")"
# d="$(dirname "$x")"

# echo "Script name (\$0): $0"
# echo "Directory (\$d): $d"
# echo "Basename (\$b): $b"
# echo "All arguments (\$@): $@"

# PYTHONPATH="${d}:${d}/libexec:$PYTHONPATH" PATH="${d}/../python3/bin:${PATH}" exec -a "$0" python3 "${d}/test.py" "$@"
```

test.py
```
# import argparse
# import sys
# from pathlib import Path
# from subprocess import run
# from lib.scylla_cloud import is_ec2, is_gce, is_azure, get_cloud_instance, out

# class DiskIsNotEmptyError(Exception):
#     def __init__(self, disk):
#         self.disk = disk
#         pass

#     def __str__(self):
#         return f"{self.disk} is not empty, abort setup"

# def check_persistent_disks_are_empty(disks):
#     for disk in disks:
#         part = out(f'lsblk -dpnr -o PTTYPE /dev/{disk}')
#         fs = out(f'lsblk -dpnr -o FSTYPE /dev/{disk}')
#         if part != '' or fs != '':
#             raise DiskIsNotEmptyError(f'/dev/{disk}')

# def get_default_devices(instance):
#     disk_names = []
#     disk_names = instance.get_local_disks()
#     if not disk_names:
#         disk_names = instance.get_remote_disks()
#         check_persistent_disks_are_empty(disk_names)
#     return [str(Path('/dev', name)) for name in disk_names]

# if __name__ == "__main__":
#   instance = get_cloud_instance()
#   print(instance._disks)
#   disk_devices = get_default_devices(instance)
#   print(disk_devices)
```
