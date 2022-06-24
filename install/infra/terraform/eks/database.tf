resource "aws_db_subnet_group" "gitpod_subnets" {
  subnet_ids = data.aws_subnet_ids.subnet_ids.ids
}

resource "aws_security_group" "rdssg" {
  name = "rdssg"
  vpc_id = module.vpc.vpc_id

  ingress {
    from_port = 0
    to_port = 3306
    protocol = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_db_instance" "gitpod" {
  allocated_storage    = 10
  max_allocated_storage = 100
  engine               = "mysql"
  engine_version       = "5.7"
  instance_class       = "db.t3.micro"
  vpc_security_group_ids = [ aws_security_group.rdssg.id ]
  identifier           = "db-${var.cluster_name}"
  name                 = "gitpod"
  username             = "gitpod"
  password             = "gitpod-qwat" # we can replace this with https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/secretsmanager_secret_version
  parameter_group_name = "default.mysql5.7"
  skip_final_snapshot  = true
  db_subnet_group_name = aws_db_subnet_group.gitpod_subnets.name
  publicly_accessible  = true
}
