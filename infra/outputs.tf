output "ec2-ip" {
  value = aws_instance.bot-docker-host.private_ip
}