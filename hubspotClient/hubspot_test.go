package hubspotClient_test

import (
	"testing"

	"go.vocdoni.io/manager/config"
	"go.vocdoni.io/manager/hubspotClient"
	"go.vocdoni.io/manager/types"
)

func TestNewClient(t *testing.T) {
	var err error
	_, err = hubspotClient.New(&config.Hubspot{ApiKey: "", BaseUrl: "https://api.hubspot.com", Enabled: true})
	if err == nil {
		t.Fatal("api key is not provided client should not be able to start")
	}
	_, err = hubspotClient.New(&config.Hubspot{ApiKey: "eu1-8167-1c7a-4710-9e0d-fb5a55cc6f16", BaseUrl: "https://api.hubspot.com", Enabled: false})
	if err == nil {
		t.Fatal("client is disabled shoul not be initialized")
	}
	_, err = hubspotClient.New(&config.Hubspot{ApiKey: "eu1-8167-1c7a-4710-9e0d-fb5a55cc6f16", BaseUrl: "https://api.hubspot.com", Enabled: true})
	if err != nil {
		t.Fatal("cannot start client")
	}
}
func TestHsAPI(t *testing.T) {
	// start client
	hs, err := hubspotClient.New(&config.Hubspot{ApiKey: "eu1-8167-1c7a-4710-9e0d-fb5a55cc6f16", BaseUrl: "https://api.hubspot.com", Enabled: true})
	if err != nil {
		t.Fatal("cannot start client")
	}
	// create company
	c := types.HubspotProperties{
		Name:              "This is a tesmporal test company",
		VocdoniEmail:      "this is the test email",
		NumberOfEmployees: "3000000",
		VocdoniType:       "aa",
		Domain:            "thetestddomain.com",
	}
	company, err := hs.CreateCompany(c)
	if err != nil {
		t.Fatal("error while creating company")
	}
	// get company
	_, err = hs.GetCompany(company.Id)
	if err != nil {
		t.Fatal("error while getting company")
	}
	// get company with wrong id
	_, err = hs.GetCompany("thisisnotanid")
	if err == nil {
		t.Fatal("company found, company should not exist because the id is invalid")
	}
	// test delete error, wrong id
	err = hs.DeleteCompany("thisisnotanid")
	if err == nil {
		t.Fatal("company deleted, company should not exist because the id is invalid")
	}
	// test delete success
	err = hs.DeleteCompany(company.Id)
	if err != nil {
		t.Fatal("error while deleting company")
	}
	// get company deleted
	_, err = hs.GetCompany(company.Id)
	if err == nil {
		t.Fatal("error while getting company, this company should be deleted")
	}
	// create property
	p := types.HsProperty{
		Name:        "test_property",
		Label:       "Test Property",
		FormField:   true,
		Hidden:      false,
		Type:        "string",
		FieldType:   "text",
		GroupName:   "companyinformation",
		Description: "This is a test property",
	}
	property, err := hs.CreateProperty(p, "company")
	if err != nil {
		t.Fatalf("Error while creating the property, err: %v", err)
	}
	// get property
	_, err = hs.GetProperty(property.Name, "company")
	if err != nil {
		t.Fatal("error while getting property")
	}
	// delete property
	err = hs.DeleteProperty(property.Name, "company")
	if err != nil {
		t.Fatal("error while deleting property")
	}
	// get deleted property
	_, err = hs.GetProperty(property.Name, "company")
	if err == nil {
		t.Fatal("error property should be deleted")
	}
}
