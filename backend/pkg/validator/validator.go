package validator

import (
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"unicode"
)

type ValidationErrors map[string]string

func (v ValidationErrors) HasErrors() bool {
	return len(v) > 0
}

func (v ValidationErrors) Add(field, message string) {
	v[field] = message
}

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
var slugRegex = regexp.MustCompile(`^[a-z0-9-]+$`)

func ValidateRegister(email, username, displayName, password string) ValidationErrors {
	errs := make(ValidationErrors)

	// Email
	email = strings.TrimSpace(email)
	if email == "" {
		errs.Add("email", "Email is required")
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs.Add("email", "Invalid email address")
	}

	// Username
	username = strings.TrimSpace(username)
	if username == "" {
		errs.Add("username", "Username is required")
	} else if len(username) < 3 {
		errs.Add("username", "Username must be at least 3 characters")
	} else if len(username) > 50 {
		errs.Add("username", "Username is too long")
	} else if !usernameRegex.MatchString(username) {
		errs.Add("username", "Username can only contain letters, numbers, _ and -")
	}

	// Display name
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		errs.Add("display_name", "Display name is required")
	} else if len(displayName) < 2 {
		errs.Add("display_name", "Display name must be at least 2 characters")
	} else if len(displayName) > 100 {
		errs.Add("display_name", "Display name is too long")
	}

	// Password
	validatePassword(password, errs)

	return errs
}

func ValidateLogin(email, password string) ValidationErrors {
	errs := make(ValidationErrors)

	email = strings.TrimSpace(email)
	if email == "" {
		errs.Add("email", "Email is required")
	} else if _, err := mail.ParseAddress(email); err != nil {
		errs.Add("email", "Invalid email address")
	}

	if password == "" {
		errs.Add("password", "Password is required")
	}

	return errs
}

func ValidateWorkspace(name, slug string) ValidationErrors {
	errs := make(ValidationErrors)

	name = strings.TrimSpace(name)
	if name == "" {
		errs.Add("name", "Workspace name is required")
	} else if len(name) < 2 {
		errs.Add("name", "Workspace name must be at least 2 characters")
	} else if len(name) > 100 {
		errs.Add("name", "Workspace name is too long")
	}

	slug = strings.TrimSpace(slug)
	if slug != "" {
		if len(slug) < 2 {
			errs.Add("slug", "Slug must be at least 2 characters")
		} else if len(slug) > 100 {
			errs.Add("slug", "Slug is too long")
		} else if !slugRegex.MatchString(slug) {
			errs.Add("slug", "Slug can only contain lowercase letters, numbers and dashes")
		}
	}

	return errs
}

func ValidateChannel(name, chType string) ValidationErrors {
	errs := make(ValidationErrors)

	name = strings.TrimSpace(name)
	if name == "" {
		errs.Add("name", "Channel name is required")
	} else if len(name) < 2 {
		errs.Add("name", "Channel name must be at least 2 characters")
	} else if len(name) > 100 {
		errs.Add("name", "Channel name is too long")
	}

	if chType != "" && chType != "public" && chType != "private" && chType != "dm" {
		errs.Add("type", "Channel type must be public, private, or dm")
	}

	return errs
}

func validatePassword(password string, errs ValidationErrors) {
	if len(password) < 8 {
		errs.Add("password", "Password must be at least 8 characters")
		return
	}

	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	missing := []string{}
	if !hasUpper {
		missing = append(missing, "one uppercase letter")
	}
	if !hasLower {
		missing = append(missing, "one lowercase letter")
	}
	if !hasDigit {
		missing = append(missing, "one number")
	}

	if len(missing) > 0 {
		errs.Add("password", fmt.Sprintf("Password must contain at least %s", strings.Join(missing, ", ")))
	}
}
