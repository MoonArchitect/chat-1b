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

data "aws_availability_zone" "servers" {
  name = "us-east-1a"
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
  instance_type = "c6g.xlarge"

  iam_instance_profile   = aws_iam_instance_profile.ssm_instance_profile.name
  subnet_id              = aws_subnet.public_subnet.id
  vpc_security_group_ids = [aws_security_group.backend_sg.id]
  availability_zone      = data.aws_availability_zone.servers.name

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
  count                  = 1
  ami                    = "ami-039338d7e5e3ed484"
  instance_type          = "c6gd.large"
  subnet_id              = aws_subnet.db_subnet.id
  vpc_security_group_ids = [aws_security_group.db_sg.id]
  private_ip             = element(["10.0.2.10", "10.0.2.11"], count.index)
  availability_zone      = data.aws_availability_zone.servers.name

  user_data = <<EOF
  {
    "scylla_yaml": {
        "cluster_name": "chat-cluster",
        "seed_provider": [{"class_name": "org.apache.cassandra.locator.SimpleSeedProvider",
                          "parameters": [{"seeds": "10.0.2.10"}]}]
    },
    "start_scylla_on_first_boot": true
  }
  EOF

  tags = {
    Name         = "scylla-db-${count.index + 1}"
    "ServerType" = "scylla-db"
  }
}

# resource "aws_ebs_volume" "db_ebs_vol" {
#   count             = 2
#   availability_zone = aws_instance.scylla_db[count.index].availability_zone
#   size             = 50
#   type             = "gp3"

#   tags = {
#     Name = "scylla-db-ebs-${count.index + 1}"
#   }
# }

# resource "aws_volume_attachment" "db_att" {
#   count        = 2
#   device_name  = "/dev/sda2"
#   volume_id    = aws_ebs_volume.db_ebs_vol[count.index].id
#   instance_id  = aws_instance.scylla_db[count.index].id
# }

resource "aws_instance" "monitoring" {
  ami                    = "ami-0093872901c1f8e0a"
  instance_type          = "t4g.micro"
  subnet_id              = aws_subnet.public_subnet.id
  vpc_security_group_ids = [aws_security_group.monitoring_sg.id]
  availability_zone      = data.aws_availability_zone.servers.name

  tags = {
    Name         = "scylla-monitoring"
    "ServerType" = "scylla-monitoring"
  }

  user_data = <<EOF
#!/bin/bash
echo "start docker"
sudo systemctl start docker
echo "enable docker"
sudo systemctl enable docker
echo "cd"
cd /opt/scylla-monitoring
echo "kill all"
sudo ./kill-all.sh
echo "start all"
sudo ./start-all.sh -v 6.2
echo "done"
EOF

}
