package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/m1ker1n/go-generics"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

const CodegenLabelPrefix = `// apigen:api `
const FuncWrapperPrefix = `wrapper`
const ApiValidatorTagPrefix = `apivalidator:"`

type CodegenOptions struct {
	Url    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method"`
}

type FuncWrapper struct {
	Decl *ast.FuncDecl

	RecvName       string
	IsStarReceiver bool
	RecvTypeName   string
	FuncName       string

	Input FuncInput

	Options CodegenOptions
}

func NewFuncWrapper(f *ast.FuncDecl) (*FuncWrapper, error) {
	var (
		recvTypeName   string
		isStarReceiver bool
	)

	switch expr := f.Recv.List[0].Type.(type) {
	case *ast.StarExpr:
		isStarReceiver = true
		i, ok := (expr.X).(*ast.Ident)
		if !ok {
			return nil, fmt.Errorf("could not get receiver type name of %s", f.Name.Name)
		}
		recvTypeName = i.Name
	case *ast.Ident:
		recvTypeName = expr.Name
	}

	var options CodegenOptions
	codegenOptionLine := generics.Filter(f.Doc.List, func(comment *ast.Comment) bool {
		return strings.Contains(comment.Text, CodegenLabelPrefix)
	})[0]
	codegenOptionsJson, ok := strings.CutPrefix(codegenOptionLine.Text, CodegenLabelPrefix)
	if !ok {
		return nil, errors.New("codegen options are not provided")
	}
	err := json.Unmarshal([]byte(codegenOptionsJson), &options)
	if err != nil {
		return nil, fmt.Errorf("could not unpack codegen options: %w", err)
	}
	if options.Method == "" {
		options.Method = http.MethodGet
	}

	input, err := FuncDeclToFuncInput(f)
	if err != nil {
		return nil, fmt.Errorf("could not parse input of function")
	}

	return &FuncWrapper{
		Decl:           f,
		RecvName:       f.Recv.List[0].Names[0].Name,
		IsStarReceiver: isStarReceiver,
		RecvTypeName:   recvTypeName,
		FuncName:       f.Name.Name,
		Input:          input,
		Options:        options,
	}, nil
}

func (f *FuncWrapper) WrapperFuncName() string {
	return FuncWrapperPrefix + f.FuncName
}

type FuncInput struct {
	RecvTypeName string
	Fields       []FuncInputStructField
}

type FuncInputStructField struct {
	Name                   string
	Type                   string
	ApiValidatorTagContent string
	Validations            []string
}

// ServeHTTPWrapper always will create function with star receiver
type ServeHTTPWrapper struct {
	RecvName     string
	RecvTypeName string

	//Wrappers[URL][Method] to access some function
	Wrappers map[string]map[string]*FuncWrapper
}

