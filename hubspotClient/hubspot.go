package hubspotClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/types"
)

type Hubspot struct {
	config *config.Hubspot
}

// Creates a company in hubspot using the v3 API
func (hs *Hubspot) CreateCompany(c *types.HubspotCompany) (types.HubspotObject, error) {
	var data types.HubspotObject
	// convert object to bytes
	bodyObj := types.HubspotObject{
		CompanyProperties: *c,
	}
	bodyJson, _ := json.Marshal(bodyObj)
	bodyBytes := bytes.NewBuffer(bodyJson)
	// generate url and make request
	url := fmt.Sprintf("%s/crm/v3/objects/companies?hapikey=%s", hs.config.BaseUrl, hs.config.ApiKey)
	resp, err := http.Post(url, "application/json", bodyBytes)
	if err != nil {
		return data, fmt.Errorf("error while creating company in hubspot, err: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusCreated {
		// Decode response into Hubspot Object
		err := json.NewDecoder(resp.Body).Decode(&data)
		if err != nil {
			return data, fmt.Errorf("error while decoding hubspot response, err: %v", err)
		}
	} else {
		return data, fmt.Errorf("error on while doing request to hubspot api, code %d", resp.StatusCode)
	}
	return data, nil
}
func New(hsc *config.Hubspot) (*Hubspot, error) {
	return &Hubspot{config: hsc}, nil
}
