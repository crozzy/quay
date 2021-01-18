package shared

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// ValidateGitHubOAuth checks that the Bitbucker OAuth credentials are correct
func ValidateGitHubOAuth(clientID, clientSecret string) bool {

	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go

	req, err := http.NewRequest("GET", "https://api.github.com/", nil)
	if err != nil {
		return false
	}
	req.SetBasicAuth(clientID, clientSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		return true
	}

	return false

}

// ValidateRequiredObject validates that a object input is not nil
func ValidateRequiredObject(input interface{}, field, fgName string) (bool, ValidationError) {

	// Check string
	if input == nil || reflect.ValueOf(input).IsNil() {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " is required",
		}
		return false, newError
	}

	// Return okay
	return true, ValidationError{}

}

// ValidateRequiredString validates that a string input is not empty
func ValidateRequiredString(input, field, fgName string) (bool, ValidationError) {

	// Check string
	if input == "" {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " is required",
		}
		return false, newError
	}

	// Return okay
	return true, ValidationError{}

}

// ValidateAtLeastOneOfBool validates that at least one of the given options is true
func ValidateAtLeastOneOfBool(inputs []bool, fields []string, fgName string) (bool, ValidationError) {

	// At first, assume none are true
	atLeastOne := false

	// Iterate through options
	for _, val := range inputs {
		if val == true {
			atLeastOne = true
			break
		}
	}

	// If at least one isnt true, return error
	if !atLeastOne {
		newError := ValidationError{
			Tags:       fields,
			FieldGroup: fgName,
			Message:    "At least one of " + strings.Join(fields, ",") + " must be enabled",
		}
		return false, newError
	}

	return true, ValidationError{}

}

// ValidateAtLeastOneOfString validates that at least one of the given options is true
func ValidateAtLeastOneOfString(inputs []string, fields []string, fgName string) (bool, ValidationError) {

	// At first, assume none are true
	atLeastOne := false

	// Iterate through options
	for _, val := range inputs {
		if val != "" {
			atLeastOne = true
			break
		}
	}

	// If at least one isnt true, return error
	if !atLeastOne {
		newError := ValidationError{
			Tags:       fields,
			FieldGroup: fgName,
			Message:    "At least one of " + strings.Join(fields, ",") + " must be present",
		}
		return false, newError
	}

	return true, ValidationError{}

}

// ValidateRedisConnection validates that a Redis connection can successfully be established
func ValidateRedisConnection(options *redis.Options, field, fgName string) (bool, ValidationError) {

	// Start client
	rdb := redis.NewClient(options)

	// Ping client
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    "Could not connect to Redis with values provided in " + field + ". Error: " + err.Error(),
		}
		return false, newError
	}

	return true, ValidationError{}

}

// ValidateIsOneOfString validates that a string is one of a given option
func ValidateIsOneOfString(input string, options []string, field string, fgName string) (bool, ValidationError) {

	// At first, assume none are true
	isOneOf := false

	// Iterate through options
	for _, val := range options {
		if input == val {
			isOneOf = true
			break
		}
	}

	// If at least one isnt true, return error
	if !isOneOf {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " must be one of " + strings.Join(options, ",") + ".",
		}
		return false, newError
	}

	return true, ValidationError{}
}

// ValidateIsURL tests a string to determine if it is a well-structured url or not.
func ValidateIsURL(input string, field string, fgName string) (bool, ValidationError) {

	_, err := url.ParseRequestURI(input)
	if err != nil {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " must be of type URL",
		}
		return false, newError
	}

	u, err := url.Parse(input)
	if err != nil || u.Scheme == "" || u.Host == "" {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " must be of type URL",
		}
		return false, newError
	}

	return true, ValidationError{}
}

// ValidateIsHostname tests a string to determine if it is a well-structured hostname or not.
func ValidateIsHostname(input string, field string, fgName string) (bool, ValidationError) {

	// trim whitespace
	input = strings.Trim(input, " ")

	// check against regex
	re, _ := regexp.Compile(`^[a-zA-Z-0-9\.]+(:[0-9]+)?$`)
	if !re.MatchString(input) {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " must be of type Hostname",
		}
		return false, newError
	}

	return true, ValidationError{}
}

