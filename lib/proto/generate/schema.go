package main

import (
	"strings"

	"github.com/ysmood/gson"
)

type objType int

const (
	objTypeStruct    objType = iota // such as object
	objTypePrimitive                // such as string, bool
)

type cdpType string

const (
	cdpTypeTypes    cdpType = "types"
	cdpTypeCommands cdpType = "commands"
	cdpTypeEvents   cdpType = "events"
)

type domain struct {
	name         string
	experimental bool
	description  string
	definitions  []*definition
	global       gson.JSON
}

func (schema *domain) find(id string) gson.JSON {
	domain := schema.name
	list := strings.Split(id, ".")
	if len(list) == 2 {
		domain, id = list[0], list[1]
	}

	for _, schema := range schema.global.Get("domains").Arr() {
		if schema.Get("domain").Str() == domain {
			for _, s := range schema.Get("types").Arr() {
				if s.Get("id").Str() == id {
					return s
				}
			}
		}
	}
	panic("cannot find: " + domain + "." + id)
}

func (schema *domain) ref(id string) bool {
	return schema.find(id).Has("properties")
}

type definition struct {
	domain       *domain
	objType      objType
	cdpType      cdpType
	typeName     string
	enum         []string
	name         string
	originName   string
	description  string
	experimental bool
	deprecated   bool
	optional     bool
	command      bool
	returnValue  bool
	props        []*definition
	skip         bool
}

func parse(schema gson.JSON) []*domain {
	patch(schema)

	list := []*domain{}

	for _, domainSchema := range schema.Get("domains").Arr() {
		list = append(list, parseDomain(schema, domainSchema))
	}

	return list
}

func parseDomain(global, schema gson.JSON) *domain {
	domain := &domain{
		name:         schema.Get("domain").Str(),
		experimental: schema.Get("experimental").Bool(),
		definitions:  []*definition{},
		global:       global,
	}

	if schema.Has("description") {
		domain.description = schema.Get("description").Str()
	}

	for _, cdpType := range []cdpType{cdpTypeTypes, cdpTypeCommands, cdpTypeEvents} {
		for _, typeSchame := range schema.Get(string(cdpType)).Arr() {
			domain.definitions = append(domain.definitions, parseDef(domain, cdpType, typeSchame)...)
		}
	}

	return domain
}

func parseDef(domain *domain, cdpType cdpType, schema gson.JSON) []*definition {
	list := []*definition{}

	switch cdpType {
	case cdpTypeTypes:
		if schema.Has("properties") {
			list = append(list, parseStruct(domain, cdpType, schema.Get("id").Str(), false, schema, "properties")...)
		} else {
			list = append(list, &definition{
				domain:       domain,
				typeName:     typeName(domain, schema),
				name:         domain.name + symbol(schema.Get("id").Str()),
				description:  schema.Get("description").Str(),
				deprecated:   schema.Get("deprecated").Bool(),
				experimental: schema.Get("experimental").Bool(),
				objType:      objTypePrimitive,
				enum:         enumList(schema),
				skip:         schema.Get("skip").Bool(),
			})
		}
	case cdpTypeCommands:
		list = append(list, parseStruct(domain, cdpType, schema.Get("name").Str(), true, schema, "parameters")...)
		if schema.Has("returns") {
			list = append(list, parseStruct(domain, cdpType, schema.Get("name").Str()+"Result", false, schema, "returns")...)
		}

	case cdpTypeEvents:
		list = append(list, parseStruct(domain, cdpType, schema.Get("name").Str(), false, schema, "parameters")...)

	default:
		panic("type error: " + schema.Str())

	}

	return list
}

func parseStruct(domain *domain, cdpType cdpType, name string, isCommand bool, schema gson.JSON, propsPath string) []*definition {
	list := []*definition{}

	props := []*definition{}
	for _, propSchema := range schema.Get(propsPath).Arr() {
		typeName := typeName(domain, propSchema)

		prop := &definition{
			objType:      objTypePrimitive,
			name:         symbol(propSchema.Get("name").Str()),
			originName:   propSchema.Get("name").Str(),
			description:  propSchema.Get("description").Str(),
			optional:     propSchema.Get("optional").Bool(),
			deprecated:   propSchema.Get("deprecated").Bool(),
			experimental: propSchema.Get("experimental").Bool(),
			typeName:     typeName,
		}

		props = append(props, prop)

		if propSchema.Has("enum") {
			enum := &definition{
				domain:      domain,
				name:        domain.name + symbol(name) + symbol(propSchema.Get("name").Str()),
				objType:     objTypePrimitive,
				description: "enum",
				enum:        enumList(propSchema),
				typeName:    typeName,
			}
			list = append(list, enum)

			prop.typeName = enum.name
		}
	}

	desc := schema.Get("description").Str()
	if !isCommand && schema.Has("returns") {
		desc = "..."
	}

	list = append(list, &definition{
		domain:       domain,
		cdpType:      cdpType,
		objType:      objTypeStruct,
		typeName:     typeName(domain, schema),
		name:         domain.name + symbol(name),
		originName:   name,
		description:  desc,
		optional:     schema.Get("optional").Bool(),
		deprecated:   schema.Get("deprecated").Bool(),
		experimental: schema.Get("experimental").Bool(),
		props:        props,
		command:      isCommand,
		returnValue:  schema.Has("returns"),
		skip:         schema.Get("skip").Bool(),
	})

	return list
}
