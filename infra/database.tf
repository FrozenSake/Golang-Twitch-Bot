# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_instance
# https://docs.aws.amazon.com/AmazonRDS/latest/APIReference/API_CreateDBInstance.html
resource "aws_db_instance" "Twitch_Chat_Bot" {
  #identifiers
  name     = ""
  username = ""
  password = ""
  
  #engine
  engine = postgres
  engine_version = "postgres13"

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
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/db_subnet_group
