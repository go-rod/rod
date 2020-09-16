package main

import (
	"strings"

	"github.com/go-rod/rod/lib/utils"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
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
	skip         bool
}

func parse(schema gjson.Result) []*domain {
	optimize(&schema.Raw)

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
				skip:         schema.Get("skip").Bool(),
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
		skip:         schema.Get("skip").Bool(),
	})

	return list
}

func optimize(json *string) {
	var err error

	set := func(path string, value interface{}) {
		*json, err = sjson.Set(*json, path, value)
		utils.E(err)
	}

	// TargetTargetInfoType
	set("domains.32.types.2.properties.1.enum", []string{
		"page", "background_page", "service_worker", "shared_worker", "browser", "other",
	})

	// PageLifecycleEventName
	set("domains.26.events.17.parameters.2.enum", []string{
		"init", "firstPaint", "firstContentfulPaint", "firstImagePaint", "firstMeaningfulPaintCandidate",
		"DOMContentLoaded", "load", "networkAlmostIdle", "firstMeaningfulPaint", "networkIdle",
	})

	// replace these with better type definition
	set("domains.19.types.3.skip", true) // Input.TimeSinceEpoch
	set("domains.24.types.5.skip", true) // Network.TimeSinceEpoch
	set("domains.24.types.6.skip", true) // Network.MonotonicTime
}
