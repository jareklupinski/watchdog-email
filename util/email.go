package util

import (
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"log"
	"net"
	"os"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func EmailIsValid(e string) bool {
	if len(e) < 3 && len(e) > 254 {
		return false
	}
	if !emailRegex.MatchString(e) {
		return false
	}
	parts := strings.Split(e, "@")
	mx, err := net.LookupMX(parts[1])
	if err != nil || len(mx) == 0 {
		return false
	}
	return true
}

func SendEmail(emailAddress string) {
	from := mail.NewEmail("⏰🐕.📧", "timer@watchdog.email")
	subject := "Your Watchdog.Email Timer has Fired!"
	to := mail.NewEmail(emailAddress, emailAddress)
	plainTextContent := "Reset your timer: http://watchdog.email/" + emailAddress
	htmlContent := "Reset your timer: <a href=\"http://watchdog.email/" + emailAddress + "\">http://watchdog.email/" + emailAddress + "</a>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))

	response, err := client.Send(message)
	if err != nil || response.StatusCode != 200 {
		if response != nil {
			log.Printf("Failed to send email to %s: %s, SendGrid Response %d: %s\n", emailAddress, err, response.StatusCode, response.Body)
		} else {
			log.Printf("Failed to send email to %s: %s, No SendGrid Response\n", emailAddress, err)
		}
	}
}
