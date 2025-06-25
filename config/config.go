/*
Package config centralizes compile-time configuration knobs used across the
application. Values here rarely change at runtime and are imported by other
packages.
*/
package config

import (
	"log/slog"
	"os"
)

var JOB_QUEUE_NUM_WORKERS = 4

var PORT = "9000" // change to os.Getenv("PORT") if you want to use an environment variable

// Mailgun configuration. Set MAILGUN_DOMAIN and MAILGUN_API_KEY environment
// variables in production. MAILGUN_API_BASE rarely changes and defaults to the
// public API endpoint.
var MAILGUN_DOMAIN = os.Getenv("MAILGUN_DOMAIN")
var MAILGUN_API_KEY = os.Getenv("MAILGUN_API_KEY")

// Base URL for the Mailgun API. Overridable for testing.
var MAILGUN_API_BASE = "https://api.mailgun.net/v3"

var SECRET_KEY = os.Getenv("SECRET_KEY")

var MONOLITH_VERSION = "0.1.0"

func init() {
	// log warnings if secret key and other environment variables are not set
	if SECRET_KEY == "" {
		slog.Warn("SECRET_KEY is not set, using default value. This is insecure for production use.")
		SECRET_KEY = "default_secret_key"
	}
	if MAILGUN_DOMAIN == "" {
		slog.Warn("MAILGUN_DOMAIN is not set, email functionality will not function properly.")
	}
	if MAILGUN_API_KEY == "" {
		slog.Warn("MAILGUN_API_KEY is not set, email functionality will not function properly.")
	}
}
