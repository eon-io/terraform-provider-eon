package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Coverage maps SDK operations to the provider surface that consumes them.
type Coverage struct {
	// Consumers maps operationId to the Terraform types ("eon_vault") or
	// internal call sites ("internal:client/token_refresher.go") that use it.
	Consumers map[string][]string
}

// TerraformConsumers returns the consumers of op that are Terraform types.
func (c Coverage) TerraformConsumers(opID string) []string {
	var out []string
	for _, name := range c.Consumers[opID] {
		if !strings.HasPrefix(name, "internal:") {
			out = append(out, name)
		}
	}
	return out
}

// sdkCall identifies a call into the generated SDK, e.g. AccountsAPI.GetSourceAccount.
type sdkCall struct {
	service string
	method  string
}

// rawCall identifies a hand-rolled HTTP call that bypasses the generated SDK.
type rawCall struct {
	method     string   // "GET", "PATCH", ... ("" when it could not be determined)
	urlPattern []string // fmt.Sprintf URL split into path segments, "%s" for params
}

// clientFunc is one function in internal/client with everything it reaches.
type clientFunc struct {
	file       string
	sdkCalls   []sdkCall
	rawCalls   []rawCall
	localCalls []string // other internal/client functions it calls
}

// extractCoverage statically analyzes internal/client and internal/provider to
// determine which SDK operations each Terraform resource/data source consumes.
//
// The provider follows a strict two-hop pattern: resources call methods on
// *client.EonClient, and those wrapper methods call the generated SDK (or, for
// endpoints missing from the SDK, build the HTTP request by hand with a
// fmt.Sprintf URL). Both hops are recovered syntactically.
func extractCoverage(providerDir string, ops []SpecOperation) (Coverage, error) {
	clientFuncs, err := scanClientPackage(filepath.Join(providerDir, "internal", "client"))
	if err != nil {
		return Coverage{}, err
	}

	// Resolve each client function to the set of operationIds it reaches,
	// following calls between wrapper functions (e.g. WaitForRestoreJobCompletion
	// -> GetRestoreJob).
	funcOps := map[string]map[string]bool{}
	var resolve func(name string, seen map[string]bool) map[string]bool
	resolve = func(name string, seen map[string]bool) map[string]bool {
		if cached, ok := funcOps[name]; ok {
			return cached
		}
		if seen[name] {
			return nil
		}
		seen[name] = true
		fn, ok := clientFuncs[name]
		if !ok {
			return nil
		}
		reached := map[string]bool{}
		for _, call := range fn.sdkCalls {
			if op := matchSDKCall(call, ops); op != "" {
				reached[op] = true
			}
		}
		for _, call := range fn.rawCalls {
			if op := matchRawCall(call, ops); op != "" {
				reached[op] = true
			}
		}
		for _, callee := range fn.localCalls {
			for op := range resolve(callee, seen) {
				reached[op] = true
			}
		}
		funcOps[name] = reached
		return reached
	}
	for name := range clientFuncs {
		resolve(name, map[string]bool{})
	}

	consumers := map[string]map[string]bool{}
	addConsumer := func(opID, name string) {
		if consumers[opID] == nil {
			consumers[opID] = map[string]bool{}
		}
		consumers[opID][name] = true
	}

	// Hop 2: which provider files call which wrapper methods.
	providerFiles, err := scanProviderPackage(filepath.Join(providerDir, "internal", "provider"))
	if err != nil {
		return Coverage{}, err
	}
	for _, pf := range providerFiles {
		consumer := pf.terraformName
		if consumer == "" {
			consumer = "internal:provider/" + pf.file
		}
		for _, method := range pf.clientCalls {
			for op := range funcOps[method] {
				addConsumer(op, consumer)
			}
		}
	}

	// Operations reached inside internal/client but never through a provider
	// file (e.g. token refresh) are internal plumbing.
	for name, fn := range clientFuncs {
		for op := range funcOps[name] {
			if len(consumers[op]) == 0 {
				addConsumer(op, "internal:client/"+fn.file)
			}
		}
	}

	out := Coverage{Consumers: map[string][]string{}}
	for op, set := range consumers {
		for name := range set {
			out.Consumers[op] = append(out.Consumers[op], name)
		}
		sort.Strings(out.Consumers[op])
	}
	return out, nil
}

