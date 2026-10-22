package hfd

// buildBindingsSchema duyệt metadata của từng BindTyped bằng reflect và sinh
// một node graph JSON mô tả kiểu In/Out, để CLI viết bindings *.d.ts type-safe.
//
// Node graph được dùng chung với hfd-cli/src/ts-emitter.mjs — đổi format ở đây
// thì phải đổi bên đó.

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
	// Name tên Go type (struct có tên hoặc type được đặt tên như `type Status string`).
	Name string `json:"name,omitempty"`
	// Pkg package của type đặt tên (rỗng nếu builtin/anonymous).
	Pkg  string      `json:"pkg,omitempty"`
	Elem *schemaNode `json:"elem,omitempty"` // array/ptr
	Key  *schemaNode `json:"key,omitempty"`  // map
	Val  *schemaNode `json:"val,omitempty"`  // map
	// Fields của struct.
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

// buildBindingsSchema in JSON stdout cho CLI.
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

// walkSchemaType chuyển reflect.Type → schemaNode. `seen` chặn struct tự tham
// chiếu vô hạn (cycle → kind "cyclic", CLI sẽ emit unknown).
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
			// struct nhúng (embedded) không có json tag → promote field.
			if f.Anonymous && f.Tag.Get("json") == "" && f.Type.Kind() == reflect.Struct {
				inner := walkSchemaType(f.Type, seen)
				n.Fields = append(n.Fields, inner.Fields...)
				continue
			}
			name := f.Name
			if tag, ok := f.Tag.Lookup("json"); ok {
				name = parseJSONName(tag, f.Name)
				if name == "" { // json:"-" → bỏ qua
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
			// type đặt tên thật (vd `type Status string`) — không phải builtin.
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

	default: // chan, func, complex… không map được qua JSON
		return &schemaNode{Kind: "any", Name: t.Name(), Pkg: t.PkgPath()}
	}
}

// parseJSONName đọc json tag: `,omitempty`… biến thể.
func parseJSONName(tag, fallback string) string {
	for i := 0; i < len(tag); i++ {
		if tag[i] == ',' {
			tag = tag[:i]
			break
		}
	}
	if tag == "" {
		return fallback // json:"" → giữ tên field gốc
	}
	return tag
}
