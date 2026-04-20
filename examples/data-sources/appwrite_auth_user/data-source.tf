data "appwrite_auth_user" "admin" {
  id = "64f2cd7e27bda9f23ab6"
}

output "user_email" {
  value = data.appwrite_auth_user.admin.email
}
