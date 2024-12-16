terraform {
    required_version = "~> 1.10.0"

    backend "s3" {
        use_lockfile = true
    }
}

provider "aws" {
    region = "ap-northeast-1"
}

resource "aws_s3_bucket" "test_bucket" {
    bucket = var.bucket_name
}

variable "bucket_name" {
     description = "The name of the S3 bucket"
     type        = string
}
