package service

import (
	"regexp"
	"strings"
)

// DynamicPromptBuilder 智能动态提示词构建器
type DynamicPromptBuilder struct {
	basePrompt        string
	DetectedLanguage  string
	DetectedFramework string
	TaskType          string
	SecurityContext   bool
	DatabaseContext   bool
}

// NewDynamicPromptBuilder 创建新的动态提示词构建器
func NewDynamicPromptBuilder() *DynamicPromptBuilder {
	return &DynamicPromptBuilder{
		basePrompt: getBasePrompt(),
	}
}

// AnalyzeRequest 分析用户请求内容
func (b *DynamicPromptBuilder) AnalyzeRequest(requestBody string) *DynamicPromptBuilder {
	requestLower := strings.ToLower(requestBody)
	
	// 检测编程语言
	b.DetectedLanguage = detectLanguage(requestLower, requestBody)
	
	// 检测框架
	b.DetectedFramework = detectFramework(requestLower)
	
	// 检测任务类型
	b.TaskType = detectTaskType(requestLower)
	
	// 检测安全敏感上下文
	b.SecurityContext = detectSecurityContext(requestLower)
	
	// 检测数据库上下文
	b.DatabaseContext = detectDatabaseContext(requestLower)
	
	return b
}

// Build 构建最终的提示词
func (b *DynamicPromptBuilder) Build() string {
	var sections []string
	
	// 1. 基础提示词 (约5000 tokens)
	sections = append(sections, b.basePrompt)
	
	// 2. 语言特定指导 (约2000-3000 tokens)
	if b.DetectedLanguage != "" {
		sections = append(sections, getLanguageGuidance(b.DetectedLanguage))
	}
	
	// 3. 框架特定指导 (约2000-3000 tokens)
	if b.DetectedFramework != "" {
		sections = append(sections, getFrameworkGuidance(b.DetectedFramework))
	}
	
	// 4. 任务类型指导 (约1000-2000 tokens)
	if b.TaskType != "" {
		sections = append(sections, getTaskGuidance(b.TaskType))
	}
	
	// 5. 安全指导 (约2000 tokens)
	if b.SecurityContext {
		sections = append(sections, getSecurityGuidance())
	}
	
	// 6. 数据库指导 (约1000 tokens)
	if b.DatabaseContext {
		sections = append(sections, getDatabaseGuidance())
	}
	
	return strings.Join(sections, "\n\n")
}

// detectLanguage 检测编程语言
func detectLanguage(requestLower, requestOriginal string) string {
	// 按优先级检测（更具体的模式优先）
	patterns := map[string][]string{
		"go": {
			`\bgo\s+`, `\bgolang\b`, `func\s+\w+\(`, `package\s+\w+`,
			`import\s+\(`, `defer\s+`, `goroutine`, `channel`,
			`.go"`, `.go'`, `\.go\b`,
		},
		"java": {
			`\bjava\b`, `class\s+\w+\s*{`, `public\s+class`, `private\s+class`,
			`@Override`, `@Autowired`, `System\.out\.println`,
			`.java"`, `.java'`, `\.java\b`,
		},
		"python": {
			`\bpython\b`, `def\s+\w+\(`, `import\s+\w+`, `from\s+\w+\s+import`,
			`if\s+__name__\s*==`, `self\.`, `\.py"`, `\.py'`, `\.py\b`,
		},
		"javascript": {
			`\bjavascript\b`, `\bjs\b`, `function\s*\(`, `const\s+\w+\s*=`,
			`let\s+\w+\s*=`, `=>\s*{`, `require\(`, `export\s+`,
			`.js"`, `.js'`, `\.js\b`, `node\.js`, `nodejs`,
		},
		"typescript": {
			`\btypescript\b`, `\bts\b`, `interface\s+\w+`, `type\s+\w+\s*=`,
			`: string`, `: number`, `: boolean`,
			`.ts"`, `.ts'`, `\.ts\b`,
		},
		"rust": {
			`\brust\b`, `fn\s+\w+`, `let\s+mut`, `impl\s+`, `pub\s+fn`,
			`cargo`, `.rs"`, `.rs'`, `\.rs\b`,
		},
		"cpp": {
			`\bc\+\+\b`, `\bcpp\b`, `#include\s*<`, `std::`, `class\s+\w+\s*{`,
			`namespace\s+`, `.cpp"`, `.cpp'`, `\.cpp\b`, `.hpp"`, `.h"`,
		},
		"csharp": {
			`\bc#\b`, `\bcsharp\b`, `using\s+System`, `namespace\s+\w+`,
			`public\s+class`, `.cs"`, `.cs'`, `\.cs\b`,
		},
		"php": {
			`\bphp\b`, `<?php`, `function\s+\w+\(`, `$\w+\s*=`,
			`.php"`, `.php'`, `\.php\b`,
		},
		"ruby": {
			`\bruby\b`, `def\s+\w+`, `end\b`, `require\s+`,
			`.rb"`, `.rb'`, `\.rb\b`,
		},
		"swift": {
			`\bswift\b`, `func\s+\w+`, `var\s+\w+:`, `let\s+\w+:`,
			`.swift"`, `.swift'`, `\.swift\b`,
		},
		"kotlin": {
			`\bkotlin\b`, `fun\s+\w+`, `val\s+\w+`, `var\s+\w+`,
			`.kt"`, `.kt'`, `\.kt\b`,
		},
	}
	
	// 检测每种语言
	scores := make(map[string]int)
	for lang, patternsForLang := range patterns {
		for _, pattern := range patternsForLang {
			if matched, _ := regexp.MatchString(pattern, requestLower); matched {
				scores[lang]++
			}
			// 也检测原始大小写（对文件扩展名很重要）
			if matched, _ := regexp.MatchString(pattern, requestOriginal); matched {
				scores[lang]++
			}
		}
	}
	
	// 找到得分最高的语言
	maxScore := 0
	detectedLang := ""
	for lang, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedLang = lang
		}
	}
	
	return detectedLang
}

