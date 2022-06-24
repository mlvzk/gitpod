output "storage" {
    sensitive = true
    value = local.storage
}

output "registry" {
    sensitive = true
    value = local.registry
}

output "database" {
    sensitive = true
    value = local.database
}
