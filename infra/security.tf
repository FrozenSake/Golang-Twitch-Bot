resource "aws_secretsmanager_secret" "db-user" {
  name        = "${var.env}/${var.service}/db-user"
  description = "The Database Username"

  tags = merge({
    Name = "Twitch Chatbot Database User Secret"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "db-user" {
  secret_id     = aws_secretsmanager_secret.db-user.id
  secret_string = aws_db_instance.twitch-chat-bot.username
}

resource "aws_secretsmanager_secret" "db-endpoint" {
  name        = "${var.env}/${var.service}/db-endpoint"
  description = "The Database endpoint"

  tags = merge({
    Name = "Twitch Chatbot Database Endpoint"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "db-endpoint" {
  secret_id     = aws_secretsmanager_secret.db-endpoint.id
  secret_string = aws_db_instance.twitch-chat-bot.endpoint
}

resource "aws_secretsmanager_secret" "db-name" {
  name        = "${var.env}/${var.service}/db-name"
  description = "The name of the database on the database instance"

  tags = merge({
    Name = "Twitch Chatbot Database Name"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "db-name" {
  secret_id = aws_secretsmanager_secret.db-name.id
  secret_string = aws_db_instance.twitch-chat-bot.name
}

resource "aws_secretsmanager_secret" "db-password" {
  name        = "${var.env}/${var.service}/db-password"
  description = "The Database password"

  tags = merge({
    Name = "Twitch Chatbot Database Password"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "db-password" {
  secret_id = aws_secretsmanager_secret.db-password.id
  secret_string = local.db-password
}

resource "aws_secretsmanager_secret" "bot-username" {
  name        = "${var.env}/${var.service}/bot-username"
  description = "The bot account's username"

  tags = merge({
    Name = "Twitch Chatbot Main Twitch Account Username"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "bot-username" {
  secret_id = aws_secretsmanager_secret.bot-username.id
  secret_string = jsondecode(file("../.env"))["botname"]
}

resource "aws_secretsmanager_secret" "bot-oauth" {
  name        = "${var.env}/${var.service}/bot-oauth"
  description = "The bot account's OAuth token"

  tags = merge({
    Name = "Twitch Chatbot Twitch Account OAuth"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "bot-oauth" {
  secret_id = aws_secretsmanager_secret.bot-oauth.id
  secret_string = jsondecode(file("../.env"))["oauth"]
}

resource "aws_secretsmanager_secret" "bot-ec2-private-key" {
  name        = "${var.env}/${var.service}/bot-ec2-private-key"
  description = "The private key for the bot ec2 instance"

  tags = merge({
    Name = "Twitch Chatbot Server Private Key"
  },local.common_tags)
}

resource "aws_secretsmanager_secret_version" "bot-ec2-private-key" {
  secret_id = aws_secretsmanager_secret.bot-ec2-private-key.id
  secret_string = tls_private_key.bot-ec2-key.private_key_pem
}