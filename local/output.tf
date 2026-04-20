output "database_name" {
  value = data.appwrite_tablesdb.main.name
}

output "bucket_name" {
  value = data.appwrite_storage_bucket.uploads.name
}

output "user_name" {
  value = data.appwrite_auth_user.john.name
}

output "team_name" {
  value = data.appwrite_auth_team.engineering.name
}

output "function_runtime" {
  value = data.appwrite_function.process_order.runtime
}

output "site_framework" {
  value = data.appwrite_site.landing_page.framework
}

output "topic_name" {
  value = data.appwrite_messaging_topic.announcements.name
}

output "webhook_url" {
  value = data.appwrite_webhook.order_events.url
}
