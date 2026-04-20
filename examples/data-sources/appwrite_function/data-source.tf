data "appwrite_function" "hello_world" {
  id = "64f2cd7e27bda9f23ab6"
}

output "function_runtime" {
  value = data.appwrite_function.hello_world.runtime
}
