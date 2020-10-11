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
	optimize(schema)

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

	list = append(list, &definition{
		domain:       domain,
		cdpType:      cdpType,
		objType:      objTypeStruct,
		typeName:     typeName(domain, schema),
		name:         domain.name + symbol(name),
		originName:   name,
		description:  schema.Get("description").Str(),
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

func optimize(json gson.JSON) {
	k := func(k, v string) gson.Query {
		return func(target interface{}) (val interface{}, has bool) {
			for _, el := range target.([]interface{}) {
				res := el.(map[string]interface{})[k]
				if res == v {
					return el, true
				}
			}
			panic("not found")
		}
	}

	getTypes := func(domain string) gson.JSON {
		res, _ := json.Gets("domains", k("domain", domain), "types")
		return res
	}

	// TargetTargetInfoType
	j, _ := getTypes("Target").Gets(k("id", "TargetInfo"), "properties", k("name", "type"))
	j.Set("enum", []string{
		"page", "background_page", "service_worker", "shared_worker", "browser", "other",
	})

	// PageLifecycleEventName
	j, _ = json.Gets("domains", k("domain", "Page"), "events", k("name", "lifecycleEvent"), "parameters", k("name", "name"))
	j.Set("enum", []string{
		"init", "firstPaint", "firstContentfulPaint", "firstImagePaint", "firstMeaningfulPaintCandidate",
		"DOMContentLoaded", "load", "networkAlmostIdle", "firstMeaningfulPaint", "networkIdle",
	})

	// replace these with better type definition
	j, _ = getTypes("Input").Gets(k("id", "TimeSinceEpoch"))
	j.Set("skip", true)
	j, _ = getTypes("Network").Gets(k("id", "TimeSinceEpoch"))
	j.Set("skip", true)
	j, _ = getTypes("Network").Gets(k("id", "MonotonicTime"))
	j.Set("skip", true)

	// fix Cookie.Expires
	j, _ = getTypes("Network").Gets(k("id", "Cookie"), "properties")
	j.Set("4", map[string]interface{}{
		"$ref":        "TimeSinceEpoch",
		"description": "Cookie expiration date",
		"name":        "expires",
	})
}
