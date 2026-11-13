package http

import (
	"errors"
	"io"
	"net/http"

	"github.com/raisin/travel-split/backend/internal/platform/apperr"
	"github.com/raisin/travel-split/backend/internal/domain/slip"
)

// uploadSlips reads every image in a multipart request (any file field) and
// returns what the AI extracted. Images stay in memory and are discarded;
// expenses are only created later, after the user confirms each one.
func (s *Server) uploadSlips(w http.ResponseWriter, r *http.Request) {
	if !s.Slips.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "slip_reader_unavailable", "ยังไม่ได้ตั้งค่า AI อ่านสลิปบนเซิร์ฟเวอร์")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, slip.MaxFiles*(slip.MaxFileSize+1<<20))
	mr, err := r.MultipartReader()
	if err != nil {
		writeErr(w, r, apperr.Invalid("invalid_upload", "ต้องส่งเป็น multipart/form-data"))
		return
	}

	var uploads []slip.Upload
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				writeErr(w, r, apperr.Invalid("upload_too_large", "อัปโหลดรวมกันใหญ่เกินไป"))
				return
			}
			writeErr(w, r, apperr.Invalid("invalid_upload", "อ่านไฟล์ที่อัปโหลดไม่ได้"))
			return
		}
		if part.FileName() == "" {
			continue // ignore non-file fields
		}
		if len(uploads) == slip.MaxFiles {
			writeErr(w, r, apperr.Invalid("too_many_files", "อัปโหลดได้ครั้งละไม่เกิน 10 รูป"))
			return
		}
		// Read one byte past the limit so oversize files are detected and
		// rejected per file instead of failing the whole request.
		data, err := io.ReadAll(io.LimitReader(part, slip.MaxFileSize+1))
		if err != nil {
			writeErr(w, r, apperr.Invalid("invalid_upload", "อ่านไฟล์ที่อัปโหลดไม่ได้"))
			return
		}
		uploads = append(uploads, slip.Upload{Filename: part.FileName(), Data: data})
	}
	if len(uploads) == 0 {
		writeErr(w, r, apperr.Invalid("no_files", "ไม่พบรูปที่อัปโหลด"))
		return
	}

	ctx := r.Context()
	results := s.Slips.Process(ctx, tripFrom(ctx).ID, memberFrom(ctx).ID, uploads)
	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}
