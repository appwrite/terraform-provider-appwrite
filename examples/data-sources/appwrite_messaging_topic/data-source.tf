data "appwrite_messaging_topic" "notifications" {
  id = "notifications"
}

output "topic_name" {
  value = data.appwrite_messaging_topic.notifications.name
}
