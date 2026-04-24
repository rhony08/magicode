// Package lsp provides Language Server Protocol client implementation.
package lsp

import "time"

// Constants for LSP operations
const (
	DiagnosticsDebounceMs    = 150
	DiagnosticsWaitTimeoutMs = 5000
	DiagnosticsFullWaitMs    = 10000
	DiagnosticsRequestMs     = 3000
	InitializeTimeoutMs      = 45000

	// Max tracked files (bounded for memory)
	MaxTrackedFiles = 50

	// File change types
	FileChangeCreated  = 1
	FileChangeChanged  = 2
	FileChangeDeleted  = 3

	// Text document sync kinds
	TextDocumentSyncNone       = 0
	TextDocumentSyncFull       = 1
	TextDocumentSyncIncremental = 2
)

// ServerID is a unique identifier for an LSP server
type ServerID string

// Well-known server IDs
const (
	ServerTypeScript ServerID = "typescript"
	ServerGo         ServerID = "go"
	ServerPython     ServerID = "python"
	ServerRust       ServerID = "rust"
	ServerJava       ServerID = "java"
	ServerC          ServerID = "c"
	ServerCpp        ServerID = "cpp"
	ServerRuby       ServerID = "ruby"
	ServerPHP        ServerID = "php"
	ServerJSON       ServerID = "json"
	ServerYAML       ServerID = "yaml"
	ServerHTML       ServerID = "html"
	ServerCSS        ServerID = "css"
)

// DiagnosticSeverity represents diagnostic severity level
type DiagnosticSeverity int

const (
	SeverityError       DiagnosticSeverity = 1
	SeverityWarning     DiagnosticSeverity = 2
	SeverityInformation DiagnosticSeverity = 3
	SeverityHint        DiagnosticSeverity = 4
)

// DiagnosticTag represents diagnostic tags
type DiagnosticTag int

const (
	TagUnnecessary  DiagnosticTag = 1
	TagDeprecated   DiagnosticTag = 2
)

// Position represents a position in a document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// Range represents a range in a document
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Diagnostic represents a LSP diagnostic
type Diagnostic struct {
	Range           Range              `json:"range"`
	Message         string             `json:"message"`
	Severity        DiagnosticSeverity `json:"severity,omitempty"`
	Code            interface{}        `json:"code,omitempty"` // string or int
	CodeDescription *CodeDescription   `json:"codeDescription,omitempty"`
	Source          string             `json:"source,omitempty"`
	Tags            []DiagnosticTag    `json:"tags,omitempty"`
	RelatedInfo     []DiagnosticRelatedInfo `json:"relatedInformation,omitempty"`
	Data            interface{}        `json:"data,omitempty"`
}

// CodeDescription represents a code description
type CodeDescription struct {
	Href string `json:"href"`
}

// DiagnosticRelatedInfo represents related diagnostic information
type DiagnosticRelatedInfo struct {
	Location Location `json:"location"`
	Message  string   `json:"message"`
}

// Location represents a location in a document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// TextDocumentItem represents a text document
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// TextDocumentIdentifier represents a text document identifier
type TextDocumentIdentifier struct {
	URI string `json:"uri"`
}

// VersionedTextDocumentIdentifier represents a versioned text document identifier
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent represents a content change event
type TextDocumentContentChangeEvent struct {
	Range       *Range `json:"range,omitempty"`
	RangeLength *int   `json:"rangeLength,omitempty"`
	Text        string `json:"text"`
}

