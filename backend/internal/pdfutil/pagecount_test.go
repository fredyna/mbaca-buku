package pdfutil

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
)

// buildPDF produces a minimal but structurally valid PDF with numPages empty
// pages, so tests can assert page counting against a known value.
func buildPDF(numPages int) []byte {
	var buf bytes.Buffer
	offsets := []int{}

	writeObj := func(body string) {
		offsets = append(offsets, buf.Len())
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", len(offsets), body)
	}

	buf.WriteString("%PDF-1.4\n")

	kids := make([]string, numPages)
	for i := 0; i < numPages; i++ {
		kids[i] = fmt.Sprintf("%d 0 R", i+3)
	}

	writeObj("<< /Type /Catalog /Pages 2 0 R >>")
	writeObj(fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), numPages))
	for i := 0; i < numPages; i++ {
		writeObj("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << >> >>")
	}

	xrefOffset := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(offsets)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(offsets)+1, xrefOffset)

	return buf.Bytes()
}

func TestPageCountReturnsActualNumberOfPages(t *testing.T) {
	for _, want := range []int{1, 3, 12} {
		t.Run(fmt.Sprintf("%d_pages", want), func(t *testing.T) {
			got, err := PageCount(bytes.NewReader(buildPDF(want)))
			if err != nil {
				t.Fatalf("PageCount returned error: %v", err)
			}
			if got != want {
				t.Errorf("PageCount = %d, want %d", got, want)
			}
		})
	}
}

func TestPageCountRejectsNonPDF(t *testing.T) {
	if _, err := PageCount(bytes.NewReader([]byte("this is not a pdf"))); err == nil {
		t.Error("PageCount accepted non-PDF data, want error")
	}
}

func TestPageCountRewindsReaderForReuse(t *testing.T) {
	data := buildPDF(3)
	r := bytes.NewReader(data)

	if _, err := PageCount(r); err != nil {
		t.Fatalf("PageCount returned error: %v", err)
	}

	rest, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading after PageCount: %v", err)
	}
	if !bytes.Equal(rest, data) {
		t.Errorf("reader not rewound: read %d bytes, want the full %d bytes", len(rest), len(data))
	}
}
