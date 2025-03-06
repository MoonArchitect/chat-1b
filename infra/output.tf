# output "backend_ip" {
#   value = aws_instance.backend.public_ip
# }
output "monitoring_public_ip" {
  value = aws_instance.monitoring.public_ip
}

output "scylla_db_private_ips" {
  value = aws_instance.scylla_db[*].private_ip
}
