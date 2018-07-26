package main

import (
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strconv"
	"strings"

	. "github.com/pikans/mealplan"
)

type ReminderGroup struct {
	Duties    []string
	TodayText string
}

var DutyGroups = map[string]ReminderGroup{
	"cook":  ReminderGroup{[]string{"Big cook", "Little cook"}, "today"},
	"clean": ReminderGroup{[]string{"Cleaner 1", "Cleaner 2"}, "tonight"},
}

func dayDeltaString(dayDelta int, todayText string) string {
	switch {
	case dayDelta == 0:
		return todayText
	case dayDelta == 1:
		return "tomorrow"
	case dayDelta == -1:
		return "yesterday"
	case dayDelta > 1:
		return fmt.Sprintf("in %d days", dayDelta)
	case dayDelta < -1:
		return fmt.Sprintf("%d days ago", -dayDelta)
	default:
		panic("impossible")
	}
}

const mailserver = "outgoing.mit.edu:smtp"
const from = "yfnkm@mit.edu"

func sendReminder(to []string, task string, mightBeCanceled bool) {
	msg :=
		`From: "pika kitchen manager" <%s>
To: %s
Subject: Reminder: you are signed up to %s

http://mealplan.pikans.org/

	`
	if mightBeCanceled {
		msg += "NOTE: not all shifts are filled, so dinner may be canceled"
	}
	body := fmt.Sprintf(msg, from, strings.Join(to, ", "), task)
	to = append(to, from) // bcc yfnkm
	err := smtp.SendMail(mailserver, nil, from, to, []byte(body))
	if err != nil {
		log.Printf("%v", err)
	}
}

// Returns whether any shifts are missing
func mightBeCanceled(data *Data, day int) bool {
	for _, duty := range Duties {
		if data.Assignments[duty][day] == "" {
			return true
		}
	}
	return false
}

func toEmail(username string) string {
	if strings.Contains(username, "@") {
		return username
	} else {
		return username + "@mit.edu"
	}
}

func main() {
	if len(os.Args) != 4 {
		log.Fatalf("wrong number of args: run %v <datapath> <duty> <daysOut>", os.Args[0])
	}
	var ok bool
	var err error
	var group ReminderGroup
	var dayDelta int

	task := os.Args[2]
	if group, ok = DutyGroups[task]; !ok {
		log.Fatalf("no task '%s'", task)
	}
	if dayDelta, err = strconv.Atoi(os.Args[3]); err != nil {
		log.Fatalf("invalid day delta '%s': %v", os.Args[2], err)
	}

	data, err := ReadData(os.Args[1])
	if err != nil {
		log.Fatalf("couldn't read data from '%s': %v", os.Args[1], err)
	}

	daysIn := DaysIn() + dayDelta
	if daysIn >= len(data.Days) {
		log.Fatalf("past the end of time: %d (0-indexed) days into %d-day plan", daysIn, len(data.Days))
	}

	to := []string{}
	for _, duty := range group.Duties {
		assignee := data.Assignments[duty][daysIn]
		if assignee != "" {
			to = append(to, assignee)
		}
	}

	taskText := fmt.Sprintf("%s %s", task, dayDeltaString(dayDelta, group.TodayText))
	sendReminder(to, taskText, mightBeCanceled(data, daysIn))
}
