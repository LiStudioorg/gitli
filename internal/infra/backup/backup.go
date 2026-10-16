// SPDX-License-Identifier: AGPL-3.0

// Package backup implements streaming tar.gz backup and restore with
// optional AES-256-GCM encryption and HMAC-SHA256 integrity verification.
//
// Encrypted archive layout (all integers big-endian):
//
//	[Magic "GLIBKUP1" (8 bytes)]
//	zero or more frames:
//	    uint32  frame length N (nonce + ciphertext, includes 16-byte GCM tag)
//	    byte[N] nonce (12 bytes) followed by GCM ciphertext
//	[uint32  hmac trailer length (always 32)]
//	[HMAC-SHA256 (32 bytes) over every byte before the trailer]
//
// The plaintext protected by GCM is the gzip-compressed tar stream, split
// into 64 KiB chunks. GCM authenticates each frame; the HMAC additionally
// covers the whole archive (including the magic) so any tampering is caught
// before extraction begins.
//
// Plain (unencrypted) archives are a clean gzip'd tarball with no header.
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Magic is the 8-byte header identifying an encrypted backup archive.
const Magic = "GLIBKUP1"

const (
	chunkSize = 64 * 1024
	nonceSize = 12
)

// Options configures Create.
type Options struct {
	SourceDir  string
	OutputFile string
	Encrypt    bool
	Key        []byte
}

// Create streams SourceDir into a tar.gz archive written to OutputFile.
// When opts.Encrypt is true, opts.Key must be exactly 32 bytes; the gzip
// stream is then wrapped in an AES-256-GCM framed encryptor and an HMAC
// trailer is appended. The archive is written to a temporary file in the
// same directory and atomically renamed into place.
func Create(ctx context.Context, opts Options) error {
	if opts.SourceDir == "" {
		return errors.New("backup: source dir is required")
	}
	if opts.OutputFile == "" {
		return errors.New("backup: output file is required")
	}
	if opts.Encrypt && len(opts.Key) != 32 {
		return fmt.Errorf("backup: key must be 32 bytes, got %d", len(opts.Key))
	}

	info, err := os.Stat(opts.SourceDir)
	if err != nil {
		return fmt.Errorf("backup: stat source: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("backup: source %q is not a directory", opts.SourceDir)
	}

	outDir := filepath.Dir(opts.OutputFile)
	tmp, err := os.CreateTemp(outDir, ".gitli-backup-*.tmp")
	if err != nil {
		return fmt.Errorf("backup: create temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if err := writeArchive(ctx, tmp, opts); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("backup: sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("backup: close temp file: %w", err)
	}
	if err := os.Rename(tmpName, opts.OutputFile); err != nil {
		return fmt.Errorf("backup: rename temp file: %w", err)
	}
	return nil
}

// writeArchive builds the archive into w, layering writers so data flows
// tar -> gzip -> (optional GCM encryptor) -> w.
func writeArchive(ctx context.Context, w io.Writer, opts Options) error {
	var (
		sink   = io.Writer(w)
		framer *encryptWriter
	)
	if opts.Encrypt {
		var err error
		framer, err = newEncryptWriter(w, opts.Key)
		if err != nil {
			return err
		}
		sink = framer
	}

	gz := gzip.NewWriter(sink)
	tw := tar.NewWriter(gz)

	if err := writeTar(ctx, tw, opts.SourceDir); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return fmt.Errorf("backup: close tar: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("backup: close gzip: %w", err)
	}
	if framer != nil {
		if err := framer.Close(); err != nil {
			return fmt.Errorf("backup: close encryptor: %w", err)
		}
	}
	return nil
}

// writeTar walks srcDir and streams each regular file into tw using paths
// relative to srcDir. Symlinks and special files are skipped.
func writeTar(ctx context.Context, tw *tar.Writer, srcDir string) error {
	root := filepath.Clean(srcDir)
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			return tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     rel + "/",
				Mode:     0o755,
			})
		}
		if !d.Type().IsRegular() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if err := tw.WriteHeader(&tar.Header{
			Typeflag: tar.TypeReg,
			Name:     rel,
			Mode:     int64(info.Mode().Perm()),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
		}); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(tw, f)
		closeErr := f.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

