locals {
  db-password = random_password.db_password.result
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_instance
# https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_CreateDBInstance.html
resource "aws_db_instance" "twitch-chat-bot" {
  #identifiers
  identifier = "twitch-chat-bot"
  name       = "TwitchBot"
  username   = "TwitchBot"
  password   = local.db-password
  
  #networking
  publicly_accessible    = false
  db_subnet_group_name   = aws_db_subnet_group.chatbot-db-subnet.id
  vpc_security_group_ids = flatten([aws_security_group.chatbot-internal-sg.id])

  #engine
  engine         = "postgres"
  engine_version = "12"

  #compute
  instance_class = "db.t2.micro"

  #storage
  storage_type          = "gp2"
  allocated_storage     = 20
  max_allocated_storage = 100

  #availability
  multi_az              = false

  #deletion
  skip_final_snapshot = false
  deletion_protection = true

  #monitoring
  performance_insights_enabled = true
  copy_tags_to_snapshot        = true
  parameter_group_name         = aws_db_parameter_group.chatbot-db-parameters.name

  #tags
  tags = merge({
    Name = "Twitch Chatbot DB"
  },local.common_tags)
}

# https://registry.terraform.io/providers/hashicorp/random/latest/docs/resources/password
resource "random_password" "db_password" {
  length  = 16
  special = true
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_subnet_group
resource "aws_db_subnet_group" "chatbot-db-subnet" {
  name       = "internal_twitch_chatbot"
  subnet_ids = flatten([aws_subnet.chatbot-subnets.*.id])
  
  description = "Chatbot DB Subnet"

  #tag
  tags = merge({
    Name = "Twitch Chatbot DB Subnet"
  },local.common_tags)
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_parameter_group
resource "aws_db_parameter_group" "chatbot-db-parameters" {
  name   = "chatbot-rds-pg"
  family = "postgres12"

  parameter {
    name = "shared_preload_libraries"
    value = "pg_stat_statements"
  }

  parameter {
    name = "pg_stat_statements.track"
    value = "ALL"
  }
}