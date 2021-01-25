# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/data-sources/availability_zones
data "aws_availability_zones" "aws-az" {
  state = "available"
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/vpc
resource "aws_vpc" "chatbot-vpc" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  #tag
  tags = merge({
    Name    = "Twitch Chatbot VPC"
  },local.common_tags)
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/subnet
resource "aws_subnet" "chatbot-subnets" {
  count                   = length(data.aws_availability_zones.aws-az.names)
  vpc_id                  = aws_vpc.chatbot-vpc.id
  cidr_block              = cidrsubnet(aws_vpc.chatbot-vpc.cidr_block, 8, count.index + 1)
  availability_zone       = data.aws_availability_zones.aws-az.names[count.index]
  map_public_ip_on_launch = true

  #tag
  tags = merge({
    Name    = "Twitch Chatbot Subnet #${count.index + 1}"
  },local.common_tags)
}


# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/internet_gateway
resource "aws_internet_gateway" "aws-igw" {
  vpc_id = aws_vpc.chatbot-vpc.id

  #tag
  tags = merge({
    Name    = "Twitch Chatbot Internet Gateway"
  },local.common_tags)
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route_table
resource "aws_route_table" "aws-route-table" {
  vpc_id = aws_vpc.chatbot-vpc.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.aws-igw.id
  }

  #tag
  tags = merge({
    Name    = "Twitch Chatbot routing table"
  },local.common_tags)
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/route_table_association
resource "aws_main_route_table_association" "aws-route-table-association" {
  vpc_id         = aws_vpc.chatbot-vpc.id
  route_table_id = aws_route_table.aws-route-table.id
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group
resource "aws_security_group" "chatbot-internal-sg" {
  name        = "twitch_chatbot_sg"
  description = "Control internal chatbot connections"
  vpc_id      = aws_vpc.chatbot-vpc.id

  ingress {
    protocol        = "-1"
    from_port       = 0
    to_port         = 0
    security_groups = [aws_security_group.chatbot-external-sg.id]
    self            = true
  }

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }

  #tag
  tags = merge({
    Name    = "Twitch Chatbot Internal security group"
  },local.common_tags)
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/security_group
resource "aws_security_group" "chatbot-external-sg"{
  name        = "external_twitch_chatbot"
  description = "Control external chatbot connections"
  vpc_id      = aws_vpc.chatbot-vpc.id

  ingress {
    protocol        = "-1"
    from_port       = 0
    to_port         = 0
    self            = true
  }

  egress {
    protocol    = "-1"
    from_port   = 0
    to_port     = 0
    cidr_blocks = ["0.0.0.0/0"]
  }

  #tag
  tags = merge({
    Name    = "Twitch Chatbot external security group"
  },local.common_tags)
}

resource "aws_security_group_rule" "chatbot-external-ssh" {
  type              = "ingress"
  from_port         = 22
  to_port           = 22
  protocol          = "tcp"
  cidr_blocks       = ["${chomp(data.http.icanhazip.body)}/32"]
  security_group_id = aws_security_group.chatbot-external-sg.id
}

data "http" "icanhazip" {
  url = "http://icanhazip.com"
}