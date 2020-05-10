package meraki

import (
	"strings"
	"time"
)

type Organization struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Network struct {
	ID                      string      `json:"id"`
	OrganizationID          string      `json:"organizationId"`
	Name                    string      `json:"name"`
	TimeZone                string      `json:"timeZone"`
	Tags                    interface{} `json:"tags"`
	ProductTypes            []string    `json:"productTypes"`
	Type                    string      `json:"type"`
	DisableMyMerakiCom      bool        `json:"disableMyMerakiCom"`
	DisableRemoteStatusPage bool        `json:"disableRemoteStatusPage"`
}

type Client struct {
	ID                 string      `json:"id"`
	Mac                string      `json:"mac"`
	Description        string      `json:"description"`
	IP                 string      `json:"ip"`
	IP6                interface{} `json:"ip6"`
	User               interface{} `json:"user"`
	FirstSeen          time.Time   `json:"firstSeen"`
	LastSeen           time.Time   `json:"lastSeen"`
	Manufacturer       string      `json:"manufacturer"`
	Os                 string      `json:"os"`
	RecentDeviceSerial string      `json:"recentDeviceSerial"`
	RecentDeviceName   string      `json:"recentDeviceName"`
	RecentDeviceMac    string      `json:"recentDeviceMac"`
	Ssid               interface{} `json:"ssid"`
	Vlan               int         `json:"vlan"`
	Switchport         interface{} `json:"switchport"`
	Usage              struct {
		Sent int `json:"sent"`
		Recv int `json:"recv"`
	} `json:"usage"`
	Status string `json:"status"`
}

func (c *Client) DNSName() string {
	name := strings.ToLower(c.Description)
	if name == "" {
		name = strings.ToLower(c.Mac)
	}
	name = strings.Replace(name, " ", "-", -1)
	name = strings.Replace(name, ":", "-", -1)
	return name
}
