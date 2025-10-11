package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/oschwald/geoip2-golang"
)

type IPLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
	Timezone    string  `json:"timezone,omitempty"`
}

var (
	geoipReader *geoip2.Reader
	geoipOnce   sync.Once
	useGeoIP    bool
)

// InitGeoIP initializes the GeoIP2 database reader
// Download GeoLite2-City.mmdb from https://dev.maxmind.com/geoip/geolite2-free-geolocation-data
func InitGeoIP(dbPath string) error {
	var err error
	geoipOnce.Do(func() {
		if _, statErr := os.Stat(dbPath); statErr == nil {
			geoipReader, err = geoip2.Open(dbPath)
			if err == nil {
				useGeoIP = true
				log.Println("GeoIP2 database loaded successfully")
			} else {
				log.Printf("Failed to load GeoIP2 database: %v, falling back to API", err)
			}
		} else {
			log.Printf("GeoIP2 database not found at %s, using API fallback", dbPath)
		}
	})
	return err
}

// CloseGeoIP closes the GeoIP2 reader
func CloseGeoIP() {
	if geoipReader != nil {
		geoipReader.Close()
	}
}

// GetClientIP extracts the real client IP from the request
// Handles various proxy headers properly
func GetClientIP(r *http.Request) string {
	// Check CF-Connecting-IP (Cloudflare)
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	// Check X-Real-IP
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Check X-Forwarded-For (most common)
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Take the first IP in the chain (client IP)
		ips := strings.Split(forwarded, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check True-Client-IP (Akamai)
	if trueIP := r.Header.Get("True-Client-IP"); trueIP != "" {
		return trueIP
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// GetIPLocation fetches location data for an IP address
// Uses MaxMind GeoIP2 database if available, falls back to ip-api.com
func GetIPLocation(ip string) (*IPLocation, error) {
	// Skip for localhost/private IPs
	if isPrivateIP(ip) {
		return &IPLocation{
			Country:     "Local",
			CountryCode: "LOCAL",
			City:        "Local Network",
		}, nil
	}

	// Try GeoIP2 database first (faster, more accurate)
	if useGeoIP && geoipReader != nil {
		return getLocationFromGeoIP(ip)
	}

	// Fall back to ip-api.com (free, no key required, 45 req/min limit)
	return getLocationFromAPI(ip)
}

// getLocationFromGeoIP uses MaxMind GeoIP2 database
func getLocationFromGeoIP(ip string) (*IPLocation, error) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return nil, fmt.Errorf("invalid IP address: %s", ip)
	}

	record, err := geoipReader.City(ipAddr)
	if err != nil {
		return nil, err
	}

	location := &IPLocation{
		Country:     record.Country.Names["en"],
		CountryCode: record.Country.IsoCode,
		City:        record.City.Names["en"],
		Latitude:    record.Location.Latitude,
		Longitude:   record.Location.Longitude,
		Timezone:    record.Location.TimeZone,
	}

	return location, nil
}

// getLocationFromAPI uses ip-api.com as fallback
func getLocationFromAPI(ip string) (*IPLocation, error) {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	url := fmt.Sprintf("http://ip-api.com/json/%s?fields=status,country,countryCode,city,lat,lon,timezone", ip)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		City        string  `json:"city"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Status != "success" {
		return nil, fmt.Errorf("API request failed")
	}

	return &IPLocation{
		Country:     apiResp.Country,
		CountryCode: apiResp.CountryCode,
		City:        apiResp.City,
		Latitude:    apiResp.Lat,
		Longitude:   apiResp.Lon,
		Timezone:    apiResp.Timezone,
	}, nil
}

// isPrivateIP checks if an IP is private/local
func isPrivateIP(ip string) bool {
	if ip == "::1" || ip == "127.0.0.1" || ip == "localhost" {
		return true
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	// Check for private IP ranges
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"169.254.0.0/16", // Link-local
		"fc00::/7",       // IPv6 private
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network != nil && network.Contains(ipAddr) {
			return true
		}
	}

	return false
}

// GetLocationString returns a human-readable location string
func (l *IPLocation) GetLocationString() string {
	if l.City != "" && l.Country != "" {
		return fmt.Sprintf("%s, %s", l.City, l.Country)
	} else if l.Country != "" {
		return l.Country
	}
	return "Unknown"
}
