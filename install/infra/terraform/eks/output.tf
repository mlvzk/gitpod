output "external_dns_settings" {
  value = [
    {
        "name" = "provider",
        "value" = "aws"
    },
    {
        "name" = "aws.region",
        "value" = "eu-west-1"
    },
    {
        "name" = "aws.credentials.secretKey",
        "value" = aws_iam_access_key.edns.secret
    },
    {
        "name" = "aws.credentials.accessKey",
        "value" = aws_iam_access_key.edns.id
    }
    ]
}

output "secretAccessKey" {
    value = try("${aws_iam_access_key.edns.secret}", "")
}

output "cert_manager_issuer" {
  value = try({
        region = "eu-west-1"
        secretAccessKeySecretRef = {
            name = "route53-credentials"
            key = "secret-access-key"
        }

        hostedZoneID  = aws_route53_zone.gitpod.zone_id
        accessKeyID = aws_iam_access_key.edns.id
  }, {})
}

output "domain_nameservers" {
  value = formatlist("%s.", resource.aws_route53_zone.gitpod.name_servers)
}

output "database" {
  sensitive = true
  value = try({
    host     = "${aws_db_instance.gitpod.address}"
    password = "gitpod-qwat" # FIXME hardcoded at this point
    port     = 3306
    username = "${aws_db_instance.gitpod.username}"
  }, {})
}

output "registry" {
  sensitive = true
  value = try({
    server  = aws_ecr_repository.gitpod.repository_url
    username = data.aws_ecr_authorization_token.gitpod.user_name
    password = data.aws_ecr_authorization_token.gitpod.password
  }, {})
}

output "storage" {
  sensitive = true
  value = try({
    access_key_id     = aws_iam_access_key.bucket_storage_user.id
    secret_access_key = aws_iam_access_key.bucket_storage_user.secret
    region            = aws_s3_bucket.gitpod-storage.region
    bucket_name       = aws_s3_bucket.gitpod-storage.id
    endpoint          = "s3.amazonaws.com"
  }, {})
}
