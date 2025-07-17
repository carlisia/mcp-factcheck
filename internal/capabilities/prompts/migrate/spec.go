package migrate

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/carlisia/mcp-factcheck/internal/capabilities"
	"github.com/carlisia/mcp-factcheck/internal/capabilities/rules"
	"github.com/mark3labs/mcp-go/mcp"
)

// MigrateRequest represents a request to migrate MCP content between versions
type MigrateRequest struct {
	// CurrentVersion is the MCP version the content is currently based on
	CurrentVersion string `json:"current_version"`
	
	// TargetVersion is the MCP version to migrate the content to
	TargetVersion string `json:"target_version"`
	
	// UpdateScope controls the type of updates to make
	// Valid values: "comprehensive" (default), "critical_only", "enhancement_focused"
	UpdateScope string `json:"update_scope,omitempty"`
}

// Update scope constants
const (
	scopeComprehensive      = "comprehensive"      // Default scope: all updates including improvements and best practices
	scopeCriticalOnly       = "critical_only"       // Only breaking changes and critical fixes
	scopeEnhancementFocused = "enhancement_focused" // Focus on improvements and new features
)

// validUpdateScopes lists all supported update scopes for content migration
var validUpdateScopes = []string{scopeComprehensive, scopeCriticalOnly, scopeEnhancementFocused}

// Pre-computed validation maps
var (
	validVersionSet = make(map[string]bool) // Valid MCP specification versions
	validScopeSet   = make(map[string]bool) // Valid update scope values
)

//go:embed templates/migrate-content.tmpl
var migrateContentTemplate string

// Cached template for performance
var migrateTemplate *template.Template

// init performs all package-level initialization in a single, structured function
func init() {
	initValidationSets()
	initTemplates()
}

// initValidationSets initializes the validation lookup maps
func initValidationSets() {
	// Initialize version set from capabilities.ValidSpecVersions
	for _, v := range capabilities.ValidSpecVersions {
		validVersionSet[v] = true
	}
	
	// Initialize scope set
	for _, s := range validUpdateScopes {
		validScopeSet[s] = true
	}
}

// initTemplates parses and caches the embedded template
func initTemplates() {
	var err error
	migrateTemplate, err = template.New("migrate-content").Funcs(template.FuncMap{
		"AccuracyCheckingRulesShort":  func() string { return rules.AccuracyCheckingShort },
		"StylePreservationGuidelines": func() string { return rules.StylePreservation },
		"SpecificationGuidanceNote":   func() string { return rules.SpecificationGuidance },
	}).Parse(migrateContentTemplate)
	if err != nil {
		panic(fmt.Sprintf("failed to parse migrate template: %v", err))
	}
}

// ParseMigrateArgs parses raw arguments into a validated MigrateRequest
func ParseMigrateArgs(args map[string]string) (*MigrateRequest, error) {
	// Extract parameters
	currentVersion, ok := args["current_version"]
	if !ok || currentVersion == "" {
		return nil, fmt.Errorf("required argument missing: current_version")
	}
	
	targetVersion, ok := args["target_version"]
	if !ok || targetVersion == "" {
		return nil, fmt.Errorf("required argument missing: target_version")
	}
	
	updateScope := args["update_scope"]
	
	// Validate and build request
	return validateMigrateRequest(currentVersion, targetVersion, updateScope)
}

// validateMigrateRequest validates and normalizes a migrate request
func validateMigrateRequest(currentVersion, targetVersion, updateScope string) (*MigrateRequest, error) {
	return newMigrateRequestBuilder().
		WithCurrentVersion(currentVersion).
		WithTargetVersion(targetVersion).
		WithUpdateScope(updateScope).
		Build()
}

// newDefaultMigrateRequest creates a MigrateRequest with default values
func newDefaultMigrateRequest() MigrateRequest {
	return MigrateRequest{
		UpdateScope: scopeComprehensive,
	}
}

// migrateRequestBuilder builds and validates MigrateRequest instances
type migrateRequestBuilder struct {
	request MigrateRequest
	errors  []error
}

// newMigrateRequestBuilder creates a new builder for MigrateRequest
func newMigrateRequestBuilder() *migrateRequestBuilder {
	return &migrateRequestBuilder{
		request: newDefaultMigrateRequest(),
	}
}

// WithCurrentVersion sets the current version
func (b *migrateRequestBuilder) WithCurrentVersion(version string) *migrateRequestBuilder {
	if version == "" {
		b.errors = append(b.errors, fmt.Errorf("current_version cannot be empty"))
	} else if !isValidVersion(version) {
		b.errors = append(b.errors, fmt.Errorf("invalid current_version: %s", version))
	}
	b.request.CurrentVersion = version
	return b
}

// WithTargetVersion sets the target version
func (b *migrateRequestBuilder) WithTargetVersion(version string) *migrateRequestBuilder {
	if version == "" {
		b.errors = append(b.errors, fmt.Errorf("target_version cannot be empty"))
	} else if !isValidVersion(version) {
		b.errors = append(b.errors, fmt.Errorf("invalid target_version: %s", version))
	}
	b.request.TargetVersion = version
	return b
}

// WithUpdateScope sets the update scope
func (b *migrateRequestBuilder) WithUpdateScope(scope string) *migrateRequestBuilder {
	if scope == "" {
		scope = scopeComprehensive
	}
	if !isValidScope(scope) {
		b.errors = append(b.errors, fmt.Errorf("invalid update_scope: %s (must be comprehensive, critical_only, or enhancement_focused)", scope))
	}
	b.request.UpdateScope = scope
	return b
}

// Build returns the built request or an error if validation failed
func (b *migrateRequestBuilder) Build() (*MigrateRequest, error) {
	if len(b.errors) > 0 {
		return nil, fmt.Errorf("validation errors: %s", joinErrors(b.errors))
	}
	return &b.request, nil
}

// joinErrors aggregates multiple errors into a clear, readable string
func joinErrors(errs []error) string {
	var errStrs []string
	for _, err := range errs {
		errStrs = append(errStrs, err.Error())
	}
	return strings.Join(errStrs, "; ")
}

// isValidVersion checks if a version is valid
func isValidVersion(version string) bool {
	return validVersionSet[version]
}

// isValidScope checks if an update scope is valid
func isValidScope(scope string) bool {
	return validScopeSet[scope]
}

// Render renders the migrate content prompt with a validated request
func Render(req *MigrateRequest) (*mcp.GetPromptResult, error) {
	var buf bytes.Buffer
	data := map[string]string{
		"current_version": req.CurrentVersion,
		"target_version":  req.TargetVersion,
		"update_scope":    req.UpdateScope,
	}
	
	if err := migrateTemplate.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	
	return &mcp.GetPromptResult{
		Messages: []mcp.PromptMessage{
			{
				Role:    "user",
				Content: mcp.NewTextContent(buf.String()),
			},
		},
	}, nil
}