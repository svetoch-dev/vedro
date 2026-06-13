package validation

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
