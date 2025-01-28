//go:build ignore

package main

import (
	"os"
	"path"
	"runtime"
	"text/template"

	"fyne.io/fyne/v2"
)

const toStringTemplate = `
type stringFrom{{ .Name }} struct {
	base
{{ if .Format }}
	format string
{{ end }}
	from {{ .Name }}
}

// {{ .Name }}ToString creates a binding that connects a {{ .Name }} data item to a String.
// Changes to the {{ .Name }} will be pushed to the String and setting the string will parse and set the
// {{ .Name }} if the parse was successful.
//
// Since: {{ .Since }}
func {{ .Name }}ToString(v {{ .Name }}) String {
	str := &stringFrom{{ .Name }}{from: v}
	v.AddListener(str)
	return str
}
{{ if .Format }}
// {{ .Name }}ToStringWithFormat creates a binding that connects a {{ .Name }} data item to a String and is
// presented using the specified format. Changes to the {{ .Name }} will be pushed to the String and setting
// the string will parse and set the {{ .Name }} if the string matches the format and its parse was successful.
//
// Since: {{ .Since }}
func {{ .Name }}ToStringWithFormat(v {{ .Name }}, format string) String {
	if format == "{{ .Format }}" { // Same as not using custom formatting.
		return {{ .Name }}ToString(v)
	}

	str := &stringFrom{{ .Name }}{from: v, format: format}
	v.AddListener(str)
	return str
}
{{ end }}
func (s *stringFrom{{ .Name }}) Get() (string, error) {
	val, err := s.from.Get()
	if err != nil {
		return "", err
	}
{{ if .ToString }}
	return {{ .ToString }}(val)
{{- else }}
	if s.format != "" {
		return fmt.Sprintf(s.format, val), nil
	}

	return format{{ .Name }}(val), nil
{{- end }}
}

func (s *stringFrom{{ .Name }}) Set(str string) error {
{{- if .FromString }}
	val, err := {{ .FromString }}(str)
	if err != nil {
		return err
	}
{{ else }}
	var val {{ .Type }}
	if s.format != "" {
		safe := stripFormatPrecision(s.format)
		n, err := fmt.Sscanf(str, safe+" ", &val) // " " denotes match to end of string
		if err != nil {
			return err
		}
		if n != 1 {
			return errParseFailed
		}
	} else {
		new, err := parse{{ .Name }}(str)
		if err != nil {
			return err
		}
		val = new
	}
{{ end }}
	old, err := s.from.Get()
	if err != nil {
		return err
	}
	if val == old {
		return nil
	}
	if err = s.from.Set(val); err != nil {
		return err
	}

	queueItem(s.DataChanged)
	return nil
}

func (s *stringFrom{{ .Name }}) DataChanged() {
	s.trigger()
}
`
const toIntTemplate = `
type intFrom{{ .Name }} struct {
	base
	from {{ .Name }}
}

// {{ .Name }}ToInt creates a binding that connects a {{ .Name }} data item to an Int.
//
// Since: 2.5
func {{ .Name }}ToInt(v {{ .Name }}) Int {
	i := &intFrom{{ .Name }}{from: v}
	v.AddListener(i)
	return i
}

func (s *intFrom{{ .Name }}) Get() (int, error) {
	val, err := s.from.Get()
	if err != nil {
		return 0, err
	}
	return {{ .ToInt }}(val)
}

func (s *intFrom{{ .Name }}) Set(v int) error {
	val, err := {{ .FromInt }}(v)
	if err != nil {
		return err
	}

	old, err := s.from.Get()
	if err != nil {
		return err
	}
	if val == old {
		return nil
	}
	if err = s.from.Set(val); err != nil {
		return err
	}

	queueItem(s.DataChanged)
	return nil
}

func (s *intFrom{{ .Name }}) DataChanged() {
	s.trigger()
}
`
const fromIntTemplate = `
type intTo{{ .Name }} struct {
	base
	from Int
}

// IntTo{{ .Name }} creates a binding that connects an Int data item to a {{ .Name }}.
//
// Since: 2.5
func IntTo{{ .Name }}(val Int) {{ .Name }} {
	v := &intTo{{ .Name }}{from: val}
	val.AddListener(v)
	return v
}

func (s *intTo{{ .Name }}) Get() ({{ .Type }}, error) {
	val, err := s.from.Get()
	if err != nil {
		return {{ .Default }}, err
	}
	return {{ .FromInt }}(val)
}

func (s *intTo{{ .Name }}) Set(val {{ .Type }}) error {
	i, err := {{ .ToInt }}(val)
	if err != nil {
		return err
	}
	old, err := s.from.Get()
	if i == old {
		return nil
	}
	if err != nil {
		return err
	}
	if err = s.from.Set(i); err != nil {
		return err
	}

	queueItem(s.DataChanged)
	return nil
}

func (s *intTo{{ .Name }}) DataChanged() {
	s.trigger()
}
`
const fromStringTemplate = `
type stringTo{{ .Name }} struct {
	base
{{ if .Format }}
	format string
{{ end }}
	from String
}

// StringTo{{ .Name }} creates a binding that connects a String data item to a {{ .Name }}.
// Changes to the String will be parsed and pushed to the {{ .Name }} if the parse was successful, and setting
// the {{ .Name }} update the String binding.
//
// Since: {{ .Since }}
func StringTo{{ .Name }}(str String) {{ .Name }} {
	v := &stringTo{{ .Name }}{from: str}
	str.AddListener(v)
	return v
}
{{ if .Format }}
// StringTo{{ .Name }}WithFormat creates a binding that connects a String data item to a {{ .Name }} and is
// presented using the specified format. Changes to the {{ .Name }} will be parsed and if the format matches and
// the parse is successful it will be pushed to the String. Setting the {{ .Name }} will push a formatted value
// into the String.
//
// Since: {{ .Since }}
func StringTo{{ .Name }}WithFormat(str String, format string) {{ .Name }} {
	if format == "{{ .Format }}" { // Same as not using custom format.
		return StringTo{{ .Name }}(str)
	}

	v := &stringTo{{ .Name }}{from: str, format: format}
	str.AddListener(v)
	return v
}
{{ end }}
func (s *stringTo{{ .Name }}) Get() ({{ .Type }}, error) {
	str, err := s.from.Get()
	if str == "" || err != nil {
		return {{ .Default }}, err
	}
{{ if .FromString }}
	return {{ .FromString }}(str)
{{- else }}
	var val {{ .Type }}
	if s.format != "" {
		n, err := fmt.Sscanf(str, s.format+" ", &val) // " " denotes match to end of string
		if err != nil {
			return {{ .Default }}, err
		}
		if n != 1 {
			return {{ .Default }}, errParseFailed
		}
	} else {
		new, err := parse{{ .Name }}(str)
		if err != nil {
			return {{ .Default }}, err
		}
		val = new
	}

	return val, nil
{{- end }}
}

func (s *stringTo{{ .Name }}) Set(val {{ .Type }}) error {
{{- if .ToString }}
	str, err := {{ .ToString }}(val)
	if err != nil {
		return err
	}
{{- else }}
	var str string
	if s.format != "" {
		str = fmt.Sprintf(s.format, val)
	} else {
		str = format{{ .Name }}(val)
	}
{{ end }}
	old, err := s.from.Get()
	if str == old {
		return err
	}

	if err = s.from.Set(str); err != nil {
		return err
	}

	queueItem(s.DataChanged)
	return nil
}

func (s *stringTo{{ .Name }}) DataChanged() {
	s.trigger()
}
`

