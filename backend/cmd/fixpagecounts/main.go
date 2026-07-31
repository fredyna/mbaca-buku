// Command fixpagecounts recomputes ebooks.total_pages from the stored PDF
// files. Page counts used to come from a form field filled in by hand, so
// older rows can disagree with the real document — which made the dashboard
// and the reader show different reading progress for the same book.
//
// Run once after deploying the automatic page count:
//
//	docker compose exec backend ./fixpagecounts
//
// Pass -dry-run to report what would change without writing.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/fredy/mbaca-buku/internal/config"
	"github.com/fredy/mbaca-buku/internal/pdfutil"
	"github.com/fredy/mbaca-buku/internal/repository"
	"github.com/fredy/mbaca-buku/internal/storage"
	"github.com/fredy/mbaca-buku/pkg/database"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "report changes without writing them")
	flag.Parse()

	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	minioStorage, err := storage.NewMinIOClient(cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ebookRepo := repository.NewEbookRepository(db)
	progressRepo := repository.NewProgressRepository(db)

	ebooks, err := ebookRepo.ListAll(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var fixed, skipped, failed int
	for _, e := range ebooks {
		obj, err := minioStorage.OpenFile(ctx, e.FileURL)
		if err != nil {
			log.Printf("%q: cannot open file: %v", e.Title, err)
			failed++
			continue
		}

		actual, err := pdfutil.PageCount(obj)
		obj.Close()
		if err != nil {
			log.Printf("%q: cannot read page count: %v", e.Title, err)
			failed++
			continue
		}

		if actual == e.TotalPages {
			skipped++
			continue
		}

		if *dryRun {
			log.Printf("%q: would change %d -> %d pages", e.Title, e.TotalPages, actual)
			fixed++
			continue
		}

		if err := ebookRepo.UpdateTotalPages(ctx, e.ID, actual); err != nil {
			log.Printf("%q: update failed: %v", e.Title, err)
			failed++
			continue
		}

		clamped, err := progressRepo.ClampToTotalPages(ctx, e.ID, actual)
		if err != nil {
			log.Printf("%q: page count fixed but progress clamp failed: %v", e.Title, err)
		}

		log.Printf("%q: %d -> %d pages (%d progress rows adjusted)", e.Title, e.TotalPages, actual, clamped)
		fixed++
	}

	log.Printf("Done: %d fixed, %d already correct, %d failed", fixed, skipped, failed)
}
