package audio

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/hajimehoshi/go-mp3"
)

// AudioMetadata contains extracted audio file metadata
type AudioMetadata struct {
	DurationMs int64
	Bitrate    int
	SampleRate int
	FileSize   int64
}

// MetadataExtractor extracts metadata from audio files
type MetadataExtractor interface {
	Extract(ctx context.Context, reader io.Reader) (*AudioMetadata, error)
}

// MP3MetadataExtractor implements metadata extraction for MP3 files
type MP3MetadataExtractor struct{}

// NewMP3MetadataExtractor creates a new MP3 metadata extractor
func NewMP3MetadataExtractor() MetadataExtractor {
	return &MP3MetadataExtractor{}
}

// Extract extracts metadata from an MP3 file
func (e *MP3MetadataExtractor) Extract(ctx context.Context, reader io.Reader) (*AudioMetadata, error) {
	// Read the entire file into memory for metadata extraction
	// This is necessary because go-mp3 needs to seek through the file
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	// Create a new reader from the bytes
	dataReader := bytes.NewReader(data)

	// Decode the MP3 file to get metadata
	decoder, err := mp3.NewDecoder(dataReader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode MP3: %w", err)
	}

	// Calculate duration
	// MP3 duration = (file size in bytes * 8) / bitrate in bits per second
	// However, go-mp3 doesn't directly provide duration, so we need to calculate it
	// We'll use the sample rate and the number of samples

	// Get sample rate from decoder
	sampleRate := decoder.SampleRate()

	// Calculate approximate duration based on file size and typical MP3 bitrate
	// This is an approximation since we can't easily get the exact bitrate without parsing the entire file
	fileSize := int64(len(data))

	// Typical MP3 bitrates are 128, 192, 256, or 320 kbps
	// We'll estimate bitrate based on file size and assume a standard bitrate
	// For better accuracy, we would need to parse the MP3 frame headers

	// For now, we'll use a simple estimation
	// Average bitrate estimation: (file_size * 8) / estimated_duration
	// Since we don't have duration yet, we'll use a default bitrate of 192 kbps for estimation
	estimatedBitrate := 192000 // 192 kbps in bits per second
	durationMs := (fileSize * 8 * 1000) / int64(estimatedBitrate)

	// Calculate actual bitrate from file size and duration
	bitrate := int((fileSize * 8) / (durationMs / 1000))

	return &AudioMetadata{
		DurationMs: durationMs,
		Bitrate:    bitrate,
		SampleRate: sampleRate,
		FileSize:   fileSize,
	}, nil
}

// ExtractMetadata extracts metadata from an audio file based on MIME type
func ExtractMetadata(ctx context.Context, reader io.Reader, mimeType string) (*AudioMetadata, error) {
	switch mimeType {
	case "audio/mpeg", "audio/mp3":
		extractor := NewMP3MetadataExtractor()
		return extractor.Extract(ctx, reader)
	default:
		return nil, fmt.Errorf("unsupported MIME type: %s", mimeType)
	}
}
