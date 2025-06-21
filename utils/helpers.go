package utils

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

// EMSGAddress represents a parsed EMSG address
type EMSGAddress struct {
	User   string
	Domain string
	Raw    string
}

// ParseEMSGAddress parses an EMSG address in the format user#domain.com
func ParseEMSGAddress(address string) (*EMSGAddress, error) {
	if address == "" {
		return nil, fmt.Errorf("address cannot be empty")
	}

	// Check for the # separator
	parts := strings.Split(address, "#")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid EMSG address format: expected user#domain.com, got %s", address)
	}

	user := strings.TrimSpace(parts[0])
	domain := strings.TrimSpace(parts[1])

	// Validate user part
	if user == "" {
		return nil, fmt.Errorf("user part cannot be empty")
	}

	// Validate user format (alphanumeric, dots, hyphens, underscores)
	userRegex := regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
	if !userRegex.MatchString(user) {
		return nil, fmt.Errorf("invalid user format: %s", user)
	}

	// Validate domain part
	if domain == "" {
		return nil, fmt.Errorf("domain part cannot be empty")
	}

	if !IsValidDomain(domain) {
		return nil, fmt.Errorf("invalid domain format: %s", domain)
	}

	return &EMSGAddress{
		User:   user,
		Domain: domain,
		Raw:    address,
	}, nil
}

// String returns the string representation of the EMSG address
func (addr *EMSGAddress) String() string {
	return addr.Raw
}

// GetEMSGDNSName returns the DNS name for EMSG TXT record lookup
func (addr *EMSGAddress) GetEMSGDNSName() string {
	return fmt.Sprintf("_emsg.%s", addr.Domain)
}

// IsValidDomain validates a domain name
func IsValidDomain(domain string) bool {
	if domain == "" {
		return false
	}

	// Basic domain validation
	domainRegex := regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	if !domainRegex.MatchString(domain) {
		return false
	}

	// Check if it's a valid hostname
	if net.ParseIP(domain) != nil {
		return false // IP addresses are not valid domain names for EMSG
	}

	// Domain must contain at least one dot
	if !strings.Contains(domain, ".") {
		return false
	}

	// Check length constraints
	if len(domain) > 253 {
		return false
	}

	// Check each label
	labels := strings.Split(domain, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}

// IsValidEMSGAddress validates an EMSG address format
func IsValidEMSGAddress(address string) bool {
	_, err := ParseEMSGAddress(address)
	return err == nil
}

// NormalizeEMSGAddress normalizes an EMSG address by trimming whitespace and converting to lowercase
func NormalizeEMSGAddress(address string) string {
	address = strings.TrimSpace(address)
	
	// Split and normalize parts
	parts := strings.Split(address, "#")
	if len(parts) != 2 {
		return address // Return as-is if invalid format
	}

	user := strings.TrimSpace(parts[0])
	domain := strings.ToLower(strings.TrimSpace(parts[1]))

	return fmt.Sprintf("%s#%s", user, domain)
}

// ExtractDomainFromEMSGAddress extracts just the domain part from an EMSG address
func ExtractDomainFromEMSGAddress(address string) (string, error) {
	addr, err := ParseEMSGAddress(address)
	if err != nil {
		return "", err
	}
	return addr.Domain, nil
}

// ExtractUserFromEMSGAddress extracts just the user part from an EMSG address
func ExtractUserFromEMSGAddress(address string) (string, error) {
	addr, err := ParseEMSGAddress(address)
	if err != nil {
		return "", err
	}
	return addr.User, nil
}

// ValidateEMSGAddressList validates a list of EMSG addresses
func ValidateEMSGAddressList(addresses []string) error {
	for i, addr := range addresses {
		if !IsValidEMSGAddress(addr) {
			return fmt.Errorf("invalid EMSG address at index %d: %s", i, addr)
		}
	}
	return nil
}

// ParseEMSGAddressList parses a list of EMSG addresses
func ParseEMSGAddressList(addresses []string) ([]*EMSGAddress, error) {
	result := make([]*EMSGAddress, len(addresses))
	for i, addr := range addresses {
		parsed, err := ParseEMSGAddress(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse address at index %d: %w", i, err)
		}
		result[i] = parsed
	}
	return result, nil
}
