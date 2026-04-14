//go:build windows && amd64
package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png" 
	_ "image/gif" 
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nfnt/resize"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"github.com/mholt/archiver/v3"

	"xxmimm/internal/xxmi"
	"xxmimm/internal/gamebanana"
)

type QuickModFile struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int64  `json:"size"`
	MD5         string `json:"md5"`
	DirectURL   string `json:"direct_url"`
}

type QuickModInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	ImageURL    string         `json:"image_url"`
	Files       []QuickModFile `json:"files"`
}


type App struct {
	ctx  context.Context
	db   *sql.DB
	root string
	spoofName string
}

type Mod struct {
	UUID        string `json:"uuid"`
	Name        string `json:"name"`
	Description string `json:"description"`
	SourceURL   string `json:"source_url"`
	Preview     string `json:"preview"`
	Loader      string `json:"loader"`
	InstallCmd  string `json:"install_cmd"`
	Installed   bool   `json:"installed"`
}

func NewApp() *App {
	return &App{root: "XXMIMM"}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	os.MkdirAll(a.root, 0755)
	var err error
	a.db, err = sql.Open("sqlite3", filepath.Join(a.root, "mods.db"))
	if err != nil { panic(err) }

	a.db.Exec(`CREATE TABLE IF NOT EXISTS mods (
		uuid TEXT PRIMARY KEY, name TEXT, description TEXT, 
		source_url TEXT, preview BLOB, loader TEXT, install_cmd TEXT, installed INTEGER DEFAULT 0
	);`)
}

func (a *App) SelectFolder() string {
	res, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "Select Mod Folder"})
	if err != nil { return "" }
	return res
}

func (a *App) sanitizePath(name string) string {
	badChars := []string{" ", "|", ".", "/", "\\", ":", "*", "?", "\"", "<", ">"}
	sanitized := name
	for _, char := range badChars {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}
	return strings.Trim(sanitized, "_")
}

func (a *App) GetMods(loader string) []Mod {
	rows, _ := a.db.Query("SELECT uuid, name, description, source_url, preview, loader, install_cmd, installed FROM mods WHERE loader = ?", loader)
	defer rows.Close()
	var mods []Mod
	for rows.Next() {
		var m Mod
		var preview []byte
		var inst int
		rows.Scan(&m.UUID, &m.Name, &m.Description, &m.SourceURL, &preview, &m.Loader, &m.InstallCmd, &inst)
		if len(preview) > 0 {
			m.Preview = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(preview)
		} else {
			m.Preview = "assets/images/no_preview.jpg"
		}
		m.Installed = inst == 1
		mods = append(mods, m)
	}
	return mods
}

func (a *App) AddMod(name, desc, cmd, srcPath, previewB64, sourceURL, loader string) string {
    fmt.Printf("[AddMod] previewB64/path received: %s\n", previewB64) // DEBUG
    
    if name == "" || srcPath == ""  { return "Missing Required Info" }
    
    id := uuid.New().String()
    dest := filepath.Join(a.root, id)
    os.MkdirAll(dest, 0755)

    copyCmd := exec.Command("robocopy", srcPath, dest, "/E", "/MT", "/R:0", "/W:0")
    copyCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
    
    err := copyCmd.Run()
    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            if exitError.ExitCode() >= 8 {
                return "Robocopy failed with exit code: " + strconv.Itoa(exitError.ExitCode())
            }
        } else {
            return "Failed to run robocopy: " + err.Error()
        }
    }
    
    var imgBytes []byte

	// Check if it's a base64 data URL (contains comma)
	if strings.Contains(previewB64, ",") {
		fmt.Printf("[AddMod] Processing as base64 data URL\n")
		raw := previewB64[strings.Index(previewB64, ",")+1:]
		unprocBytes, _ := base64.StdEncoding.DecodeString(raw)
		
		img, _, err := image.Decode(bytes.NewReader(unprocBytes))
		if err == nil {
			bounds := img.Bounds()
			w, h := bounds.Dx(), bounds.Dy()
			
			var newW, newH int
			if w > h {
				newW = 512
				newH = (h * 512) / w
			} else {
				newH = 512
				newW = (w * 512) / h
			}

			resizedImg := resize.Resize(uint(newW), uint(newH), img, resize.Lanczos3)
			
			buf := new(bytes.Buffer)
			jpeg.Encode(buf, resizedImg, &jpeg.Options{Quality: 85})
			imgBytes = buf.Bytes()
		}
	} else if previewB64 != "" && !strings.HasPrefix(previewB64, "http") {
		// It's a file path (not empty, not base64, not http URL)
		fmt.Printf("[AddMod] Processing as file path: %s\n", previewB64)
		unprocBytes, err := os.ReadFile(previewB64)
		if err != nil {
			fmt.Printf("[AddMod] Failed to read file: %v\n", err)
		} else {
			img, _, err := image.Decode(bytes.NewReader(unprocBytes))
			if err != nil {
				fmt.Printf("[AddMod] Failed to decode image: %v\n", err)
			} else {
				bounds := img.Bounds()
				w, h := bounds.Dx(), bounds.Dy()
				
				var newW, newH int
				if w > h {
					newW = 512
					newH = (h * 512) / w
				} else {
					newH = 512
					newW = (w * 512) / h
				}

				resizedImg := resize.Resize(uint(newW), uint(newH), img, resize.Lanczos3)
				
				buf := new(bytes.Buffer)
				jpeg.Encode(buf, resizedImg, &jpeg.Options{Quality: 85})
				imgBytes = buf.Bytes()
				fmt.Printf("[AddMod] Successfully processed image from path\n")
			}
		}
	}

    _, err = a.db.Exec("INSERT INTO mods VALUES (?,?,?,?,?,?,?,0)", id, name, desc, sourceURL, imgBytes, loader, cmd)
    if err != nil { return err.Error() }
    return "Success"
}

