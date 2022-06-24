terraform {
  required_providers {
    aws = {
        version = " ~> 3.0"
        source = "registry.terraform.io/hashicorp/aws"
    }
  }
}

provider "aws" {
  region  = "eu-west-1"
}
