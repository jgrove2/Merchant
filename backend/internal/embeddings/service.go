package embeddings

import (
	"context"
	"fmt"
	"log"

	"github.com/nlpodyssey/cybertron/pkg/models/bert"
	"github.com/nlpodyssey/cybertron/pkg/tasks"
	"github.com/nlpodyssey/cybertron/pkg/tasks/textencoding"
)

// Service defines the interface for generating embeddings
type Service interface {
	Generate(text string) ([]float32, error)
	Close() error
}

type localService struct {
	model textencoding.Interface
}

// NewService initializes the local embedding model (all-MiniLM-L6-v2)
func NewService() (Service, error) {
	log.Println("Initializing local embedding model (all-MiniLM-L6-v2)...")

	// Load the model. Cybertron handles downloading/caching automatically.
	// We use the default configuration which usually targets all-MiniLM-L6-v2.
	// Ideally, we explicitly specify the model to be safe.
	model, err := tasks.Load[textencoding.Interface](&tasks.Config{
		ModelsDir: "models", // Store models in a local "models" directory
		ModelName: "sentence-transformers/all-MiniLM-L6-v2",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load embedding model: %w", err)
	}

	return &localService{model: model}, nil
}

// Generate creates a vector embedding for the given text
func (s *localService) Generate(text string) ([]float32, error) {
	if s.model == nil {
		return nil, fmt.Errorf("model not initialized")
	}

	result, err := s.model.Encode(context.Background(), text, int(bert.MeanPooling))
	if err != nil {
		return nil, err
	}

	// Cybertron returns a mat.Matrix interface.
	// For most vectors, it should be a dense matrix we can iterate or extract data from.
	// Check the concrete type or use the Data() method if available,
	// but Encode returns a *textencoding.Response which has a Vector field of type mat.Matrix.

	// Use Dims() instead of Rows/Columns.
	// Error indicates Dims returns a single int value.
	// This usually means it's treated as a flat size or 1D length in this context,
	// or maybe it's just returning the number of dimensions (rank)?
	// If it returns a single int, let's assume it is the rank, or if it is the total size?
	// But `dims :=` suggests we expect a slice. The error `invalid argument: dims (variable of type int) for built-in len`
	// confirms Dims() returns `int`.

	// If Dims() returns int, it might be the Size() of the vector?
	// Or maybe it's the number of rows?

	// Let's use Rows() and Columns() again. If they were undefined before, maybe we are looking at `mat.Tensor` interface
	// which is what Encode returns (wrapped in Vector).
	// If `Vector` is a `mat.Tensor`, let's check what methods it has.
	// It has Shape() []int usually.

	shape := result.Vector.Shape()

	rows, cols := 0, 0
	if len(shape) == 1 {
		rows = 1
		cols = shape[0]
	} else if len(shape) >= 2 {
		rows = shape[0]
		cols = shape[1]
	}

	// Create float32 slice
	vec32 := make([]float32, 0, rows*cols)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			v := result.Vector.At(r, c)
			vec32 = append(vec32, float32(v.Item().F64()))
		}
	}

	return vec32, nil
}

func (s *localService) Close() error {
	// Cybertron models don't always have an explicit Close, but if resources need freeing
	// we would do it here. For now, it's a no-op as the library manages memory.
	return nil
}