const treeBindTemplate = `
// {{ .Name }}Tree supports binding a tree of {{ .Type }} values.
//
{{ if eq .Name "Untyped" -}}
// Since: 2.5
{{- else -}}
// Since: 2.4
{{- end }}
type {{ .Name }}Tree interface {
	DataTree

	Append(parent, id string, value {{ .Type }}) error
	Get() (map[string][]string, map[string]{{ .Type }}, error)
	GetValue(id string) ({{ .Type }}, error)
	Prepend(parent, id string, value {{ .Type }}) error
	Remove(id string) error
	Set(ids map[string][]string, values map[string]{{ .Type }}) error
	SetValue(id string, value {{ .Type }}) error
}

// External{{ .Name }}Tree supports binding a tree of {{ .Type }} values from an external variable.
//
{{ if eq .Name "Untyped" -}}
// Since: 2.5
{{- else -}}
// Since: 2.4
{{- end }}
type External{{ .Name }}Tree interface {
	{{ .Name }}Tree

	Reload() error
}

// New{{ .Name }}Tree returns a bindable tree of {{ .Type }} values.
//
{{ if eq .Name "Untyped" -}}
// Since: 2.5
{{- else -}}
// Since: 2.4
{{- end }}
func New{{ .Name }}Tree() {{ .Name }}Tree {
	t := &bound{{ .Name }}Tree{val: &map[string]{{ .Type }}{}}
	t.ids = make(map[string][]string)
	t.items = make(map[string]DataItem)
	return t
}

// Bind{{ .Name }}Tree returns a bound tree of {{ .Type }} values, based on the contents of the passed values.
// The ids map specifies how each item relates to its parent (with id ""), with the values being in the v map.
// If your code changes the content of the maps this refers to you should call Reload() to inform the bindings.
//
// Since: 2.4
func Bind{{ .Name }}Tree(ids *map[string][]string, v *map[string]{{ .Type }}) External{{ .Name }}Tree {
	if v == nil {
		return New{{ .Name }}Tree().(External{{ .Name }}Tree)
	}

	t := &bound{{ .Name }}Tree{val: v, updateExternal: true}
	t.ids = make(map[string][]string)
	t.items = make(map[string]DataItem)

	for parent, children := range *ids {
		for _, leaf := range children {
			t.appendItem(bind{{ .Name }}TreeItem(v, leaf, t.updateExternal), leaf, parent)
		}
	}

	return t
}

type bound{{ .Name }}Tree struct {
	treeBase

	updateExternal bool
	val            *map[string]{{ .Type }}
}

func (t *bound{{ .Name }}Tree) Append(parent, id string, val {{ .Type }}) error {
	t.lock.Lock()
	ids, ok := t.ids[parent]
	if !ok {
		ids = make([]string, 0)
	}

	t.ids[parent] = append(ids, id)
	v := *t.val
	v[id] = val

	trigger, err := t.doReload()
	t.lock.Unlock()

	if trigger {
		t.trigger()
	}

	return err
}

func (t *bound{{ .Name }}Tree) Get() (map[string][]string, map[string]{{ .Type }}, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.ids, *t.val, nil
}

func (t *bound{{ .Name }}Tree) GetValue(id string) ({{ .Type }}, error) {
	t.lock.RLock()
	defer t.lock.RUnlock()

	if item, ok := (*t.val)[id]; ok {
		return item, nil
	}

	return {{ .Default }}, errOutOfBounds
}

func (t *bound{{ .Name }}Tree) Prepend(parent, id string, val {{ .Type }}) error {
	t.lock.Lock()
	ids, ok := t.ids[parent]
	if !ok {
		ids = make([]string, 0)
	}

	t.ids[parent] = append([]string{id}, ids...)
	v := *t.val
	v[id] = val

	trigger, err := t.doReload()
	t.lock.Unlock()

	if trigger {
		t.trigger()
	}

	return err
}

// Remove takes the specified id out of the tree.
// It will also remove any child items from the data structure.
//
// Since: 2.5
func (t *bound{{ .Name }}Tree) Remove(id string) error {
	t.lock.Lock()
	t.removeChildren(id)
	delete(t.ids, id)
	v := *t.val
	delete(v, id)

	trigger, err := t.doReload()
	t.lock.Unlock()

	if trigger {
		t.trigger()
	}

	return err
}

func (t *bound{{ .Name }}Tree) removeChildren(id string) {
	for _, cid := range t.ids[id] {
		t.removeChildren(cid)

		delete(t.ids, cid)
		v := *t.val
		delete(v, cid)
	}
}

func (t *bound{{ .Name }}Tree) Reload() error {
	t.lock.Lock()
	trigger, err := t.doReload()
	t.lock.Unlock()

	if trigger {
		t.trigger()
	}

	return err
}

func (t *bound{{ .Name }}Tree) Set(ids map[string][]string, v map[string]{{ .Type }}) error {
	t.lock.Lock()
	t.ids = ids
	*t.val = v

	trigger, err := t.doReload()
	t.lock.Unlock()

	if trigger {
		t.trigger()
	}

	return err
}

func (t *bound{{ .Name }}Tree) doReload() (fire bool, retErr error) {
	updated := []string{}
	for id := range *t.val {
		found := false
		for child := range t.items {
			if child == id { // update existing
				updated = append(updated, id)
				found = true
				break
			}
		}
		if found {
			continue
		}

		// append new
		t.appendItem(bind{{ .Name }}TreeItem(t.val, id, t.updateExternal), id, parentIDFor(id, t.ids))
		updated = append(updated, id)
		fire = true
	}

	for id := range t.items {
		remove := true
		for _, done := range updated {
			if done == id {
				remove = false
				break
			}
		}

		if remove { // remove item no longer present
			fire = true
			t.deleteItem(id, parentIDFor(id, t.ids))
		}
	}

	for id, item := range t.items {
		var err error
		if t.updateExternal {
			err = item.(*boundExternal{{ .Name }}TreeItem).setIfChanged((*t.val)[id])
		} else {
			err = item.(*bound{{ .Name }}TreeItem).doSet((*t.val)[id])
		}
		if err != nil {
			retErr = err
		}
	}
	return
}

func (t *bound{{ .Name }}Tree) SetValue(id string, v {{ .Type }}) error {
	t.lock.Lock()
	(*t.val)[id] = v
	t.lock.Unlock()

	item, err := t.GetItem(id)
	if err != nil {
		return err
	}
	return item.({{ .Name }}).Set(v)
}

func bind{{ .Name }}TreeItem(v *map[string]{{ .Type }}, id string, external bool) {{ .Name }} {
	if external {
		ret := &boundExternal{{ .Name }}TreeItem{old: (*v)[id]}
		ret.val = v
		ret.id = id
		return ret
	}

	return &bound{{ .Name }}TreeItem{id: id, val: v}
}

type bound{{ .Name }}TreeItem struct {
	base

	val *map[string]{{ .Type }}
	id  string
}

func (t *bound{{ .Name }}TreeItem) Get() ({{ .Type }}, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	v := *t.val
	if item, ok := v[t.id]; ok {
		return item, nil
	}

	return {{ .Default }}, errOutOfBounds
}

func (t *bound{{ .Name }}TreeItem) Set(val {{ .Type }}) error {
	return t.doSet(val)
}

func (t *bound{{ .Name }}TreeItem) doSet(val {{ .Type }}) error {
	t.lock.Lock()
	(*t.val)[t.id] = val
	t.lock.Unlock()

	t.trigger()
	return nil
}

type boundExternal{{ .Name }}TreeItem struct {
	bound{{ .Name }}TreeItem

	old {{ .Type }}
}

func (t *boundExternal{{ .Name }}TreeItem) setIfChanged(val {{ .Type }}) error {
	t.lock.Lock()
	{{- if eq .Comparator "" }}
	if val == t.old {
		t.lock.Unlock()
		return nil
	}
	{{- else }}
	if {{ .Comparator }}(val, t.old) {
		t.lock.Unlock()
		return nil
	}
	{{- end }}
	(*t.val)[t.id] = val
	t.old = val
	t.lock.Unlock()

	t.trigger()
	return nil
}
`

