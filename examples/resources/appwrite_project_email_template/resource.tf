resource "appwrite_project_email_template" "verification" {
  type    = "verification"
  locale  = "en"
  subject = "Verify your {{project}} account"
  message = <<-HTML
    <p>Hello {{user}},</p>
    <p>Follow <a href="{{redirect}}">this link</a> to verify your email address.</p>
    <p>The {{project}} team</p>
  HTML

  # Appwrite accepts custom templates only while custom SMTP is enabled.
  depends_on = [appwrite_project_smtp.example]
}

resource "appwrite_project_email_template" "verification_pt_br" {
  type        = "verification"
  locale      = "pt-br"
  subject     = "Confirme sua conta {{project}}"
  message     = "<p>Olá {{user}}, confirme seu e-mail: <a href=\"{{redirect}}\">{{redirect}}</a></p>"
  sender_name = "Equipe Example"

  depends_on = [appwrite_project_smtp.example]
}
