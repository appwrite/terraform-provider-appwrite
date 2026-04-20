data "appwrite_tablesdb" "main" {
  id = appwrite_tablesdb.main.id
}

data "appwrite_storage_bucket" "uploads" {
  id = appwrite_storage_bucket.uploads.id
}

data "appwrite_auth_user" "john" {
  id = appwrite_auth_user.john.id
}

data "appwrite_auth_team" "engineering" {
  id = appwrite_auth_team.engineering.id
}

data "appwrite_function" "process_order" {
  id = appwrite_function.process_order.id
}

data "appwrite_site" "landing_page" {
  id = appwrite_site.landing_page.id
}

data "appwrite_messaging_topic" "announcements" {
  id = appwrite_messaging_topic.announcements.id
}

data "appwrite_webhook" "order_events" {
  id = appwrite_webhook.order_events.id
}
