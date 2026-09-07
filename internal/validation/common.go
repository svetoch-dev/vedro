package validation

import "regexp"

var EmailPattern = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

type ValidationResult struct {
	Valid   bool
	Message string
}

func Valid() ValidationResult {
	return ValidationResult{Valid: true}
}

func Invalid(message string) ValidationResult {
	return ValidationResult{
		Valid:   false,
		Message: message,
	}
}

func ValidateLocation(location string, fn func(location string) *ValidationResult) ValidationResult {
	if location == "" {
		return Invalid("location is an empty string")
	}

	// per provider validation
	if fn != nil {
		v := fn(location)

		if v != nil {
			return *v
		}
	}

	if !regionalPattern.MatchString(location) {
		return Invalid("unsupported bucket location")
	}

	return Valid()
}

func ValidateNameImmutability(
	specName string,
	externalName string,
	objectName string,
) ValidationResult {
	if specName != "" &&
		externalName != "" &&
		externalName != specName {
		return Invalid("name cannot be changed after creation")
	}

	if specName == "" &&
		externalName != "" &&
		externalName != objectName {
		return Invalid(
			"metadata.name cannot be used as the name source if spec.name was used and CR is created",
		)
	}

	return Valid()
}
