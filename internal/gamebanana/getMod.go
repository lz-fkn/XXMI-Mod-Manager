package gamebanana

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	baseURL        = "https://gamebanana.com/apiv11/Mod/"
	requestTimeout = 15 * time.Second
)

type FileEntry struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DirectURL   string `json:"direct_url"`
	Size        int64  `json:"size"`
	MD5         string `json:"md5"`
}

type ModData struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	ImageURL    string      `json:"image_url"`
	Files       []FileEntry `json:"files"`
}

func getDirectURL(dlURL string, client *http.Client) string {
	resp, err := client.Head(dlURL)
	if err != nil {
		return dlURL
	}
	defer resp.Body.Close()
	return resp.Request.URL.String()
}

func FetchModInfo(modURL string) (interface{}, string) {
	re := regexp.MustCompile(`https?://(?:www\.)?gamebanana\.com/mods/(\d+)`)
	matches := re.FindStringSubmatch(modURL)

	if len(matches) < 2 {
		return nil, "[gbapi] not a mod url"
	}

	modID := matches[1]
	props := "_sName,_sText,_aPreviewMedia,_aFiles"
	apiURL := fmt.Sprintf("%s%s?_csvProperties=%s", baseURL, modID, props)

	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Sprintf("[gbapi] connection error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Sprintf("[gbapi] api returned status: %d", resp.StatusCode)
	}

	var raw map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, "[gbapi] failed to parse response"
	}

	result := ModData{
		Name:        getString(raw, "_sName"),
		Description: getString(raw, "_sText"),
	}

	if preview, ok := raw["_aPreviewMedia"].(map[string]interface{}); ok {
		if images, ok := preview["_aImages"].([]interface{}); ok && len(images) > 0 {
			if firstImg, ok := images[0].(map[string]interface{}); ok {
				base := getString(firstImg, "_sBaseUrl")
				file := getString(firstImg, "_sFile")
				if base != "" && file != "" {
					result.ImageURL = fmt.Sprintf("%s/%s", strings.ReplaceAll(base, "\\/", "/"), file)
				}
			}
		}
	}

	if files, ok := raw["_aFiles"].([]interface{}); ok {
		for _, f := range files {
			if fm, ok := f.(map[string]interface{}); ok {
				dlURL := strings.ReplaceAll(getString(fm, "_sDownloadUrl"), "\\/", "/")
				entry := FileEntry{
					ID:          int(getFloat(fm, "_idRow")),
					Name:        getString(fm, "_sFile"),
					Description: getString(fm, "_sDescription"),
					DirectURL:   getDirectURL(dlURL, client),
					Size:        int64(getFloat(fm, "_nFilesize")),
					MD5:         getString(fm, "_sMd5Checksum"),
				}
				result.Files = append(result.Files, entry)
			}
		}
	}

	return result, ""
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if val, ok := m[key].(float64); ok {
		return val
	}
	return 0
}