package embedding

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmbedder(t *testing.T) {
	// Use temp directory for model cache during tests
	tmpDir, err := os.MkdirTemp("", "pearls-embedding-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	modelDir := filepath.Join(tmpDir, "models")

	t.Log("Creating embedder (this will download the model on first run)...")
	embedder, err := New(modelDir)
	if err != nil {
		t.Fatalf("create embedder: %v", err)
	}
	defer embedder.Close()

	// Test single embedding
	t.Run("SingleEmbed", func(t *testing.T) {
		text := "The quick brown fox jumps over the lazy dog"
		embedding, err := embedder.Embed(text)
		if err != nil {
			t.Fatalf("embed: %v", err)
		}

		if len(embedding) != EmbeddingDim {
			t.Errorf("embedding dim = %d, want %d", len(embedding), EmbeddingDim)
		}

		// Verify embedding is not all zeros
		var sum float32
		for _, v := range embedding {
			sum += v * v
		}
		if sum == 0 {
			t.Error("embedding should not be all zeros")
		}
		t.Logf("Embedding norm squared: %f", sum)
	})

	// Test batch embedding
	t.Run("BatchEmbed", func(t *testing.T) {
		texts := []string{
			"Hello world",
			"Goodbye world",
			"The weather is nice today",
		}

		embeddings, err := embedder.EmbedBatch(texts)
		if err != nil {
			t.Fatalf("embed batch: %v", err)
		}

		if len(embeddings) != len(texts) {
			t.Errorf("got %d embeddings, want %d", len(embeddings), len(texts))
		}

		for i, emb := range embeddings {
			if len(emb) != EmbeddingDim {
				t.Errorf("embedding[%d] dim = %d, want %d", i, len(emb), EmbeddingDim)
			}
		}
	})

	// Test semantic similarity
	t.Run("SemanticSimilarity", func(t *testing.T) {
		texts := []string{
			"The cat sat on the mat",
			"A feline rested on the rug",
			"The stock market crashed today",
		}

		embeddings, err := embedder.EmbedBatch(texts)
		if err != nil {
			t.Fatalf("embed batch: %v", err)
		}

		// Calculate cosine similarities
		sim01 := cosineSimilarity(embeddings[0], embeddings[1])
		sim02 := cosineSimilarity(embeddings[0], embeddings[2])

		t.Logf("Similarity (cat/feline): %f", sim01)
		t.Logf("Similarity (cat/stock): %f", sim02)

		// "cat on mat" should be more similar to "feline on rug" than "stock market"
		if sim01 <= sim02 {
			t.Errorf("expected cat/feline similarity (%f) > cat/stock similarity (%f)", sim01, sim02)
		}
	})
}

func TestDefaultModelDir(t *testing.T) {
	dir := DefaultModelDir()
	if dir == "" {
		t.Error("DefaultModelDir should not be empty")
	}
	t.Logf("Default model dir: %s", dir)
}

// cosineSimilarity calculates the cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float32) float32 {
	// Newton's method for square root
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
