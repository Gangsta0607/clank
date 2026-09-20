package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const repoReleasesURL = "https://api.github.com/repos/Gangsta0607/clank/releases/latest"

type githubRelease struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
}

func updateCacheDir() (string, error) {
	dir, err := clankDir()
	if err != nil {
		return "", err
	}
	cache := filepath.Join(dir, "tmp")
	if err := os.MkdirAll(cache, 0700); err != nil {
		return "", err
	}
	return cache, nil
}

func cachedBinaryPath(version string) (string, error) {
	dir, err := updateCacheDir()
	if err != nil {
		return "", err
	}
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	return filepath.Join(dir, fmt.Sprintf("clank_%s_%s_%s%s", version, runtime.GOOS, runtime.GOARCH, ext)), nil
}

func cmdUpdate(args []string) int {
	var autoYes bool
	for _, a := range args {
		if a == "-y" || a == "--yes" {
			autoYes = true
		}
	}

	execPath, err := os.Executable()
	if err != nil {
		fail(M.UpdBinFail, err)
		return exitConfig
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		fail(M.UpdLinkFail, err)
		return exitConfig
	}

	sp := startSpinner(M.UpdChecking)
	rel, err := fetchLatestRelease()
	sp.stopSpinner()
	if err != nil {
		fail(M.UpdCheckErr, err)
		return exitAPI
	}

	latestVer := strings.TrimSpace(rel.TagName)
	currentVer := strings.TrimSpace(version)

	// Нормализуем для сравнения
	normLatest := strings.TrimPrefix(latestVer, "v")
	normCurrent := strings.TrimPrefix(currentVer, "v")

	cacheFile, _ := cachedBinaryPath(latestVer)
	hasCache := false
	if cacheFile != "" {
		if fi, err := os.Stat(cacheFile); err == nil && fi.Size() > 0 {
			hasCache = true
		}
	}

	if normLatest == normCurrent && !hasCache {
		fmt.Printf(M.UpdLatest, currentVer)
		return exitOK
	}

	fmt.Printf(M.UpdAvail, latestVer, currentVer)
	if strings.TrimSpace(rel.Body) != "" {
		fmt.Printf(M.UpdNotes, indentBlock(truncateRunes(stripChecksumsSection(rel.Body), 1000), "  "))
	}

	if hasCache {
		info(M.UpdCached, cacheFile)
	}

	if !autoYes && !confirmYN(fmt.Sprintf(M.UpdAsk, execPath, latestVer), true) {
		fmt.Println(M.UpdAbort)
		return exitOK
	}

	var newBinData []byte
	if hasCache {
		newBinData, err = os.ReadFile(cacheFile)
		if err != nil {
			warn(M.UpdCacheErr, err)
			hasCache = false
		}
	}

	if !hasCache {
		assetURL, assetName, isZip, err := findMatchingAsset(rel)
		if err != nil {
			fail("%v", err)
			return exitConfig
		}

		sp = startSpinner(fmt.Sprintf(M.UpdDown, assetName))
		archiveData, err := downloadAsset(assetURL)
		sp.stopSpinner()
		if err != nil {
			fail(M.UpdDownErr, err)
			return exitAPI
		}

		if err := verifyArchive(archiveData, assetName, rel); err != nil {
			if errors.Is(err, errNoChecksums) {
				warn(M.UpdNoSums)
			} else {
				fail("%v", err)
				return exitConfig
			}
		} else {
			detail(M.UpdSumOK, assetName)
		}

		newBinData, err = extractBinary(archiveData, isZip)
		if err != nil {
			fail(M.UpdUnpErr, err)
			return exitConfig
		}

		if cacheFile != "" {
			if err := os.WriteFile(cacheFile, newBinData, 0755); err == nil {
				detail(M.UpdKept, cacheFile)
			}
		}
	}

	// Попытка применить обновление
	if err := applyUpdate(execPath, newBinData); err != nil {
		fail(M.UpdApplyErr, execPath, err)
		if cacheFile != "" {
			fmt.Printf(M.UpdSudoHint)
		}
		return exitConfig
	}

	// Удаляем кэш после успешной установки
	if cacheFile != "" {
		_ = os.Remove(cacheFile)
	}

	fmt.Printf(M.UpdDone, latestVer)
	return exitOK
}

