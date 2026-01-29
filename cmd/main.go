package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tree_sitter_js "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tree_sitter_python "github.com/tree-sitter/tree-sitter-python/bindings/go"
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

type Replacement struct {
	StartByte uint
	EndByte   uint
	Name      string
	Params    string
	Result    string
	Prompt    string
}

type LanguageConfig struct {
	Name     string
	Language *sitter.Language
	Query    string
	Marker   string
}

const UniversalMarker = `(?i)lx\(['"]([^'"]+)['"]\)`

var supportedLanguages = map[string]LanguageConfig{
	".go": {
		Name:     "Go",
		Language: sitter.NewLanguage(tree_sitter_go.Language()),
		Query: `(function_declaration
					name: (identifier) @fn.name
					parameters: (parameter_list) @fn.params
					result: [
						(type_identifier) (parameter_list) (pointer_type) (qualified_type)
					]? @fn.result
					body: (block) @fn.body) @entire`,
		Marker: `lx\.Generate\(['"]([^'"]+)['"]\)`,
	},
	".js": {
		Name:     "JavaScript",
		Language: sitter.NewLanguage(tree_sitter_js.Language()),
		Query: `(function_declaration
					name: (identifier) @fn.name
					parameters: (formal_parameters) @fn.params
					body: (statement_block) @fn.body) @entire`,
		Marker: `lx\.Generate\(['"]([^'"]+)['"]\)`,
	},
	".py": {
		Name:     "Python",
		Language: sitter.NewLanguage(tree_sitter_python.Language()),
		Query: `(function_definition
					name: (identifier) @fn.name
					parameters: (parameters) @fn.params
					body: (block) @fn.body) @entire`,
		Marker: `lx\.Generate\(['"]([^'"]+)['"]\)`,
	},
}

func processFile(path string, cfg LanguageConfig, ai LLMProvider) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	parser := sitter.NewParser()
	parser.SetLanguage(cfg.Language)
	tree := parser.Parse(content, nil)

	query, qErr := sitter.NewQuery(cfg.Language, cfg.Query)
	if qErr != nil {
		return qErr
	}

	cursor := sitter.NewQueryCursor()
	matches := cursor.Matches(query, tree.RootNode(), content)

	var pending []Replacement

	for {
		m := matches.Next()
		if m == nil {
			break
		}

		var fnName, fnParams, fnResult, fnBody string
		var entireStart, entireEnd uint

		for i := range m.Captures {
			cap := m.Captures[i]
			name := query.CaptureNames()[cap.Index]
			text := string(content[cap.Node.StartByte():cap.Node.EndByte()])

			switch name {
			case "entire":
				entireStart = cap.Node.StartByte()
				entireEnd = cap.Node.EndByte()
			case "fn.name":
				fnName = text
			case "fn.params":
				fnParams = text
			case "fn.result":
				fnResult = text
			case "fn.body":
				fnBody = text
			}
		}

		prompt := extractPromptContent(fnBody, cfg)
		if prompt != "" {
			pending = append(pending, Replacement{
				StartByte: entireStart,
				EndByte:   entireEnd,
				Name:      fnName,
				Params:    fnParams,
				Result:    fnResult,
				Prompt:    prompt,
			})
		}
	}

	if len(pending) == 0 {
		return nil
	}

	newContent := string(content)
	ext := filepath.Ext(path)

	for i := len(pending) - 1; i >= 0; i-- {
		target := pending[i]
		fmt.Printf("\t-> [%s] %s 함수 처리 중...\n", path, target.Name)

		systemPrompt := buildSystemPrompt(target, cfg.Name)
		generated, err := ai.GenerateCode(systemPrompt)
		if err != nil {
			log.Printf("[error] AI 에러: %v", err)
			continue
		}

		replacementCode := cleanAICode(generated)
		handleDependencies(replacementCode, path)
		newContent = newContent[:target.StartByte] + replacementCode + newContent[target.EndByte:]
	}

	writeErr := os.WriteFile(path, []byte(newContent), 0644)
	if writeErr == nil {
		runPostProcess(path, ext)
	}
	return writeErr
}

func cleanAICode(code string) string {
	// LLM 답변에서 불필요한 마크다운 태그만 제거
	code = strings.TrimSpace(code)
	re := regexp.MustCompile("(?s)```(?:[a-z]*)\n?(.*?)\n?```")
	if match := re.FindStringSubmatch(code); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return code
}

func extractPromptContent(body string, langCfg LanguageConfig) string {
	officialRe := regexp.MustCompile(langCfg.Marker)
	if match := officialRe.FindStringSubmatch(body); len(match) > 1 {
		return match[1]
	}
	universalRe := regexp.MustCompile(UniversalMarker)
	if match := universalRe.FindStringSubmatch(body); len(match) > 1 {
		return match[1]
	}
	return ""
}

func handleDependencies(code string, path string) {
	re := regexp.MustCompile(`(?i)//\s*lx-dep:\s*([^\s\n]+)`)
	matches := re.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return
	}
	fmt.Printf("\n[%s] Dependency:\n", path)
	fmt.Println(strings.Repeat("-", 40))
	for _, m := range matches {
		fmt.Printf("\t사용된 패키지: %s\n", m[1])
	}
	fmt.Println(strings.Repeat("-", 40))
}

func runPostProcess(path string, ext string) {
	switch ext {
	case ".go":
		exec.Command("goimports", "-w", path).Run()
		exec.Command("go", "mod", "tidy").Run()
	case ".py":
		if err := exec.Command("ruff", "format", path).Run(); err != nil {
			exec.Command("black", path).Run()
		}
	}
}

func buildSystemPrompt(target Replacement, lang string) string {
	sig := fmt.Sprintf("%s %s%s %s", lang, target.Name, target.Params, target.Result)

	return fmt.Sprintf(`Implement ONLY this %s function: %s
Task: %s
- Code ONLY. No package/import/explanation.
- If external libs, start with: // lx-dep: <name>`, lang, sig, target.Prompt)
}

// --- 5. AI 및 메인 로직 ---

func (g *GeminiClient) GenerateCode(systemPrompt string) (string, error) {
	resp, err := g.client.Models.GenerateContent(context.Background(), g.model, genai.Text(systemPrompt), nil)
	if err != nil {
		return "", err
	}
	return resp.Text(), nil
}

func NewGeminiClient(apiKey, model string) (*GeminiClient, error) {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: apiKey})
	return &GeminiClient{client: client, model: model}, err
}

var version = "dev"

func main() {
	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "출력: 버전")
	flag.Parse()

	if showVersion {
		fmt.Printf("lx %s\n", version)
		return
	}

	targetDir := "."
	if args := flag.Args(); len(args) > 0 {
		targetDir = args[0]
	}

	configPath := "lx-config.yaml"
	location := "Project Local"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		home, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(home, "lx-config.yaml")
			location = "Global (Home Dir)"
		}
	}

	configData, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("[error] 설정 에러: %v", err)
	}

	var config Config
	yaml.Unmarshal(configData, &config)

	fmt.Printf("[success] 설정 로드 완료: %s (%s)\n", configPath, location)
	fmt.Printf("AI: [%s] / 모델: [%s]\n", config.Provider, config.Model)
	fmt.Println(strings.Repeat("-", 50))

	ai, _ := NewGeminiClient(config.ApiKey, config.Model)

	filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if cfg, ok := supportedLanguages[ext]; ok {
			return processFile(path, cfg, ai)
		}
		return nil
	})

	fmt.Println("\n[success] 모든 작업이 완료되었습니다.")
}
