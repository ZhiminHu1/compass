package vector

import (
	"fmt"
	"os"
	"strings"
	"unicode"
)

// ChunkConfig configures how documents are split into chunks
type ChunkConfig struct {
	ChunkSize        int  // Maximum chunk size in characters
	ChunkOverlap     int  // Overlap between chunks
	MinChunkSize     int  // Minimum chunk size to keep
	SplitByParagraph bool // Whether to prioritize paragraph splitting
}

// DefaultChunkConfig returns the default chunk configuration
func DefaultChunkConfig() ChunkConfig {
	return ChunkConfig{
		ChunkSize:        getEnvInt("CHUNK_SIZE", 1000),
		ChunkOverlap:     getEnvInt("CHUNK_OVERLAP", 200),
		MinChunkSize:     getEnvInt("MIN_CHUNK_SIZE", 100),
		SplitByParagraph: true,
	}
}

// getEnvInt reads an integer from environment variable
func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		if _, err := fmt.Sscanf(val, "%d", &result); err == nil {
			return result
		}
	}
	return defaultVal
}

// Chunk represents a text chunk with metadata
type Chunk struct {
	Content    string
	ChunkIndex int
}

// ChunkDocument splits a document into chunks based on the configuration
func ChunkDocument(content string, config ChunkConfig) []Chunk {
	if config.ChunkSize <= 0 {
		config.ChunkSize = 1000
	}
	if config.ChunkOverlap < 0 {
		config.ChunkOverlap = 0
	}
	if config.MinChunkSize <= 0 {
		config.MinChunkSize = 100
	}

	// Normalize content
	content = strings.TrimSpace(content)
	if content == "" {
		return []Chunk{}
	}

	var chunks []Chunk

	if config.SplitByParagraph {
		chunks = splitByParagraph(content, config)
	}

	// If paragraph splitting didn't produce good results, fall back to sentence splitting
	if len(chunks) == 0 {
		chunks = splitBySentence(content, config)
	}

	// Filter out chunks that are too small
	var filteredChunks []Chunk
	for _, chunk := range chunks {
		if len(chunk.Content) >= config.MinChunkSize {
			filteredChunks = append(filteredChunks, chunk)
		}
	}

	// Re-index chunks
	for i := range filteredChunks {
		filteredChunks[i].ChunkIndex = i
	}

	return filteredChunks
}

// splitByParagraph splits content by paragraph boundaries first
func splitByParagraph(content string, config ChunkConfig) []Chunk {
	var chunks []Chunk

	// Split by double newlines (paragraphs)
	paragraphs := strings.Split(content, "\n\n")

	var currentChunk strings.Builder
	currentIndex := 0

	for _, paragraph := range paragraphs {
		paragraph = strings.TrimSpace(paragraph)
		if paragraph == "" {
			continue
		}

		// If adding this paragraph would exceed chunk size
		if currentChunk.Len()+len(paragraph) > config.ChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			content := currentChunk.String()
			if len(content) >= config.MinChunkSize {
				chunks = append(chunks, Chunk{
					Content:    content,
					ChunkIndex: currentIndex,
				})
				currentIndex++
			}

			// Start new chunk with overlap
			currentChunk.Reset()

			// Add overlap from previous chunk
			if config.ChunkOverlap > 0 && len(content) > 0 {
				overlap := getTailOverlap(content, config.ChunkOverlap)
				currentChunk.WriteString(overlap)
				currentChunk.WriteString("\n\n")
			}
		}

		currentChunk.WriteString(paragraph)
		currentChunk.WriteString("\n\n")
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		if len(content) >= config.MinChunkSize {
			chunks = append(chunks, Chunk{
				Content:    content,
				ChunkIndex: currentIndex,
			})
		}
	}

	// Handle large paragraphs that exceed chunk size
	chunks = handleLargeChunks(chunks, config)

	return chunks
}

// splitBySentence splits content by sentence boundaries
func splitBySentence(content string, config ChunkConfig) []Chunk {
	var chunks []Chunk

	sentences := splitIntoSentences(content)

	var currentChunk strings.Builder
	currentIndex := 0

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if sentence == "" {
			continue
		}

		// If adding this sentence would exceed chunk size
		if currentChunk.Len()+len(sentence) > config.ChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			content := currentChunk.String()
			if len(content) >= config.MinChunkSize {
				chunks = append(chunks, Chunk{
					Content:    content,
					ChunkIndex: currentIndex,
				})
				currentIndex++
			}

			// Start new chunk with overlap
			currentChunk.Reset()

			// Add overlap from previous chunk
			if config.ChunkOverlap > 0 && len(content) > 0 {
				overlap := getTailOverlap(content, config.ChunkOverlap)
				currentChunk.WriteString(overlap)
				currentChunk.WriteString(" ")
			}
		}

		currentChunk.WriteString(sentence)
		currentChunk.WriteString(" ")
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		content := strings.TrimSpace(currentChunk.String())
		if len(content) >= config.MinChunkSize {
			chunks = append(chunks, Chunk{
				Content:    content,
				ChunkIndex: currentIndex,
			})
		}
	}

	return chunks
}

// splitIntoSentences splits text into sentences
func splitIntoSentences(text string) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		// Check for sentence endings
		if isSentenceEnd(runeAt(runes, i)) {
			// Check if it's the end of the string or next char is space/quote/paren
			next := runeAt(runes, i+1)
			if next == 0 || unicode.IsSpace(next) || next == '"' || next == '\'' || next == ')' || next == ']' {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	// Add remaining text
	if current.Len() > 0 {
		sentences = append(sentences, strings.TrimSpace(current.String()))
	}

	return sentences
}

// isSentenceEnd checks if a rune is a sentence ending punctuation
func isSentenceEnd(r rune) bool {
	return r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?'
}

// runeAt safely returns a rune at index or 0 if out of bounds
func runeAt(runes []rune, i int) rune {
	if i < 0 || i >= len(runes) {
		return 0
	}
	return runes[i]
}

// getTailOverlap gets the last N characters from text, trying to break at word boundary
func getTailOverlap(text string, size int) string {
	if size <= 0 || len(text) == 0 {
		return ""
	}

	if size >= len(text) {
		return text
	}

	// Get the tail
	tail := text[len(text)-size:]

	// Try to find a word boundary
	if firstSpace := strings.Index(tail, " "); firstSpace > 0 {
		return tail[firstSpace+1:]
	}

	return tail
}

// handleLargeChunks splits chunks that are still too large
func handleLargeChunks(chunks []Chunk, config ChunkConfig) []Chunk {
	var result []Chunk

	for _, chunk := range chunks {
		if len(chunk.Content) <= config.ChunkSize {
			result = append(result, chunk)
			continue
		}

		// Split large chunk
		subChunks := forceSplit(chunk.Content, config.ChunkSize, config.ChunkOverlap)
		for i, sc := range subChunks {
			result = append(result, Chunk{
				Content:    sc,
				ChunkIndex: chunk.ChunkIndex + i,
			})
		}
	}

	return result
}

// forceSplit splits text into fixed-size chunks
func forceSplit(text string, size, overlap int) []string {
	var chunks []string

	runes := []rune(text)
	start := 0

	for start < len(runes) {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}

		chunk := string(runes[start:end])
		chunks = append(chunks, chunk)

		start = end - overlap
		if start < 0 {
			start = 0
		}
	}

	return chunks
}
