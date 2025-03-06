terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
}

# data "aws_ami" "ubuntu" {
#   most_recent = true

#   filter {
#     name   = "name"
#     values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-arm64-server-*"]
#   }

#   filter {
#     name   = "virtualization-type"
#     values = ["hvm"]
#   }
# }

resource "aws_instance" "backend" {
  ami           = "ami-00a37593d56e9b1bf"
  instance_type = "t4g.micro"

  iam_instance_profile   = aws_iam_instance_profile.ssm_instance_profile.name
  subnet_id              = aws_subnet.public_subnet.id
  vpc_security_group_ids = [aws_security_group.backend_sg.id]

  tags = {
    Name         = "backend"
    "ServerType" = "backend"
  }
}

# resource "aws_volume_attachment" "ebs_att" {
#   device_name = "/dev/sda1"
#   volume_id   = aws_ebs_volume.ebs_vol.id
#   instance_id = aws_instance.backend.id
# }

# resource "aws_ebs_volume" "ebs_vol" {
#   availability_zone = aws_instance.backend.availability_zone
#   size             = 10
#   type             = "gp3"
# }

resource "aws_instance" "scylla_db" {
  count                  = 2
  ami                    = "ami-003c3ecbf6affbec0"
  instance_type          = "t4g.micro"
  subnet_id              = aws_subnet.db_subnet.id
  vpc_security_group_ids = [aws_security_group.db_sg.id]
  private_ip             = element(["10.0.2.10", "10.0.2.11"], count.index)

  user_data = <<EOF
  {
    "scylla_yaml": {
        "cluster_name": "chat-cluster",
        "seed_provider": [{"class_name": "org.apache.cassandra.locator.SimpleSeedProvider",
                          "parameters": [{"seeds": "10.0.2.10,10.0.2.11"}]}]
    },
    "start_scylla_on_first_boot": true
  }
  EOF

  tags = {
    Name         = "scylla-db-${count.index + 1}"
    "ServerType" = "scylla-db"
  }
}

resource "aws_instance" "monitoring" {
  ami                    = "ami-0093872901c1f8e0a"
  instance_type          = "t4g.micro"
  subnet_id              = aws_subnet.public_subnet.id
  vpc_security_group_ids = [aws_security_group.monitoring_sg.id]

  tags = {
    Name         = "scylla-monitoring"
    "ServerType" = "scylla-monitoring"
  }

  user_data = <<-EOF
#!/bin/bash
sudo systemctl start docker
sudo systemctl enable docker
cd /opt/scylla-monitoring
sudo ./start-all.sh -v 6.2
EOF

}

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
