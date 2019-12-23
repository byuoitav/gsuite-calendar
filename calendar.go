package gsuite

import (
	"context"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/byuoitav/scheduler/calendars"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Calendar struct {
	UserEmail string
	//Path to file with service account credentials
	CredentialsPath string
	RoomId          string
}

func (c *Calendar) GetEvents(ctx context.Context) ([]calendars.Event, error) {
	calSvc, err := c.AuthenticateClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot authenticate client: %w", err)
	}

	calID, err := c.GetCalendarID(calSvc)
	if err != nil {
		return nil, err
	}

	currentTime := time.Now()
	currentDayBeginning := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	currentDayEnding := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, currentTime.Location())

	eventList, err := calSvc.Events.List(calID).Fields("items(summary, start, end)").TimeMin(currentDayBeginning.Format("2006-01-02T15:04:05-07:00")).TimeMax(currentDayEnding.Format("2006-01-02T15:04:05-07:00")).Do()
	if err != nil {
		return nil, fmt.Errorf("unable to retrieve events: %w", err)
	}

	var events []calendars.Event
	for _, event := range eventList.Items {
		eventStart, _ := time.Parse(event.Start.DateTime, "2006-01-02T15:04:05-07:00")
		eventEnd, _ := time.Parse(event.End.DateTime, "2006-01-02T15:04:05-07:00")

		events = append(events, calendars.Event{
			Title:     event.Summary,
			StartTime: eventStart,
			EndTime:   eventEnd})
	}

	return events, err
}

func (c *Calendar) CreateEvent(ctx context.Context, event calendars.Event) error {
	calSvc, err := c.AuthenticateClient(ctx)
	if err != nil {
		return fmt.Errorf("cannot authenticate client: %w", err)
	}

	calID, err := c.GetCalendarID(calSvc)
	if err != nil {
		return err
	}

	//Translate event into g suite calendar event object
	newEvent := &calendar.Event{
		Summary: event.Title,
		Start: &calendar.EventDateTime{
			DateTime: event.StartTime.Format("2006-01-02T15:04:05-07:00"),
		},
		End: &calendar.EventDateTime{
			DateTime: event.EndTime.Format("2006-01-02T15:04:05-07:00"),
		},
	}

	newEvent, err = calSvc.Events.Insert(calID, newEvent).Do()
	if err != nil {
		return fmt.Errorf("unable to create event: %w", err)
	}

	return nil
}

func (c *Calendar) GetCalendarID(calSvc *calendar.Service) (string, error) {
	calList, err := calSvc.CalendarList.List().Fields("items").Do()
	if err != nil {
		return "", fmt.Errorf("unable to retrieve calendar list: %w", err)
	}

	for _, cal := range calList.Items {
		if cal.Summary == c.RoomId {
			return cal.Id, nil
		}
	}
	return "", fmt.Errorf("room: %s does not have an assigned calendar", c.RoomId)
}

func (c *Calendar) AuthenticateClient(ctx context.Context) (*calendar.Service, error) {
	data, err := ioutil.ReadFile(c.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("can't read project key file: %w", err)
	}

	conf, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/calendar")
	if err != nil {
		return nil, fmt.Errorf("can't sign JWT: %w", err)
	}

	if len(c.UserEmail) > 0 {
		conf.Subject = c.UserEmail
	}

	ts := conf.TokenSource(ctx)

	service, err := calendar.NewService(ctx, option.WithTokenSource(ts))
	if err != nil {
		return nil, fmt.Errorf("can't make calendar service: %w", err)
	}

	return service, nil
}
