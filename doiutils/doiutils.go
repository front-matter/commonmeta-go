// Package doiutils provides a set of functions to work with DOIs
package doiutils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// PrefixFromUrl extracts DOI prefix from URL
func PrefixFromUrl(str string) (string, error) {
	u, err := url.Parse(str)
	if err != nil {
		return "", err
	}
	if u.Host == "" || u.Host != "doi.org" || !strings.HasPrefix(u.Path, "/10.") {
		return "", nil
	}
	path := strings.Split(u.Path, "/")
	return path[1], nil
}

// NormalizeDOI normalizes a DOI
func NormalizeDOI(doi string) string {
	doistr, ok := ValidateDOI(doi)
	if !ok {
		return ""
	}
	resolver := DOIResolver(doi, false)
	return resolver + strings.ToLower(doistr)
}

// ValidateDOI validates a DOI
func ValidateDOI(doi string) (string, bool) {
	r, err := regexp.Compile(`^(?:(http|https):/(/)?(dx\.)?(doi\.org|handle\.stage\.datacite\.org|handle\.test\.datacite\.org)/)?(doi:)?(10\.\d{4,5}/.+)$`)
	if err != nil {
		log.Printf("Error compiling regex: %v", err)
		return "", false
	}
	matched := r.FindStringSubmatch(doi)
	if len(matched) == 0 {
		return "", false
	}
	return matched[6], true
}

// ValidatePrefix validates a DOI prefix for a given DOI
func ValidatePrefix(doi string) (string, bool) {
	r, err := regexp.Compile(`^(?:(http|https):/(/)?(dx\.)?(doi\.org|handle\.stage\.datacite\.org|handle\.test\.datacite\.org)/)?(doi:)?(10\.\d{4,5})`)
	if err != nil {
		log.Printf("Error compiling regex: %v", err)
		return "", false
	}
	matched := r.FindStringSubmatch(doi)
	if len(matched) == 0 {
		return "", false
	}
	return matched[6], true
}

// DOIResolver returns a DOI resolver for a given DOI
func DOIResolver(doi string, sandbox bool) string {
	d, err := url.Parse(doi)
	if err != nil {
		return ""
	}
	if d.Host == "stage.datacite.org" || sandbox {
		return "https://handle.stage.datacite.org/"
	}
	return "https://doi.org/"
}

// GetDOIRA returns the DOI registration agency for a given DOI or prefix
func GetDOIRA(doi string) (string, bool) {
	prefix, ok := ValidatePrefix(doi)
	if !ok {
		return "", false
	}
	type Response []struct {
		DOI string `json:"DOI"`
		RA  string `json:"RA"`
	}
	var result Response
	resp, err := http.Get(fmt.Sprintf("https://doi.org/ra/%s", prefix))
	if err != nil {
		return "", false
	}
	if resp.StatusCode == 404 {
		return "", false
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", false
	}
	return result[0].RA, true
}

// IsRogueScholarDOI checks if a DOI is from Rogue Scholar
func IsRogueScholarDOI(doi string) bool {
	var rogueScholarPrefixes = []string{
		"10.34732",
		"10.53731",
		"10.54900",
		"10.57689",
		"10.59348",
		"10.59349",
		"10.59350",
	}
	prefix, ok := ValidatePrefix(doi)
	if !ok {
		return false
	}
	return slices.Contains(rogueScholarPrefixes, prefix)
}
