package introspect

import (
	"fmt"
	"sort"
	"time"

	"github.com/justrnr500/pearls/internal/pearl"
)

// GeneratedPearl pairs a Pearl with its markdown content.
type GeneratedPearl struct {
	Pearl            pearl.Pearl
	GeneratedContent string
}

// GeneratePearls creates pearls from introspected database tables.
// prefix is the dot-separated namespace prefix (e.g. "mydb").
// tables maps schema names to their discovered tables.
// envVar is the environment variable holding the connection string.
func GeneratePearls(prefix string, tables map[string][]Table, envVar string) []GeneratedPearl {
	now := time.Now()
	var results []GeneratedPearl

	// Database pearl
	dbPearl := GeneratedPearl{
		Pearl: pearl.Pearl{
			ID:        prefix,
			Name:      pearl.LastSegment(prefix),
			Namespace: pearl.ParentNamespace(prefix),
			Type:      pearl.TypeDatabase,
			Status:    pearl.StatusActive,
			CreatedBy: "pearls-introspect",
			CreatedAt: now,
			UpdatedAt: now,
			Connection: &pearl.ConnectionInfo{
				Host: fmt.Sprintf("${%s}", envVar),
			},
		},
	}
	results = append(results, dbPearl)

	// Sort schema names for deterministic output
	schemaNames := make([]string, 0, len(tables))
	for schema := range tables {
		schemaNames = append(schemaNames, schema)
	}
	sort.Strings(schemaNames)

	for _, schema := range schemaNames {
		tbls := tables[schema]
		schemaID := prefix + "." + schema

		// Schema pearl
		schemaPearl := GeneratedPearl{
			Pearl: pearl.Pearl{
				ID:        schemaID,
				Name:      schema,
				Namespace: prefix,
				Type:      pearl.TypeSchema,
				Status:    pearl.StatusActive,
				Parent:    prefix,
				CreatedBy: "pearls-introspect",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}
		results = append(results, schemaPearl)

		for _, tbl := range tbls {
			tableID := schemaID + "." + tbl.Name

			// Build references from foreign keys
			var refs []string
			for _, fk := range tbl.ForeignKeys {
				refSchema := fk.ReferencesSchema
				if refSchema == "" {
					refSchema = schema
				}
				refID := prefix + "." + refSchema + "." + fk.ReferencesTable
				refs = append(refs, refID)
			}

			content := GenerateTableContent(tbl, prefix)

			tablePearl := GeneratedPearl{
				Pearl: pearl.Pearl{
					ID:         tableID,
					Name:       tbl.Name,
					Namespace:  schemaID,
					Type:       pearl.TypeTable,
					Status:     pearl.StatusActive,
					Parent:     schemaID,
					References: refs,
					CreatedBy:  "pearls-introspect",
					CreatedAt:  now,
					UpdatedAt:  now,
				},
				GeneratedContent: content,
			}
			results = append(results, tablePearl)
		}
	}

	return results
}