func matchSDKCall(call sdkCall, ops []SpecOperation) string {
	for _, op := range ops {
		if strings.EqualFold(call.service, op.SDKService()) && strings.EqualFold(call.method, op.SDKMethod()) {
			return op.ID
		}
	}
	return ""
}

// matchRawCall matches a fmt.Sprintf URL pattern (plus HTTP method, when
// known) against the spec's paths, treating "%s" as a wildcard segment.
func matchRawCall(call rawCall, ops []SpecOperation) string {
	for _, op := range ops {
		if call.method != "" && call.method != op.Method {
			continue
		}
		specSegs := strings.Split(strings.TrimPrefix(op.Path, "/"), "/")
		if len(specSegs) != len(call.urlPattern) {
			continue
		}
		match := true
		for i, seg := range specSegs {
			isParam := strings.HasPrefix(seg, "{")
			if isParam && call.urlPattern[i] == "%s" {
				continue
			}
			if !isParam && seg == call.urlPattern[i] {
				continue
			}
			match = false
			break
		}
		if match {
			return op.ID
		}
	}
	return ""
}

// scanClientPackage parses every non-test file in internal/client and records,
// per function, the SDK calls, raw HTTP calls, and intra-package calls it makes.
func scanClientPackage(dir string) (map[string]clientFunc, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, notTestFile, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", dir, err)
	}

	funcs := map[string]clientFunc{}
	// First pass: collect all function names so local calls can be recognized.
	names := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					names[fn.Name.Name] = true
				}
			}
		}
	}

	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			base := filepath.Base(fileName)
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Body == nil {
					continue
				}
				cf := clientFunc{file: base}
				recv := receiverName(fn)
				httpMethods := map[string]bool{}
				var urls [][]string

				ast.Inspect(fn.Body, func(n ast.Node) bool {
					switch node := n.(type) {
					case *ast.SelectorExpr:
						// http.MethodGet and friends may appear anywhere in the
						// function (usually as an argument to NewRequestWithContext).
						if x, ok := node.X.(*ast.Ident); ok && x.Name == "http" && strings.HasPrefix(node.Sel.Name, "Method") {
							httpMethods[strings.ToUpper(strings.TrimPrefix(node.Sel.Name, "Method"))] = true
						}
					case *ast.CallExpr:
						sel, ok := node.Fun.(*ast.SelectorExpr)
						if !ok {
							return true
						}
						// c.client.<Service>API.<Method>(...) or r.apiClient.<Service>API.<Method>(...)
						if inner, ok := sel.X.(*ast.SelectorExpr); ok && strings.HasSuffix(inner.Sel.Name, "API") {
							cf.sdkCalls = append(cf.sdkCalls, sdkCall{service: inner.Sel.Name, method: sel.Sel.Name})
							return true
						}
						// fmt.Sprintf("<url with /v1/>", ...)
						if x, ok := sel.X.(*ast.Ident); ok && x.Name == "fmt" && sel.Sel.Name == "Sprintf" && len(node.Args) > 0 {
							if lit, ok := node.Args[0].(*ast.BasicLit); ok && strings.Contains(lit.Value, "/v1/") {
								pattern := strings.Trim(lit.Value, "`\"")
								segs := strings.Split(pattern, "/")
								// Drop the leading base-URL placeholder ("%s").
								if len(segs) > 0 && segs[0] == "%s" {
									segs = segs[1:]
								}
								urls = append(urls, segs)
							}
							return true
						}
						// c.OtherWrapper(...): a call to another function in this package.
						if x, ok := sel.X.(*ast.Ident); ok && recv != "" && x.Name == recv && names[sel.Sel.Name] {
							cf.localCalls = append(cf.localCalls, sel.Sel.Name)
						}
					}
					return true
				})

				// Attach the function's HTTP method to its raw URLs. Wrapper
				// functions make one hand-rolled call each; when a function
				// somehow mixes several methods, leave it ambiguous and let
				// path-shape matching decide.
				method := ""
				if len(httpMethods) == 1 {
					for m := range httpMethods {
						method = m
					}
				}
				for _, u := range urls {
					cf.rawCalls = append(cf.rawCalls, rawCall{method: method, urlPattern: u})
				}
				// Functions with the same name can exist on different types
				// (EonClient and MockEonClient both have CreateBackupPolicy);
				// merge them so the mock's empty body cannot mask the real
				// wrapper's SDK calls.
				if prev, ok := funcs[fn.Name.Name]; ok {
					cf.sdkCalls = append(cf.sdkCalls, prev.sdkCalls...)
					cf.rawCalls = append(cf.rawCalls, prev.rawCalls...)
					cf.localCalls = append(cf.localCalls, prev.localCalls...)
					if len(cf.sdkCalls) == 0 && len(cf.rawCalls) == 0 && len(cf.localCalls) == 0 {
						cf.file = prev.file
					}
				}
				funcs[fn.Name.Name] = cf
			}
		}
	}
	return funcs, nil
}

