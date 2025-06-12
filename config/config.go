/*
Package config centralizes compile-time configuration knobs used across the
application. Values here rarely change at runtime and are imported by other
packages.
*/
package config

import (
	"os"
)

var JOB_QUEUE_NUM_WORKERS = 4

var PORT = os.Getenv("PORT")

// Mailgun configuration. Set MAILGUN_DOMAIN and MAILGUN_API_KEY environment
// variables in production. MAILGUN_API_BASE rarely changes and defaults to the
// public API endpoint.
var MAILGUN_DOMAIN = os.Getenv("MAILGUN_DOMAIN")
var MAILGUN_API_KEY = os.Getenv("MAILGUN_API_KEY")

// Base URL for the Mailgun API. Overridable for testing.
var MAILGUN_API_BASE = "https://api.mailgun.net/v3"

var SECRET_KEY = os.Getenv("SECRET_KEY")
