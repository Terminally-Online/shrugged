package golang

import (
	"bytes"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

type StructField struct {
	Name string
	Type string
	Tag  string
}

type EnumValue struct {
	Name  string
	Value string
}

func mergeStructFile(filePath string, structName string, fields []StructField, newImports []string) ([]byte, error) {
	existing, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, existing, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	structFound := false
	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if typeSpec.Name.Name != structName {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structFound = true
		structType.Fields = buildFieldList(fields)
		return false
	})

	if !structFound {
		addStructDecl(file, structName, fields)
	}

	ensureImports(file, newImports)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func mergeTableFile(filePath string, structName string, extensionName string, fields []StructField, newImports []string) ([]byte, error) {
	existing, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, existing, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	extensionFields, _ := ExtractExtensionFields(filePath)
	coveredFields := make(map[string]bool)
	if extFields, ok := extensionFields[extensionName]; ok {
		for _, f := range extFields {
			coveredFields[f.Name] = true
		}
	}

	var filteredFields []StructField
	for _, f := range fields {
		if f.Type == "" {
			filteredFields = append(filteredFields, f)
			continue
		}
		if !coveredFields[f.Name] {
			filteredFields = append(filteredFields, f)
		}
	}

	filteredImports := filterImportsForFields(filteredFields, newImports)

	mainStructFound := false
	extensionStructFound := false

	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if typeSpec.Name.Name == extensionName {
			extensionStructFound = true
			return true
		}

		if typeSpec.Name.Name != structName {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		mainStructFound = true
		structType.Fields = buildFieldListWithEmbed(filteredFields)
		return false
	})

	if !extensionStructFound {
		addEmptyStructDecl(file, extensionName)
	}

	if !mainStructFound {
		addStructDeclWithEmbed(file, structName, extensionName, filteredFields)
	}

	removeUnusedImports(file, filteredImports)
	ensureImports(file, filteredImports)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func removeUnusedImports(file *ast.File, neededImports []string) {
	needed := make(map[string]bool)
	for _, imp := range neededImports {
		needed[imp] = true
	}

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}

		var keptSpecs []ast.Spec
		for _, spec := range genDecl.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if !ok {
				keptSpecs = append(keptSpecs, spec)
				continue
			}

			path := strings.Trim(importSpec.Path.Value, `"`)
			if path == "encoding/json" && !needed["encoding/json"] {
				continue
			}
			if path == "time" && !needed["time"] {
				continue
			}

			keptSpecs = append(keptSpecs, spec)
		}

		genDecl.Specs = keptSpecs
	}
}

func filterImportsForFields(fields []StructField, imports []string) []string {
	neededImports := make(map[string]bool)

	for _, f := range fields {
		if strings.HasPrefix(f.Type, "time.") || f.Type == "time.Time" || f.Type == "*time.Time" {
			neededImports["time"] = true
		}
		if strings.HasPrefix(f.Type, "json.") || strings.Contains(f.Type, "json.RawMessage") {
			neededImports["encoding/json"] = true
		}
	}

	var filtered []string
	for _, imp := range imports {
		if neededImports[imp] {
			filtered = append(filtered, imp)
		}
	}

	return filtered
}

func mergeEnumFile(filePath string, typeName string, values []EnumValue) ([]byte, error) {
	existing, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, existing, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	typeFound := false
	var constDeclToUpdate *ast.GenDecl

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if genDecl.Tok == token.TYPE {
			for _, spec := range genDecl.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if typeSpec.Name.Name == typeName {
					typeFound = true
					break
				}
			}
		}

		if genDecl.Tok == token.CONST {
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				if len(valueSpec.Names) > 0 && strings.HasPrefix(valueSpec.Names[0].Name, typeName) {
					constDeclToUpdate = genDecl
					break
				}
			}
		}
	}

	if !typeFound {
		addTypeAlias(file, typeName, "string")
	}

	if constDeclToUpdate != nil {
		constDeclToUpdate.Specs = buildEnumSpecs(typeName, values)
	} else {
		addEnumConsts(file, typeName, values)
	}

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func buildFieldList(fields []StructField) *ast.FieldList {
	var astFields []*ast.Field
	for _, f := range fields {
		field := &ast.Field{
			Names: []*ast.Ident{{Name: f.Name}},
			Type:  parseTypeExpr(f.Type),
		}
		if f.Tag != "" {
			field.Tag = &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`" + f.Tag + "`",
			}
		}
		astFields = append(astFields, field)
	}
	return &ast.FieldList{List: astFields}
}

func parseTypeExpr(typeStr string) ast.Expr {
	if strings.HasPrefix(typeStr, "*") {
		return &ast.StarExpr{
			X: parseTypeExpr(typeStr[1:]),
		}
	}
	if strings.HasPrefix(typeStr, "[]") {
		return &ast.ArrayType{
			Elt: parseTypeExpr(typeStr[2:]),
		}
	}
	if strings.Contains(typeStr, ".") {
		parts := strings.SplitN(typeStr, ".", 2)
		return &ast.SelectorExpr{
			X:   &ast.Ident{Name: parts[0]},
			Sel: &ast.Ident{Name: parts[1]},
		}
	}
	return &ast.Ident{Name: typeStr}
}

func addStructDecl(file *ast.File, name string, fields []StructField) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: name},
				Type: &ast.StructType{
					Fields: buildFieldList(fields),
				},
			},
		},
	}

	insertPos := findInsertPosition(file)
	file.Decls = insertDecl(file.Decls, insertPos, decl)
}