// ValidateHostIsReachable will check if a get request returns a 200 status code
func ValidateHostIsReachable(opts Options, input string, field string, fgName string) (bool, ValidationError) {

	// Get protocol
	u, _ := url.Parse(input)
	scheme := u.Scheme

	// Get raw hostname without protocol
	url := strings.TrimPrefix(input, "https://")
	url = strings.TrimPrefix(url, "http://")

	// Set timeout
	timeout := 3 * time.Second

	// Switch on protocol
	switch scheme {
	case "http":

		_, err := net.DialTimeout("tcp", url, timeout)
		if err != nil {
			newError := ValidationError{
				Tags:       []string{field},
				FieldGroup: fgName,
				Message:    err.Error(),
			}
			return false, newError
		}

	case "https":

		config, err := GetTlsConfig(opts)
		if err != nil {
			newError := ValidationError{
				Tags:       []string{field},
				FieldGroup: fgName,
				Message:    err.Error(),
			}
			return false, newError
		}
		dialer := &net.Dialer{
			Timeout: timeout,
		}

		_, err = tls.DialWithDialer(dialer, "tcp", url, config)
		if err != nil {
			newError := ValidationError{
				Tags:       []string{field},
				FieldGroup: fgName,
				Message:    "Cannot reach " + input + ". Error: " + err.Error(),
			}
			return false, newError
		}

	}

	return true, ValidationError{}

}

// ValidateFileExists will check if a path exists on the current machine
func ValidateFileExists(input string, field string, fgName string) (bool, ValidationError) {

	// Check path
	if _, err := os.Stat(input); os.IsNotExist(err) {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    "Cannot access the file " + input,
		}
		return false, newError
	}

	return true, ValidationError{}

}

// ValidateTimePattern validates that a string has the pattern ^[0-9]+(w|m|d|h|s)$
func ValidateTimePattern(input string, field string, fgName string) (bool, ValidationError) {

	re := regexp.MustCompile(`^[0-9]+(w|m|d|h|s)$`)
	matches := re.FindAllString(input, -1)

	// If the pattern is not matched
	if len(matches) != 1 {
		newError := ValidationError{
			Tags:       []string{field},
			FieldGroup: fgName,
			Message:    field + " must have the regex pattern ^[0-9]+(w|m|d|h|s)$",
		}
		return false, newError
	}

	return true, ValidationError{}
}

// ValidateCertsPresent validates that all required certificates are present in the options struct
func ValidateCertsPresent(opts Options, requiredCertNames []string, fgName string) (bool, ValidationError) {

	// If no certificates are passed
	if opts.Certificates == nil {
		newError := ValidationError{
			Tags:       []string{"Certificates"},
			FieldGroup: fgName,
			Message:    "Certificates are required for SSL but are not present",
		}
		return false, newError
	}

	// Check that all required certificates are present
	for _, certName := range requiredCertNames {

		// Check that cert has been included
		if _, ok := opts.Certificates[certName]; !ok {
			newError := ValidationError{
				Tags:       []string{"Certificates"},
				FieldGroup: fgName,
				Message:    "Certificate " + certName + " is required for " + fgName + " .",
			}
			return false, newError
		}
	}

	return true, ValidationError{}

}

// ValidateCertPairWithHostname will validate that a public private key pair are valid and have the correct hostname
func ValidateCertPairWithHostname(cert, key []byte, hostname string, fgName string) (bool, ValidationError) {

	// Load key pair, this will check the public, private keys are paired
	certChain, err := tls.X509KeyPair(cert, key)
	if err != nil {
		newError := ValidationError{
			Tags:       []string{"Certificates"},
			FieldGroup: fgName,
			Message:    err.Error(),
		}
		return false, newError
	}

	certificate, err := x509.ParseCertificate(certChain.Certificate[0])

	// Make sure port is removed
	cleanHost, _, err := net.SplitHostPort(hostname)
	if err != nil {
		cleanHost = hostname
	}

	err = certificate.VerifyHostname(cleanHost)
	if err != nil {
		newError := ValidationError{
			Tags:       []string{"Certificates"},
			FieldGroup: fgName,
			Message:    err.Error(),
		}
		return false, newError
	}

	return true, ValidationError{}

}
