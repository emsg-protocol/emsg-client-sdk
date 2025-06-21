package test

import (
	"testing"

	"github.com/emsg-protocol/emsg-client-sdk/utils"
)

func TestParseEMSGAddress(t *testing.T) {
	testCases := []struct {
		input    string
		expected *utils.EMSGAddress
		hasError bool
	}{
		{
			input: "alice#example.com",
			expected: &utils.EMSGAddress{
				User:   "alice",
				Domain: "example.com",
				Raw:    "alice#example.com",
			},
			hasError: false,
		},
		{
			input: "bob.smith#test.org",
			expected: &utils.EMSGAddress{
				User:   "bob.smith",
				Domain: "test.org",
				Raw:    "bob.smith#test.org",
			},
			hasError: false,
		},
		{
			input: "user_123#sub.domain.co.uk",
			expected: &utils.EMSGAddress{
				User:   "user_123",
				Domain: "sub.domain.co.uk",
				Raw:    "user_123#sub.domain.co.uk",
			},
			hasError: false,
		},
		{
			input:    "",
			expected: nil,
			hasError: true,
		},
		{
			input:    "alice@example.com",
			expected: nil,
			hasError: true,
		},
		{
			input:    "alice#",
			expected: nil,
			hasError: true,
		},
		{
			input:    "#example.com",
			expected: nil,
			hasError: true,
		},
		{
			input:    "alice#invalid_domain",
			expected: nil,
			hasError: true,
		},
		{
			input:    "alice#192.168.1.1",
			expected: nil,
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result, err := utils.ParseEMSGAddress(tc.input)

			if tc.hasError {
				if err == nil {
					t.Errorf("Expected error for input %s, but got none", tc.input)
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error for input %s: %v", tc.input, err)
				return
			}

			if result.User != tc.expected.User {
				t.Errorf("Expected user %s, got %s", tc.expected.User, result.User)
			}

			if result.Domain != tc.expected.Domain {
				t.Errorf("Expected domain %s, got %s", tc.expected.Domain, result.Domain)
			}

			if result.Raw != tc.expected.Raw {
				t.Errorf("Expected raw %s, got %s", tc.expected.Raw, result.Raw)
			}
		})
	}
}

func TestEMSGAddressGetEMSGDNSName(t *testing.T) {
	addr, err := utils.ParseEMSGAddress("alice#example.com")
	if err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}

	expected := "_emsg.example.com"
	actual := addr.GetEMSGDNSName()

	if actual != expected {
		t.Errorf("Expected DNS name %s, got %s", expected, actual)
	}
}

func TestEMSGAddressString(t *testing.T) {
	addr, err := utils.ParseEMSGAddress("alice#example.com")
	if err != nil {
		t.Fatalf("Failed to parse address: %v", err)
	}

	expected := "alice#example.com"
	actual := addr.String()

	if actual != expected {
		t.Errorf("Expected string %s, got %s", expected, actual)
	}
}

func TestIsValidDomain(t *testing.T) {
	validDomains := []string{
		"example.com",
		"sub.example.com",
		"test.org",
		"a.b.c.d.e.f",
		"xn--nxasmq6b.xn--j6w193g", // IDN domain
		"123.example.com",
		"example-test.com",
	}

	invalidDomains := []string{
		"",
		"localhost",
		"example",
		"192.168.1.1",
		"::1",
		"example..com",
		".example.com",
		"example.com.",
		"-example.com",
		"example-.com",
		"very-long-domain-name-that-exceeds-the-maximum-allowed-length-for-a-domain-name-which-is-253-characters-long-and-this-domain-name-is-definitely-longer-than-that-limit-so-it-should-be-considered-invalid-by-our-validation-function.com",
	}

	for _, domain := range validDomains {
		t.Run("valid_"+domain, func(t *testing.T) {
			if !utils.IsValidDomain(domain) {
				t.Errorf("Domain %s should be valid", domain)
			}
		})
	}

	for _, domain := range invalidDomains {
		t.Run("invalid_"+domain, func(t *testing.T) {
			if utils.IsValidDomain(domain) {
				t.Errorf("Domain %s should be invalid", domain)
			}
		})
	}
}

