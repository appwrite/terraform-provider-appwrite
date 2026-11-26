resource "appwrite_escalation" "on_call" {
  name = "On call"

  steps = [
    { roles = ["owner"] },
    { delay = 900, roles = ["developer"], user_ids = [var.on_call_user_id] },
  ]

  repeat = 1
}

variable "on_call_user_id" {
  type = string
}