type bindValues struct {
	Name, Type, Default  string
	Format, Since        string
	SupportsPreferences  bool
	FromString, ToString string // function names...
	Comparator           string // comparator function name
	FromInt, ToInt       string // function names...
}

func newFile(name string) (*os.File, error) {
	_, dirname, _, _ := runtime.Caller(0)
	filepath := path.Join(path.Dir(dirname), name+".go")
	os.Remove(filepath)
	f, err := os.Create(filepath)
	if err != nil {
		fyne.LogError("Unable to open file "+f.Name(), err)
		return nil, err
	}

	f.WriteString(`// auto-generated
// **** THIS FILE IS AUTO-GENERATED, PLEASE DO NOT EDIT IT **** //

package binding
`)
	return f, nil
}

func writeFile(f *os.File, t *template.Template, d any) {
	if err := t.Execute(f, d); err != nil {
		fyne.LogError("Unable to write file "+f.Name(), err)
	}
}

func main() {
	convertFile, err := newFile("convert")
	if err != nil {
		return
	}
	defer convertFile.Close()
	convertFile.WriteString(`
import (
	"fmt"

	"fyne.io/fyne/v2"
)

func internalFloatToInt(val float64) (int, error) {
	return int(val), nil
}

func internalIntToFloat(val int) (float64, error) {
	return float64(val), nil
}
`)

	listFile, err := newFile("bindlists")
	if err != nil {
		return
	}
	defer listFile.Close()
	listFile.WriteString(`
import (
	"bytes"

	"fyne.io/fyne/v2"
)
`)

	treeFile, err := newFile("bindtrees")
	if err != nil {
		return
	}
	defer treeFile.Close()
	treeFile.WriteString(`
import (
	"bytes"

	"fyne.io/fyne/v2"
)
`)

	fromString := template.Must(template.New("fromString").Parse(fromStringTemplate))
	fromInt := template.Must(template.New("fromInt").Parse(fromIntTemplate))
	toInt := template.Must(template.New("toInt").Parse(toIntTemplate))
	toString := template.Must(template.New("toString").Parse(toStringTemplate))
	tree := template.Must(template.New("tree").Parse(treeBindTemplate))
	binds := []bindValues{
		{Name: "Bool", Type: "bool", Default: "false", Format: "%t", SupportsPreferences: true},
		{Name: "Bytes", Type: "[]byte", Default: "nil", Since: "2.2", Comparator: "bytes.Equal"},
		{Name: "Float", Type: "float64", Default: "0.0", Format: "%f", SupportsPreferences: true, ToInt: "internalFloatToInt", FromInt: "internalIntToFloat"},
		{Name: "Int", Type: "int", Default: "0", Format: "%d", SupportsPreferences: true},
		{Name: "Rune", Type: "rune", Default: "rune(0)"},
		{Name: "String", Type: "string", Default: "\"\"", SupportsPreferences: true},
		{Name: "Untyped", Type: "any", Default: "nil", Since: "2.1"},
		{Name: "URI", Type: "fyne.URI", Default: "fyne.URI(nil)", Since: "2.1",
			FromString: "uriFromString", ToString: "uriToString", Comparator: "compareURI"},
	}
	for _, b := range binds {
		if b.Since == "" {
			b.Since = "2.0"
		}

		writeFile(treeFile, tree, b)
		if b.Name == "Untyped" {
			continue // any is special, we have it in binding.go instead
		}

		if b.Format != "" || b.ToString != "" {
			writeFile(convertFile, toString, b)
		}
		if b.FromInt != "" {
			writeFile(convertFile, fromInt, b)
		}
		if b.ToInt != "" {
			writeFile(convertFile, toInt, b)
		}
	}
	// add StringTo... at the bottom of the convertFile for correct ordering
	for _, b := range binds {
		if b.Since == "" {
			b.Since = "2.0"
		}

		if b.Format != "" || b.FromString != "" {
			writeFile(convertFile, fromString, b)
		}
	}
}
