package store

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"rental/internal/domain"
)

func (s *Store) PutDocument(personID int, data []byte) error {
	// Decode outside the lock, then recheck existence while committing the image.
	if len(data) == 0 || len(data) > domain.MaxDocumentBytes {
		return fmt.Errorf("%w: Bild muss zwischen 1 Byte und 5 MiB groß sein", domain.ErrInvalid)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || int64(cfg.Width)*int64(cfg.Height) > 20000000 {
		return fmt.Errorf("%w: Gültiges JPG/PNG mit maximal 20 Megapixeln erforderlich", domain.ErrInvalid)
	}
	if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("%w: Beschädigte Bilddatei: %v", domain.ErrInvalid, err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.people[personID]; !ok {
		return domain.ErrNotFound
	}
	mime := "image/png"
	if format == "jpeg" {
		mime = "image/jpeg"
	}
	s.documents[personID] = domain.Document{Data: bytes.Clone(data), ContentType: mime}
	return nil
}
func (s *Store) GetDocument(personID int) (domain.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	doc, ok := s.documents[personID]
	if !ok {
		return domain.Document{}, domain.ErrNotFound
	}
	doc.Data = bytes.Clone(doc.Data)
	return doc, nil
}
func (s *Store) DeleteDocument(personID int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.people[personID]; !ok {
		return domain.ErrNotFound
	}
	delete(s.documents, personID)
	return nil
}
