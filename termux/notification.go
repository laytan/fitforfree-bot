package termux

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"time"
)

// NotificationAction specifies a command to execute
type NotificationAction string

// NotificationPriority is how the notification is handled by the os high/low/max/min/default
type NotificationPriority string

func (p NotificationPriority) validate() bool {
	for _, prio := range p.validPriorities() {
		if prio == p {
			return true
		}
	}
	return false
}

func (p NotificationPriority) validPriorities() []NotificationPriority {
	return []NotificationPriority{"high", "low", "max", "min", "default"}
}

// NotificationType is the type of notification, media or default
type NotificationType string

func (t NotificationType) validate() bool {
	for _, nType := range t.validTypes() {
		if t == nType {
			return true
		}
	}
	return false
}

func (t NotificationType) validTypes() []NotificationType {
	return []NotificationType{"media", "default"}
}

// NotificationLED sets led options
type NotificationLED struct {
	// Set an LED color, should be rrggbb without #
	Color string `json:"--led-color,omitempty"`
	// How long the led should be off
	Off time.Duration `json:"--led-off,omitempty"`
	// How long the led should be on
	On time.Duration `json:"--led-on,omitempty"`
}

// NotificateOptions are the options to use when showing a notification
// More info: https://wiki.termux.com/wiki/Termux-notification
type NotificateOptions struct {
	// Notification title
	Title string `json:"-t,omitempty"`
	// Notification content
	Content string `json:"-c,omitempty"`
	// On clicking the notification
	Action NotificationAction `json:"--action,omitempty"`
	// Only alert on initial send, not on edits etc
	AlertOnce bool `json:"--alert-once"`
	// Buttons, there can be 3 buttons in a notification
	Buttons []NotificationButton `json:"buttons,omitempty"`
	// Notifications with the same group will be grouped together
	Group string `json:"--group,omitempty"`
	// Override an existing notification by id
	ID uint `json:"-i,omitempty"`
	// Show an image, should be an absolute path
	ImagePath string `json:"--image-path,omitempty"`
	// Led options
	LED *NotificationLED `json:"led,omitempty"`
	// Action to execute on deleting/clearing the notification
	OnDelete NotificationAction `json:"--on-delete,omitempty"`
	// Pin the notification
	OnGoing bool `json:"--ongoing"`
	// What priority is the notification
	Priority NotificationPriority `json:"--priority,omitempty"`
	// Play a sound with the notification
	Sound bool `json:"--sound"`
	// Specify a pattern to vibrate on
	VibratePattern []time.Duration `json:"vibratePattern,omitempty"`
	// Type default or media notification
	Type NotificationType `json:"--type,omitempty"`
}

func (o *NotificateOptions) validate() []string {
	errs := make([]string, 0)
	if !o.Type.validate() {
		errs = append(errs, fmt.Sprintf("Notification type %s is invalid. Valid types: %v", o.Type, o.Type.validTypes()))
	}

	if !o.Priority.validate() {
		errs = append(errs, fmt.Sprintf("Notification priority %s is invalid. Valid types: %v", o.Priority, o.Priority.validPriorities()))
	}
	return errs
}

func (o *NotificateOptions) String() string {
	// Convert the struct to json
	js, _ := json.Marshal(o)
	// Convert the json to a map
	asMap := make(map[string]interface{})
	json.Unmarshal(js, &asMap)

	var opts string

	for k, v := range asMap {
		switch k {
		// Boolean case, add the option if the value is true
		case "--alert-once", "--ongoing", "--sound":
			if v.(bool) == true {
				opts += " " + k
			}
		case "buttons":
			asSlice := v.([]interface{})

			// Max of 3 buttons
			if len(asSlice) > 3 {
				asSlice = asSlice[:2]
			}

			for i, btn := range asSlice {
				asMap := btn.(map[string]interface{})

				text, exists := asMap["Text"]
				action, existsA := asMap["Action"]
				if !exists || !existsA {
					log.Println("Button without text or without action is not allowed")
					break
				}

				opts += fmt.Sprintf(" button%d %s button%d-action %s", i+1, text, i+1, action)
			}
		case "led":
			for mk, mv := range v.(map[string]interface{}) {
				if mk == "--led-color" {
					opts += fmt.Sprintf(" %s %s", mk, mv)
				} else {
					// Turn the time.Duration nanoseconds into milliseconds
					opts += fmt.Sprintf(" %s %d", mk, uint(mv.(float64)/1000000))
				}
			}
		case "vibratePattern":
			opts += " --vibrate "
			asSlice := v.([]interface{})
			opts += fmt.Sprintf("%d", uint(asSlice[0].(float64)/1000000))
			for _, t := range asSlice[1:] {
				opts += fmt.Sprintf(",%d", uint(t.(float64)/1000000))
			}
			opts += " "
		default:
			opts += fmt.Sprintf(" %s %v", k, v)
		}
	}

	return opts
}

// NotificationButton shown in a notification
type NotificationButton struct {
	// Command to execute on clicking the button
	Action NotificationAction
	// Button text
	Text string
}

// Notificate sends a notification
func Notificate(title string, content string, opts *NotificateOptions) error {
	if opts == nil {
		opts = &NotificateOptions{
			Title:   title,
			Content: content,
		}
	}

	if errs := opts.validate(); len(errs) != 0 {
		var errMsg string
		for _, err := range errs {
			errMsg += fmt.Sprintf("%s\n", err)
		}
		return errors.New(errMsg)
	}

	cmd := exec.Command("termux-notification", opts.String())
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