var (
	packageImportsTmpl = template.Must(template.New("packageImportsTmpl").Parse(
		`package {{.}}

import (
	"encoding/json"
	"errors"
	"net/http"
)`))

	varsTmpl = template.Must(template.New("varsTmpl").Parse(
		`var (
	errUnauthorized = errors.New("unauthorized")
	errNotFound = errors.New("unknown method")
	errStatusNotAcceptable = errors.New("bad method")
)`))

	typesTmpl = template.Must(template.New("typesTmpl").Parse(strings.ReplaceAll(
		`type httpResponse struct {
	Err      string ♂json:"error"♂
	Response any    ♂json:"response,omitempty"♂
}

func (r httpResponse) write(w http.ResponseWriter, status int) {
	marshal, _ := json.Marshal(r)
	w.WriteHeader(status)
	_, _ = w.Write(marshal)
}`, `♂`, "`")))

	serveHTTPTmpl = template.Must(template.New("serveHTTPTmpl").Parse(
		`func ({{$serveRecvName := .RecvName}}{{$serveRecvName}} *{{.RecvTypeName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path { {{range $url, $methods := .Wrappers}}
	case "{{$url}}":
		switch r.Method { {{range $method, $wrapper := $methods}}
		case "{{$method}}":
			{{$serveRecvName}}.{{$wrapper.WrapperFuncName}}(w, r){{end}}
		default:
			httpResponse{Err: errStatusNotAcceptable.Error()}.write(w, http.StatusNotAcceptable)
		}{{end}}
	default:
		httpResponse{Err: errNotFound.Error()}.write(w, http.StatusNotFound)
	}
}`))

	funcTmpl = template.Must(template.New("wrapperTmpl").Parse(
		`func ({{.RecvName}} {{if .IsStarReceiver}}*{{end}}{{.RecvTypeName}}) {{.WrapperFuncName}}(w http.ResponseWriter, r *http.Request) { {{if .Options.Auth}}
	if authorized := auth(w, r); !authorized {
		return
	}{{end}}
	//get&validate
	//call {{.RecvName}}.{{.WrapperFuncName}}(r.Context(), in)
	//return results
}`))

	authTmpl = template.Must(template.New("authTmpl").Parse(
		`func auth(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("X-Auth") != "100500" {
		httpResponse{Err: errUnauthorized.Error()}.write(w, http.StatusForbidden)
		return false
	}
	return true
}`))
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, err := os.Create(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	funcDecls := generics.Map(node.Decls, DeclToFuncDecl)
	funcWrappers, err := generics.TryMap(funcDecls, NewFuncWrapper)
	if err != nil {
		log.Fatal(err)
	}

	serveWrappers := make(map[string]*ServeHTTPWrapper)
	for _, f := range funcWrappers {
		if _, serveHTTPWrapperExists := serveWrappers[f.RecvTypeName]; !serveHTTPWrapperExists {
			serveWrappers[f.RecvTypeName] = &ServeHTTPWrapper{
				RecvName:     f.RecvName,
				RecvTypeName: f.RecvTypeName,
				Wrappers:     make(map[string]map[string]*FuncWrapper),
			}
		}
		curServeWrapper := serveWrappers[f.RecvTypeName]

		if _, groupByUrlExists := curServeWrapper.Wrappers[f.Options.Url]; !groupByUrlExists {
			curServeWrapper.Wrappers[f.Options.Url] = make(map[string]*FuncWrapper)
		}
		curUrl := curServeWrapper.Wrappers[f.Options.Url]

		if _, methodOccupied := curUrl[f.Options.Method]; methodOccupied {
			log.Fatalf("2 handlers are on the same URL and Method: %s, %s", curUrl[f.Options.Method].FuncName, f.FuncName)
		}
		curUrl[f.Options.Method] = f
	}

	err = packageImportsTmpl.Execute(out, node.Name.Name)
	if err != nil {
		log.Fatal(err)
	}
	newLines(out, 2)

	err = varsTmpl.Execute(out, nil)
	if err != nil {
		log.Fatal(err)
	}
	newLines(out, 2)

	err = typesTmpl.Execute(out, nil)
	if err != nil {
		log.Fatal(err)
	}
	newLines(out, 2)

	for _, serveWrapper := range serveWrappers {
		err := serveHTTPTmpl.Execute(out, serveWrapper)
		if err != nil {
			log.Fatal(err)
		}
		newLines(out, 2)
	}

	for _, funcWrapper := range funcWrappers {
		err := funcTmpl.Execute(out, funcWrapper)
		if err != nil {
			log.Fatal(err)
		}
		newLines(out, 2)
	}

	err = authTmpl.Execute(out, nil)
	if err != nil {
		log.Fatal(err)
	}
	newLines(out, 2)

	fmt.Printf("%#v", serveWrappers)
}

func newLines(w io.Writer, amount int) {
	str := strings.Repeat("\n", amount)
	_, _ = fmt.Fprintf(w, str)
}

func DeclToFuncDecl(d ast.Decl) (*ast.FuncDecl, error) {
	r, ok := (d).(*ast.FuncDecl)
	if !ok {
		return nil, errors.New("can't cast ast.Decl to *ast.FuncDecl")
	}

	if !DeclContains1CodegenLabel(r, CodegenLabelPrefix) {
		return nil, errors.New("func has 2 codegen labels, must be only 1")
	}

	if !DeclHasReceiver(r) {
		return nil, errors.New("function must have receiver")
	}

	return r, nil
}

func DeclContains1CodegenLabel(d *ast.FuncDecl, label string) bool {
	if d == nil {
		return false
	}

	if d.Doc == nil {
		return false
	}

	exists := false
	for _, line := range d.Doc.List {
		if strings.Contains(line.Text, label) {
			if exists {
				return false
			}
			exists = true
		}
	}
	return exists
}

func DeclHasReceiver(d *ast.FuncDecl) bool {
	if d == nil {
		return false
	}

	if d.Recv == nil {
		return false
	}

	if len(d.Recv.List) == 0 {
		return false
	}

	if len(d.Recv.List[0].Names) == 0 {
		return false
	}

	return true
}

func FuncDeclToFuncInput(f *ast.FuncDecl) (FuncInput, error) {
	//first parameter is context
	ident, _ := (f.Type.Params.List[1].Type).(*ast.Ident)
	typeSpec, _ := (ident.Obj.Decl).(*ast.TypeSpec)
	structType, _ := (typeSpec.Type).(*ast.StructType)

	result := FuncInput{
		RecvTypeName: ident.Name,
	}

	fields := make([]FuncInputStructField, 0, len(structType.Fields.List))
	for _, field := range structType.Fields.List {
		fieldTypeAsIdent, _ := (field.Type).(*ast.Ident)
		switch fieldTypeAsIdent.Name {
		case "int", "string":
		default:
			continue
		}

		_, afterTag, ok := strings.Cut(field.Tag.Value, ApiValidatorTagPrefix)
		if !ok {
			continue
		}
		tagContent, _, ok := strings.Cut(afterTag, `"`)

		fields = append(fields, FuncInputStructField{
			Name:                   field.Names[0].Name,
			Type:                   fieldTypeAsIdent.Name,
			ApiValidatorTagContent: tagContent,
			Validations:            strings.Split(tagContent, ","),
		})
	}
	result.Fields = fields
	return result, nil
}
