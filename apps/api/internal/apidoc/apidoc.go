// Package apidoc finishes the OpenAPI document ekman generates, for `server openapi` (and so
// for `make gen`, openapi.json and the web's generated types).
//
// ekman v0.1 documents every Go string type as a plain string and has no hook to say more.
// Here, a string type with an Enum() method becomes a named enum schema ("people.Status"), and
// every property and parameter of that type points at it, so the web's types narrow to the
// real values instead of string. Types whose values are only known at run time (role names)
// are passed in Document's extra. A route whose input implements Multipart gets its
// multipart/form-data body documented.
//
// To make a new string type an enum: give it `func (T) Enum() []string`, then run `make gen`.
package apidoc

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strings"
	"unicode"

	"github.com/sakib412/ekman"
)

// Enum is a string type with a closed set of values.
type Enum interface{ Enum() []string }

// Multipart is an input whose body is a multipart form with one file field.
type Multipart interface{ MultipartFile() string }

type obj = map[string]any

var (
	enumType      = reflect.TypeFor[Enum]()
	multipartType = reflect.TypeFor[Multipart]()
	jsonMarshaler = reflect.TypeFor[json.Marshaler]()
)

// Document returns app's OpenAPI document with enums (and multipart bodies) filled in. extra
// adds enum values for types without an Enum method.
func Document(app *ekman.App, extra map[reflect.Type][]string) ([]byte, error) {
	raw, err := app.OpenAPI()
	if err != nil {
		return nil, err
	}
	var doc obj
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	p := &patcher{enums: map[reflect.Type][]string{}, byName: map[string]reflect.Type{}, seen: map[reflect.Type]bool{}}
	for t, vals := range extra {
		p.enums[t] = vals
	}
	routes := app.Routes()
	if routes == nil {
		return nil, fmt.Errorf("apidoc: the app failed validation")
	}
	for _, r := range routes {
		p.walk(r.In)
		p.walk(r.Out)
	}

	components, _ := doc["components"].(obj)
	schemas, _ := components["schemas"].(obj)
	if schemas == nil {
		return nil, fmt.Errorf("apidoc: no components.schemas")
	}
	for t, vals := range p.enums {
		name := typeName(t)
		if prev, ok := schemas[name]; ok {
			if m, _ := prev.(obj); m["enum"] == nil {
				return nil, fmt.Errorf("apidoc: enum %s collides with a schema of the same name", name)
			}
		}
		schemas[name] = obj{"type": "string", "enum": vals}
	}
	for name, s := range schemas {
		t, ok := p.byName[name]
		if !ok {
			t, ok = p.byName[strings.TrimSuffix(name, "Body")] // ekman's body-only variant
		}
		if m, isObj := s.(obj); ok && isObj {
			p.patchStruct(m, t)
		}
	}

	paths, _ := doc["paths"].(obj)
	for _, r := range routes {
		if r.In == nil || r.Method == "" {
			continue
		}
		item, _ := paths[openAPIPath(r.Pattern)].(obj)
		op, _ := item[strings.ToLower(r.Method)].(obj)
		if op == nil {
			continue
		}
		params, _ := op["parameters"].([]any)
		for _, prm := range params {
			m, _ := prm.(obj)
			if f, ok := paramField(r.In, m["in"], m["name"]); ok {
				if s := p.schemaFor(deref(f.Type)); s != nil { // an absent parameter is just absent
					m["schema"] = s
				}
			}
		}
		if in := reflect.PointerTo(r.In); in.Implements(multipartType) {
			field := reflect.Zero(r.In).Interface().(Multipart).MultipartFile()
			op["requestBody"] = obj{"required": true, "content": obj{"multipart/form-data": obj{"schema": obj{
				"type":       "object",
				"required":   []string{field},
				"properties": obj{field: obj{"type": "string", "format": "binary"}},
			}}}}
		}
	}
	return json.MarshalIndent(doc, "", "  ")
}

