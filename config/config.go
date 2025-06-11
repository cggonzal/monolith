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
