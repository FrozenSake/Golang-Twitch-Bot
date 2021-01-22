## https://registry.terraform.io/
terraform {
    required_version = ">= 0.12"
}

# https://registry.terraform.io/providers/hashicorp/aws/latest/docs
provider "aws" {
    version    = "~> 3.25"
    region     = var.aws_region
    access_key = var.aws_access_key
    secret_key = var.aws_secret_key
}