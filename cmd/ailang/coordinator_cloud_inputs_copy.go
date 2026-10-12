package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/sunholo-data/ailang/internal/messaging"
)

type stagedInputFile struct {
	source   string
	dest     string
	index    int
	explicit bool
}

func stageTaskInput(ctx context.Context, temp, clone string, input messaging.TaskInput, index int, remaining *int64) ([]stagedInputFile, taskInputProvenance, error) {
	report := taskInputProvenance{Repo: input.Repo, Ref: input.Ref, Digests: map[string]string{}}
	if input.Path != "" {
		if err := validateInputDataPath(input.Path); err != nil {
			return nil, report, err
		}
	}
	if err := checkInputAncestors(clone, input.Path); err != nil {
		return nil, report, err
	}
	source := filepath.Join(clone, filepath.FromSlash(input.Path))
	info, err := os.Lstat(source)
	if err != nil {
		return nil, report, fmt.Errorf("source %q: %w", input.Path, err)
	}
	if !info.IsDir() && !info.Mode().IsRegular() {
		return nil, report, fmt.Errorf("source must be a regular file or directory")
	}
	if input.SHA256 != "" && !info.Mode().IsRegular() {
		return nil, report, fmt.Errorf("sha256 requires a regular single file")
	}
	dest := strings.TrimSuffix(input.Dest, "/")
	if dest == "" {
		dest = fmt.Sprintf(".incoming/%d", index+1)
	}
	if err := validateInputDataPath(dest); err != nil {
		return nil, report, err
	}
	report.Dest = dest
	stage := filepath.Join(temp, fmt.Sprintf("stage-%d", index))
	var files []stagedInputFile
	err = filepath.WalkDir(source, func(p string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(source, p)
		if err != nil {
			return err
		}
		// The clone's own metadata is never input. Nested metadata is refused.
		if input.Path == "" && rel == ".git" {
			return filepath.SkipDir
		}
		if rel != "." {
			if err := validateInputDataPath(rel); err != nil {
				return err
			}
		}
		entryInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if entryInfo.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("source symlink refused: %q", rel)
		}
		if entry.IsDir() {
			return nil
		}
		if !entryInfo.Mode().IsRegular() {
			return fmt.Errorf("source is not a regular file: %q", rel)
		}
		name := filepath.ToSlash(rel)
		if !info.IsDir() {
			name = filepath.Base(source)
		}
		target := filepath.Join(dest, filepath.FromSlash(name))
		// For a single file, a trailing slash names a destination directory;
		// otherwise an explicit dest names the exact new file.
		if !info.IsDir() && input.Dest != "" && !strings.HasSuffix(input.Dest, "/") {
			target = dest
			report.Dest = dest
		}
		if err := validateInputDestination(target, input.Dest != ""); err != nil {
			return err
		}
		staged := filepath.Join(stage, filepath.FromSlash(name))
		digest, err := stageInputFile(ctx, p, staged, remaining)
		if err != nil {
			return err
		}
		report.Digests[name] = digest
		files = append(files, stagedInputFile{source: staged, dest: target, index: index, explicit: input.Dest != ""})
		return nil
	})
	if err != nil {
		return nil, report, err
	}
	if input.SHA256 != "" {
		name := filepath.Base(source)
		if !strings.EqualFold(report.Digests[name], input.SHA256) {
			return nil, report, fmt.Errorf("sha256 mismatch for %q", input.Path)
		}
		report.Verified = append(report.Verified, name)
	}
	if info.IsDir() {
		verified, err := verifyInputManifest(stage, report.Digests)
		if err != nil {
			return nil, report, err
		}
		report.Verified = append(report.Verified, verified...)
	}
	return files, report, nil
}

// Stage bytes and compute the digest in the same read. LimitReader includes one
// sentinel byte so an input growing during the read still cannot evade the cap.
func stageInputFile(ctx context.Context, source, dest string, remaining *int64) (string, error) {
	if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
		return "", err
	}
	in, err := os.Open(source)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(out, hash), io.LimitReader(&contextInputReader{ctx: ctx, reader: in}, *remaining+1))
	closeErr := out.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n > *remaining {
		return "", fmt.Errorf("aggregate copied bytes exceed %d-byte limit", maxTaskInputBytes)
	}
	*remaining -= n
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

type contextInputReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextInputReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}

// Delivery uses exclusive creation, then rolls back only paths it created if
// any later write fails. Existing workspace directories are left untouched.
func deliverTaskInputFiles(ctx context.Context, workspace string, files []stagedInputFile) (err error) {
	var createdFiles, createdDirs []string
	defer func() {
		if err != nil {
			for i := len(createdFiles) - 1; i >= 0; i-- {
				_ = os.Remove(createdFiles[i])
			}
			for i := len(createdDirs) - 1; i >= 0; i-- {
				_ = os.Remove(createdDirs[i])
			}
		}
	}()
	for _, file := range files {
		if e := ctx.Err(); e != nil {
			return fmt.Errorf("inputs[%d]: %w", file.index, e)
		}
		if e := checkInputDestination(workspace, file.dest, file.explicit); e != nil {
			return fmt.Errorf("inputs[%d]: %w", file.index, e)
		}
		target := filepath.Join(workspace, file.dest)
		var missing []string
		for dir := filepath.Dir(target); dir != workspace; dir = filepath.Dir(dir) {
			info, e := os.Lstat(dir)
			if os.IsNotExist(e) {
				missing = append(missing, dir)
				continue
			}
			if e != nil {
				return fmt.Errorf("inputs[%d]: %w", file.index, e)
			}
			if !info.IsDir() {
				return fmt.Errorf("inputs[%d]: parent is not a directory: %s", file.index, dir)
			}
			break
		}
		for i := len(missing) - 1; i >= 0; i-- {
			if e := os.Mkdir(missing[i], 0o755); e != nil {
				return fmt.Errorf("inputs[%d]: create destination: %w", file.index, e)
			}
			createdDirs = append(createdDirs, missing[i])
		}
		if e := deliverInputFile(file.source, target, &createdFiles); e != nil {
			return fmt.Errorf("inputs[%d]: deliver %q: %w", file.index, file.dest, e)
		}
	}
	return nil
}

func deliverInputFile(source, dest string, created *[]string) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	*created = append(*created, dest)
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
