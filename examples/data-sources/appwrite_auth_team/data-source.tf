data "appwrite_auth_team" "engineers" {
  id = "engineers"
}

output "team_name" {
  value = data.appwrite_auth_team.engineers.name
}
