package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
)

// Implementation of the API Credentials interface based on cookies authentication
type CookieCredentials struct {
	Client               *http.Client
	PersistentParameters *PersistentParameters
	RuntimeParameters    *RuntimeParameters
}

// Structure that is serialized in JSON to store the user credentials
type AuthFile struct {
	// Cookies to make the requests
	Cookies []*http.Cookie `json:"cookies"`

	// Parameters used by the requests (user id, etc ...)
	PersistentParameters *PersistentParameters `json:"persistantParameters"`
}

// Create a new CookieCredentials given a slice of cookies and the PersistentParameters
func NewCookieCredentials(cookies []*http.Cookie, parameters *PersistentParameters) *CookieCredentials {
	// Create a cookie jar for the client
	jar, _ := cookiejar.New(nil)
	cookiesUrl, _ := url.Parse("https://photos.google.com")

	// Add cookies in the jar
	jar.SetCookies(cookiesUrl, cookies)

	return &CookieCredentials{
		Client: &http.Client{
			Jar: jar,
		},
		PersistentParameters: parameters,
		RuntimeParameters:    &RuntimeParameters{},
	}
}

// Restore an CookieCredentials object from a JSON
func NewCookieCredentialsFromJson(in io.Reader) (*CookieCredentials, error) {

	// Parse AuthFile
	authFile := AuthFile{}
	if err := json.NewDecoder(in).Decode(&authFile); err != nil {
		return nil, fmt.Errorf("auth: Can't read the JSON AuthFile (%v)", err)
	}

	return NewCookieCredentials(authFile.Cookies, authFile.PersistentParameters), nil
}

// Restore an CookieCredentials object from a JSON file
func NewCookieCredentialsFromFile(fileName string) (*CookieCredentials, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("auth: Can't open %v", fileName)
	}
	defer file.Close()

	return NewCookieCredentialsFromJson(file)
}

// func (c *CookieCredentials) GetClient() *http.Client {
// 	return c.client
// }

// func (c *CookieCredentials) GetPersistentParameters() (*PersistentParameters, error) {
// 	if c.persistentParameters == nil {
// 		return nil, fmt.Errorf("persistent parameters not set")
// 	}
// 	return c.persistentParameters, nil
// }

// func (c *CookieCredentials) SetPersistentParameters(pp *PersistentParameters) {
// 	c.persistentParameters = pp
// }

// func (c *CookieCredentials) GetRuntimeParameters() *RuntimeParameters {
// 	return c.runtimeParameters
// }

// Serialize the CookieCredentials object into a JSON object, to be restored in the future using
// NewCookieCredentialsFromJson
func (c *CookieCredentials) Serialize(out io.Writer) error {
	cookiesUrl, _ := url.Parse("https://photos.google.com")
	cookies := c.Client.Jar.Cookies(cookiesUrl)

	for _, cookie := range cookies {
		if cookie.Name == "OTZ" {
			cookie.Domain = "photos.google.com"
		} else {
			cookie.Domain = ".google.com"
		}
		cookie.Path = "/"
	}

	return json.NewEncoder(out).Encode(&AuthFile{
		Cookies:              cookies,
		PersistentParameters: c.PersistentParameters,
	})
}

// Serialize the CookieCredentials object into a JSON file, to be restored in the future using
// NewCookieCredentialsFromJsonFile
func (c *CookieCredentials) SerializeToFile(fileName string) error {
	file, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("auth: Can't create the file %v (%v)", fileName, err)
	}
	defer file.Close()

	return c.Serialize(file)
}

// Result of a credentials test
type CredentialsTestResult struct {
	// False if the cookies are not valid anymore
	Valid bool

	// Reason to explain a negative Valid field value
	Reason string
}

// Test the CookieCredentials object to see if the authentication cookies are valid.
// Note that this method can return false-positive results, but if it return a CredentialsTestResult with false as Valid
// the cookies are not valid for sure.
// An eventual as second return parameter try to explain why we can't determine the credentials validity
func (c *CookieCredentials) TestCredentials() (*CredentialsTestResult, error) {
	// To check if the cookies are valid, make a request to the Google Photos Login and check if we're redirected
	req, err := http.NewRequest("GET", "https://photos.google.com/login", nil)
	if err != nil {
		return nil, err
	}

	// Make the request
	res, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth: Can't send an HTTPS reuqest to check cookies validity (%v)", err)
	}
	if res.Request.URL.String() != "https://photos.google.com/" {
		return &CredentialsTestResult{
			Valid:  false,
			Reason: "Google didn't redirect us to the Photos Homepage while accessing the Login page",
		}, nil
	}

	// All seems all right
	return &CredentialsTestResult{
		Valid: true,
	}, nil
}
