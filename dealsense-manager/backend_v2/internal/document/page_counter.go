package document

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/sirupsen/logrus"
	"github.com/unidoc/unioffice/presentation"
)

// GetActualPageCount returns the actual page/slide count for a document
// This is more accurate than heuristic-based estimation
func GetActualPageCount(fileData io.Reader, mimeType string) (int, error) {
	switch mimeType {
	case "application/pdf":
		return getPDFPageCount(fileData)
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return getPPTXSlideCount(fileData)
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		// DOCX doesn't have a simple page count - use heuristic fallback
		return -1, fmt.Errorf("DOCX page counting not supported, use file size heuristic")
	default:
		return -1, fmt.Errorf("unsupported file type for page counting: %s", mimeType)
	}
}

// getPDFPageCount gets the actual page count from a PDF file
func getPDFPageCount(fileData io.Reader) (int, error) {
	// Read all data into memory (required by pdfcpu)
	data, err := io.ReadAll(fileData)
	if err != nil {
		return 0, fmt.Errorf("failed to read PDF data: %w", err)
	}

	// Create a reader from bytes
	reader := bytes.NewReader(data)

	// Get page count using pdfcpu API
	// We use PageCount which requires a ReadSeeker
	pageCount, err := api.PageCount(reader, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to count PDF pages: %w", err)
	}

	logrus.Infof("PDF has %d pages (actual count)", pageCount)
	return pageCount, nil
}

// getPPTXSlideCount gets the actual slide count from a PPTX file
func getPPTXSlideCount(fileData io.Reader) (int, error) {
	// Read all data into memory
	data, err := io.ReadAll(fileData)
	if err != nil {
		return 0, fmt.Errorf("failed to read PPTX data: %w", err)
	}

	// Create a reader from bytes
	reader := bytes.NewReader(data)

	// Open PPTX presentation
	pres, err := presentation.Read(reader, int64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("failed to read PPTX presentation: %w", err)
	}

	// Get slide count
	slideCount := len(pres.Slides())
	logrus.Infof("PPTX has %d slides (actual count)", slideCount)
	
	return slideCount, nil
}

// EstimatePageCountFallback estimates page count from file size as fallback
// This is used when actual counting fails or is not supported
func EstimatePageCountFallback(fileSize int64, mimeType string) int {
	// Average page sizes (in bytes):
	// PDF with images: ~300KB per page
	// PDF text only: ~50KB per page
	// PPTX: ~200KB per page
	// DOCX: ~100KB per page

	avgSizePerPage := int64(300 * 1024) // Default: 300KB per page for PDF with images

	switch mimeType {
	case "application/pdf":
		avgSizePerPage = 300 * 1024
	case "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		avgSizePerPage = 200 * 1024
	case "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
		avgSizePerPage = 100 * 1024
	}

	estimatedPages := int(fileSize / avgSizePerPage)
	if estimatedPages == 0 {
		estimatedPages = 1
	}

	logrus.Infof("Estimated page count: %d (fallback heuristic)", estimatedPages)
	return estimatedPages
}

// GetPageCountWithFallback tries to get actual page count, falls back to estimation
func GetPageCountWithFallback(fileData io.Reader, fileSize int64, mimeType string) (int, bool) {
	// Try to get actual page count
	actualCount, err := GetActualPageCount(fileData, mimeType)
	if err == nil && actualCount > 0 {
		logrus.Infof("Using actual page count: %d", actualCount)
		return actualCount, true // true = actual count
	}

	// Log why we're falling back
	if err != nil {
		logrus.Debugf("Could not get actual page count: %v. Using fallback estimation.", err)
	}

	// Fall back to estimation
	estimatedCount := EstimatePageCountFallback(fileSize, mimeType)
	logrus.Infof("Using estimated page count: %d", estimatedCount)
	return estimatedCount, false // false = estimated
}

