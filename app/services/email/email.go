package email

import (
	"encoding/json"

	"monolith/app/jobs"
	"monolith/app/models"
)

// SendEmail enqueues an email job using the global job queue.
// subject and body are the email contents. sender is the "from" address.
// recipients is the list of destination email addresses.
func SendEmail(subject, body, sender string, recipients []string) error {
	p := jobs.EmailPayload{Subject: subject, Body: body, Sender: sender, To: recipients}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	return jobs.GetJobQueue().AddJob(models.JobTypeEmail, b)
}
