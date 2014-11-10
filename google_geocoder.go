// Modified by Michael Nixon to add lots of error checking
// Before modification, this library panics on the slightest error.
// Also added support for API key.
package geo

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	//"hash"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

// This struct contains all the funcitonality
// of interacting with the Google Maps Geocoding Service
type GoogleGeocoder struct{}

// This struct contains selected fields from Google's Geocoding Service response
type googleGeocodeResponse struct {
	Error_message string
	Status        string
	Results       []struct {
		FormattedAddress string `json:"formatted_address"`
		Geometry         struct {
			Location struct {
				Lat float64
				Lng float64
			}
		}
	}
}

// This is the error that consumers receive when there
// are no results from the geocoding request.
var googleZeroResultsError = errors.New("ZERO_RESULTS")

// This contains the base URL for the Google Geocoder API.
var googleGeocodeURL = "https://maps.googleapis.com/maps/api/geocode/json"
var googleGeocodeURLbase = "/maps/api/geocode/json"

// Note:  In the next major revision (1.0.0), it is planned
//        That Geocoders should adhere to the `geo.Geocoder`
//        interface and provide versioning of APIs accordingly.
// Sets the base URL for the Google Geocoding API.
func SetGoogleGeocodeURL(newGeocodeURL string) {
	googleGeocodeURL = newGeocodeURL
}

// Issues a request to the google geocoding service and forwards the passed in params string
// as a URL-encoded entity.  Returns an array of byes as a result, or an error if one occurs during the process.
func (g *GoogleGeocoder) Request(params string) ([]byte, error) {
	client := &http.Client{}

	fullUrl := fmt.Sprintf("%s?%s", googleGeocodeURL, params)

	// TODO Potentially refactor out from MapQuestGeocoder as well
	req, err := http.NewRequest("GET", fullUrl, nil)
	if err != nil {
		return nil, err
	}

	resp, requestErr := client.Do(req)
	if requestErr != nil {
		return nil, requestErr
	}

	data, dataReadErr := ioutil.ReadAll(resp.Body)

	if dataReadErr != nil {
		return nil, dataReadErr
	}

	return data, nil
}

// Geocodes the passed in query string and returns a pointer to a new Point struct.
// Returns an error if the underlying request cannot complete.
func (g *GoogleGeocoder) Geocode(query string) (*Point, error) {
	url_safe_query := url.QueryEscape(query)
	data, err := g.Request(fmt.Sprintf("address=%s", url_safe_query))
	if err != nil {
		return nil, err
	}

	lat, lng, err := g.extractLatLngFromResponse(data)
	if err != nil {
		return nil, err
	}

	p := &Point{lat: lat, lng: lng}

	return p, nil
}

// Extracts the first lat and lng values from a Google Geocoder Response body.
func (g *GoogleGeocoder) extractLatLngFromResponse(data []byte) (float64, float64, error) {
	res := &googleGeocodeResponse{}
	err := json.Unmarshal(data, &res)
	if err != nil {
		return 0, 0, err
	}
	if len(res.Results) == 0 {
		return 0, 0, googleZeroResultsError
	}

	lat := res.Results[0].Geometry.Location.Lat
	lng := res.Results[0].Geometry.Location.Lng

	return lat, lng, nil
}

// Reverse geocodes the pointer to a Point struct and returns the first address that matches
// or returns an error if the underlying request cannot complete.
func (g *GoogleGeocoder) ReverseGeocode(p *Point, apikey string) (string, error) {
	var queryurl string
	var s string

	if apikey != "" {
		s = fmt.Sprintf("%f,%f", p.lat, p.lng)
		queryurl = "language=ja&latlng=" + s + "&key=" + url.QueryEscape(apikey)
	} else {
		queryurl = fmt.Sprintf("language=ja&latlng=%f,%f", p.lat, p.lng)
	}

	data, err := g.Request(queryurl)
	if err != nil {
		return "", err
	}

	resStr, err := g.extractAddressFromResponse(data)
	if err != nil {
		return "", err
	}

	return resStr, nil
}

// Reverse geocodes the pointer to a Point struct and returns the first address that matches
// or returns an error if the underlying request cannot complete.
func (g *GoogleGeocoder) ReverseGeocodePremier(p *Point, username string, key string) (string, error) {
	var queryurl string
	var s string

	s = fmt.Sprintf("%f,%f", p.lat, p.lng)
	queryurl = "language=ja&latlng=" + s + "&client=" + username

	// Calculate hash
	decodedkeyarray, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", err
	}
	s = googleGeocodeURLbase + "?" + queryurl
	hash := hmac.New(sha1.New, decodedkeyarray)
	hash.Write([]byte(s))
	signaturebinary := hash.Sum(nil)

	// base64.URLEncoding doesn't work, so I did a cheap workaround for now with
	// strings.Replace. Works fine, but I'll tidy this up later.

	signaturebase64 := base64.URLEncoding.EncodeToString(signaturebinary)
	signaturebase64 = strings.Replace(signaturebase64, "+", "-", -1)
	signaturebase64 = strings.Replace(signaturebase64, "/", "_", -1)
	signaturebase64 = strings.Replace(signaturebase64, "=", ",", -1)
	queryurl += "&signature=" + signaturebase64

	data, err := g.Request(queryurl)
	if err != nil {
		return "", err
	}

	resStr, err := g.extractAddressFromResponse(data)
	if err != nil {
		return "", err
	}

	return resStr, nil
}

// Geocodes the passed in query string and returns a pointer to a new Point struct.
// Returns an error if the underlying request cannot complete.
func (g *GoogleGeocoder) GeocodePremier(address string, username string, key string) (*Point, error) {
	if address == "" {
		return nil, errors.New("address is empty.")
	}

	var queryurl string
	var s string

	queryurl = fmt.Sprintf("language=ja&address=%s&client=%s", url.QueryEscape(address), username)

	// Calculate hash
	decodedkeyarray, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	s = googleGeocodeURLbase + "?" + queryurl
	hash := hmac.New(sha1.New, decodedkeyarray)
	hash.Write([]byte(s))
	signaturebinary := hash.Sum(nil)

	// base64.URLEncoding doesn't work, so I did a cheap workaround for now with
	// strings.Replace. Works fine, but I'll tidy this up later.
	signaturebase64 := base64.URLEncoding.EncodeToString(signaturebinary)
	signaturebase64 = strings.Replace(signaturebase64, "+", "-", -1)
	signaturebase64 = strings.Replace(signaturebase64, "/", "_", -1)
	signaturebase64 = strings.Replace(signaturebase64, "=", ",", -1)
	queryurl += "&signature=" + signaturebase64

	data, err := g.Request(queryurl)
	if err != nil {
		return nil, err
	}

	lat, lng, err := g.extractLatLngFromResponse(data)
	if err != nil {
		return nil, err
	}

	p := &Point{lat: lat, lng: lng}

	return p, nil
}

// Returns an Address from a Google Geocoder Response body.
func (g *GoogleGeocoder) extractAddressFromResponse(data []byte) (string, error) {
	//var s string
	//s = string(data[:])
	//fmt.Println("debug: response: " + s)
	res := &googleGeocodeResponse{}
	err := json.Unmarshal(data, &res)
	if err != nil {
		return "", err
	}

	if len(res.Results) == 0 {
		return "", errors.New("Failed: (" + res.Status + ") " + res.Error_message)
	} else {
		return res.Results[0].FormattedAddress, nil
	}
}
