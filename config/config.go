package config

import (
	"os"
)

var JOB_QUEUE_NUM_WORKERS = 4

var PORT = os.Getenv("PORT")
