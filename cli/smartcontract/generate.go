package smartcontract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"reflect"
	"strings"

	"github.com/nspcc-dev/neo-go/internal/random"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract/manifest"
	"github.com/nspcc-dev/neo-go/pkg/util"
	"github.com/urfave/cli"
)

const srcTmpl = `
{{- define "HEADER" -}}
// {{.Name}} {{.Comment}}
func (c *Client) {{.Name}}({{range $index, $arg := .Arguments -}}
	{{- if ne $index 0}}, {{end}}
		{{- .Name}} {{scTypeToGo .Type}}
	{{- end}}) {{if .ReturnType }}({{ .ReturnType }}, error){{else}}error{{end}} 
{{- end -}}

{{- define "ARGS" -}}
	args := make([]smartcontract.Parameter, {{ len .Arguments }})
	{{range $index, $arg := .Arguments -}}
	args[{{$index}}] = smartcontract.Parameter{Type: {{ scType $arg.Type }}, Value: {{ scName $arg.Type $arg.Name -}} }
	{{end}}
{{- end -}}

{{- define "CHECKRETURN" -}}
	if err != nil {
		return {{if .ReturnType }}{{ .ReturnValue }}, {{end}}err
	}
{{- end -}}

{{- define "SAFE" -}}
{{ template "HEADER" . }} {
	{{ if .Arguments }}{{ template "ARGS" . }}{{- else -}}{{end}}
	result, err := (*client.Client)(c).InvokeFunction(contractHash, "
		{{- lowerFirst .Name }}", {{if .Arguments}}args{{else}}nil{{end}}, nil)
	{{ template "CHECKRETURN" . }}

	{{if .ReturnType -}}
	err = client.GetInvocationError(result)
	{{ template "CHECKRETURN" . }}

	return {{ .Converter }}(result.Stack)
	{{- else -}}
	return client.GetInvocationError(result)
	{{- end}}
}
{{- end -}}
package {{.PackageName}}

import (
	"github.com/nspcc-dev/neo-go/pkg/rpc/client"
	"github.com/nspcc-dev/neo-go/pkg/smartcontract"
	"github.com/nspcc-dev/neo-go/pkg/util"
{{range $m, $key := .Imports}}	"{{ $m }}"
{{end}})

var contractHash = {{ printf "%#v" .Hash }}

// Client is a wrapper over RPC-client mirroring methods of smartcontract.
type Client client.Client
{{range $m := .SafeMethods}}
{{template "SAFE" $m }}
{{end}}`

func printValue(v interface{}) string {
	if v == nil {
		return "nil"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Interface, reflect.Slice:
		if rv.IsNil() {
			return "nil"
		}
	case reflect.String:
		return "``"
	}
	return fmt.Sprintf("%#v", v)
}

func printType(v interface{}) string {
	if v == nil {
		return "interface{}"
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Map, reflect.Interface, reflect.Slice:
		if rv.IsNil() {
			return "interface{}"
		}
	}
	return fmt.Sprintf("%T", v)
}

func scType(s smartcontract.ParamType) string {
	switch s {
	case smartcontract.AnyType:
		return "smartcontract.AnyType"
	case smartcontract.BoolType:
		return "smartcontract.BoolType"
	case smartcontract.IntegerType:
		return "smartcontract.IntegerType"
	case smartcontract.ByteArrayType:
		return "smartcontract.ByteArrayType"
	case smartcontract.StringType:
		return "smartcontract.StringType"
	case smartcontract.Hash160Type:
		return "smartcontract.Hash160Type"
	case smartcontract.Hash256Type:
		return "smartcontract.Hash256Type"
	case smartcontract.PublicKeyType:
		return "smartcontract.PublicKeyType"
	case smartcontract.SignatureType:
		return "smartcontract.SignatureType"
	case smartcontract.ArrayType:
		return "smartcontract.ArrayType"
	case smartcontract.MapType:
		return "smartcontract.MapType"
	case smartcontract.InteropInterfaceType:
		return "smartcontract.InteropInterfaceType"
	case smartcontract.VoidType:
		return ""
	default:
		return "smartcontract.AnyType"
	}
}

func scName(typ smartcontract.ParamType, name string) string {
	switch typ {
	case smartcontract.Hash160Type, smartcontract.Hash256Type:
		return name + ".BytesBE()"
	default:
		return name
	}
}

