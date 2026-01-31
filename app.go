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

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/nfnt/resize"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx  context.Context
	db   *sql.DB
	root string
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
		}
		m.Installed = inst == 1
		mods = append(mods, m)
	}
	return mods
}

func (a *App) AddMod(name, desc, cmd, srcPath, previewB64, sourceURL, loader string) string {
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
	if strings.Contains(previewB64, ",") {
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
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Base(filepath.Dir(exe))
}