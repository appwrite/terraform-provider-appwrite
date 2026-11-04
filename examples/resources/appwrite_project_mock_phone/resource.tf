# Lets an app store reviewer sign in with a phone number without receiving an SMS.
resource "appwrite_project_mock_phone" "app_review" {
  number = "+15555550100"
  otp    = "123456"
}
