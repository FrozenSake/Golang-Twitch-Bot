# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/instance
resource "aws_instance" "bot-docker-host" {
  ami           = "ami-02e44367276fe7adc"
  instance_type = "t3.micro"
  key_name      = aws_key_pair.bot-ec2-key.key_name

  subnet_id = aws_subnet.chatbot-subnets[0].id
  
  iam_instance_profile = aws_iam_instance_profile.bot-ec2-profile.name
  security_groups      = [
    aws_security_group.chatbot-internal-sg.id,
    aws_security_group.chatbot-external-sg.id
  ]

  user_data = data.template_file.bot-user-data.rendered

  root_block_device {
    volume_size = 40
  }

  tags = merge({
    Name = "Twitch Chatbot Server"
  },local.common_tags)
}

data "template_file" "bot-user-data" {
  template = "${file("./user_data")}"
  vars = {
    region = var.aws_region
    env = var.env
    service = var.service
    role = aws_iam_role.bot-ec2-role.name
  }
}

resource "aws_key_pair" "bot-ec2-key" {
  key_name = "bot-ec2-key"
  public_key = tls_private_key.bot-ec2-key.public_key_openssh
  tags = merge({
    Name = "Twitch Chatbot Server Keypair"
  },local.common_tags)
}

resource "tls_private_key" "bot-ec2-key" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_iam_instance_profile" "bot-ec2-profile" {
  name = "chatbot_profile"
  role = aws_iam_role.bot-ec2-role.name
}

resource "aws_iam_role" "bot-ec2-role" {
  name        = "chatbot_ec2_role"
  path        = "/"
  description = "The role to attach to the chatbot ec2"

  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
              "Service": "ec2.amazonaws.com"
            },
            "Action": "sts:AssumeRole"
        }
    ]
}
EOF

  tags = merge({
    Name = "Twitch Chatbot Server IAM Role"
  },local.common_tags)
}

resource "aws_iam_role_policy_attachment" "bot-ec2-secrets-policy-attachment" {
  role       = aws_iam_role.bot-ec2-role.name
  policy_arn = aws_iam_policy.bot-secrets-policy.arn
}

resource "aws_iam_policy" "bot-secrets-policy" {
  name        = "chatbot_secretsmanager_policy"
  path        = "/"
  description = "Allow the chatbot ec2 to access the proper secrets in secretsmanager."

  policy = <<EOF
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
              "secretsmanager:GetSecretValue"
            ],
            "Resource": [
              "arn:aws:secretsmanager:ca-central-1:280028325900:secret:sandbox/twitch-chatbot/db-user*",
              "arn:aws:secretsmanager:ca-central-1:280028325900:secret:sandbox/twitch-chatbot/db-password*",
              "arn:aws:secretsmanager:ca-central-1:280028325900:secret:sandbox/twitch-chatbot/db-name*",
              "arn:aws:secretsmanager:ca-central-1:280028325900:secret:sandbox/twitch-chatbot/db-endpoint*",
              "arn:aws:secretsmanager:ca-central-1:280028325900:secret:sandbox/twitch-chatbot/bot-username*",
              "arn:aws:secretsmanager:ca-central-1:280028325900:secret:sandbox/twitch-chatbot/bot-oauth*"
            ]
        }
    ]
}
EOF
}
