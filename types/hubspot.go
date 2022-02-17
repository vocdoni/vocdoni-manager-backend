package types

type HubspotProperties struct {
	Name              string `json:"name,omitempty"`
	VocdoniEmail      string `json:"vocdoni_email,omitempty"`
	NumberOfEmployees string `json:"numberofemployees,omitempty"`
	VocdoniType       string `json:"vocdoni_type,omitempty"`
	Domain            string `json:"domain,omitempty"`
}
type HubspotObject struct {
	Id         string            `json:"id,omitempty"`
	CreatedAt  string            `json:"createdAt,omitempty"`
	UpdatedAt  string            `json:"updatedAt,omitempty"`
	Archived   bool              `json:"archived,omitempty"`
	Properties HubspotProperties `json:"properties,omitempty"`
}

type HsCompany struct {
	Id         string         `json:"id,omitempty"`
	CreatedAt  string         `json:"createdAt,omitempty"`
	UpdatedAt  string         `json:"updatedAt,omitempty"`
	Archived   bool           `json:"archived,omitempty"`
	Properties HsCompanyProps `json:"properties,omitempty"`
}
type HsCompanyProps struct {
	Name              string `json:"name,omitempty"`
	Email             string `json:"email,omitempty"`
	NumberOfEmployees string `json:"numberofemployees,omitempty"`
	VocdoniType       string `json:"vocdoni_type,omitempty"`
	Domain            string `json:"domain,omitempty"`
}

type HsProperty struct {
	Id           string             `json:"id,omitempty"`
	CreatedAt    string             `json:"createdAt,omitempty"`
	UpdatedAt    string             `json:"updatedAt,omitempty"`
	Archived     bool               `json:"archived,omitempty"`
	Name         string             `json:"name,omitempty"`
	Description  string             `json:"description,omitempty"`
	Label        string             `json:"label,omitempty"`
	Type         string             `json:"type,omitempty"`
	FieldType    string             `json:"fieldType,omitempty"`
	GroupName    string             `json:"groupName,omitempty"`
	Options      []HsPropertyOption `json:"options,omitempty"`
	DisplayOrder int                `json:"displayOrder,omitempty"`
	Hidden       bool               `json:"hidden,omitempty"`
	FormField    bool               `json:"formField,omitempty"`
}

type HsPropertyOption struct {
	Label        string `json:"label,omitempty"`
	Value        string `json:"value,omitempty"`
	Description  string `json:"description,omitempty"`
	DisplayOrder int    `json:"displayOrder,omitempty"`
	Hidden       bool   `json:"hidden,omitempty"`
}
