# Golang-Twitch-Bot

# Runs in the cloud.

This bot used to run on your local and use a .env file. It's currently been reworked to run in the cloud. If you wanted to run your own version, you'd need to create a .env file in the root directory with a JSON containing your bot's username/OAUTH, then run terraform apply in the infra directory, SSH to the created server, install docker, build the container, access the secrets from the EC2 instance, load them into the environment in the docker container, and then run it.
