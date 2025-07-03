package validator

import (
	"strings"
)

// expandClaimForSearch expands a claim into multiple search queries to improve spec matching
func expandClaimForSearch(claim string) []string {
	queries := []string{claim} // Always include original

	// Normalize the claim
	normalized := strings.ToLower(claim)

	// Handle compound claims with "and"
	if strings.Contains(normalized, " and ") {
		// Split compound claims and search for each part
		parts := strings.Split(normalized, " and ")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				queries = append(queries, part)
			}
		}
	}

	// Common expansions for initialization/capability concepts
	if strings.Contains(normalized, "initialization") || strings.Contains(normalized, "initialize") {
		queries = append(queries,
			"initialization phase client server",
			"initialize request response",
			"first interaction between client and server",
		)
	}

	if strings.Contains(normalized, "capability") || strings.Contains(normalized, "capabilities") {
		queries = append(queries,
			"exchange capabilities",
			"capability negotiation",
			"declare capabilities",
			"negotiate capabilities",
		)
	}

	if strings.Contains(normalized, "protocol version") {
		queries = append(queries,
			"protocol version compatibility",
			"establish protocol version",
			"version compatibility",
		)
	}

	// Tool-related expansions
	if strings.Contains(normalized, "tools/call") || strings.Contains(normalized, "tool invocation") {
		queries = append(queries,
			"tools/call request",
			"invoke tool",
			"tool invocation",
			"Client->>Server: tools/call",
		)
	}

	// Tool and Resource exposure expansions
	if strings.Contains(normalized, "exposes") && (strings.Contains(normalized, "tools") || strings.Contains(normalized, "resources")) {
		queries = append(queries,
			"servers expose tools",
			"servers expose resources",
			"expose resources and tools",
			"inputSchema JSON Schema defining expected parameters",
			"tool inputSchema",
		)
	}

	if strings.Contains(normalized, "tool") && strings.Contains(normalized, "result") {
		queries = append(queries,
			"tool result",
			"Server-->>Client: Tool result",
			"receive results",
		)
	}

	// Resource-related expansions
	if strings.Contains(normalized, "resource") {
		queries = append(queries,
			"expose resources",
			"resources with URI",
			"resource name title mimeType",
		)
	}

	// Security-related expansions
	if strings.Contains(normalized, "security") || strings.Contains(normalized, "best practice") {
		queries = append(queries,
			"security best practices",
			"security considerations",
			"SHOULD implement security",
			"trust safety security",
		)
	}

	// Schema-related expansions
	if strings.Contains(normalized, "schema") || strings.Contains(normalized, "json schema") {
		queries = append(queries,
			"inputSchema outputSchema",
			"tool schema",
			"JSON Schema",
			"metadata describing schema",
			"tools inputSchema JSON Schema",
			"JSON Schema defining expected parameters",
			"resources JSON Schema",
			"schema validation",
		)
	}

	// MCP general expansions
	if strings.Contains(normalized, "mcp") && strings.Contains(normalized, "standardized") {
		queries = append(queries,
			"Model Context Protocol provides standardized",
			"MCP provides a standardized way",
			"standardized way for servers",
		)
	}

	return queries
}
