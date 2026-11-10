package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Arm is one compiler under test, copied into its own empty directory so the
// stdlib resolver's <binary>/../std tier finds nothing and the binary's embedded
// stdlib is the one measured (design doc §6.3).
type Arm struct {
	Name    string `json:"name"` // baseline | candidate
	Origin  string `json:"origin"`
	Bin     string `json:"bin"`
	SHA256  string `json:"sha256"`
	Version string `json:"version"`
	Dir     string `json:"-"`
}

const releaseRepo = "sunholo-data/ailang"

var releaseTagRE = regexp.MustCompile(`^v\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?$`)

// systemStdlibDirs are the resolver's "system" tier (internal/stdlibroot). A
// stdlib installed there would be measured instead of each binary's own.
var systemStdlibDirs = []string{"/usr/local/share/ailang/std", "/usr/share/ailang/std"}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// releaseAsset is the release tarball name for this platform (release.yml naming).
func releaseAsset() string {
	arch := map[string]string{"amd64": "x64", "arm64": "arm64"}[runtime.GOARCH]
	return runtime.GOOS + "." + arch + ".ailang.tar.gz"
}

// ResolveBaseline turns BASELINE (a path, a release tag, or "latest") into a
// local binary. Release assets are downloaded once per tag into
// <state>/baselines/<tag>/ and sha256-verified against the release's .sha256 file;
// a mismatch is an instrument failure, never a warning.
func ResolveBaseline(spec, stateDir string, offline bool) (path, origin string, err error) {
	if fi, statErr := os.Stat(spec); statErr == nil && !fi.IsDir() {
		return spec, "path " + spec, nil
	}
	if spec != "latest" && !releaseTagRE.MatchString(spec) {
		return "", "", fmt.Errorf("BASELINE %q is neither an existing file, a release tag (vX.Y.Z) nor \"latest\"", spec)
	}
	tag := spec
	if spec == "latest" {
		if offline {
			return "", "", fmt.Errorf("BASELINE=latest needs the network; name a tag or path with --offline")
		}
		if tag, err = ghOut("release", "view", "-R", releaseRepo, "--json", "tagName", "--jq", ".tagName"); err != nil {
			return "", "", err
		}
	}
	dir := filepath.Join(stateDir, "baselines", tag)
	bin := filepath.Join(dir, "ailang")
	asset := releaseAsset()
	origin = fmt.Sprintf("release %s (%s, sha256 verified)", tag, asset)
	if _, err := os.Stat(bin); err == nil {
		return bin, origin + ", cached", nil
	}
	if offline {
		return "", "", fmt.Errorf("offline and baseline %s is not cached at %s", tag, bin)
	}
	dl := filepath.Join(dir, "download")
	if err := os.MkdirAll(dl, 0o755); err != nil {
		return "", "", err
	}
	if _, err := ghOut("release", "download", tag, "-R", releaseRepo, "-D", dl, "--clobber", "-p", asset, "-p", asset+".sha256"); err != nil {
		return "", "", err
	}
	want, err := os.ReadFile(filepath.Join(dl, asset+".sha256"))
	if err != nil {
		return "", "", err
	}
	got, err := fileSHA256(filepath.Join(dl, asset))
	if err != nil {
		return "", "", err
	}
	fields := strings.Fields(string(want))
	if len(fields) == 0 || fields[0] != got {
		return "", "", fmt.Errorf("baseline %s: sha256 mismatch: release says %q, file is %s", tag, strings.TrimSpace(string(want)), got)
	}
	if err := extractBinary(filepath.Join(dl, asset), bin); err != nil {
		return "", "", err
	}
	return bin, origin, nil
}

func extractBinary(tgz, dest string) error {
	f, err := os.Open(tgz)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return fmt.Errorf("%s holds no ailang binary", tgz)
		}
		if err != nil {
			return err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == "ailang" {
			tmp := dest + ".tmp"
			out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
			return os.Rename(tmp, dest)
		}
	}
}

// PrepareArm copies bin into <runDir>/arms/<name>/bin/ailang and creates the
// arm's private HOME, cache, state and temp dirs, then proves the copy starts.
func PrepareArm(name, bin, origin, runDir string) (*Arm, error) {
	fi, err := os.Stat(bin)
	if err != nil {
		return nil, fmt.Errorf("%s binary: %w", name, err)
	}
	if fi.IsDir() || fi.Mode()&0o111 == 0 {
		return nil, fmt.Errorf("%s binary %s is not an executable file", name, bin)
	}
	dir := filepath.Join(runDir, "arms", name)
	for _, sub := range []string{"bin", "home", "cache", "state", "tmp", "xdg-cache", "xdg-data", "xdg-config"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, err
		}
	}
	dst := filepath.Join(dir, "bin", "ailang")
	in, err := os.Open(bin)
	if err != nil {
		return nil, err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return nil, err
	}
	if err := out.Close(); err != nil {
		return nil, err
	}
	sum, err := fileSHA256(dst)
	if err != nil {
		return nil, err
	}
	a := &Arm{Name: name, Origin: origin, Bin: dst, SHA256: sum, Dir: dir}
	cmd := exec.Command(dst, "--version")
	cmd.Env, cmd.Dir = a.Env(), filepath.Join(dir, "tmp")
	v, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s binary does not start (%s --version): %v: %s", name, bin, err, strings.TrimSpace(string(v)))
	}
	sc := bufio.NewScanner(strings.NewReader(string(v)))
	if sc.Scan() {
		a.Version = strings.TrimSpace(sc.Text())
	}
	return a, nil
}

// Env is the arm's child environment: every AILANG_* and XDG_* variable of the
// operator is dropped, and HOME, caches, state and temp all point into the arm's
// scratch dir. AILANG_PACKAGE_ROOT is added by the runner (read-only by contract).
func (a *Arm) Env() []string {
	var env []string
	for _, kv := range os.Environ() {
		k := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			k = kv[:i]
		}
		if strings.HasPrefix(k, "AILANG_") || strings.HasPrefix(k, "XDG_") || k == "HOME" || k == "TMPDIR" || k == "USERPROFILE" {
			continue
		}
		env = append(env, kv)
	}
	return append(env,
		"HOME="+filepath.Join(a.Dir, "home"),
		"TMPDIR="+filepath.Join(a.Dir, "tmp"),
		"XDG_CACHE_HOME="+filepath.Join(a.Dir, "xdg-cache"),
		"XDG_DATA_HOME="+filepath.Join(a.Dir, "xdg-data"),
		"XDG_CONFIG_HOME="+filepath.Join(a.Dir, "xdg-config"),
		"AILANG_CACHE_DIR="+filepath.Join(a.Dir, "cache"),
		"AILANG_STATE_DIR="+filepath.Join(a.Dir, "state"),
		"AILANG_TRACE=off",
		"NO_COLOR=1",
	)
}
