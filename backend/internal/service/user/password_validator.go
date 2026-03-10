package user

import (
	"context"
	"fmt"
	"strings"
	"unicode"

	"github.com/antigravity/prometheus/internal/repository"
)

type PasswordValidator struct {
	policyRepo repository.PasswordPolicyRepository
}

func NewPasswordValidator(policyRepo repository.PasswordPolicyRepository) *PasswordValidator {
	return &PasswordValidator{policyRepo: policyRepo}
}

// Validate checks the password against the current policy.
// Returns a list of human-readable violation messages (German).
func (v *PasswordValidator) Validate(ctx context.Context, password string, username string) []string {
	policy, err := v.policyRepo.Get(ctx)
	if err != nil {
		// If policy can't be loaded, only enforce a sane minimum
		if len(password) < 4 {
			return []string{"Passwort muss mindestens 4 Zeichen lang sein"}
		}
		return nil
	}

	var violations []string

	if len(password) < policy.MinLength {
		violations = append(violations, fmt.Sprintf("Passwort muss mindestens %d Zeichen lang sein", policy.MinLength))
	}

	if policy.MaxLength > 0 && len(password) > policy.MaxLength {
		violations = append(violations, fmt.Sprintf("Passwort darf maximal %d Zeichen lang sein", policy.MaxLength))
	}

	if policy.RequireUppercase {
		hasUpper := false
		for _, r := range password {
			if unicode.IsUpper(r) {
				hasUpper = true
				break
			}
		}
		if !hasUpper {
			violations = append(violations, "Passwort muss mindestens einen Grossbuchstaben enthalten")
		}
	}

	if policy.RequireLowercase {
		hasLower := false
		for _, r := range password {
			if unicode.IsLower(r) {
				hasLower = true
				break
			}
		}
		if !hasLower {
			violations = append(violations, "Passwort muss mindestens einen Kleinbuchstaben enthalten")
		}
	}

	if policy.RequireDigit {
		hasDigit := false
		for _, r := range password {
			if unicode.IsDigit(r) {
				hasDigit = true
				break
			}
		}
		if !hasDigit {
			violations = append(violations, "Passwort muss mindestens eine Ziffer enthalten")
		}
	}

	if policy.RequireSpecial {
		hasSpecial := false
		for _, r := range password {
			if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
				hasSpecial = true
				break
			}
		}
		if !hasSpecial {
			violations = append(violations, "Passwort muss mindestens ein Sonderzeichen enthalten")
		}
	}

	if policy.DisallowUsername && username != "" {
		if strings.Contains(strings.ToLower(password), strings.ToLower(username)) {
			violations = append(violations, "Passwort darf nicht den Benutzernamen enthalten")
		}
	}

	return violations
}