func (a *App) UpdateMod(uuid, name, desc, cmd, previewB64, sourceURL string) string {
	if uuid == "" || name == "" { return "Missing Required Info" }

	query := "UPDATE mods SET name=?, description=?, install_cmd=?, source_url=?"
	args := []interface{}{name, desc, cmd, sourceURL}

	if strings.Contains(previewB64, ",") {
		raw := previewB64[strings.Index(previewB64, ",")+1:]
		unprocBytes, err := base64.StdEncoding.DecodeString(raw)
		if err == nil {
			img, _, err := image.Decode(bytes.NewReader(unprocBytes))
			if err == nil {
				bounds := img.Bounds()
				w, h := bounds.Dx(), bounds.Dy()
				var newW, newH int
				if w > h {
					newW = 512
					newH = (h * 512) / w
				} else {
					newH = 512
					newW = (w * 512) / h
				}

				resizedImg := resize.Resize(uint(newW), uint(newH), img, resize.Lanczos3)
				buf := new(bytes.Buffer)
				jpeg.Encode(buf, resizedImg, &jpeg.Options{Quality: 85})

				query += ", preview=?"
				args = append(args, buf.Bytes())
			}
		}
	}

	query += " WHERE uuid=?"
	args = append(args, uuid)

	_, err := a.db.Exec(query, args...)
	if err != nil { return err.Error() }
	return "Success"
}

func (a *App) SaveChanges(states map[string]bool) {
	for id, active := range states {
		var cmdStr string
		var modName string
		a.db.QueryRow("SELECT name, install_cmd FROM mods WHERE uuid = ?", id).Scan(&modName, &cmdStr)
		
		modDir, _ := filepath.Abs(filepath.Join(a.root, id))
		safeModName := a.sanitizePath(modName)
		lines := strings.Split(cmdStr, ";")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" { continue }

			if strings.HasPrefix(strings.ToLower(line), "link folder to ") {
				targetDir := strings.TrimSpace(line[15:])
				fullTarget, _ := filepath.Abs(filepath.Join(targetDir, safeModName))
				
				if active {
					a.mklink(modDir, fullTarget)
				} else {
					os.Remove(fullTarget)
				}
				continue
			}

			if strings.HasPrefix(strings.ToLower(line), "link files to ") {
				targetDir := strings.TrimSpace(line[14:])
				fullTargetDir, _ := filepath.Abs(targetDir)
				
				entries, _ := os.ReadDir(modDir)
				for _, e := range entries {
					srcFile := filepath.Join(modDir, e.Name())
					dstFile := filepath.Join(fullTargetDir, e.Name())
					
					if active {
						a.mklink(srcFile, dstFile)
					} else {
						os.Remove(dstFile)
					}
				}
				continue
			}
		}

		status := 0
		if active { status = 1 }
		a.db.Exec("UPDATE mods SET installed = ? WHERE uuid = ?", status, id)
	}
}