func addEmptyStructDecl(file *ast.File, name string) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: name},
				Type: &ast.StructType{
					Fields: &ast.FieldList{},
				},
			},
		},
	}

	insertPos := findInsertPosition(file)
	file.Decls = insertDecl(file.Decls, insertPos, decl)
}

func addStructDeclWithEmbed(file *ast.File, name string, embedName string, fields []StructField) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: name},
				Type: &ast.StructType{
					Fields: buildFieldListWithEmbed(fields),
				},
			},
		},
	}

	insertPos := findInsertPosition(file)
	file.Decls = insertDecl(file.Decls, insertPos, decl)
}

func buildFieldListWithEmbed(fields []StructField) *ast.FieldList {
	var astFields []*ast.Field
	for i, f := range fields {
		if i == 0 && f.Type == "" {
			astFields = append(astFields, &ast.Field{
				Type: &ast.Ident{Name: f.Name},
			})
			continue
		}
		field := &ast.Field{
			Names: []*ast.Ident{{Name: f.Name}},
			Type:  parseTypeExpr(f.Type),
		}
		if f.Tag != "" {
			field.Tag = &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`" + f.Tag + "`",
			}
		}
		astFields = append(astFields, field)
	}
	return &ast.FieldList{List: astFields}
}

func addTypeAlias(file *ast.File, name string, underlying string) {
	decl := &ast.GenDecl{
		Tok: token.TYPE,
		Specs: []ast.Spec{
			&ast.TypeSpec{
				Name: &ast.Ident{Name: name},
				Type: &ast.Ident{Name: underlying},
			},
		},
	}

	insertPos := findInsertPosition(file)
	file.Decls = insertDecl(file.Decls, insertPos, decl)
}

func addEnumConsts(file *ast.File, typeName string, values []EnumValue) {
	decl := &ast.GenDecl{
		Tok:    token.CONST,
		Lparen: 1,
		Specs:  buildEnumSpecs(typeName, values),
	}

	insertPos := findTypePosition(file, typeName)
	if insertPos < 0 {
		insertPos = findInsertPosition(file)
	} else {
		insertPos++
	}
	file.Decls = insertDecl(file.Decls, insertPos, decl)
}

func buildEnumSpecs(typeName string, values []EnumValue) []ast.Spec {
	var specs []ast.Spec
	for _, v := range values {
		specs = append(specs, &ast.ValueSpec{
			Names: []*ast.Ident{{Name: v.Name}},
			Type:  &ast.Ident{Name: typeName},
			Values: []ast.Expr{
				&ast.BasicLit{Kind: token.STRING, Value: `"` + v.Value + `"`},
			},
		})
	}
	return specs
}

func ensureImports(file *ast.File, imports []string) {
	if len(imports) == 0 {
		return
	}

	existing := make(map[string]bool)
	var importDecl *ast.GenDecl

	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.IMPORT {
			continue
		}
		importDecl = genDecl
		for _, spec := range genDecl.Specs {
			importSpec, ok := spec.(*ast.ImportSpec)
			if ok {
				path := strings.Trim(importSpec.Path.Value, `"`)
				existing[path] = true
			}
		}
	}

	var toAdd []string
	for _, imp := range imports {
		if !existing[imp] {
			toAdd = append(toAdd, imp)
		}
	}

	if len(toAdd) == 0 {
		return
	}

	if importDecl == nil {
		importDecl = &ast.GenDecl{
			Tok:    token.IMPORT,
			Lparen: 1,
		}
		file.Decls = append([]ast.Decl{importDecl}, file.Decls...)
	}

	for _, imp := range toAdd {
		importDecl.Specs = append(importDecl.Specs, &ast.ImportSpec{
			Path: &ast.BasicLit{Kind: token.STRING, Value: `"` + imp + `"`},
		})
	}

	if len(importDecl.Specs) > 1 {
		importDecl.Lparen = 1
	}
}

func findInsertPosition(file *ast.File) int {
	for i, decl := range file.Decls {
		if _, ok := decl.(*ast.GenDecl); ok {
			genDecl := decl.(*ast.GenDecl)
			if genDecl.Tok != token.IMPORT {
				return i
			}
		}
	}
	return len(file.Decls)
}

func findTypePosition(file *ast.File, typeName string) int {
	for i, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if ok && typeSpec.Name.Name == typeName {
				return i
			}
		}
	}
	return -1
}

func insertDecl(decls []ast.Decl, pos int, decl ast.Decl) []ast.Decl {
	if pos >= len(decls) {
		return append(decls, decl)
	}
	decls = append(decls[:pos+1], decls[pos:]...)
	decls[pos] = decl
	return decls
}

func ExtractExtensionFields(filePath string) (map[string][]StructField, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]StructField)

	ast.Inspect(file, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		if !strings.HasSuffix(typeSpec.Name.Name, "Extension") {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		var fields []StructField
		for _, field := range structType.Fields.List {
			if len(field.Names) == 0 {
				continue
			}

			fieldName := field.Names[0].Name
			fieldType := exprToString(field.Type)

			var tag string
			if field.Tag != nil {
				tag = strings.Trim(field.Tag.Value, "`")
			}

			fields = append(fields, StructField{
				Name: fieldName,
				Type: fieldType,
				Tag:  tag,
			})
		}

		result[typeSpec.Name.Name] = fields
		return true
	})

	return result, nil
}

func exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + exprToString(t.X)
	case *ast.ArrayType:
		if t.Len == nil {
			return "[]" + exprToString(t.Elt)
		}
		return "[...]" + exprToString(t.Elt)
	case *ast.SelectorExpr:
		return exprToString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + exprToString(t.Key) + "]" + exprToString(t.Value)
	default:
		return ""
	}
}
