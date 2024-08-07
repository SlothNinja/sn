package sn

import (
	"os"
)

type options struct {
	projectID        string
	url              string
	frontEndURL      string
	backEndURL       string
	port             string
	frontEndPort     string
	backEndPort      string
	secretsProjectID string
	secretsDSURL     string
	prefix           string
	home             string
}

// WithProjectID sets the Google Cloud Project.
// Overrides value set by GOOGLE_CLOUD_PROJECT environment variable.
// CAUTION: Likely only suitable for development environment as Google Cloud environment sets GOOGLE_CLOUD_PROJECT
func WithProjectID(id string) Option {
	return func(cl *Client) *Client {
		cl.projectID = id
		return cl
	}
}

func getProjectID() string {
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}

// GetProjectID returns value of the Google Cloud Project for the client.
func (cl *Client) GetProjectID() string {
	return cl.projectID
}

// WithURL sets the URL for the service, with protocol but without port.
// Also set FrontEndURL and BackEndURL, if not otherwise defined.
// Overrides value set by URL environment variable
// For example, https://www.slothninja.com
func WithURL(url string) Option {
	return func(cl *Client) *Client {
		cl.url = url
		if cl.frontEndURL == "" {
			cl.frontEndURL = url
		}
		if cl.backEndURL == "" {
			cl.backEndURL = url
		}
		return cl
	}
}

func getURL() string {
	return os.Getenv("URL")
}

// GetURL returns the URL for the service.
func (cl *Client) GetURL() string {
	return cl.url
}

// WithFrontEndURL sets the URL for the front end of the service, with protocol but without port.
// For example, https://www.slothninja.com
// If not set via WithFrontEndURL or via FE_URL environment variable, fallsback to WithURL value
func WithFrontEndURL(url string) Option {
	return func(cl *Client) *Client {
		cl.frontEndURL = url
		return cl
	}
}

func getFrontEndURL() string {
	if url, found := os.LookupEnv("FE_URL"); found {
		return url
	}
	return getURL()
}

// GetFrontEndURL returns the URL for the front end of the service.
func (cl *Client) GetFrontEndURL() string {
	return cl.frontEndURL
}

// WithBackEndURL sets the URL for the back end of the service, with protocol but without port.
// For example, https://www.slothninja.com
// If not set via WithBackEndURL or via BE_URL environment variable, fallsback to WithURL value
func WithBackEndURL(url string) Option {
	return func(cl *Client) *Client {
		cl.backEndURL = url
		return cl
	}
}

func getBackEndURL() string {
	if url, found := os.LookupEnv("BE_URL"); found {
		return url
	}
	return getURL()
}

// GetBackEndURL returns the URL for the back end of the service.
func (cl *Client) GetBackEndURL() string {
	return cl.backEndURL
}

// WithPort sets the port for the service.
// Also sets front end port and back end port, if not otherwise defined.
// Overrides value set by PORT environment variable
func WithPort(port string) Option {
	return func(cl *Client) *Client {
		cl.port = port
		if cl.frontEndPort == "" {
			cl.frontEndPort = port
		}
		if cl.backEndPort == "" {
			cl.backEndPort = port
		}
		return cl
	}
}

func getPort() string {
	return os.Getenv("PORT")
}

// GetPort return the value of the port.
func (cl *Client) GetPort() string {
	return cl.port
}

// WithFrontEndPort sets the port for the front end of the service.
// If not set via WithFrontEndPort, the FE_PORT environment variable, or the PORT environment variable,
// then the front end port fallsback to the WithPort value
func WithFrontEndPort(port string) Option {
	return func(cl *Client) *Client {
		cl.frontEndPort = port
		return cl
	}
}

func getFrontEndPort() string {
	if port, found := os.LookupEnv("FE_PORT"); found {
		return port
	}
	return getPort()
}

// GetFrontEndPort return the value of the front end port.
func (cl *Client) GetFrontEndPort() string {
	return cl.frontEndPort
}

// WithBackEndPort sets the port for the back end of the service.
// If not set via WithBackEndPort, the BE_PORT environment variable, or the PORT environment variable,
// then the back end port fallsback to the WithPort value
func WithBackEndPort(port string) Option {
	return func(cl *Client) *Client {
		cl.backEndPort = port
		return cl
	}
}

func getBackEndPort() string {
	if port, found := os.LookupEnv("BE_PORT"); found {
		return port
	}
	return getPort()
}

// GetBackEndPort return the value of the back end port.
func (cl *Client) GetBackEndPort() string {
	return cl.backEndPort
}

// WithSecretsProjectID used to set the SECRETS_PROJECT_ID option for the client
func WithSecretsProjectID(id string) Option {
	return func(cl *Client) *Client {
		cl.secretsProjectID = id
		return cl
	}
}

func getSecretsProjectID() string {
	if id, found := os.LookupEnv("SECRETS_PROJECT_ID"); found {
		return id
	}
	return "user-slothninja-games"
}

// GetSecretsProjectID used to get the SECRETS_PROJECT_ID option specified for the client
func (cl *Client) GetSecretsProjectID() string {
	return cl.secretsProjectID
}

// WithSecretsDSURL used to set the url for the secrets datastore
func WithSecretsDSURL(url string) Option {
	return func(cl *Client) *Client {
		cl.secretsDSURL = url
		return cl
	}
}

func getSecretsDSURL() string {
	if url, found := os.LookupEnv("SECRETS_DS_URL"); found {
		return url
	}
	if IsProduction() {
		return "user.slothninja.com"
	}
	return "localhost:8086"
}

// GetSecretsDSURL returns the url for the secrets datastore
func (cl *Client) GetSecretsDSURL() string {
	return cl.secretsDSURL
}

// WithPrefix used to set a url prefix for backend actions
func WithPrefix(prefix string) Option {
	return func(cl *Client) *Client {
		cl.prefix = prefix
		return cl
	}
}

func getPrefix() string {
	if prefix, found := os.LookupEnv("PREFIX"); found {
		return prefix
	}
	return "/sn"
}

// GetPrefix used to get the url prefix for backend actions
func (cl *Client) GetPrefix() string {
	return cl.prefix
}

// WithHome used to set the url path to the home page
func WithHome(path string) Option {
	return func(cl *Client) *Client {
		cl.home = path
		return cl
	}
}

func getHome() string {
	if prefix, found := os.LookupEnv("HOME"); found {
		return prefix
	}
	return "/"
}

// GetHome used to get the url path to the home page
func (cl *Client) GetHome() string {
	return cl.home
}

// Option type for functions used to set client options
type Option func(*Client) *Client