// Restore extracts archiveFile into destDir. If the archive begins with the
// Magic header it is treated as encrypted: a 32-byte key is required, the
// HMAC is validated before extraction, then the stream is decrypted,
// gunzipped and untarred. Entry paths escaping destDir are rejected.
func Restore(ctx context.Context, archiveFile, destDir string, key []byte) error {
	if archiveFile == "" {
		return errors.New("backup: archive file is required")
	}
	if destDir == "" {
		return errors.New("backup: dest dir is required")
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("backup: create dest dir: %w", err)
	}

	f, err := os.Open(archiveFile)
	if err != nil {
		return fmt.Errorf("backup: open archive: %w", err)
	}
	defer func() { _ = f.Close() }()

	magic := make([]byte, len(Magic))
	n, err := io.ReadFull(f, magic)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return fmt.Errorf("backup: read magic: %w", err)
	}
	magic = magic[:n]

	var src io.Reader
	if string(magic) == Magic {
		if len(key) != 32 {
			return errors.New("backup: archive is encrypted, a 32-byte key is required")
		}
		// Two-pass over the seekable file: first verify the HMAC over the
		// whole ciphertext, then rewind and stream-decrypt. This keeps memory
		// bounded even for large archives.
		if err := verifyHMAC(f, key); err != nil {
			return err
		}
		if _, err := f.Seek(int64(len(Magic)), io.SeekStart); err != nil {
			return fmt.Errorf("backup: seek archive: %w", err)
		}
		dr, err := newDecryptReader(f, key)
		if err != nil {
			return err
		}
		src = dr
	} else {
		src = newPrefixReader(magic, f)
	}

	gz, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("backup: gzip reader: %w", err)
	}
	defer func() { _ = gz.Close() }()
	return readTar(ctx, gz, destDir)
}

// readTar extracts a tar stream into destDir, guarding against path escape.
func readTar(ctx context.Context, r io.Reader, destDir string) error {
	root := filepath.Clean(destDir)
	tr := tar.NewReader(r)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("backup: read tar: %w", err)
		}

		target, err := safeJoin(root, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("backup: mkdir %q: %w", target, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("backup: mkdir %q: %w", filepath.Dir(target), err)
			}
			if err := extractFile(tr, target, os.FileMode(hdr.Mode).Perm()); err != nil {
				return err
			}
		}
	}
}

// safeJoin joins name onto root, rejecting any path that escapes root.
func safeJoin(root, name string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(name))
	target := filepath.Join(root, clean)
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return "", fmt.Errorf("backup: resolve %q: %w", name, err)
	}
	if rel == ".." || filepath.IsAbs(rel) ||
		(len(rel) >= 3 && rel[0] == '.' && rel[1] == '.' && (rel[2] == '/' || rel[2] == os.PathSeparator)) {
		return "", fmt.Errorf("backup: path %q escapes destination", name)
	}
	return target, nil
}

func extractFile(r io.Reader, target string, mode os.FileMode) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("backup: create %q: %w", target, err)
	}
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		return fmt.Errorf("backup: write %q: %w", target, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("backup: close %q: %w", target, closeErr)
	}
	return nil
}

// prefixReader replays a small buffered prefix before delegating to r,
// avoiding any seek so the whole flow stays streaming.
type prefixReader struct {
	prefix []byte
	off    int
	r      io.Reader
}

func newPrefixReader(prefix []byte, r io.Reader) *prefixReader {
	return &prefixReader{prefix: prefix, r: r}
}

func (p *prefixReader) Read(b []byte) (int, error) {
	if p.off < len(p.prefix) {
		n := copy(b, p.prefix[p.off:])
		p.off += n
		return n, nil
	}
	return p.r.Read(b)
}

// encryptWriter frames plaintext into GCM-sealed chunks and writes them,
// followed by an HMAC-SHA256 trailer over all emitted bytes.
type encryptWriter struct {
	dst io.Writer
	gcm cipher.AEAD
	mac hashMac
	buf []byte
}

// hashMac is the subset of hash.Hash used for the HMAC trailer.
type hashMac interface {
	io.Writer
	Sum([]byte) []byte
}

func newEncryptWriter(w io.Writer, key []byte) (*encryptWriter, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("backup: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("backup: new gcm: %w", err)
	}
	e := &encryptWriter{
		dst: w,
		gcm: gcm,
		mac: hmac.New(sha256.New, key),
		buf: make([]byte, 0, chunkSize),
	}
	if _, err := e.writeRaw([]byte(Magic)); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *encryptWriter) Write(p []byte) (int, error) {
	written := 0
	for len(p) > 0 {
		n := chunkSize - len(e.buf)
		if n > len(p) {
			n = len(p)
		}
		e.buf = append(e.buf, p[:n]...)
		p = p[n:]
		written += n
		if len(e.buf) == chunkSize {
			if err := e.flushChunk(); err != nil {
				return written, err
			}
		}
	}
	return written, nil
}

func (e *encryptWriter) flushChunk() error {
	if len(e.buf) == 0 {
		return nil
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("backup: read nonce: %w", err)
	}
	sealed := e.gcm.Seal(nil, nonce, e.buf, nil)
	frame := make([]byte, 0, nonceSize+len(sealed))
	frame = append(frame, nonce...)
	frame = append(frame, sealed...)

	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(frame)))
	if _, err := e.writeRaw(lenBuf[:]); err != nil {
		return err
	}
	if _, err := e.writeRaw(frame); err != nil {
		return err
	}
	e.buf = e.buf[:0]
	return nil
}

