package geo

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

// This struct contains all the funcitonality
// of interacting with the Google Maps Geocoding Service
type GoogleGeocoder struct{}

// This struct contains selected fields from Google's Geocoding Service response
type googleGeocodeResponse struct {
	Results []struct {
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
var googleGeocodeURL = "http://maps.googleapis.com/maps/api/geocode/json"

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

	fullUrl := fmt.Sprintf("%s?sensor=false&%s", googleGeocodeURL, params)

	// TODO Potentially refactor out from MapQuestGeocoder as well
	req, _ := http.NewRequest("GET", fullUrl, nil)
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
	json.Unmarshal(data, &res)

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

	if apikey != "" {
		queryurl = fmt.Sprintf("key="+url.QueryEscape(apikey)+"&language=ja&latlng=%f,%f", p.lat, p.lng)
	} else {
		queryurl = fmt.Sprintf("language=ja&latlng=%f,%f", p.lat, p.lng)
	}

	data, err := g.Request(queryurl)
	if err != nil {
		return "", err
	}

	resStr := g.extractAddressFromResponse(data)

	return resStr, nil
}

// Returns an Address from a Google Geocoder Response body.
func (g *GoogleGeocoder) extractAddressFromResponse(data []byte) string {
	res := &googleGeocodeResponse{}
	json.Unmarshal(data, &res)

	return res.Results[0].FormattedAddress
}
