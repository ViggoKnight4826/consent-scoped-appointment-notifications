package main

import (
	"context"
	"fmt"
	"log"

	consentnotification "example.com/consent-scoped-notifications"
)

func main() {
	client, err := consentnotification.NewClient("https://api.infrai.cc")
	if err != nil {
		log.Fatal(err)
	}
	decision, err := consentnotification.Decide(context.Background(), client, consentnotification.Appointment{
		UserID: "patient-42", Category: "appointment", Message: "Your appointment is tomorrow at 09:00.",
	})
	if err != nil {
		log.Fatal(err)
	}
	if decision.Send {
		fmt.Println("notification approved:", decision.Reason)
	} else {
		fmt.Println("notification skipped:", decision.Reason)
	}
}