// TextDocumentEdit represents a text document edit
type TextDocumentEdit struct {
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`
	Edits        []TextEdit                      `json:"edits"`
}

// TextEdit represents a text edit
type TextEdit struct {
	Range Range `json:"range"`
	Text  string `json:"newText"`
}

// FileEvent represents a file event
type FileEvent struct {
	URI   string `json:"uri"`
	Type  int    `json:"type"` // created, changed, deleted
}

// PublishDiagnosticsParams represents publish diagnostics params
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Version     *int         `json:"version,omitempty"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// DocumentDiagnosticParams represents document diagnostic params
type DocumentDiagnosticParams struct {
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DocumentDiagnosticReport represents document diagnostic report
type DocumentDiagnosticReport struct {
	Kind  string       `json:"kind"` // "full" or "unchanged"
	Result []Diagnostic `json:"items,omitempty"`
}

// WorkspaceDiagnosticParams represents workspace diagnostic params
type WorkspaceDiagnosticParams struct {
	PreviousResultIds []PreviousResultId `json:"previousResultIds,omitempty"`
}

// PreviousResultId represents a previous result ID
type PreviousResultId struct {
	URI    string `json:"uri"`
	Value  string `json:"value"`
}

// WorkspaceDiagnosticReport represents workspace diagnostic report
type WorkspaceDiagnosticReport struct {
	Items []WorkspaceDiagnosticItem `json:"items,omitempty"`
}

// WorkspaceDiagnosticItem represents a workspace diagnostic item
type WorkspaceDiagnosticItem struct {
	URI         string       `json:"uri,omitempty"`
	Version     *int         `json:"version,omitempty"`
	Result      []Diagnostic `json:"items,omitempty"`
	Kind        string       `json:"kind"`
}

// ServerCapabilities represents LSP server capabilities
type ServerCapabilities struct {
	TextDocumentSync        interface{}        `json:"textDocumentSync,omitempty"`
	CompletionProvider      *CompletionOptions `json:"completionProvider,omitempty"`
	HoverProvider           interface{}        `json:"hoverProvider,omitempty"`
	SignatureHelpProvider   *SignatureHelpOptions `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider      interface{}        `json:"definitionProvider,omitempty"`
	TypeDefinitionProvider  interface{}        `json:"typeDefinitionProvider,omitempty"`
	ImplementationProvider  interface{}        `json:"implementationProvider,omitempty"`
	ReferencesProvider      interface{}        `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider interface{}   `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider  interface{}       `json:"documentSymbolProvider,omitempty"`
	CodeActionProvider      interface{}       `json:"codeActionProvider,omitempty"`
	CodeLensProvider        *CodeLensOptions   `json:"codeLensProvider,omitempty"`
	DocumentFormattingProvider interface{} `json:"documentFormattingProvider,omitempty"`
	DocumentRangeFormattingProvider interface{} `json:"documentRangeFormattingProvider,omitempty"`
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
	RenameProvider          interface{}        `json:"renameProvider,omitempty"`
	DiagnosticProvider      interface{}        `json:"diagnosticProvider,omitempty"`
	FoldingRangeProvider    interface{}        `json:"foldingRangeProvider,omitempty"`
	ExecuteCommandProvider  *ExecuteCommandOptions `json:"executeCommandProvider,omitempty"`
	Workspace               *WorkspaceCapabilities `json:"workspace,omitempty"`
}

// CompletionOptions represents completion options
type CompletionOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	ResolveProvider      bool     `json:"resolveProvider,omitempty"`
	CompletionItem       interface{} `json:"completionItem,omitempty"`
}

// SignatureHelpOptions represents signature help options
type SignatureHelpOptions struct {
	TriggerCharacters   []string `json:"triggerCharacters,omitempty"`
	ReTriggerCharacters []string `json:"retriggerCharacters,omitempty"`
}

// CodeLensOptions represents code lens options
type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

// DocumentOnTypeFormattingOptions represents document on type formatting options
type DocumentOnTypeFormattingOptions struct {
	FirstTriggerCharacter string   `json:"firstTriggerCharacter"`
	MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
}

// ExecuteCommandOptions represents execute command options
type ExecuteCommandOptions struct {
	Commands []string `json:"commands"`
}

// WorkspaceCapabilities represents workspace capabilities
type WorkspaceCapabilities struct {
	WorkspaceFolders *WorkspaceFoldersCapabilities `json:"workspaceFolders,omitempty"`
	FileOperations   *FileOperationsCapabilities   `json:"fileOperations,omitempty"`
}

// WorkspaceFoldersCapabilities represents workspace folders capabilities
type WorkspaceFoldersCapabilities struct {
	Supported           bool `json:"supported,omitempty"`
	ChangeNotifications interface{} `json:"changeNotifications,omitempty"`
}

// FileOperationsCapabilities represents file operations capabilities
type FileOperationsCapabilities struct {
	DidCreate  *FileOperationOptions `json:"didCreate,omitempty"`
	WillCreate *FileOperationOptions `json:"willCreate,omitempty"`
	DidRename  *FileOperationOptions `json:"didRename,omitempty"`
	WillRename *FileOperationOptions `json:"willRename,omitempty"`
	DidDelete  *FileOperationOptions `json:"didDelete,omitempty"`
	WillDelete *FileOperationOptions `json:"willDelete,omitempty"`
}

// FileOperationOptions represents file operation options
type FileOperationOptions struct {
	Filters []FileOperationFilter `json:"filters,omitempty"`
}

// FileOperationFilter represents file operation filter
type FileOperationFilter struct {
	Scheme   string             `json:"scheme,omitempty"`
	Pattern  FileOperationPattern `json:"pattern"`
}

// FileOperationPattern represents file operation pattern
type FileOperationPattern struct {
	Glob   string `json:"glob"`
	Matches string `json:"matches,omitempty"`
	Options *FileOperationPatternOptions `json:"options,omitempty"`
}

// FileOperationPatternOptions represents file operation pattern options
type FileOperationPatternOptions struct {
	IgnoreCase bool `json:"ignoreCase,omitempty"`
}

// InitializeParams represents initialize params
type InitializeParams struct {
	ProcessID        *int                   `json:"processId,omitempty"`
	ClientInfo       *ClientInfo            `json:"clientInfo,omitempty"`
	Locale           string                 `json:"locale,omitempty"`
	RootPath         *string                `json:"rootPath,omitempty"`
	RootURI          *string                `json:"rootUri,omitempty"`
	Capabilities     ClientCapabilities     `json:"capabilities"`
	Trace            string                 `json:"trace,omitempty"` // "off", "messages", "verbose"
	WorkspaceFolders []WorkspaceFolder      `json:"workspaceFolders,omitempty"`
}

// ClientInfo represents client info
type ClientInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// ClientCapabilities represents client capabilities
type ClientCapabilities struct {
	Workspace   *WorkspaceClientCapabilities `json:"workspace,omitempty"`
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`
	General     *GeneralClientCapabilities    `json:"general,omitempty"`
	Experimental interface{}                  `json:"experimental,omitempty"`
}

// WorkspaceClientCapabilities represents workspace client capabilities
type WorkspaceClientCapabilities struct {
	ApplyEdit              bool                      `json:"applyEdit,omitempty"`
	WorkspaceEdit          *WorkspaceEditCapabilities `json:"workspaceEdit,omitempty"`
 DidChangeConfiguration  *DidChangeConfigurationCapabilities `json:"didChangeConfiguration,omitempty"`
 DidChangeWatchedFiles   *DidChangeWatchedFilesCapabilities `json:"didChangeWatchedFiles,omitempty"`
 Symbol                  *SymbolCapabilities `json:"symbol,omitempty"`
 ExecuteCommand          *ExecuteCommandCapabilities `json:"executeCommand,omitempty"`
 WorkspaceFolders        bool `json:"workspaceFolders,omitempty"`
 Configuration           bool `json:"configuration,omitempty"`
 Diagnostic              *DiagnosticWorkspaceCapabilities `json:"diagnostic,omitempty"`
}

// TextDocumentClientCapabilities represents text document client capabilities
type TextDocumentClientCapabilities struct {
	Synchronization *TextDocumentSyncClientCapabilities `json:"synchronization,omitempty"`
	Completion      *CompletionClientCapabilities `json:"completion,omitempty"`
	Hover           *HoverClientCapabilities `json:"hover,omitempty"`
	SignatureHelp   *SignatureHelpClientCapabilities `json:"signatureHelp,omitempty"`
	Declaration     *DeclarationClientCapabilities `json:"declaration,omitempty"`
	Definition      *DefinitionClientCapabilities `json:"definition,omitempty"`
	TypeDefinition  *TypeDefinitionClientCapabilities `json:"typeDefinition,omitempty"`
	Implementation  *ImplementationClientCapabilities `json:"implementation,omitempty"`
	References      *ReferenceClientCapabilities `json:"references,omitempty"`
	DocumentHighlight *DocumentHighlightClientCapabilities `json:"documentHighlight,omitempty"`
	DocumentSymbol   *DocumentSymbolClientCapabilities `json:"documentSymbol,omitempty"`
	CodeAction       *CodeActionClientCapabilities `json:"codeAction,omitempty"`
	CodeLens         *CodeLensClientCapabilities `json:"codeLens,omitempty"`
	DocumentLink     *DocumentLinkClientCapabilities `json:"documentLink,omitempty"`
	ColorProvider    *DocumentColorClientCapabilities `json:"colorProvider,omitempty"`
 Formatting       *FormattingClientCapabilities `json:"formatting,omitempty"`
 RangeFormatting  *RangeFormattingClientCapabilities `json:"rangeFormatting,omitempty"`
 OnTypeFormatting *OnTypeFormattingClientCapabilities `json:"onTypeFormatting,omitempty"`
 Rename           *RenameClientCapabilities `json:"rename,omitempty"`
 PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`
 FoldingRange     *FoldingRangeClientCapabilities `json:"foldingRange,omitempty"`
 Diagnostic       *DiagnosticTextDocumentCapabilities `json:"diagnostic,omitempty"`
}

// GeneralClientCapabilities represents general client capabilities
type GeneralClientCapabilities struct {
	StaleRequestSupport *StaleRequestSupportCapabilities `json:"staleRequestSupport,omitempty"`
 RegularExpressions   *RegularExpressionsCapabilities `json:"regularExpressions,omitempty"`
 Markdown            *MarkdownCapabilities `json:"markdown,omitempty"`
}

// WorkspaceFolder represents a workspace folder
type WorkspaceFolder struct {
	URI  string `json:"uri"`
	Name string `json:"name"`
}

// InitializeResult represents initialize result
type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities"`
	ServerInfo   *ServerInfo        `json:"serverInfo,omitempty"`
}

// ServerInfo represents server info
type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
}

// FileState represents tracked file state (bounded - don't store full text)
type FileState struct {
	URI        string
	Version    int
	LastAccess time.Time
	// Don't store full text - only track version for memory efficiency
}

// DiagnosticEvent represents a diagnostic event
type DiagnosticEvent struct {
	ServerID ServerID
	URI      string
	Diagnostics []Diagnostic
}

// ===========================================
// Client Capability Types (simplified)
// ===========================================

// WorkspaceEditCapabilities represents workspace edit capabilities
type WorkspaceEditCapabilities struct {
	DocumentChanges bool `json:"documentChanges,omitempty"`
}

// DidChangeConfigurationCapabilities represents did change configuration capabilities
type DidChangeConfigurationCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DidChangeWatchedFilesCapabilities represents did change watched files capabilities
type DidChangeWatchedFilesCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// SymbolCapabilities represents symbol capabilities
type SymbolCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// ExecuteCommandCapabilities represents execute command capabilities
type ExecuteCommandCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DiagnosticWorkspaceCapabilities represents diagnostic workspace capabilities
type DiagnosticWorkspaceCapabilities struct {
	RefreshSupport bool `json:"refreshSupport,omitempty"`
}

// TextDocumentSyncClientCapabilities represents text document sync client capabilities
type TextDocumentSyncClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	WillSave            bool `json:"willSave,omitempty"`
	WillSaveWaitUntil   bool `json:"willSaveWaitUntil,omitempty"`
	DidSave             bool `json:"didSave,omitempty"`
}

