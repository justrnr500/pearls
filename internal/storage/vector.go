package storage

import (
	"fmt"

	sqlite_vec "github.com/asg017/sqlite-vec-go-bindings/cgo"
)

// SemanticResult represents a search result with similarity score.
type SemanticResult struct {
	ID       string
	Distance float32 // Lower = more similar (L2 distance)
}

// InsertEmbedding stores a pearl's embedding in the vector index.
func (d *DB) InsertEmbedding(id string, embedding []float32) error {
	serialized, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return fmt.Errorf("serialize embedding: %w", err)
	}

	_, err = d.db.Exec(
		"INSERT INTO pearl_embeddings(rowid, embedding) VALUES ((SELECT rowid FROM pearls WHERE id = ?), ?)",
		id, serialized,
	)
	if err != nil {
		return fmt.Errorf("insert embedding: %w", err)
	}

	return nil
}

// UpdateEmbedding replaces a pearl's embedding in the vector index.
func (d *DB) UpdateEmbedding(id string, embedding []float32) error {
	serialized, err := sqlite_vec.SerializeFloat32(embedding)
	if err != nil {
		return fmt.Errorf("serialize embedding: %w", err)
	}

	// Delete existing and insert new (vec0 tables don't support UPDATE)
	_, err = d.db.Exec(
		"DELETE FROM pearl_embeddings WHERE rowid = (SELECT rowid FROM pearls WHERE id = ?)",
		id,
	)
	if err != nil {
		return fmt.Errorf("delete old embedding: %w", err)
	}

	_, err = d.db.Exec(
		"INSERT INTO pearl_embeddings(rowid, embedding) VALUES ((SELECT rowid FROM pearls WHERE id = ?), ?)",
		id, serialized,
	)
	if err != nil {
		return fmt.Errorf("insert new embedding: %w", err)
	}

	return nil
}

// DeleteEmbedding removes a pearl's embedding from the vector index.
func (d *DB) DeleteEmbedding(id string) error {
	_, err := d.db.Exec(
		"DELETE FROM pearl_embeddings WHERE rowid = (SELECT rowid FROM pearls WHERE id = ?)",
		id,
	)
	if err != nil {
		return fmt.Errorf("delete embedding: %w", err)
	}

	return nil
}

// SearchSemantic finds pearls by vector similarity.
// Returns results ordered by distance (closest first).
func (d *DB) SearchSemantic(queryEmbedding []float32, limit int) ([]SemanticResult, error) {
	if limit <= 0 {
		limit = 10
	}

	serialized, err := sqlite_vec.SerializeFloat32(queryEmbedding)
	if err != nil {
		return nil, fmt.Errorf("serialize query: %w", err)
	}

	// Use CTE with k constraint for compatibility with JOINs
	// (LIMIT alone doesn't work with JOINs in sqlite-vec)
	rows, err := d.db.Query(`
		WITH knn_matches AS (
			SELECT rowid, distance
			FROM pearl_embeddings
			WHERE embedding MATCH ?
			AND k = ?
		)
		SELECT p.id, k.distance
		FROM knn_matches k
		JOIN pearls p ON p.rowid = k.rowid
		ORDER BY k.distance
	`, serialized, limit)
	if err != nil {
		return nil, fmt.Errorf("search embeddings: %w", err)
	}
	defer rows.Close()

	var results []SemanticResult
	for rows.Next() {
		var r SemanticResult
		if err := rows.Scan(&r.ID, &r.Distance); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// ClearEmbeddings removes all embeddings from the vector index.
// Used for rebuilding the index.
func (d *DB) ClearEmbeddings() error {
	_, err := d.db.Exec("DELETE FROM pearl_embeddings")
	if err != nil {
		return fmt.Errorf("clear embeddings: %w", err)
	}
	return nil
}

// HasEmbedding checks if a pearl has an embedding.
func (d *DB) HasEmbedding(id string) (bool, error) {
	var count int
	err := d.db.QueryRow(
		"SELECT COUNT(*) FROM pearl_embeddings WHERE rowid = (SELECT rowid FROM pearls WHERE id = ?)",
		id,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check embedding: %w", err)
	}
	return count > 0, nil
}

// EmbeddingCount returns the number of embeddings in the index.
func (d *DB) EmbeddingCount() (int, error) {
	var count int
	err := d.db.QueryRow("SELECT COUNT(*) FROM pearl_embeddings").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count embeddings: %w", err)
	}
	return count, nil
}