func (a *App) OpenBrowser(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}

func (a *App) installLink(src, dst string, wildcard bool) {
	if wildcard {
		entries, _ := os.ReadDir(src)
		for _, e := range entries {
			a.mklink(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name()))
		}
	} else {
		a.mklink(src, dst)
	}
}

func (a *App) removeLink(src, dst string, wildcard bool) {
	if wildcard {
		entries, _ := os.ReadDir(src)
		for _, e := range entries {
			os.Remove(filepath.Join(dst, e.Name()))
		}
	} else {
		os.Remove(dst)
	}
}

func (a *App) mklink(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	_, err := os.Stat(src)
	if err != nil {
		return err
	}

	if _, err := os.Lstat(dst); err == nil {
		return nil
	}

	err = os.Symlink(src, dst)
	if err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	return nil
}

func (a *App) DeleteMod(id string) string {
	_, err := a.db.Exec("DELETE FROM mods WHERE uuid = ?", id)
	if err != nil { return err.Error() }
	err = os.RemoveAll(filepath.Join(a.root, id))
	if err != nil { return err.Error() }
	return "Success"
}

func (a *App) GetParentFolderName() string {
	if a.spoofName != "" {
		return a.spoofName
	}

	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Base(filepath.Dir(exe))
}

func (a *App) StartGameWithMods(loader string) string {
	launcherPath, err := xxmi.GetLauncherFilepath()
	if err != nil {
		return err.Error()
	}
	cmd := exec.Command(launcherPath, "--nogui", "--xxmi", loader)
	cmd.Dir = filepath.Dir(launcherPath)
	err = cmd.Start()
	if err != nil {
		return err.Error()
	}
	return "Success"
}

func (a *App) StartGameWithoutMods(loader string) string {
	gameDir, gameExe, err := xxmi.GetGameFilepath(loader)
	if err != nil {
		return err.Error()
	}
	fullPath := filepath.Join(gameDir, gameExe)
	cmd := exec.Command(fullPath)
	cmd.Dir = gameDir
	err = cmd.Start()
	if err != nil {
		return err.Error()
	}
	return "Success"
}

func (a *App) OpenModFolder(uuid string) string {
	fullPath, err := filepath.Abs(filepath.Join(a.root, uuid))
	if err != nil {
		return err.Error()
	}
	// i dunno if it should be selected or nah
	// cmd := exec.Command("explorer.exe", "/select,", fullPath)
	cmd := exec.Command("explorer.exe", fullPath)
	if err := cmd.Start(); err != nil {
		return err.Error()
	}
	return "Success"
}

func (a *App) FetchQuickModInfo(url string) string {
	data, errStr := gamebanana.FetchModInfo(url)
	if errStr != "" {
		return errStr
	}
	
	var modData gamebanana.ModData
	
	// Handle both *ModData and ModData returns
	if ptr, ok := data.(*gamebanana.ModData); ok {
		modData = *ptr
	} else if val, ok := data.(gamebanana.ModData); ok {
		modData = val
	} else {
		return fmt.Sprintf("[quickimport] invalid data type: %T", data)
	}
	
	// Return debug info as error if no files found
	if len(modData.Files) == 0 {
		return fmt.Sprintf("[debug] Type: %T, Name: %s, Files count: %d, Raw files: %+v", data, modData.Name, len(modData.Files), modData.Files)
	}
	
	files := make([]QuickModFile, 0, len(modData.Files))
	for _, f := range modData.Files {
		files = append(files, QuickModFile{
			ID:          f.ID,
			Name:        f.Name,
			Description: f.Description,
			Size:        f.Size,
			MD5:         f.MD5,
			DirectURL:   f.DirectURL,
		})
	}
	
	info := QuickModInfo{
		Name:        modData.Name,
		Description: modData.Description,
		ImageURL:    modData.ImageURL,
		Files:       files,
	}
	
	jsonBytes, err := json.Marshal(info)
	if err != nil {
		return fmt.Sprintf("[quickimport] json marshal error: %v", err)
	}
	
	return string(jsonBytes)
}

