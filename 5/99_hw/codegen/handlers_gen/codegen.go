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

const CodegenLabel = `// apigen:api `
const FuncWrapperPrefix = `wrapper`

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
		return strings.Contains(comment.Text, CodegenLabel)
	})[0]
	codegenOptionsJson, ok := strings.CutPrefix(codegenOptionLine.Text, CodegenLabel)
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

	return &FuncWrapper{
		Decl:           f,
		RecvName:       f.Recv.List[0].Names[0].Name,
		IsStarReceiver: isStarReceiver,
		RecvTypeName:   recvTypeName,
		FuncName:       f.Name.Name,
		Options:        options,
	}, nil
}

func (f *FuncWrapper) WrapperFuncName() string {
	return FuncWrapperPrefix + f.FuncName
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
	"net/http"
)`))

	serveHTTPTmpl = template.Must(template.New("serveHTTPTmpl").Parse(
		`func ({{$serveRecvName := .RecvName}}{{$serveRecvName}} *{{.RecvTypeName}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path { {{range $url, $methods := .Wrappers}}
	case "{{$url}}":
		switch r.Method { {{range $method, $wrapper := $methods}}
		case "GET":
			{{$serveRecvName}}.{{$wrapper.WrapperFuncName}}(w, r){{end}}
		default:
			http.NotFound(w, r)
		}{{end}}
	default:
		http.NotFound(w, r)
	}
}`))

	funcTmpl = template.Must(template.New("wrapperTmpl").Parse(
		`func ({{.RecvName}} {{if .IsStarReceiver}}*{{end}}{{.RecvTypeName}}) {{.WrapperFuncName}}(w http.ResponseWriter, r *http.Request) { {{if .Options.Auth}}
	//auth{{end}}
	//get&validate
	//call {{.RecvName}}.{{.WrapperFuncName}}(r.Context(), in)
	//return results
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

	err = packageImportsTmpl.Execute(out, node.Name.Name)
	if err != nil {
		log.Fatal(err)
	}
	newLines(out, 2)

	funcDecls := generics.Map(node.Decls, DeclToFuncDecl)
	funcWrappers, err := generics.TryMap(funcDecls, NewFuncWrapper)
	if err != nil {
		log.Fatal(err)
	}

	serveWrappers := make(map[string]*ServeHTTPWrapper)
	for _, f := range funcWrappers {
		if _, serveHTTPWrapperExists := serveWrappers[f.RecvTypeName]; !serveHTTPWrapperExists {
			serveWrappers[f.RecvTypeName] = &ServeHTTPWrapper{
				RecvTypeName: f.RecvTypeName,
				Wrappers:     make(map[string]map[string]*FuncWrapper),
			}
		}
		curServeWrapper := serveWrappers[f.RecvTypeName]
		if curServeWrapper.RecvName == "" && f.RecvName != "" {
			curServeWrapper.RecvName = f.RecvName
		}

		if _, groupByUrlExists := curServeWrapper.Wrappers[f.Options.Url]; !groupByUrlExists {
			curServeWrapper.Wrappers[f.Options.Url] = make(map[string]*FuncWrapper)
		}
		curUrl := curServeWrapper.Wrappers[f.Options.Url]

		if _, methodOccupied := curUrl[f.Options.Method]; methodOccupied {
			log.Fatalf("2 handlers are on the same URL and Method: %s, %s", curUrl[f.Options.Method].FuncName, f.FuncName)
		}
		curUrl[f.Options.Method] = f
	}

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

	if !DeclContains1CodegenLabel(r, CodegenLabel) {
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
