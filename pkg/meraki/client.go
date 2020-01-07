package meraki

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Api struct {
	apiKey  string
	baseURL string
}

func New(apiKey string, opts ...func(*Api)) *Api {
	m := Api{
		apiKey:  apiKey,
		baseURL: "https://api.meraki.com/api/v0/",
	}
	for _, option := range opts {
		option(&m)
	}
	return &m
}

func BaseURL(url string) func(*Api) {
	return func(c *Api) {
		c.baseURL = url
	}
}

func (c *Api) FindOrganization(name string) (*Organization, error) {
	orgs, err := c.Organizations()
	if err != nil {
		return nil, err
	}

	var org *Organization
	for _, o := range orgs {
		if strings.ToLower(name) == strings.ToLower(o.Name) {
			org = o
			break
		}
	}

	return org, nil
}

func (c *Api) FindNetwork(organizationID, name string) (*Network, error) {
	nws, err := c.Networks(organizationID)
	if err != nil {
		return nil, err
	}

	var network *Network
	for _, n := range nws {
		if strings.ToLower(name) == strings.ToLower(n.Name) {
			network = n
			break
		}
	}

	return network, nil
}

func (c *Api) Organizations() ([]*Organization, error) {
	var orgs []*Organization
	resp, err := c.get("organizations")
	if err != nil {
		return nil, err
	}

	if resp != nil {
		err = json.Unmarshal(resp, &orgs)
		if err != nil {
			return nil, err
		}
	}
	return orgs, err
}

func (c *Api) Networks(organizationID string) ([]*Network, error) {
	var networks []*Network
	resp, err := c.get(fmt.Sprintf("organizations/%s/networks", organizationID))
	if err != nil {
		return nil, err
	}
	if resp != nil {
		err = json.Unmarshal(resp, &networks)
		if err != nil {
			return nil, err
		}
	}
	return networks, nil
}

func (c *Api) Clients(networkID string) ([]*Client, error) {
	var clients []*Client
	resp, err := c.get(fmt.Sprintf("networks/%s/clients", networkID))
	if err != nil {
		return nil, err
	}
	if resp != nil {
		err = json.Unmarshal(resp, &clients)
		if err != nil {
			return nil, err
		}
	}
	return clients, nil
}

func (c *Api) OnlineClients(networkID string) ([]*Client, error) {
	var clients []*Client
	allClients, err := c.Clients(networkID)
	if err != nil {
		return nil, err
	}
	for _, client := range allClients {
		if client.Status == "Online" {
			clients = append(clients, client)
		}
	}
	return clients, nil
}

func (c *Api) get(path string) ([]byte, error) {
	url := c.baseURL + path + "?perPage=1000"
	httpClient := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Cisco-Meraki-API-Key", c.apiKey)
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	log.WithField("path", path).WithField("status", resp.StatusCode).Info("request")
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected response status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
