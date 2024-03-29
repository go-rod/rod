// This file is generated by "./lib/proto/generate"

package proto

/*

Schema

This domain is deprecated.

*/

// SchemaDomain Description of the protocol domain.
type SchemaDomain struct {
	// Name Domain name.
	Name string `json:"name"`

	// Version Domain version.
	Version string `json:"version"`
}

// SchemaGetDomains Returns supported domains.
type SchemaGetDomains struct{}

// ProtoReq name.
func (m SchemaGetDomains) ProtoReq() string { return "Schema.getDomains" }

// Call the request.
func (m SchemaGetDomains) Call(c Client) (*SchemaGetDomainsResult, error) {
	var res SchemaGetDomainsResult
	return &res, call(m.ProtoReq(), m, &res, c)
}

// SchemaGetDomainsResult ...
type SchemaGetDomainsResult struct {
	// Domains List of supported domains.
	Domains []*SchemaDomain `json:"domains"`
}
