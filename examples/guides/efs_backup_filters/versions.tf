terraform {
  required_providers {
    eon = {
      source  = "eon-io/eon"
      version = "~> 1.0"
    }
  }
}

provider "eon" {
}
