package main

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/ysmood/kit"
	"github.com/ysmood/rod/lib/launcher"
)

func getSchema() gjson.Result {
	l := launcher.New()

	defer func() {
		p, err := os.FindProcess(l.PID())
		if err == nil {
			_ = p.Kill()
		}
	}()

	u := l.Launch()
	parsed, err := url.Parse(u)
	kit.E(err)
	parsed.Scheme = "http"
	parsed.Path = "/json/protocol"

	data := kit.Req(parsed.String()).MustString()

	kit.E(kit.OutputFile("tmp/proto.json", data, nil))

	return gjson.Parse(data)
}

func mapType(n string) string {
	return map[string]string{
		"boolean": "bool",
		"number":  "float64",
		"integer": "int64",
		"string":  "string",
		"binary":  "[]byte",
		"object":  "map[string]JSON",
		"any":     "JSON",
	}[n]
}

func typeName(domain *domain, schema gjson.Result) string {
	typeName := schema.Get("type").String()

	if typeName == "array" {
		item := schema.Get("items")

		if item.Get("type").Exists() {
			typeName = "[]" + mapType(item.Get("type").String())
		} else {
			ref := item.Get("$ref").String()
			if domain.ref(ref) {
				typeName = "[]*" + refName(domain.name, ref)
			} else {
				typeName = "[]" + refName(domain.name, ref)
			}
		}
	} else if schema.Get("$ref").Exists() {
		ref := schema.Get("$ref").String()
		if domain.ref(ref) {
			typeName += "*"
		}
		typeName += refName(domain.name, ref)
	} else {
		typeName = mapType(typeName)
	}

	switch typeName {
	case "NetworkTimeSinceEpoch", "InputTimeSinceEpoch":
		typeName = "*TimeSinceEpoch"
	case "NetworkMonotonicTime":
		typeName = "*MonotonicTime"
	}

	return typeName
}

func enumList(schema gjson.Result) []string {
	var enum []string
	if schema.Get("enum").Exists() {
		enum = []string{}
		for _, v := range schema.Get("enum").Array() {
			if v.Type != gjson.String {
				panic("enum type error")
			}
			enum = append(enum, v.String())
		}
	}

	return enum
}

func jsonTag(name string, optional bool) string {
	jsonTagValue := name
	if optional {
		jsonTagValue += ",omitempty"
	}
	return fmt.Sprintf("`json:\"%s\"`", jsonTagValue)
}

func refName(domain, id string) string {
	if strings.Contains(id, ".") {
		return symbol(id)
	}
	return domain + symbol(id)
}

// make sure golint works fine
func symbol(n string) string {
	if n == "" {
		return ""
	}

	n = strings.Replace(n, ".", "", -1)

	dashed := regexp.MustCompile(`[-_]`).Split(n, -1)
	if len(dashed) > 1 {
		converted := []string{}
		for _, part := range dashed {
			converted = append(converted, strings.ToUpper(part[:1])+part[1:])
		}
		n = strings.Join(converted, "")
	}

	n = strings.ToUpper(n[:1]) + n[1:]

	n = replaceLower(n, "Id")
	n = replaceLower(n, "Css")
	n = replaceLower(n, "Url")
	n = replaceLower(n, "Uuid")
	n = replaceLower(n, "Xml")
	n = replaceLower(n, "Http")
	n = replaceLower(n, "Dns")
	n = replaceLower(n, "Cpu")
	n = replaceLower(n, "Mime")
	n = replaceLower(n, "Json")
	n = replaceLower(n, "Html")
	n = replaceLower(n, "Guid")
	n = replaceLower(n, "Sql")
	n = replaceLower(n, "Eof")
	n = replaceLower(n, "Api")

	return n
}

func replaceLower(n, word string) string {
	return regexp.MustCompile(word+`([A-Z-_]|$)`).ReplaceAllStringFunc(n, func(s string) string {
		return strings.ToUpper(s)
	})
}
