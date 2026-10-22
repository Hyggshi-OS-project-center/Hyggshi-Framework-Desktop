package hfd

// buildBindingsSchema walks the metadata of each BindTyped via reflect and builds
// a JSON node graph describing the In/Out types, so the CLI can emit type-safe *.d.ts bindings.
//
// The node graph format is shared with hfd-cli/src/ts-emitter.mjs — if you change
// the format here, you must change it there too.

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

var timeType = reflect.TypeOf(time.Time{})

type schemaNode struct {
	// Kind: struct | array | map | ptr | string | number | bool | any | cyclic
	Kind string `json:"kind"`
	// Name is the Go type name (a named struct or a named type such as `type Status string`).
	Name string `json:"name,omitempty"`
	// Pkg is the package of the named type (empty for builtin/anonymous).
	Pkg  string      `json:"pkg,omitempty"`
	Elem *schemaNode `json:"elem,omitempty"` // array/ptr
	Key  *schemaNode `json:"key,omitempty"`  // map
	Val  *schemaNode `json:"val,omitempty"`  // map
	// Fields of the struct.
	Fields []schemaField `json:"fields,omitempty"`
}

type schemaField struct {
	Name string      `json:"name"`
	Type *schemaNode `json:"type"`
}

type bindingSchema struct {
	Name string      `json:"name"`
	In   *schemaNode `json:"in"`
	Out  *schemaNode `json:"out"`
}

type bindingsSchema struct {
	Bindings []bindingSchema `json:"bindings"`
}

// buildBindingsSchema prints the JSON to stdout for the CLI.
func buildBindingsSchema() []byte {
	typedMu.Lock()
	entries := make([]typedEntry, len(typedEntries))
	copy(entries, typedEntries)
	typedMu.Unlock()

	s := bindingsSchema{}
	for _, e := range entries {
		seen := map[reflect.Type]bool{}
		s.Bindings = append(s.Bindings, bindingSchema{
			Name: e.Name,
			In:   walkSchemaType(e.In, seen),
			Out:  walkSchemaType(e.Out, seen),
		})
	}

	data, err := json.Marshal(s)
	if err != nil {
		return []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
	}
	return append(data, '\n')
}

// walkSchemaType converts a reflect.Type → schemaNode. `seen` prevents infinite
// self-referencing structs (cycle → kind "cyclic", the CLI emits unknown).
func walkSchemaType(t reflect.Type, seen map[reflect.Type]bool) *schemaNode {
	switch t.Kind() {
	case reflect.Struct:
		if t == timeType {
			return &schemaNode{Kind: "string", Name: "Time", Pkg: "time"}
		}
		n := &schemaNode{Kind: "struct", Name: t.Name(), Pkg: t.PkgPath()}
		if seen[t] && n.Name != "" {
			return &schemaNode{Kind: "cyclic", Name: n.Name, Pkg: n.Pkg}
		}
		seen[t] = true
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			// embedded struct (no json tag) → promote fields.
			if f.Anonymous && f.Tag.Get("json") == "" && f.Type.Kind() == reflect.Struct {
				inner := walkSchemaType(f.Type, seen)
				n.Fields = append(n.Fields, inner.Fields...)
				continue
			}
			name := f.Name
			if tag, ok := f.Tag.Lookup("json"); ok {
				name = parseJSONName(tag, f.Name)
				if name == "" { // json:"-" → skip
					continue
				}
			}
			n.Fields = append(n.Fields, schemaField{
				Name: name,
				Type: walkSchemaType(f.Type, seen),
			})
		}
		return n

	case reflect.Pointer:
		return &schemaNode{Kind: "ptr", Elem: walkSchemaType(t.Elem(), seen)}

	case reflect.Slice, reflect.Array:
		return &schemaNode{Kind: "array", Elem: walkSchemaType(t.Elem(), seen)}

	case reflect.Map:
		return &schemaNode{
			Kind: "map",
			Key:  walkSchemaType(t.Key(), seen),
			Val:  walkSchemaType(t.Elem(), seen),
		}

	case reflect.String:
		if t.Name() != "" && t.PkgPath() != "" {
			// a real named type (e.g. `type Status string`) — not builtin.
			return &schemaNode{Kind: "string", Name: t.Name(), Pkg: t.PkgPath()}
		}
		return &schemaNode{Kind: "string"}

	case reflect.Bool:
		if t.Name() != "" && t.PkgPath() != "" {
			return &schemaNode{Kind: "bool", Name: t.Name(), Pkg: t.PkgPath()}
		}
		return &schemaNode{Kind: "bool"}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Uintptr, reflect.Float32, reflect.Float64:
		if t.Name() != "" && t.PkgPath() != "" {
			return &schemaNode{Kind: "number", Name: t.Name(), Pkg: t.PkgPath()}
		}
		return &schemaNode{Kind: "number"}

	case reflect.Interface:
		return &schemaNode{Kind: "any"}

	default: // chan, func, complex… cannot be mapped through JSON
		return &schemaNode{Kind: "any", Name: t.Name(), Pkg: t.PkgPath()}
	}
}

// parseJSONName parses the json tag: `,omitempty`… variants.
func parseJSONName(tag, fallback string) string {
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			tag = tag[:i]
			break
		}
	}
	if tag == "" {
		return fallback // json:"" → keep the original field name
	}
	return tag
}
