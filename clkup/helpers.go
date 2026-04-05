package clkup

import (
	"fmt"
	"regexp"
)

func validateEmail(email string) error {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		return fmt.Errorf("invalid email address format: %s", email)
	}
	return nil
}

func validatePhone(phone string) error {
	// "+1 123 456 7890"
	re := regexp.MustCompile(`^\+\d{1,3}\s?\d{4,14}$`)
	if !re.MatchString(phone) {
		return fmt.Errorf("invalid phone format, must include country code (e.g. +1 1234567890): %s", phone)
	}
	return nil
}

// Used for URL, DropDown, Text
func NewStringField(val string) SetCustomFieldPayload {
	return SetCustomFieldPayload{Value: val}
}

func NewEmailField(email string) (SetCustomFieldPayload, error) {
	if err := validateEmail(email); err != nil {
		return SetCustomFieldPayload{}, err
	}
	return SetCustomFieldPayload{Value: email}, nil
}

func NewPhoneField(phone string) (SetCustomFieldPayload, error) {
	if err := validatePhone(phone); err != nil {
		return SetCustomFieldPayload{}, err
	}
	return SetCustomFieldPayload{Value: phone}, nil
}

func NewNumberField(val interface{}) SetCustomFieldPayload {
	return SetCustomFieldPayload{Value: val} // Pass int, float64, etc.
}

func NewDateField(timestamp int64, hasTime bool) SetCustomFieldPayload {
	return SetCustomFieldPayload{
		Value:        timestamp,
		ValueOptions: map[string]bool{"time": hasTime},
	}
}

// used for both Tasks and Users
type AddRemPayload struct {
	Add []string `json:"add,omitempty"`
	Rem []string `json:"rem,omitempty"`
}

func NewAddRemField(add, rem []string) SetCustomFieldPayload {
	return SetCustomFieldPayload{Value: AddRemPayload{Add: add, Rem: rem}}
}

func NewManualProgressField(current int) SetCustomFieldPayload {
	return SetCustomFieldPayload{Value: map[string]int{"current": current}}
}

// need to add geocoding and reverse geocoding API for location CFs
