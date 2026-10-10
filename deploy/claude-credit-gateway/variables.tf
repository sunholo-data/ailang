variable "provision" {
  description = "Create authority/project/IAM/secret resources only after infrastructure review. Defaults off."
  type        = bool
  default     = false
}
variable "deploy_runtime" {
  description = "After provisioning the signing secret value and IAM hardening, create service/jobs. Account remains disabled."
  type        = bool
  default     = false
}
variable "authority_project_id" {
  type    = string
  default = "ailang-credit-authority"
}
variable "billing_account" {
  description = "Billing account for the new private authority project; no secret values."
  type        = string
  default     = ""
}
variable "folder_id" {
  type    = string
  default = "389195706883"
}
variable "production_project" {
  type    = string
  default = "ailang-multivac"
}
variable "region" {
  type    = string
  default = "europe-west1"
}
variable "gateway_image" {
  description = "Verified coordinator image @sha256 digest, containing credit-gateway serve. Buildpack entrypoint retained."
  type        = string
  default     = ""
}
variable "operators" {
  description = "Exact Google identity emails authorized for grant confirmation and account control, distinct from coordinators."
  type        = set(string)
  default     = []
}
variable "organization" {
  type    = string
  default = ""
}
variable "workspace" {
  type    = string
  default = ""
}
variable "environments" {
  description = "Existing dev/prod dependencies; egress mirrors existing policy when configured. Both lanes use one authority."
  type = map(object({
    project          = string
    prefix           = string
    coordinator      = string
    agent_image      = string
    agent_go_image   = string
    github_secret    = string
    topic_prefix     = string
    artifacts_bucket = string
    config_env       = optional(map(string), {})
    egress           = optional(object({ network = string, subnetwork = string }))
  }))
  default = {}
}