// Close flushes any buffered plaintext and writes the HMAC trailer.
func (e *encryptWriter) Close() error {
	if err := e.flushChunk(); err != nil {
		return err
	}
	sum := e.mac.Sum(nil)
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(sum)))
	if _, err := e.writeRaw(lenBuf[:]); err != nil {
		return err
	}
	_, err := e.writeRaw(sum)
	return err
}

// writeRaw writes to dst and folds the bytes into the running HMAC.
func (e *encryptWriter) writeRaw(b []byte) (int, error) {
	n, err := e.dst.Write(b)
	if n > 0 {
		_, _ = e.mac.Write(b[:n])
	}
	return n, err
}

// decryptReader streams the decrypted plaintext frame by frame. It expects
// src to be positioned at the first frame, immediately after the Magic
// header, and that the HMAC trailer has already been verified.
type decryptReader struct {
	src     io.Reader
	gcm     cipher.AEAD
	pending []byte
	eof     bool
}

func newDecryptReader(src io.Reader, key []byte) (*decryptReader, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("backup: new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("backup: new gcm: %w", err)
	}
	return &decryptReader{src: src, gcm: gcm}, nil
}

// verifyHMAC streams the archive from the start, hashing the Magic header and
// every ciphertext frame, and compares the trailing HMAC without buffering
// the body.
func verifyHMAC(f io.ReadSeeker, key []byte) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("backup: seek archive: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	magic := make([]byte, len(Magic))
	if _, err := io.ReadFull(f, magic); err != nil {
		return fmt.Errorf("backup: read magic: %w", err)
	}
	if string(magic) != Magic {
		return errors.New("backup: invalid encrypted archive magic")
	}
	_, _ = mac.Write(magic)

	maxFrame := chunkSize + nonceSize + 16
	frame := make([]byte, maxFrame)
	for {
		var lenBuf [4]byte
		if _, err := io.ReadFull(f, lenBuf[:]); err != nil {
			return fmt.Errorf("backup: read frame length: %w", err)
		}
		frameLen := int(binary.BigEndian.Uint32(lenBuf[:]))
		if frameLen == sha256.Size {
			want := make([]byte, sha256.Size)
			if _, err := io.ReadFull(f, want); err != nil {
				return fmt.Errorf("backup: read hmac: %w", err)
			}
			if !hmac.Equal(mac.Sum(nil), want) {
				return errors.New("backup: HMAC verification failed")
			}
			return nil
		}
		if frameLen < nonceSize+16 || frameLen > maxFrame {
			return fmt.Errorf("backup: invalid frame length %d", frameLen)
		}
		_, _ = mac.Write(lenBuf[:])
		if _, err := io.ReadFull(f, frame[:frameLen]); err != nil {
			return fmt.Errorf("backup: read frame body: %w", err)
		}
		_, _ = mac.Write(frame[:frameLen])
	}
}

func (d *decryptReader) Read(p []byte) (int, error) {
	if len(d.pending) > 0 {
		n := copy(p, d.pending)
		d.pending = d.pending[n:]
		return n, nil
	}
	if d.eof {
		return 0, io.EOF
	}
	for {
		var lenBuf [4]byte
		if _, err := io.ReadFull(d.src, lenBuf[:]); err != nil {
			return 0, fmt.Errorf("backup: read frame length: %w", err)
		}
		frameLen := int(binary.BigEndian.Uint32(lenBuf[:]))
		if frameLen == sha256.Size {
			// HMAC trailer: no more data frames.
			if _, err := io.CopyN(io.Discard, d.src, int64(sha256.Size)); err != nil {
				return 0, fmt.Errorf("backup: read hmac: %w", err)
			}
			d.eof = true
			return 0, io.EOF
		}
		if frameLen < nonceSize+16 {
			return 0, fmt.Errorf("backup: invalid frame length %d", frameLen)
		}
		frame := make([]byte, frameLen)
		if _, err := io.ReadFull(d.src, frame); err != nil {
			return 0, fmt.Errorf("backup: read frame body: %w", err)
		}
		nonce := frame[:nonceSize]
		plain, err := d.gcm.Open(nil, nonce, frame[nonceSize:], nil)
		if err != nil {
			return 0, fmt.Errorf("backup: decrypt frame: %w", err)
		}
		if len(plain) == 0 {
			continue
		}
		n := copy(p, plain)
		if n < len(plain) {
			d.pending = plain[n:]
		}
		return n, nil
	}
}
