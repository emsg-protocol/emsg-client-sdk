package dns

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// EMSGServerInfo represents the information about an EMSG server
type EMSGServerInfo struct {
	URL       string `json:"url"`
	PublicKey string `json:"pubkey,omitempty"`
	Version   string `json:"version,omitempty"`
}

// ResolverConfig holds configuration for DNS resolution
type ResolverConfig struct {
	Timeout time.Duration
	Retries int
}

// DefaultResolverConfig returns a default resolver configuration
func DefaultResolverConfig() *ResolverConfig {
	return &ResolverConfig{
		Timeout: 10 * time.Second,
		Retries: 3,
	}
}

// Resolver handles EMSG DNS resolution
type Resolver struct {
	config *ResolverConfig
}

// NewResolver creates a new DNS resolver with the given configuration
func NewResolver(config *ResolverConfig) *Resolver {
	if config == nil {
		config = DefaultResolverConfig()
	}
	return &Resolver{config: config}
}

// ResolveDomain resolves an EMSG domain to server information
func (r *Resolver) ResolveDomain(domain string) (*EMSGServerInfo, error) {
	if domain == "" {
		return nil, fmt.Errorf("domain cannot be empty")
	}

	// Construct the EMSG DNS name
	dnsName := fmt.Sprintf("_emsg.%s", domain)

	// Perform TXT record lookup
	txtRecords, err := r.lookupTXT(dnsName)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup TXT records for %s: %w", dnsName, err)
	}

	if len(txtRecords) == 0 {
		return nil, fmt.Errorf("no TXT records found for %s", dnsName)
	}

	// Try to parse each TXT record
	for _, record := range txtRecords {
		serverInfo, err := r.parseTXTRecord(record)
		if err != nil {
			continue // Try next record
		}
		return serverInfo, nil
	}

	return nil, fmt.Errorf("no valid EMSG server information found in TXT records for %s", dnsName)
}

// lookupTXT performs a TXT record lookup with retries
func (r *Resolver) lookupTXT(name string) ([]string, error) {
	var lastErr error
	
	for i := 0; i < r.config.Retries; i++ {
		txtRecords, err := net.LookupTXT(name)
		if err == nil {
			return txtRecords, nil
		}
		lastErr = err
		
		if i < r.config.Retries-1 {
			time.Sleep(time.Duration(i+1) * time.Second)
		}
	}
	
	return nil, lastErr
}

// parseTXTRecord parses a TXT record to extract EMSG server information
func (r *Resolver) parseTXTRecord(record string) (*EMSGServerInfo, error) {
	record = strings.TrimSpace(record)
	
	// Try JSON format first
	if strings.HasPrefix(record, "{") && strings.HasSuffix(record, "}") {
		return r.parseJSONRecord(record)
	}
	
	// Try URL format
	if strings.HasPrefix(record, "http://") || strings.HasPrefix(record, "https://") {
		return r.parseURLRecord(record)
	}
	
	// Try key-value format (e.g., "url=https://example.com pubkey=abc123")
	return r.parseKeyValueRecord(record)
}

// parseJSONRecord parses a JSON-formatted TXT record
func (r *Resolver) parseJSONRecord(record string) (*EMSGServerInfo, error) {
	var serverInfo EMSGServerInfo
	err := json.Unmarshal([]byte(record), &serverInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON record: %w", err)
	}
	
	if serverInfo.URL == "" {
		return nil, fmt.Errorf("missing URL in JSON record")
	}
	
	// Validate URL
	if err := r.validateURL(serverInfo.URL); err != nil {
		return nil, fmt.Errorf("invalid URL in JSON record: %w", err)
	}
	
	return &serverInfo, nil
}

// parseURLRecord parses a simple URL-only TXT record
func (r *Resolver) parseURLRecord(record string) (*EMSGServerInfo, error) {
	if err := r.validateURL(record); err != nil {
		return nil, fmt.Errorf("invalid URL record: %w", err)
	}
	
	return &EMSGServerInfo{
		URL: record,
	}, nil
}

// parseKeyValueRecord parses a key-value formatted TXT record
func (r *Resolver) parseKeyValueRecord(record string) (*EMSGServerInfo, error) {
	serverInfo := &EMSGServerInfo{}
	
	// Split by spaces and parse key=value pairs
	parts := strings.Fields(record)
	for _, part := range parts {
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		
		key := strings.ToLower(strings.TrimSpace(kv[0]))
		value := strings.TrimSpace(kv[1])
		
		switch key {
		case "url":
			serverInfo.URL = value
		case "pubkey", "publickey":
			serverInfo.PublicKey = value
		case "version":
			serverInfo.Version = value
		}
	}
	
	if serverInfo.URL == "" {
		return nil, fmt.Errorf("missing URL in key-value record")
	}
	
	// Validate URL
	if err := r.validateURL(serverInfo.URL); err != nil {
		return nil, fmt.Errorf("invalid URL in key-value record: %w", err)
	}
	
	return serverInfo, nil
}

// validateURL validates that a URL is properly formatted and uses HTTP/HTTPS
func (r *Resolver) validateURL(urlStr string) error {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use HTTP or HTTPS scheme, got: %s", parsedURL.Scheme)
	}
	
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must have a host")
	}
	
	return nil
}

// ResolveEMSGAddress resolves an EMSG address to server information
func (r *Resolver) ResolveEMSGAddress(address string) (*EMSGServerInfo, error) {
	// Extract domain from address
	parts := strings.Split(address, "#")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid EMSG address format: %s", address)
	}
	
	domain := parts[1]
	return r.ResolveDomain(domain)
}

// CacheEntry represents a cached DNS resolution result
type CacheEntry struct {
	ServerInfo *EMSGServerInfo
	Timestamp  time.Time
	TTL        time.Duration
}

// CachedResolver wraps a resolver with caching capabilities
type CachedResolver struct {
	resolver *Resolver
	cache    map[string]*CacheEntry
	defaultTTL time.Duration
}

// NewCachedResolver creates a new cached resolver
func NewCachedResolver(config *ResolverConfig, ttl time.Duration) *CachedResolver {
	if ttl == 0 {
		ttl = 5 * time.Minute // Default TTL
	}
	
	return &CachedResolver{
		resolver:   NewResolver(config),
		cache:      make(map[string]*CacheEntry),
		defaultTTL: ttl,
	}
}

// ResolveDomain resolves a domain with caching
func (cr *CachedResolver) ResolveDomain(domain string) (*EMSGServerInfo, error) {
	// Check cache first
	if entry, exists := cr.cache[domain]; exists {
		if time.Since(entry.Timestamp) < entry.TTL {
			return entry.ServerInfo, nil
		}
		// Cache expired, remove entry
		delete(cr.cache, domain)
	}
	
	// Resolve from DNS
	serverInfo, err := cr.resolver.ResolveDomain(domain)
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	cr.cache[domain] = &CacheEntry{
		ServerInfo: serverInfo,
		Timestamp:  time.Now(),
		TTL:        cr.defaultTTL,
	}
	
	return serverInfo, nil
}