func scTypeToGo(typ smartcontract.ParamType) (string, string, string) {
	switch typ {
	case smartcontract.AnyType, smartcontract.InteropInterfaceType:
		return "interface{}", "nil", "client.TopItemFromStack"
	case smartcontract.BoolType:
		return "bool", "false", "client.TopBoolFromStack"
	case smartcontract.IntegerType:
		return "int64", "0", "client.TopIntFromStack"
	case smartcontract.ByteArrayType, smartcontract.SignatureType, smartcontract.PublicKeyType:
		return "[]byte", "nil", "client.TopBytesFromStack"
	case smartcontract.StringType:
		return "string", "``", "client.TopStringFromStack"
	case smartcontract.Hash160Type:
		return "util.Uint160", "util.Uint160{}", "client.TopUint160FromStack"
	case smartcontract.Hash256Type:
		return "util.Uint256", "util.Uint256{}", "client.TopUint256FromStack"
	case smartcontract.ArrayType:
		return "[]stackitem.Item", "nil", "client.TopArrayFromStack"
	case smartcontract.MapType:
		return "*stackitem.Map", "nil", "client.TopMapFromStack"
	case smartcontract.VoidType:
		return "", "", ""
	default:
		panic(fmt.Sprintf("unexpected type: %s", typ))
	}
}

func upperFirst(s string) string {
	return strings.ToUpper(s[0:1]) + s[1:]
}
func lowerFirst(s string) string {
	return strings.ToLower(s[0:1]) + s[1:]
}

func Generate(arg contractTmpl) (string, error) {
	fm := template.FuncMap{
		"lowerFirst": lowerFirst,
		"scType":     scType,
		"scName":     scName,
		"scTypeToGo": func(s smartcontract.ParamType) string {
			typ, _, _ := scTypeToGo(s)
			return typ
		},
		"printValue": printValue,
		"printType":  printType,
	}
	tmp := template.New("test").Funcs(fm)
	tmp, err := tmp.Parse(srcTmpl)
	if err != nil {
		return "", err
	}
	b := bytes.NewBuffer(nil)
	if err := tmp.Execute(b, arg); err != nil {
		return "", err
	}
	return b.String(), nil
}

type (
	contractTmpl struct {
		PackageName string
		Imports     map[string]struct{}
		Hash        util.Uint160
		SafeMethods []methodTmpl
	}

	methodTmpl struct {
		Name        string
		Comment     string
		Arguments   []manifest.Parameter
		ReturnType  string
		ReturnValue string
		Converter   string
	}
)

// contractGenerateWrapper deploys contract.
func contractGenerateWrapper(ctx *cli.Context) error {
	manifestFile := ctx.String("manifest")
	if len(manifestFile) == 0 {
		return cli.NewExitError(errNoManifestFile, 1)
	}

	manifestBytes, err := ioutil.ReadFile(manifestFile)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to read manifest file: %w", err), 1)
	}

	m := &manifest.Manifest{}
	err = json.Unmarshal(manifestBytes, m)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("failed to restore manifest file: %w", err), 1)
	}

	ctr := contractTmpl{
		PackageName: ctx.String("package"),
		Imports:     map[string]struct{}{},
		Hash:        random.Uint160(),
	}

	converters := make(map[string]string)
	for _, s := range ctx.StringSlice("return") {
		ss := strings.SplitN(s, ":", 2)
		if len(ss) != 2 {
			return cli.NewExitError(fmt.Errorf("invalid return override: %s", s), 1)
		}
		converters[ss[0]] = ss[1]
	}

	for _, m := range m.ABI.Methods {
		if m.Name[0] == '_' || !m.Safe {
			continue
		}
		typ, val, conv := scTypeToGo(m.ReturnType)
		if m.ReturnType == smartcontract.MapType || m.ReturnType == smartcontract.ArrayType {
			ctr.Imports["github.com/nspcc-dev/neo-go/pkg/vm/stackitem"] = struct{}{}
		}
		mtd := methodTmpl{
			Name:        upperFirst(m.Name),
			ReturnType:  typ,
			ReturnValue: val,
			Comment:     fmt.Sprintf("invokes `%s` method of contract.", m.Name),
			Arguments:   m.Parameters,
			Converter:   conv,
		}
		if c, ok := converters[m.Name]; ok {
			switch c {
			case "iterator":
				mtd.Converter = "client.TopIterableFromStack"
				mtd.ReturnType = "[]interface{}"
				mtd.ReturnValue = "nil"
			case "keys":
				mtd.Converter = "client.TopPublicKeysFromStack"
				mtd.ReturnType = "keys.PublicKeys"
				mtd.ReturnValue = "nil"
			}
		} else {
		}
		ctr.SafeMethods = append(ctr.SafeMethods, mtd)
	}

	s, err := Generate(ctr)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("error during generation: %w", err), 1)
	}

	err = ioutil.WriteFile(ctx.String("out"), []byte(s), os.ModePerm)
	if err != nil {
		return cli.NewExitError(fmt.Errorf("error during write: %w", err), 1)
	}
	return nil
}
