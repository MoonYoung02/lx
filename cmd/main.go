package main

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"google.golang.org/genai"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider string `yaml:"provider"`
	ApiKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
}

type LLMProvider interface {
	GenerateCode(systemPrompt string) (string, error)
}

type GeminiClient struct {
	client *genai.Client
	model  string
}

func NewGeminiClient(apiKey string, model string) (*GeminiClient, error) {
	ctx := context.Background()
	cfg := &genai.ClientConfig{APIKey: apiKey}
	client, err := genai.NewClient(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &GeminiClient{client: client, model: model}, nil
}

func (g *GeminiClient) GenerateCode(systemPrompt string) (string, error) {
	resp, err := g.client.Models.GenerateContent(context.Background(), g.model, genai.Text(systemPrompt), nil)
	if err != nil {
		return "", err
	}
	code := resp.Text()
	code = strings.TrimPrefix(code, "```go")
	code = strings.TrimPrefix(code, "```")
	code = strings.TrimSuffix(code, "```")
	return strings.TrimSpace(code), nil
}

var version = "dev"

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "출력: 현재 버전")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: lx [options] [target_directory]\n\nOptions:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if showVersion {
		fmt.Printf("lx %s\n", version)
		return
	}

	targetDir := "."
	args := flag.Args()
	if len(args) > 0 {
		targetDir = args[0]
	}

	configData, err := os.ReadFile("lx-config.yaml")
	if err != nil {
		log.Fatalf("lx-config.yaml 확인 필요: %v", err)
	}
	var config Config
	yaml.Unmarshal(configData, &config)

	var ai LLMProvider
	if config.Provider == "gemini" {
		client, err := NewGeminiClient(config.ApiKey, config.Model)
		if err != nil {
			log.Fatalf("Gemini 초기화 실패: %v", err)
		}
		ai = client
	}

	fmt.Printf("[%s] 분석 중...\n", targetDir)

	fset := token.NewFileSet()
	err = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || !strings.HasSuffix(path, ".go") {
			return nil
		}
		processFile(fset, path, ai)
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
}

func processFile(fset *token.FileSet, filePath string, ai LLMProvider) {
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return
	}

	updated := false
	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}

		var userPrompt string
		for _, stmt := range fn.Body.List {
			if expr, ok := stmt.(*ast.ExprStmt); ok {
				if prompt := getLXPrompt(expr); prompt != "" {
					userPrompt = prompt
					break
				}
			}
		}

		if userPrompt != "" {
			fmt.Printf("[%s] -> '%s' 처리 중...\n", filePath, fn.Name.Name)

			systemPrompt := buildSystemPrompt(fn, userPrompt)
			generatedCode, err := ai.GenerateCode(systemPrompt)
			if err != nil {
				log.Printf("AI 실패: %v", err)
				continue
			}

			fn.Body = parseToPureBlock(generatedCode)
			updated = true
		}
	}

	if updated {
		saveFile(filePath, node)
	}
}

func parseToPureBlock(code string) *ast.BlockStmt {
	code = strings.TrimSpace(code)

	if strings.HasPrefix(code, "{") && strings.HasSuffix(code, "}") {
		code = strings.TrimPrefix(code, "{")
		code = strings.TrimSuffix(code, "}")
	}

	lines := strings.Split(code, "\n")
	var bodyLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "import (") || trimmed == ")" {

			continue
		}
		bodyLines = append(bodyLines, line)
	}
	pureBody := strings.Join(bodyLines, "\n")

	dummyFile := "package main\nfunc dummy() {\n" + pureBody + "\n}"

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", dummyFile, parser.ParseComments)
	if err != nil {

		log.Printf("AI 코드 파싱 에러: %v\n[시도했던 코드]:\n%s", err, dummyFile)
		return nil
	}

	if len(f.Decls) > 0 {
		return f.Decls[0].(*ast.FuncDecl).Body
	}
	return nil
}

func getLXPrompt(stmt *ast.ExprStmt) string {
	call, ok := stmt.X.(*ast.CallExpr)
	if !ok {
		return ""
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return ""
	}

	if x, ok := sel.X.(*ast.Ident); ok && strings.ToLower(x.Name) == "lx" && sel.Sel.Name == "Generate" {
		if len(call.Args) > 0 {
			if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
				return strings.Trim(lit.Value, "\"")
			}
		}
	}
	return ""
}

func saveFile(filePath string, node *ast.File) {
	f, err := os.Create(filePath)
	if err != nil {
		log.Fatal(err)
	}

	format.Node(f, token.NewFileSet(), node)
	f.Close()

	exec.Command("goimports", "-w", filePath).Run()
}

func buildSystemPrompt(fn *ast.FuncDecl, userPrompt string) string {
	signature := fmt.Sprintf("func %s(%s) (%s)", fn.Name.Name, getFieldString(fn.Type.Params), getFieldString(fn.Type.Results))

	const promptTemplate = `You are a Go expert. Implement the ENTIRE function body logic.
RULES:
1. Output ONLY the Go code logic that goes INSIDE the function curly braces.
2. DO NOT include 'import' statements. If you need a package, just use it (e.g., time.Now()).
3. DO NOT include the function signature or curly braces { }.
4. Ensure the code is self-contained and matches the signature.

Signature: %s
Task: %s`

	return fmt.Sprintf(promptTemplate, signature, userPrompt)
}

func getFieldString(fields *ast.FieldList) string {
	if fields == nil {
		return ""
	}
	var parts []string
	for _, field := range fields.List {
		typeName := ""
		if t, ok := field.Type.(*ast.Ident); ok {
			typeName = t.Name
		}
		if len(field.Names) > 0 {
			for _, name := range field.Names {
				parts = append(parts, fmt.Sprintf("%s %s", name.Name, typeName))
			}
		} else {
			parts = append(parts, typeName)
		}
	}
	return strings.Join(parts, ", ")
}
