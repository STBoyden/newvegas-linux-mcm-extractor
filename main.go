package main

import (
	"archive/zip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/bodgit/sevenzip"
	"github.com/ncruces/zenity"
)

type archiveFormat int

const (
	zipFormat archiveFormat = iota
	sevenzipFormat
)

var tmpDir, tmpDirError = os.MkdirTemp("", "newvegas-linux-mcm-extractor-*")

type DownloadLinkResponse []struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
	URI       string `json:"URI"`
}

func (d DownloadLinkResponse) String() string {
	b, _ := json.MarshalIndent(d, "", "  ")
	return string(b)
}

func showError(err error) {
	if err := zenity.Error(err.Error()); err != nil {
		panic(err)
	}

	os.Exit(1)
}

func downloadFile(url, modName, modStub, modFileExtension string) (out *os.File, outFileSize int, err error) {
	var totalBytesRead int

	downloadsDir := tmpDir + "/downloads"
	_, err = os.Stat(downloadsDir)

	if err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(downloadsDir, 0o750); err != nil {
			return nil, 0, err
		}
	} else if err != nil {
		return nil, 0, err
	}

	resp, err := http.Get(url)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()

	progressDialog, err := zenity.Progress(
		zenity.Title(fmt.Sprintf("Downloading \"%s\"", modName)),
		zenity.MaxValue(int(resp.ContentLength)),
	)
	if err != nil {
		return nil, 0, err
	}
	defer progressDialog.Close()

	f, err := os.Create(fmt.Sprintf("%s/%s.%s", downloadsDir, modStub, modFileExtension))
	if err != nil {
		return nil, 0, err
	}

	buf := make([]byte, 1024)
	for {
		bytesRead, err := resp.Body.Read(buf)
		if err == io.EOF {
			break
		} else if err != nil {
			progressDialog.Complete()
			showError(err)
		}

		if bytesRead == 0 && totalBytesRead == 0 {
			progressDialog.Text("Starting download...")
		} else if bytesRead > 0 {
			totalBytesRead += bytesRead
			f.Write(buf[:bytesRead])
			progressDialog.Text(fmt.Sprintf("Downloaded %d/%d bytes (%.2f%%)", totalBytesRead, resp.ContentLength, float32(totalBytesRead)/float32(resp.ContentLength)*100.0))
			progressDialog.Value(totalBytesRead)
		}
	}

	zenity.Info(fmt.Sprintf("Finished downloading \"%s\"", modName), zenity.Title(modName), zenity.NoCancel())
	progressDialog.Complete()

	return f, totalBytesRead, nil
}

func downloadMod(apiKey, modName, modStub, modFileExtension string, modID, modFileID int) (file *os.File, modFileSize int, err error) {
	var modFile *os.File
	var fileSize int

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.nexusmods.com/v1/games/newvegas/mods/%d/files/%d/download_link.json", modID, modFileID),
		nil,
	)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("apikey", apiKey)

	r, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer r.Body.Close()

	if r.Body == nil {
		return nil, 0, errors.New("response from Nexus Mods API is empty")
	}

	var response DownloadLinkResponse
	err = json.NewDecoder(r.Body).Decode(&response)
	if err != nil {
		showError(err)
	}

	hasCDN := false
	for _, e := range response {
		if e.ShortName != "Nexus CDN" {
			continue
		}

		modFile, fileSize, err = downloadFile(e.URI, modName, modStub, modFileExtension)
		if err != nil {
			return nil, 0, err
		}

		hasCDN = true
		break
	}

	if !hasCDN {
		return nil, 0, errors.New("nexus CDN not found in response")
	}

	return modFile, fileSize, nil
}

func cleanUpFiles(files ...*os.File) {
	for _, file := range files {
		file.Close()

		os.Remove(file.Name())
	}
}

func extractFileFrom7Z(f *sevenzip.File, outDir string, out *os.File) (filePath string, err error) {
	if err := os.Mkdir(outDir, 0o750); err != nil {
		return "", err
	}

	fd, err := f.Open()
	if err != nil {
		return "", err
	}
	defer fd.Close()

	p := fmt.Sprintf("%s/%s", outDir, f.Name)

	if out == nil {
		out, err = os.Create(p)
		if err != nil {
			return "", err
		}
	}

	_, err = io.Copy(out, fd)
	if err != nil {
		return "", err
	}

	return p, nil
}

func extractAndCopyFomod(modFile *os.File, modFileSize int64, modName, modStub string) error {
	reader, err := sevenzip.NewReader(modFile, modFileSize)
	if err != nil {
		return err
	}

	for _, f := range reader.File {
		if !strings.Contains(f.Name, ".fomod") {
			continue
		}

		filePath, err := extractFileFrom7Z(f, tmpDir+"/"+modStub, nil)
		if err != nil {
			return err
		}

		fomodFile, err := os.Open(filePath)
		if err != nil {
			return err
		}
		fomodInfo, err := fomodFile.Stat()
		if err != nil {
			return err
		}

		homeDir, _ := os.UserHomeDir()

		outputZipPath, err := zenity.SelectFileSave(
			zenity.Title("Output file for The Mod Configuration Menu"),
			zenity.Filename(fmt.Sprintf(homeDir+"/%s - Repacked.7z", modName)),
		)
		if err != nil {
			return err
		}

		outputZip, err := os.Create(outputZipPath)
		if err != nil {
			return err
		}

		zipWriter := zip.NewWriter(outputZip)

		r, err := sevenzip.NewReader(fomodFile, fomodInfo.Size())
		if r == nil {
			showError(errors.New("fomod archive is empty"))
		}
		for _, f := range r.File {
			if strings.Contains(f.Name, "fomod") {
				continue
			}

			w, err := zipWriter.Create(f.Name)
			if err != nil {
				return err
			}

			fr, err := f.Open()
			if err != nil {
				return err
			}

			if _, err := io.Copy(w, fr); err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	if tmpDirError != nil {
		showError(tmpDirError)
	}

	apiKey, err := zenity.Entry(
		"Please enter a Nexus Mods API key. Available at https://next.nexusmods.com/settings/api-keys",
		zenity.DisallowEmpty(),
		zenity.Width(200),
	)
	if err != nil {
		showError(err)
	}

	mcmMod, mcmModFileSize, err := downloadMod(apiKey, "The Mod Configuration Menu", "mcm", "7z", 42507, 105803)
	if err != nil {
		showError(err)
	}

	if err := extractAndCopyFomod(mcmMod, int64(mcmModFileSize), "The Mod Configuration Menu", "mcm"); err != nil {
		showError(err)
	}

	wmmMod, wmmModFileSize, err := downloadMod(apiKey, "The Weapon Mod Menu", "wmm", "7z", 44515, 1000001686)
	if err != nil {
		showError(err)
	}

	if err := extractAndCopyFomod(wmmMod, int64(wmmModFileSize), "The Weapon Mod Menu", "wmm"); err != nil {
		showError(err)
	}

	cleanUpFiles(mcmMod, wmmMod)
}
