package k8s

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

const (
	// APIEndpoint defines the base path for kubernetes API resources.
	APIEndpoint        = "/api/v1"
	defaultService     = "/namespaces/default/services"
	extentionsEndpoint = "/apis/extensions/v1beta1"
	defaultIngress     = "/ingresses"
)

// Client is a client for the Kubernetes master.
type Client struct {
	endpointURL string
	httpClient  *http.Client
}

// NewClient returns a new Kubernetes client.
// The provided host is an url (scheme://hostname[:port]) of a
// Kubernetes master without any path.
// The provided client is an authorized http.Client used to perform requests to the Kubernetes API master.
func NewClient(baseURL string, client *http.Client) (*Client, error) {
	validURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL %q: %v", baseURL, err)
	}
	return &Client{
		endpointURL: strings.TrimSuffix(validURL.String(), "/"),
		httpClient:  client,
	}, nil
}

// GetIngresses returns all services in the cluster
func (c *Client) GetIngresses(predicate func(Ingress) bool) ([]Ingress, error) {
	getURL := c.endpointURL + extentionsEndpoint + defaultIngress

	// Make request to Kubernetes API
	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: GET %q : %v", getURL, err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: GET %q: %v", getURL, err)
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body for GET %q: %v", getURL, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error %d GET %q: %q: %v", res.StatusCode, getURL, string(body), err)
	}

	var ingressList IngressList
	if err := json.Unmarshal(body, &ingressList); err != nil {
		return nil, fmt.Errorf("failed to decode list of ingress resources: %v", err)
	}
	ingresses := ingressList.Items[:0]
	for _, ingress := range ingressList.Items {
		if predicate(ingress) {
			ingresses = append(ingresses, ingress)
		}
	}
	return ingresses, nil
}

// GetServices returns all services in the cluster
func (c *Client) GetServices(predicate func(Service) bool) ([]Service, error) {
	getURL := c.endpointURL + APIEndpoint + defaultService

	// Make request to Kubernetes API
	req, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: GET %q : %v", getURL, err)
	}
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: GET %q: %v", getURL, err)
	}

	body, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read request body for GET %q: %v", getURL, err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error %d GET %q: %q: %v", res.StatusCode, getURL, string(body), err)
	}

	var serviceList ServiceList
	if err := json.Unmarshal(body, &serviceList); err != nil {
		return nil, fmt.Errorf("failed to decode list of services resources: %v", err)
	}
	services := serviceList.Items[:0]
	for _, service := range serviceList.Items {
		if predicate(service) {
			services = append(services, service)
		}
	}
	return services, nil
}