type patcher struct {
	enums  map[reflect.Type][]string
	byName map[string]reflect.Type // component name -> named struct type
	seen   map[reflect.Type]bool
}

// walk records every named struct and enum type reachable from t.
func (p *patcher) walk(t reflect.Type) {
	if t == nil || p.seen[t] {
		return
	}
	p.seen[t] = true
	if t.Kind() == reflect.String && t.Implements(enumType) {
		if _, ok := p.enums[t]; !ok {
			p.enums[t] = reflect.Zero(t).Interface().(Enum).Enum()
		}
		return
	}
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice, reflect.Array, reflect.Map:
		p.walk(t.Elem())
	case reflect.Struct:
		if t.Implements(jsonMarshaler) || reflect.PointerTo(t).Implements(jsonMarshaler) {
			return
		}
		if t.Name() != "" {
			p.byName[typeName(t)] = t
		}
		for _, f := range reflect.VisibleFields(t) {
			if f.IsExported() {
				p.walk(f.Type)
			}
		}
	}
}

// patchStruct points the properties of t's schema whose Go type involves an enum at it.
func (p *patcher) patchStruct(s obj, t reflect.Type) {
	props, _ := s["properties"].(obj)
	if props == nil {
		return
	}
	for _, f := range reflect.VisibleFields(t) {
		if !f.IsExported() || (f.Anonymous && deref(f.Type).Kind() == reflect.Struct) {
			continue
		}
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "-" || hasOpt(opts, "string") {
			continue
		}
		if name == "" {
			name = f.Name
		}
		if _, ok := props[name]; !ok {
			continue // a path, query or header parameter, not a body property
		}
		if es := p.schemaFor(f.Type); es != nil {
			props[name] = es
		}
	}
}

// schemaFor is t's schema when t involves an enum type, else nil (keep ekman's).
func (p *patcher) schemaFor(t reflect.Type) obj {
	if _, ok := p.enums[t]; ok {
		return obj{"$ref": "#/components/schemas/" + typeName(t)}
	}
	switch t.Kind() {
	case reflect.Pointer:
		if inner := p.schemaFor(t.Elem()); inner != nil {
			return obj{"anyOf": []any{inner, obj{"type": "null"}}}
		}
	case reflect.Slice, reflect.Array:
		if inner := p.schemaFor(t.Elem()); inner != nil {
			return obj{"type": "array", "items": inner}
		}
	case reflect.Map:
		if inner := p.schemaFor(t.Elem()); inner != nil {
			return obj{"type": "object", "additionalProperties": inner}
		}
	}
	return nil
}

func paramField(in reflect.Type, where, name any) (reflect.StructField, bool) {
	if deref(in).Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	src, _ := where.(string)
	want, _ := name.(string)
	for _, f := range reflect.VisibleFields(deref(in)) {
		if tag, ok := f.Tag.Lookup(src); ok && strings.EqualFold(strings.Split(tag, ",")[0], want) {
			return f, true
		}
	}
	return reflect.StructField{}, false
}

// typeName matches ekman's component names: "<package>.<Type>".
func typeName(t reflect.Type) string {
	n := t.Name()
	if pkg := path.Base(t.PkgPath()); pkg != "." && pkg != "" {
		n = pkg + "." + n
	}
	return strings.Map(func(r rune) rune {
		if r == '.' || r == '-' || r == '_' || r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)) {
			return r
		}
		return '_'
	}, n)
}

// openAPIPath matches ekman's: ServeMux wildcards {name...} and {$} become OpenAPI templates.
func openAPIPath(p string) string {
	p = strings.ReplaceAll(p, "...}", "}")
	return strings.TrimSuffix(p, "{$}")
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func hasOpt(opts, want string) bool {
	for _, o := range strings.Split(opts, ",") {
		if o == want {
			return true
		}
	}
	return false
}
