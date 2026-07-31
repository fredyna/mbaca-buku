// Package pdfutil derives objective properties from PDF files so the server
// never has to trust client-supplied metadata.
package pdfutil

import (
	"fmt"
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// PageCount reports how many pages the PDF in rs contains. The reader is
// rewound to the start before returning, so callers can upload the same
// stream afterwards.
func PageCount(rs io.ReadSeeker) (int, error) {
	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("rewinding pdf: %w", err)
	}

	conf := model.NewDefaultConfiguration()
	conf.ValidationMode = model.ValidationRelaxed

	n, err := api.PageCount(rs, conf)
	if err != nil {
		return 0, fmt.Errorf("reading pdf page count: %w", err)
	}

	if _, err := rs.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("rewinding pdf: %w", err)
	}

	if n < 1 {
		return 0, fmt.Errorf("pdf reports %d pages", n)
	}

	return n, nil
}
