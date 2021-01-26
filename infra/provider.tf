## https://registry.terraform.io/
terraform {
    required_version = ">= 0.12"
    experiments      = [variable_validation]
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs
provider "aws" {
    version    = ">= 3.25"
    region     = var.aws_region
}

provider "random" {
  version = ">= 3.0.1"
}

provider "tls" {
  version = ">= 3.0.0"
}

provider "http" {
  version = ">= 2.0.0"
}

provider "template" {
  version = ">= 2.2.0"
}