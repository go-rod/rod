package main

import (
	"strings"

	"github.com/tidwall/gjson"
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
	deprecated   bool
	definitions  []*definition
	global       gjson.Result
}

func (schema *domain) find(id string) gjson.Result {
	domain := schema.name
	list := strings.Split(id, ".")
	if len(list) == 2 {
		domain, id = list[0], list[1]
	}

	for _, schema := range schema.global.Get("domains").Array() {
		if schema.Get("domain").String() == domain {
			for _, s := range schema.Get("types").Array() {
				if s.Get("id").String() == id {
					return s
				}
			}
		}
	}
	panic("cannot find: " + domain + "." + id)
}

func (schema *domain) ref(id string) bool {
	return schema.find(id).Get("properties").Exists()
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
}

func parse(schema gjson.Result) []*domain {
	list := []*domain{}

	for _, domainSchema := range schema.Get("domains").Array() {
		list = append(list, parseDomain(schema, domainSchema))
	}

	return list
}

func parseDomain(global, schema gjson.Result) *domain {
	domain := &domain{
		name:         schema.Get("domain").String(),
		experimental: schema.Get("experimental").Bool(),
		definitions:  []*definition{},
		global:       global,
	}

	for _, cdpType := range []cdpType{cdpTypeTypes, cdpTypeCommands, cdpTypeEvents} {
		for _, typeSchame := range schema.Get(string(cdpType)).Array() {
			domain.definitions = append(domain.definitions, parseDef(domain, cdpType, typeSchame)...)
		}
	}

	return domain
}

func parseDef(domain *domain, cdpType cdpType, schema gjson.Result) []*definition {
	list := []*definition{}

	switch cdpType {
	case cdpTypeTypes:
		if schema.Get("properties").Exists() {
			list = append(list, parseStruct(domain, cdpType, schema.Get("id").String(), false, schema, "properties")...)
		} else {
			list = append(list, &definition{
				domain:       domain,
				typeName:     typeName(domain, schema),
				name:         domain.name + symbol(schema.Get("id").String()),
				description:  schema.Get("description").String(),
				deprecated:   schema.Get("deprecated").Bool(),
				experimental: schema.Get("experimental").Bool(),
				objType:      objTypePrimitive,
				enum:         enumList(schema),
			})
		}
	case cdpTypeCommands:
		list = append(list, parseStruct(domain, cdpType, schema.Get("name").String(), true, schema, "parameters")...)
		if schema.Get("returns").Exists() {
			list = append(list, parseStruct(domain, cdpType, schema.Get("name").String()+"Result", false, schema, "returns")...)
		}

	case cdpTypeEvents:
		list = append(list, parseStruct(domain, cdpType, schema.Get("name").String(), false, schema, "parameters")...)

	default:
		panic("type error: " + schema.Raw)

	}

	return list
}

func parseStruct(domain *domain, cdpType cdpType, name string, isCommand bool, schema gjson.Result, propsPath string) []*definition {
	list := []*definition{}

	props := []*definition{}
	for _, propSchema := range schema.Get(propsPath).Array() {
		typeName := typeName(domain, propSchema)

		prop := &definition{
			objType:      objTypePrimitive,
			name:         symbol(propSchema.Get("name").String()),
			originName:   propSchema.Get("name").String(),
			description:  propSchema.Get("description").String(),
			optional:     propSchema.Get("optional").Bool(),
			deprecated:   propSchema.Get("deprecated").Bool(),
			experimental: propSchema.Get("experimental").Bool(),
			typeName:     typeName,
		}

		props = append(props, prop)

		if propSchema.Get("enum").Exists() {
			enum := &definition{
				domain:      domain,
				name:        domain.name + symbol(name) + symbol(propSchema.Get("name").String()),
				objType:     objTypePrimitive,
				description: "enum",
				enum:        enumList(propSchema),
				typeName:    typeName,
			}
			list = append(list, enum)

			prop.typeName = enum.name
		}
	}

	list = append(list, &definition{
		domain:       domain,
		cdpType:      cdpType,
		objType:      objTypeStruct,
		typeName:     typeName(domain, schema),
		name:         domain.name + symbol(name),
		originName:   name,
		description:  schema.Get("description").String(),
		optional:     schema.Get("optional").Bool(),
		deprecated:   schema.Get("deprecated").Bool(),
		experimental: schema.Get("experimental").Bool(),
		props:        props,
		command:      isCommand,
		returnValue:  schema.Get("returns").Exists(),
	})

	return list
}
