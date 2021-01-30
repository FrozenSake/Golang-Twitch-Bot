# Golang-Twitch-Bot

# Runs in the cloud.

This bot used to run on your local and use a .env file. It's currently been reworked to run in the cloud. If you wanted to run your own version, you'd need to create a .env file in the root directory with a JSON containing your bot's username/OAUTH, then run terraform apply in the infra directory, SSH to the created server, install docker, build the container, access the secrets from the EC2 instance, load them into the environment in the docker container, and then run it.


# Improvement thoughts:

## Command Query Optimizations:

Basically you can do a "select distinct" query in sql, which will return all unique values for a particular column. So when setting up a channel, I could save on database hits by starting up with a "select distinct triggers" query, then building a list, and checking against the list (CPU time) in stead of a database query (database CPU time, network latency, local CPU time)
And adding and removing a command can just edit the list as it does the add/remove from the DB.