// detectFramework 检测框架
func detectFramework(requestLower string) string {
	frameworks := map[string][]string{
		"react": {`\breact\b`, `useState`, `useEffect`, `jsx`, `tsx`},
		"vue": {`\bvue\b`, `v-if`, `v-for`, `<template>`, `composition api`},
		"angular": {`\bangular\b`, `@Component`, `@Injectable`, `ngOnInit`},
		"spring": {`\bspring\b`, `@SpringBootApplication`, `@RestController`, `@Autowired`},
		"django": {`\bdjango\b`, `models.Model`, `views.py`, `urls.py`},
		"flask": {`\bflask\b`, `@app.route`, `Flask(__name__)`},
		"express": {`\bexpress\b`, `app.get`, `app.post`, `req.body`},
		"gin": {`\bgin\b`, `gin.Default()`, `c.JSON`, `gin.Context`},
		"fastapi": {`\bfastapi\b`, `@app.get`, `@app.post`, `FastAPI()`},
		"rails": {`\brails\b`, `ActiveRecord`, `has_many`, `belongs_to`},
		"laravel": {`\blaravel\b`, `Eloquent`, `Route::`, `Artisan`},
	}
	
	for framework, keywords := range frameworks {
		for _, keyword := range keywords {
			if strings.Contains(requestLower, keyword) {
				return framework
			}
		}
	}
	
	return ""
}

// detectTaskType 检测任务类型
func detectTaskType(requestLower string) string {
	taskPatterns := map[string][]string{
		"refactoring": {`refactor`, `重构`, `优化代码`, `improve code`, `clean up`},
		"bugfix": {`bug`, `fix`, `error`, `错误`, `修复`, `问题`, `issue`},
		"feature": {`add`, `implement`, `create`, `新增`, `实现`, `添加`, `feature`},
		"testing": {`test`, `测试`, `unit test`, `integration test`, `mock`},
		"api": {`api`, `endpoint`, `route`, `接口`, `rest`, `graphql`},
		"database": {`database`, `sql`, `query`, `数据库`, `migration`, `schema`},
		"ui": {`ui`, `界面`, `layout`, `component`, `style`, `css`},
		"performance": {`performance`, `optimize`, `性能`, `slow`, `慢`, `cache`},
	}
	
	for taskType, keywords := range taskPatterns {
		for _, keyword := range keywords {
			if strings.Contains(requestLower, keyword) {
				return taskType
			}
		}
	}
	
	return ""
}

// detectSecurityContext 检测安全敏感上下文
func detectSecurityContext(requestLower string) bool {
	securityKeywords := []string{
		"auth", "login", "password", "token", "jwt", "session",
		"security", "permission", "role", "access control",
		"认证", "登录", "密码", "权限", "安全",
		"encryption", "decrypt", "hash", "crypto",
	}
	
	for _, keyword := range securityKeywords {
		if strings.Contains(requestLower, keyword) {
			return true
		}
	}
	
	return false
}

// detectDatabaseContext 检测数据库上下文
func detectDatabaseContext(requestLower string) bool {
	dbKeywords := []string{
		"sql", "database", "query", "select", "insert", "update", "delete",
		"table", "数据库", "查询", "orm", "postgres", "mysql", "mongodb",
	}
	
	for _, keyword := range dbKeywords {
		if strings.Contains(requestLower, keyword) {
			return true
		}
	}
	
	return false
}
