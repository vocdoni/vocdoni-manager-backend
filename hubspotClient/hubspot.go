package hubspotClient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	log "go.vocdoni.io/dvote/log"
	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/types"
)

type Hubspot struct {
	config *config.Hubspot
}

func New(hsc *config.Hubspot) (*Hubspot, error) {
	if hsc.ApiKey == "" {
		return nil, fmt.Errorf("invalid api key")
	}
	if !hsc.Enabled {
		return nil, fmt.Errorf("hubspot api not enabled")
	}
	return &Hubspot{config: hsc}, nil
}

func (hs *Hubspot) GetProperty(propertyName string, objectName string) (*types.HubspotProperties, error) {
	var data types.HubspotProperties
	// make url and http request
	url := fmt.Sprintf("%s/crm/v3/properties/%s/%s?hapikey=%s", hs.config.BaseUrl, objectName, propertyName, hs.config.ApiKey)
	err := httpWrapper(http.MethodGet, url, nil, &data)
	if err != nil {
		return nil, fmt.Errorf("error while making the http request, err: %v", err)
	}
	return &data, nil
}

func (hs *Hubspot) CreateProperty(p types.HsProperty, objectName string) (*types.HsProperty, error) {
	var data types.HsProperty
	// generate url and make request
	url := fmt.Sprintf("%s/crm/v3/properties/%s?hapikey=%s", hs.config.BaseUrl, objectName, hs.config.ApiKey)
	err := httpWrapper(http.MethodPost, url, p, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}
func (hs *Hubspot) DeleteProperty(propertyName string, objectName string) error {
	// generate url and make request
	url := fmt.Sprintf("%s/crm/v3/properties/%s/%s?hapikey=%s", hs.config.BaseUrl, objectName, propertyName, hs.config.ApiKey)
	err := httpWrapper(http.MethodDelete, url, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

/*************/
/* COMPANIES */
/*************/

// Gets a hubspot company by its id
func (hs *Hubspot) GetCompany(id string) (*types.HubspotProperties, error) {
	var data types.HubspotProperties
	// make url and http request
	url := fmt.Sprintf("%s/crm/v3/objects/companies/%s?hapikey=%s", hs.config.BaseUrl, id, hs.config.ApiKey)
	err := httpWrapper(http.MethodGet, url, nil, &data)
	if err != nil {
		return nil, fmt.Errorf("error while making the http request, err: %v", err)
	}
	return &data, nil
}

// Creates a company in hubspot using the v3 API
func (hs *Hubspot) CreateCompany(c types.HubspotProperties) (*types.HubspotObject, error) {
	var data types.HubspotObject
	// set post data
	bodyObj := types.HubspotObject{
		Properties: c,
	}
	// generate url and make request
	url := fmt.Sprintf("%s/crm/v3/objects/companies?hapikey=%s", hs.config.BaseUrl, hs.config.ApiKey)
	err := httpWrapper(http.MethodPost, url, bodyObj, &data)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// Deletes a company in hubspot using the v3 API
func (hs *Hubspot) DeleteCompany(id string) error {
	// generate url and make request
	url := fmt.Sprintf("%s/crm/v3/objects/companies/%s?hapikey=%s", hs.config.BaseUrl, id, hs.config.ApiKey)
	err := httpWrapper(http.MethodDelete, url, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (hs *Hubspot) InitializeProperties() {

	var (
		p   types.HsProperty
		err error
	)
	// Create necessary properties
	if _, err = hs.GetProperty("vocdoni_email", "company"); err != nil {
		// Error because property does not exist
		p = types.HsProperty{
			Name:        "vocdoni_email",
			Label:       "Email",
			FormField:   true,
			Hidden:      false,
			Type:        "string",
			FieldType:   "text",
			GroupName:   "companyinformation",
			Description: "This property is used to store the email that the entity used in vocdoni.app",
		}
		if _, err := hs.CreateProperty(p, "company"); err != nil {
			log.Warnf("could not create vocdoni_email property, err: %s", err.Error())
		} else {
			log.Info("vocdoni_email property created in hubspot")
		}
	} else {
		log.Info("vocdoni_email property already exists in hubspot, omitting")
	}
	if _, err = hs.GetProperty("vocdoni_type", "company"); err != nil {
		p = types.HsProperty{
			Name:        "vocdoni_type",
			Label:       "Entity Type",
			FormField:   true,
			Hidden:      false,
			Type:        "enumeration",
			FieldType:   "select",
			GroupName:   "companyinformation",
			Description: "This property is used to store the entity type that the entity in selected vocdoni.app",
			Options: []types.HsPropertyOption{{
				Label:        "Association or Non-Profit Organization",
				Value:        "non-profit",
				DisplayOrder: 1,
				Hidden:       false,
			},
				{
					Label:        "Informal Organization",
					Value:        "informal",
					DisplayOrder: 2,
					Hidden:       false,
				},
				{
					Label:        "Company",
					Value:        "company",
					DisplayOrder: 3,
					Hidden:       false,
				},
				{
					Label:        "Cooperative",
					Value:        "cooperative",
					DisplayOrder: 4,
					Hidden:       false,
				},
				{
					Label:        "Trade Union",
					Value:        "trade-union",
					DisplayOrder: 5,
					Hidden:       false,
				},
				{
					Label:        "City Council",
					Value:        "city-council",
					DisplayOrder: 6,
					Hidden:       false,
				},
				{
					Label:        "Other Public Institutions",
					Value:        "other-public",
					DisplayOrder: 7,
					Hidden:       false,
				},
				{
					Label:        "Individual",
					Value:        "individual",
					DisplayOrder: 8,
					Hidden:       false,
				},
				{
					Label:        "Others",
					Value:        "other",
					DisplayOrder: 9,
					Hidden:       false,
				},
			},
		}
		if _, err := hs.CreateProperty(p, "company"); err != nil {
			log.Warnf("could not create vocdoni_type property, err: %s", err.Error())
		} else {
			log.Info("vocdoni_type property created in hubspot")
		}
	} else {
		log.Info("vocdoni_type property already exists in hubspot, omitting")
	}
}

func httpWrapper(method string, url string, body interface{}, res interface{}) error {
	var (
		expectedStatus int
		resp           *http.Response
		err            error
	)
	switch method {
	case http.MethodGet:
		expectedStatus = 200
		resp, err = http.Get(url)
		if err != nil {
			return fmt.Errorf("error while making the http request, err: %v", err)
		}
		defer resp.Body.Close()
	case http.MethodPost:
		expectedStatus = 201
		bodyJson, _ := json.Marshal(body)
		bodyBytes := bytes.NewBuffer(bodyJson)
		resp, err = http.Post(url, "application/json", bodyBytes)
		if err != nil {
			return fmt.Errorf("error while making the http request, err: %v", err)
		}
		defer resp.Body.Close()
	case http.MethodDelete:
		expectedStatus = 204
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodDelete, url, nil)
		if err != nil {
			return fmt.Errorf("error while creating the http request, err: %v", err)
		}
		resp, err = client.Do(req)
		if err != nil {
			return fmt.Errorf("error while making the http request, err: %v", err)
		}
		defer resp.Body.Close()
	default:
		return fmt.Errorf("method not suported")
	}
	if body != nil {
		err := json.NewDecoder(resp.Body).Decode(res)
		if err != nil {
			return fmt.Errorf("error while decoding hubspot response, err: %v", err)
		}
	}
	if resp.StatusCode != expectedStatus {
		return fmt.Errorf("error on while making request to hubspot api, code %d, res: %v", resp.StatusCode, res)
	}
	return nil
}
