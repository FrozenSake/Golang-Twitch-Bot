resource "aws_iam_user" "bot" {
  name = "chatbot"
  path = "/twitch/"
}

resource "aws_iam_access_key" "bot" {
  user = aws_iam_user.bot.name
}

resource "aws_iam_user_policy" "bot-secrets-policy" {
  name        = "chatbot_secretsmanager_policy"
  user        = aws_iam_user.bot.name

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