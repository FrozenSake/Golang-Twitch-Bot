locals {
  common_tags = {
    service = var.service
    env     = var.env
    ver     = var.service-version
  }
}

# AWS connection & authentication
variable "aws_region" {
  type        = string
  description = "AWS region"
}

#Tagging Vars
variable "env" {
  type        = string
  description = "The environment for this component"
  validation {
    condition     = can(regex("sandbox|staging|production", var.env))
    error_message = "Environment is one of: sandbox, staging, production."
  }
}
variable "service-version" {
  type        = string
  description = "The component version"
  validation {
    condition     = can(regex("v?\\d+\\.\\d+\\.\\d+", var.service-version))
    error_message = "Version should match the regex v?\\d+\\.\\d+\\.\\d+ - e.g.: 10.0.54 or v10.0.54."
  }
}
variable "service" {
  type        = string
  description = "The component service"
  validation {
    condition     = can(regex("\\w+-\\w+", var.service))
    error_message = "Service should be two words, split by a hyphen, eg: twitch-chatbot. I recommend domain-role."
  }
}