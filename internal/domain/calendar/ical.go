package calendar

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func ExportICS(event *CalendarEvent) ([]byte, error) {
	if err := event.Validate(); err != nil {
		return nil, err
	}
	formatTime := func(value time.Time) string { return value.UTC().Format("20060102T150405Z") }
	lines := []string{
		"BEGIN:VCALENDAR", "VERSION:2.0", "PRODID:-//Cashflow//Calendar//EN", "BEGIN:VEVENT",
		fmt.Sprintf("UID:%d@cashflow", event.ID), "SUMMARY:" + escapeICS(event.Name),
		"DTSTART:" + formatTime(event.Start), "DTEND:" + formatTime(event.Stop),
	}
	if event.Description != "" {
		lines = append(lines, "DESCRIPTION:"+escapeICS(event.Description))
	}
	if event.Location != "" {
		lines = append(lines, "LOCATION:"+escapeICS(event.Location))
	}
	lines = append(lines, "END:VEVENT", "END:VCALENDAR", "")
	return []byte(strings.Join(lines, "\r\n")), nil
}

func ParseICS(data []byte) (*CalendarEvent, error) {
	event := &CalendarEvent{}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inEvent := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		switch line {
		case "BEGIN:VEVENT":
			inEvent = true
		case "END:VEVENT":
			inEvent = false
		}
		if !inEvent || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key, value := parts[0], unescapeICS(parts[1])
		switch key {
		case "UID":
			if at := strings.Index(value, "@"); at > 0 {
				event.ID, _ = strconv.ParseInt(value[:at], 10, 64)
			}
		case "SUMMARY":
			event.Name = value
		case "DESCRIPTION":
			event.Description = value
		case "LOCATION":
			event.Location = value
		case "DTSTART":
			parsed, err := time.Parse("20060102T150405Z", value)
			if err != nil {
				return nil, invalidCalendar("invalid DTSTART in iCal")
			}
			event.Start = parsed
		case "DTEND":
			parsed, err := time.Parse("20060102T150405Z", value)
			if err != nil {
				return nil, invalidCalendar("invalid DTEND in iCal")
			}
			event.Stop = parsed
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(event.Name) == "" || event.Start.IsZero() || event.Stop.IsZero() || !event.Stop.After(event.Start) {
		return nil, invalidCalendar("iCal event requires a name and valid start/stop")
	}
	event.Duration = event.Stop.Sub(event.Start).Hours()
	return event, nil
}

func unescapeICS(value string) string {
	value = strings.ReplaceAll(value, `\n`, "\n")
	value = strings.ReplaceAll(value, `\,`, ",")
	value = strings.ReplaceAll(value, `\;`, ";")
	return strings.ReplaceAll(value, `\\`, `\`)
}

func escapeICS(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, ";", `\;`)
	value = strings.ReplaceAll(value, ",", `\,`)
	return strings.ReplaceAll(value, "\n", `\n`)
}