// Step 2: Download, verify and extract selected file
func (a *App) DownloadAndExtract(fileID int, directURL string, size int64, md5hash, imageURL, modName, modDesc, sourceURL string) string {
	result := map[string]string{
		"name":        modName,
		"description": modDesc,
		"source_url":  sourceURL,
	}
	
	// Generate temp folder names (8 random hex digits)
	randBytes := make([]byte, 4)
	rand.Read(randBytes)
	hexStr := hex.EncodeToString(randBytes)
	
	downloadDir := filepath.Join(os.TempDir(), fmt.Sprintf("xxmimmD-%s", hexStr))
	extractDir := filepath.Join(os.TempDir(), fmt.Sprintf("xxmimmE-%s", hexStr))
	
	os.MkdirAll(downloadDir, 0755)
	os.MkdirAll(extractDir, 0755)
	
	// Extract filename from URL or use original name
	fileName := filepath.Base(directURL)
	if fileName == "" || fileName == "." {
		fileName = "modfile.zip"
	}
	// Clean filename
	fileName = strings.Split(fileName, "?")[0]
	
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(directURL)
	if err != nil {
		os.RemoveAll(downloadDir)
		return fmt.Sprintf("[download] failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		os.RemoveAll(downloadDir)
		return fmt.Sprintf("[download] server returned status: %d", resp.StatusCode)
	}
	
	filePath := filepath.Join(downloadDir, fileName)
	out, err := os.Create(filePath)
	if err != nil {
		os.RemoveAll(downloadDir)
		return fmt.Sprintf("[download] create file failed: %v", err)
	}
	
	written, err := io.Copy(out, resp.Body)
	out.Close()
	if err != nil {
		os.RemoveAll(downloadDir)
		return fmt.Sprintf("[download] write failed: %v", err)
	}
	
	// Verify size first
	if written != size {
		os.RemoveAll(downloadDir)
		return fmt.Sprintf("[verify] size mismatch: got %d bytes, expected %d", written, size)
	}
	
	// Verify MD5 hash
	if md5hash != "" {
		file, err := os.Open(filePath)
		if err != nil {
			os.RemoveAll(downloadDir)
			return fmt.Sprintf("[verify] open for hash failed: %v", err)
		}
		
		hasher := md5.New()
		if _, err := io.Copy(hasher, file); err != nil {
			file.Close()
			os.RemoveAll(downloadDir)
			return fmt.Sprintf("[verify] hash computation failed: %v", err)
		}
		file.Close()
		
		computedHash := hex.EncodeToString(hasher.Sum(nil))
		if computedHash != md5hash {
			os.RemoveAll(downloadDir)
			return fmt.Sprintf("[verify] hash mismatch: got %s, expected %s", computedHash, md5hash)
		}
	}
	
	// Extract archive (handles zip, rar, 7z)
	if err := archiver.Unarchive(filePath, extractDir); err != nil {
		os.RemoveAll(downloadDir)
		os.RemoveAll(extractDir)
		return fmt.Sprintf("[extract] failed (not a valid archive?): %v", err)
	}
	
	result["extract_path"] = extractDir
	result["temp_download_dir"] = downloadDir
	
	// Download preview image
		// Download preview image
	if imageURL != "" {
		imgResp, err := client.Get(imageURL)
		if err == nil && imgResp.StatusCode == http.StatusOK {
			imgPath := filepath.Join(downloadDir, "XXMIMM-Preview.jpg")
			imgOut, err := os.Create(imgPath)
			if err == nil {
				_, copyErr := io.Copy(imgOut, imgResp.Body)
				imgOut.Close()
				if copyErr == nil {
					result["preview_path"] = imgPath
					fmt.Printf("%s", imgPath)
					if _, err := os.Stat(imgPath); err != nil {
						fmt.Printf("[DownloadAndExtract] WARNING: Image file does not exist after save: %v\n", err)
					}
				}
			}
			imgResp.Body.Close()
		}
	}
	
	jsonBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("[quickimport] result marshal error: %v", err)
	}
	
	return string(jsonBytes)
}

// Cleanup temp directories (call this after successful import or on cancel)
func (a *App) CleanupTempDirs(dirs []string) {
	for _, dir := range dirs {
		os.RemoveAll(dir)
	}
}