// providerFile is one file in internal/provider with the Terraform type it
// defines (if any) and the client wrapper methods it calls.
type providerFile struct {
	file          string
	terraformName string // "eon_vault", or "" for shared helper files
	kind          string // "resource", "data_source", or ""
	clientCalls   []string
}

// scanProviderPackage parses every non-test file in internal/provider and
// records the Terraform type each file registers (via the
// `resp.TypeName = req.ProviderTypeName + "_x"` convention) and every
// `<recv>.client.<Method>(...)` call it makes.
func scanProviderPackage(dir string) ([]providerFile, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, notTestFile, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", dir, err)
	}

	var out []providerFile
	for _, pkg := range pkgs {
		for fileName, file := range pkg.Files {
			pf := providerFile{file: filepath.Base(fileName)}
			calls := map[string]bool{}

			ast.Inspect(file, func(n ast.Node) bool {
				switch node := n.(type) {
				case *ast.FuncDecl:
					// Metadata(ctx, req resource.MetadataRequest, ...) vs datasource.
					if node.Name.Name == "Metadata" && node.Recv != nil {
						for _, param := range node.Type.Params.List {
							if sel, ok := param.Type.(*ast.SelectorExpr); ok {
								if x, ok := sel.X.(*ast.Ident); ok && sel.Sel.Name == "MetadataRequest" {
									switch x.Name {
									case "resource":
										pf.kind = "resource"
									case "datasource":
										pf.kind = "data_source"
									}
								}
							}
						}
					}
				case *ast.BinaryExpr:
					// req.ProviderTypeName + "_vault"
					if node.Op == token.ADD {
						if sel, ok := node.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "ProviderTypeName" {
							if lit, ok := node.Y.(*ast.BasicLit); ok {
								pf.terraformName = "eon" + strings.Trim(lit.Value, "\"")
							}
						}
					}
				case *ast.CallExpr:
					// r.client.GetVault(...) / d.client.ListVaults(...)
					if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
						if inner, ok := sel.X.(*ast.SelectorExpr); ok && inner.Sel.Name == "client" {
							calls[sel.Sel.Name] = true
						}
					}
				}
				return true
			})

			for c := range calls {
				pf.clientCalls = append(pf.clientCalls, c)
			}
			sort.Strings(pf.clientCalls)
			out = append(out, pf)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].file < out[j].file })
	return out, nil
}

func receiverName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 || len(fn.Recv.List[0].Names) == 0 {
		return ""
	}
	return fn.Recv.List[0].Names[0].Name
}

func notTestFile(fi fs.FileInfo) bool {
	return !strings.HasSuffix(fi.Name(), "_test.go")
}
