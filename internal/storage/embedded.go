package storage

import (
	_ "embed"
	"fmt"
)

// Embed the embeddings files directly into the binary
var (
	//go:embed embeddings/draft.json
	embeddingsDraft []byte

	//go:embed embeddings/draft-fine.json
	embeddingsDraftFine []byte

	//go:embed embeddings/2025-06-18.json
	embeddings20250618 []byte

	//go:embed embeddings/2025-06-18-fine.json
	embeddings20250618Fine []byte

	//go:embed embeddings/2025-03-26.json
	embeddings20250326 []byte

	//go:embed embeddings/2025-03-26-fine.json
	embeddings20250326Fine []byte

	//go:embed embeddings/2024-11-05.json
	embeddings20241105 []byte

	//go:embed embeddings/2024-11-05-fine.json
	embeddings20241105Fine []byte
)

// embeddingsMap provides access to embedded embeddings by version
var embeddingsMap = map[string][]byte{
	"draft":           embeddingsDraft,
	"draft-fine":      embeddingsDraftFine,
	"2025-06-18":      embeddings20250618,
	"2025-06-18-fine": embeddings20250618Fine,
	"2025-03-26":      embeddings20250326,
	"2025-03-26-fine": embeddings20250326Fine,
	"2024-11-05":      embeddings20241105,
	"2024-11-05-fine": embeddings20241105Fine,
}

// LoadEmbeddedEmbeddings loads embeddings from embedded data
func LoadEmbeddedEmbeddings(version string) ([]byte, error) {
	data, ok := embeddingsMap[version]
	if !ok {
		return nil, fmt.Errorf("embedded embeddings not found for version: %s", version)
	}
	return data, nil
}

// GetAvailableVersions returns all embedded versions
func GetAvailableVersions() []string {
	versions := make([]string, 0, len(embeddingsMap))
	for v := range embeddingsMap {
		versions = append(versions, v)
	}
	return versions
}
