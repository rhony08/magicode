// Package lsp provides language mappings for LSP servers.
package lsp

import (
	"path/filepath"
	"strings"
)

// LanguageMapping maps languages to their LSP server configurations
type LanguageMapping struct {
	ServerID    ServerID
	LanguageID  string
	Extensions  []string
	Command     []string // Command to start the LSP server
	Args        []string // Additional arguments
	InstallHint string   // Hint for installing the server
}

// LanguageMappings are the well-known language server mappings
var LanguageMappings = map[ServerID]LanguageMapping{
	ServerTypeScript: {
		ServerID:   ServerTypeScript,
		LanguageID: "typescript",
		Extensions: []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs"},
		Command:    []string{"typescript-language-server", "--stdio"},
		InstallHint: "npm install -g typescript-language-server typescript",
	},
	ServerGo: {
		ServerID:   ServerGo,
		LanguageID: "go",
		Extensions: []string{".go"},
		Command:    []string{"gopls"},
		InstallHint: "go install golang.org/x/tools/gopls@latest",
	},
	ServerPython: {
		ServerID:   ServerPython,
		LanguageID: "python",
		Extensions: []string{".py", ".pyi", ".pyw"},
		Command:    []string{"pylsp"}, // or "pyright", "ruff-lsp"
		InstallHint: "pip install python-lsp-server",
	},
	ServerRust: {
		ServerID:   ServerRust,
		LanguageID: "rust",
		Extensions: []string{".rs"},
		Command:    []string{"rust-analyzer"},
		InstallHint: "rustup component add rust-analyzer",
	},
	ServerJava: {
		ServerID:   ServerJava,
		LanguageID: "java",
		Extensions: []string{".java"},
		Command:    []string{"jdtls"},
		InstallHint: "Install Eclipse JDT Language Server",
	},
	ServerC: {
		ServerID:   ServerC,
		LanguageID: "c",
		Extensions: []string{".c", ".h"},
		Command:    []string{"clangd"},
		InstallHint: "Install clangd (part of LLVM)",
	},
	ServerCpp: {
		ServerID:   ServerCpp,
		LanguageID: "cpp",
		Extensions: []string{".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx"},
		Command:    []string{"clangd"},
		InstallHint: "Install clangd (part of LLVM)",
	},
	ServerRuby: {
		ServerID:   ServerRuby,
		LanguageID: "ruby",
		Extensions: []string{".rb", ".rake"},
		Command:    []string{"solargraph", "stdio"},
		InstallHint: "gem install solargraph",
	},
	ServerPHP: {
		ServerID:   ServerPHP,
		LanguageID: "php",
		Extensions: []string{".php"},
		Command:    []string{"intelephense", "--stdio"},
		InstallHint: "npm install -g intelephense",
	},
	ServerJSON: {
		ServerID:   ServerJSON,
		LanguageID: "json",
		Extensions: []string{".json", ".jsonc"},
		Command:    []string{"vscode-json-languageserver", "--stdio"},
		InstallHint: "npm install -g vscode-json-languageserver",
	},
	ServerYAML: {
		ServerID:   ServerYAML,
		LanguageID: "yaml",
		Extensions: []string{".yaml", ".yml"},
		Command:    []string{"yaml-language-server", "--stdio"},
		InstallHint: "npm install -g yaml-language-server",
	},
	ServerHTML: {
		ServerID:   ServerHTML,
		LanguageID: "html",
		Extensions: []string{".html", ".htm"},
		Command:    []string{"vscode-html-languageserver", "--stdio"},
		InstallHint: "npm install -g vscode-html-languageserver",
	},
	ServerCSS: {
		ServerID:   ServerCSS,
		LanguageID: "css",
		Extensions: []string{".css", ".scss", ".less"},
		Command:    []string{"vscode-css-languageserver", "--stdio"},
		InstallHint: "npm install -g vscode-css-languageserver",
	},
}

// ExtensionToLanguage maps file extensions to language IDs
var ExtensionToLanguage = buildExtensionMap()

func buildExtensionMap() map[string]string {
	m := make(map[string]string)
	for _, mapping := range LanguageMappings {
		for _, ext := range mapping.Extensions {
			m[ext] = mapping.LanguageID
		}
	}
	return m
}

// GetLanguageFromPath returns the language ID for a file path
func GetLanguageFromPath(path string) string {
	ext := filepath.Ext(path)
	if lang, ok := ExtensionToLanguage[ext]; ok {
		return lang
	}
	return ""
}

// GetLanguageFromExtension returns the language ID for an extension
func GetLanguageFromExtension(ext string) string {
	if lang, ok := ExtensionToLanguage[ext]; ok {
		return lang
	}
	return ""
}

// GetServerIDFromLanguage returns the server ID for a language
func GetServerIDFromLanguage(lang string) ServerID {
	for id, mapping := range LanguageMappings {
		if mapping.LanguageID == lang {
			return id
		}
	}
	return ServerID("")
}

// GetServerFromPath returns the server mapping for a file path
func GetServerFromPath(path string) *LanguageMapping {
	lang := GetLanguageFromPath(path)
	if lang == "" {
		return nil
	}
	serverID := GetServerIDFromLanguage(lang)
	if serverID == "" {
		return nil
	}
	if mapping, ok := LanguageMappings[serverID]; ok {
		return &mapping
	}
	return nil
}

// GetServerFromExtension returns the server mapping for an extension
func GetServerFromExtension(ext string) *LanguageMapping {
	lang := GetLanguageFromExtension(ext)
	if lang == "" {
		return nil
	}
	serverID := GetServerIDFromLanguage(lang)
	if serverID == "" {
		return nil
	}
	if mapping, ok := LanguageMappings[serverID]; ok {
		return &mapping
	}
	return nil
}

// GetExtensions returns all known extensions
func GetExtensions() []string {
	exts := []string{}
	for ext := range ExtensionToLanguage {
		exts = append(exts, ext)
	}
	return exts
}

// GetLanguageIDs returns all known language IDs
func GetLanguageIDs() []string {
	langs := make(map[string]bool)
	for _, mapping := range LanguageMappings {
		langs[mapping.LanguageID] = true
	}
	result := []string{}
	for lang := range langs {
		result = append(result, lang)
	}
	return result
}

// IsLanguageFile checks if a path is a known language file
func IsLanguageFile(path string) bool {
	return GetLanguageFromPath(path) != ""
}

// MatchesExtension checks if a path matches any of the given extensions
func MatchesExtension(path string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range extensions {
		if strings.ToLower(e) == ext {
			return true
		}
	}
	return false
}

// URIFromFilepath converts a filepath to a file URI
func URIFromFilepath(path string) string {
	// Normalize path
	absPath := filepath.Clean(path)
	
	// Handle Windows paths
	if len(absPath) >= 2 && absPath[1] == ':' {
		// Windows: C:\path -> file:///C:/path
		absPath = strings.ReplaceAll(absPath, "\\", "/")
		return "file:///" + absPath
	}
	
	// Unix: /path -> file:///path
	return "file://" + absPath
}

// FilepathFromURI converts a file URI to a filepath
func FilepathFromURI(uri string) string {
	if !strings.HasPrefix(uri, "file://") {
		return ""
	}
	
	path := strings.TrimPrefix(uri, "file://")
	
	// Handle Windows paths
	if len(path) >= 3 && path[0] == '/' && path[2] == ':' {
		// file:///C:/path -> C:\path
		path = path[1:] // Remove leading /
		path = strings.ReplaceAll(path, "/", "\\")
	}
	
	return filepath.Clean(path)
}