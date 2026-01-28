// Package embedding provides text embedding generation using local models.
package embedding

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/knights-analytics/hugot"
	"github.com/knights-analytics/hugot/pipelines"
)

const (
	// ModelName is the HuggingFace model identifier
	ModelName = "sentence-transformers/all-MiniLM-L6-v2"
	// EmbeddingDim is the output dimension of all-MiniLM-L6-v2
	EmbeddingDim = 384
)

// Embedder generates vector embeddings from text using a local model.
type Embedder struct {
	session  *hugot.Session
	pipeline *pipelines.FeatureExtractionPipeline
	mu       sync.Mutex
}

// New creates an embedder, downloading the model if needed.
// modelDir specifies where to cache the model (e.g., ~/.pearls/models).
func New(modelDir string) (*Embedder, error) {
	// Expand ~ in path
	if len(modelDir) > 0 && modelDir[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		modelDir = filepath.Join(home, modelDir[1:])
	}

	// Ensure model directory exists
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return nil, fmt.Errorf("create model dir: %w", err)
	}

	// Create session with Go backend (no external dependencies)
	session, err := hugot.NewGoSession()
	if err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	// Download model if needed (cached automatically)
	downloadOptions := hugot.NewDownloadOptions()
	downloadOptions.OnnxFilePath = "onnx/model.onnx" // Use the standard (non-quantized) model
	modelPath, err := hugot.DownloadModel(ModelName, modelDir, downloadOptions)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("download model: %w", err)
	}

	// Create feature extraction pipeline
	config := hugot.FeatureExtractionConfig{
		ModelPath: modelPath,
		Name:      "pearls-embedder",
	}

	pipeline, err := hugot.NewPipeline(session, config)
	if err != nil {
		session.Destroy()
		return nil, fmt.Errorf("create pipeline: %w", err)
	}

	return &Embedder{
		session:  session,
		pipeline: pipeline,
	}, nil
}

// Close releases resources held by the embedder.
func (e *Embedder) Close() error {
	if e.session != nil {
		e.session.Destroy()
	}
	return nil
}

// Embed generates a 384-dimensional embedding for the input text.
func (e *Embedder) Embed(text string) ([]float32, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result, err := e.pipeline.RunPipeline([]string{text})
	if err != nil {
		return nil, fmt.Errorf("run pipeline: %w", err)
	}

	if len(result.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	return result.Embeddings[0], nil
}

// EmbedBatch generates embeddings for multiple texts (more efficient than individual calls).
func (e *Embedder) EmbedBatch(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	result, err := e.pipeline.RunPipeline(texts)
	if err != nil {
		return nil, fmt.Errorf("run pipeline: %w", err)
	}

	return result.Embeddings, nil
}

// DefaultModelDir returns the default model cache directory (~/.pearls/models).
func DefaultModelDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".pearls/models"
	}
	return filepath.Join(home, ".pearls", "models")
}