// CompletionClientCapabilities represents completion client capabilities
type CompletionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// HoverClientCapabilities represents hover client capabilities
type HoverClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// SignatureHelpClientCapabilities represents signature help client capabilities
type SignatureHelpClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DeclarationClientCapabilities represents declaration client capabilities
type DeclarationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DefinitionClientCapabilities represents definition client capabilities
type DefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// TypeDefinitionClientCapabilities represents type definition client capabilities
type TypeDefinitionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// ImplementationClientCapabilities represents implementation client capabilities
type ImplementationClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// ReferenceClientCapabilities represents reference client capabilities
type ReferenceClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentHighlightClientCapabilities represents document highlight client capabilities
type DocumentHighlightClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentSymbolClientCapabilities represents document symbol client capabilities
type DocumentSymbolClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// CodeActionClientCapabilities represents code action client capabilities
type CodeActionClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// CodeLensClientCapabilities represents code lens client capabilities
type CodeLensClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentLinkClientCapabilities represents document link client capabilities
type DocumentLinkClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DocumentColorClientCapabilities represents document color client capabilities
type DocumentColorClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// FormattingClientCapabilities represents formatting client capabilities
type FormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// RangeFormattingClientCapabilities represents range formatting client capabilities
type RangeFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// OnTypeFormattingClientCapabilities represents on type formatting client capabilities
type OnTypeFormattingClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// RenameClientCapabilities represents rename client capabilities
type RenameClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// PublishDiagnosticsClientCapabilities represents publish diagnostics client capabilities
type PublishDiagnosticsClientCapabilities struct {
	RelatedInformation bool `json:"relatedInformation,omitempty"`
	VersionSupport     bool `json:"versionSupport,omitempty"`
	DataSupport        bool `json:"dataSupport,omitempty"`
}

// FoldingRangeClientCapabilities represents folding range client capabilities
type FoldingRangeClientCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
}

// DiagnosticTextDocumentCapabilities represents diagnostic text document capabilities
type DiagnosticTextDocumentCapabilities struct {
	DynamicRegistration bool `json:"dynamicRegistration,omitempty"`
	RelatedDocumentSupport bool `json:"relatedDocumentSupport,omitempty"`
}

// StaleRequestSupportCapabilities represents stale request support capabilities
type StaleRequestSupportCapabilities struct {
	Cancel bool `json:"cancel,omitempty"`
}

// RegularExpressionsCapabilities represents regular expressions capabilities
type RegularExpressionsCapabilities struct {
	Engine string `json:"engine,omitempty"`
}

// MarkdownCapabilities represents markdown capabilities
type MarkdownCapabilities struct {
	Version string `json:"version,omitempty"`
}