func fetchLatestRelease() (githubRelease, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest(http.MethodGet, repoReleasesURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("User-Agent", "clank/"+version)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return githubRelease{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return githubRelease{}, err
	}
	return rel, nil
}

func findMatchingAsset(rel githubRelease) (url, name string, isZip bool, err error) {
	targetOS := runtime.GOOS
	targetArch := runtime.GOARCH

	expectedExt := ".tar.gz"
	if targetOS == "windows" {
		expectedExt = ".zip"
		isZip = true
	}

	for _, a := range rel.Assets {
		lower := strings.ToLower(a.Name)
		if strings.Contains(lower, targetOS) && strings.Contains(lower, targetArch) && strings.HasSuffix(lower, expectedExt) {
			return a.BrowserDownloadURL, a.Name, isZip, nil
		}
	}

	return "", "", false, fmt.Errorf(M.UpdNoAsset,
		targetOS, targetArch, rel.TagName)
}

func downloadAsset(url string) ([]byte, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "clank/"+version)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func extractBinary(archiveData []byte, isZip bool) ([]byte, error) {
	if isZip {
		zr, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
		if err != nil {
			return nil, err
		}
		for _, f := range zr.File {
			base := strings.ToLower(filepath.Base(f.Name))
			if strings.HasPrefix(base, "clank") && strings.HasSuffix(base, ".exe") {
				rc, err := f.Open()
				if err != nil {
					return nil, err
				}
				defer rc.Close()
				return io.ReadAll(rc)
			}
		}
		return nil, fmt.Errorf(M.UpdNoExeZip)
	}

	gr, err := gzip.NewReader(bytes.NewReader(archiveData))
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		base := strings.ToLower(filepath.Base(hdr.Name))
		if hdr.Typeflag == tar.TypeReg && strings.HasPrefix(base, "clank") && !strings.HasSuffix(base, ".md") && !strings.HasSuffix(base, ".txt") {
			return io.ReadAll(tr)
		}
	}

	return nil, fmt.Errorf(M.UpdNoExeTar)
}

func applyUpdate(targetPath string, newBinary []byte) error {
	dir := filepath.Dir(targetPath)

	// Записываем во временный файл в той же директории, чтобы rename был атомарным в пределах одной ФС
	tmpFile, err := os.CreateTemp(dir, "clank_update_*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(newBinary); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(0755); err != nil {
		tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	// На Windows открытый exe нельзя сразу перезаписать, но можно переименовать
	if runtime.GOOS == "windows" {
		oldPath := targetPath + ".old"
		_ = os.Remove(oldPath)
		if err := os.Rename(targetPath, oldPath); err != nil {
			return err
		}
	}

	return os.Rename(tmpPath, targetPath)
}

// errNoChecksums — в релизе нет checksums.txt, проверять не по чему.
// Не фатально (старые релизы собирались без него): ставим с предупреждением.
var errNoChecksums = errors.New("no checksums")

// stripChecksumsSection вырезает из тела релиза сгенерированный блок
// контрольных сумм (заголовок ## … + fenced-блок) — в `clank update`
// показываем только человеческие заметки. Заголовок без fenced-блока
// за наш не считаем и оставляем как есть.
func stripChecksumsSection(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	i := 0
	for i < len(lines) {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "## ") && (strings.Contains(t, "Контрольные суммы") ||
			strings.Contains(strings.ToLower(t), "checksum") ||
			strings.Contains(t, "SHA")) {
			j := i + 1
			for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
				j++
			}
			if j < len(lines) && strings.TrimSpace(lines[j]) == "```" {
				k := j + 1
				for k < len(lines) && strings.TrimSpace(lines[k]) != "```" {
					k++
				}
				i = k + 1 // за закрывающим fence (или конец, если его нет)
				continue
			}
		}
		out = append(out, lines[i])
		i++
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

// verifyArchive сверяет скачанный архив с checksums.txt из того же релиза.
func verifyArchive(data []byte, assetName string, rel githubRelease) error {
	var sumsURL string
	for _, a := range rel.Assets {
		if a.Name == "checksums.txt" {
			sumsURL = a.BrowserDownloadURL
			break
		}
	}
	if sumsURL == "" {
		return errNoChecksums
	}
	sumsData, err := downloadAsset(sumsURL)
	if err != nil {
		return errNoChecksums
	}
	want := ""
	for _, line := range strings.Split(string(sumsData), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == assetName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return errNoChecksums
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf(M.UpdSumBad, assetName, want, got)
	}
	return nil
}