func TestIsValidEMSGAddress(t *testing.T) {
	validAddresses := []string{
		"alice#example.com",
		"bob.smith#test.org",
		"user_123#sub.domain.co.uk",
		"test-user#example-domain.com",
	}

	invalidAddresses := []string{
		"",
		"alice@example.com",
		"alice#",
		"#example.com",
		"alice#invalid_domain",
		"alice#192.168.1.1",
		"alice bob#example.com",
		"alice#example..com",
	}

	for _, addr := range validAddresses {
		t.Run("valid_"+addr, func(t *testing.T) {
			if !utils.IsValidEMSGAddress(addr) {
				t.Errorf("Address %s should be valid", addr)
			}
		})
	}

	for _, addr := range invalidAddresses {
		t.Run("invalid_"+addr, func(t *testing.T) {
			if utils.IsValidEMSGAddress(addr) {
				t.Errorf("Address %s should be invalid", addr)
			}
		})
	}
}

func TestNormalizeEMSGAddress(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "alice#example.com",
			expected: "alice#example.com",
		},
		{
			input:    "  alice#example.com  ",
			expected: "alice#example.com",
		},
		{
			input:    "alice#EXAMPLE.COM",
			expected: "alice#example.com",
		},
		{
			input:    "  alice  #  EXAMPLE.COM  ",
			expected: "alice#example.com",
		},
		{
			input:    "invalid@format",
			expected: "invalid@format", // Returns as-is for invalid format
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := utils.NormalizeEMSGAddress(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestExtractDomainFromEMSGAddress(t *testing.T) {
	domain, err := utils.ExtractDomainFromEMSGAddress("alice#example.com")
	if err != nil {
		t.Fatalf("Failed to extract domain: %v", err)
	}

	if domain != "example.com" {
		t.Errorf("Expected domain example.com, got %s", domain)
	}

	// Test invalid address
	_, err = utils.ExtractDomainFromEMSGAddress("invalid@format")
	if err == nil {
		t.Error("Expected error for invalid address format")
	}
}

func TestExtractUserFromEMSGAddress(t *testing.T) {
	user, err := utils.ExtractUserFromEMSGAddress("alice#example.com")
	if err != nil {
		t.Fatalf("Failed to extract user: %v", err)
	}

	if user != "alice" {
		t.Errorf("Expected user alice, got %s", user)
	}

	// Test invalid address
	_, err = utils.ExtractUserFromEMSGAddress("invalid@format")
	if err == nil {
		t.Error("Expected error for invalid address format")
	}
}

func TestValidateEMSGAddressList(t *testing.T) {
	validList := []string{
		"alice#example.com",
		"bob#test.org",
		"charlie#sub.domain.com",
	}

	err := utils.ValidateEMSGAddressList(validList)
	if err != nil {
		t.Errorf("Unexpected error for valid address list: %v", err)
	}

	invalidList := []string{
		"alice#example.com",
		"invalid@format",
		"charlie#sub.domain.com",
	}

	err = utils.ValidateEMSGAddressList(invalidList)
	if err == nil {
		t.Error("Expected error for invalid address list")
	}
}

func TestParseEMSGAddressList(t *testing.T) {
	addresses := []string{
		"alice#example.com",
		"bob#test.org",
		"charlie#sub.domain.com",
	}

	parsed, err := utils.ParseEMSGAddressList(addresses)
	if err != nil {
		t.Fatalf("Failed to parse address list: %v", err)
	}

	if len(parsed) != len(addresses) {
		t.Errorf("Expected %d parsed addresses, got %d", len(addresses), len(parsed))
	}

	for i, addr := range parsed {
		if addr.String() != addresses[i] {
			t.Errorf("Expected address %s, got %s", addresses[i], addr.String())
		}
	}

	// Test with invalid address
	invalidAddresses := []string{
		"alice#example.com",
		"invalid@format",
	}

	_, err = utils.ParseEMSGAddressList(invalidAddresses)
	if err == nil {
		t.Error("Expected error for invalid address in list")
	}
}
