package main

import (
	"archive/tar"
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
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	base        = "/usr/local/addons/ccu-modbus/"
	statePath   = "/usr/local/etc/config/addons/ccu-modbus/self-update.json"
	lockPath    = "/usr/local/etc/config/addons/ccu-modbus/self-update.lock"
	stagePath   = "/usr/local/tmp/new_addon.tar.gz"
	apiURL      = "https://api.github.com/repos/WolfHenk/CCU-Modbus/releases/latest"
	assetPrefix = "https://github.com/WolfHenk/CCU-Modbus/releases/download/"
)

var versionRE = regexp.MustCompile(`^v?([0-9]+)\.([0-9]+)\.([0-9]+)$`)
var checksumRE = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

type release struct {
	Tag        string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}
type state struct {
	OK              bool   `json:"ok"`
	State           string `json:"state"`
	Message         string `json:"message"`
	Updated         int64  `json:"updated"`
	Installed       string `json:"installed,omitempty"`
	Latest          string `json:"latest,omitempty"`
	Available       bool   `json:"available,omitempty"`
	RestartRequired bool   `json:"restart_required,omitempty"`
	Error           string `json:"error,omitempty"`
}

var httpClient = &http.Client{Timeout: 80 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
	if req.URL.Scheme != "https" || len(via) > 10 {
		return errors.New("unsichere Weiterleitung")
	}
	return nil
}}

func ver(s string) ([3]int, error) {
	var nums [3]int
	m := versionRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return nums, errors.New("ungueltige Version")
	}
	for i := 0; i < 3; i++ {
		n, e := strconv.Atoi(m[i+1])
		if e != nil {
			return nums, e
		}
		nums[i] = n
	}
	return nums, nil
}
func newer(a, b string) (bool, error) {
	x, e := ver(a)
	if e != nil {
		return false, e
	}
	y, e := ver(b)
	if e != nil {
		return false, e
	}
	for i := 0; i < 3; i++ {
		if x[i] != y[i] {
			return x[i] > y[i], nil
		}
	}
	return false, nil
}
func installed() (string, error) {
	b, e := os.ReadFile(base + "VERSION")
	return strings.TrimSpace(string(b)), e
}
func latest() (string, string, string, error) {
	var rel release
	b, e := download(apiURL, 2<<20)
	if e != nil {
		return "", "", "", e
	}
	if e = json.Unmarshal(b, &rel); e != nil {
		return "", "", "", e
	}
	if _, e = ver(rel.Tag); e != nil {
		return "", "", "", e
	}
	if rel.Draft || rel.Prerelease {
		return "", "", "", errors.New("kein stabiles Release")
	}
	v := strings.TrimPrefix(rel.Tag, "v")
	names := []string{"ccu-modbus-" + v + ".tar.gz", "ccu-modbus-" + v + ".tar.gz.sha256"}
	urls := make(map[string]string)
	for _, a := range rel.Assets {
		urls[a.Name] = a.URL
	}
	for _, n := range names {
		if urls[n] != assetPrefix+"v"+v+"/"+n {
			return "", "", "", errors.New("Release-Datei fehlt oder URL ungueltig: " + n)
		}
	}
	return v, urls[names[0]], urls[names[1]], nil
}
func download(url string, limit int64) ([]byte, error) {
	req, e := http.NewRequest("GET", url, nil)
	if e != nil {
		return nil, e
	}
	req.Header.Set("User-Agent", "CCU-Modbus-Update/0.2.1")
	resp, e := httpClient.Do(req)
	if e != nil {
		return nil, e
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Download HTTP %d", resp.StatusCode)
	}
	b, e := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if e != nil {
		return nil, e
	}
	if int64(len(b)) > limit {
		return nil, errors.New("Download zu gross")
	}
	return b, nil
}
func validateArchive(data []byte, v string) error {
	gz, e := gzip.NewReader(bytes.NewReader(data))
	if e != nil {
		return e
	}
	defer gz.Close()
	t := tar.NewReader(gz)
	found := false
	count := 0
	for {
		hdr, e := t.Next()
		if e == io.EOF {
			break
		}
		if e != nil {
			return e
		}
		count++
		if count > 500 {
			return errors.New("zu viele Archiveintraege")
		}
		n := hdr.Name
		if strings.HasPrefix(n, "/") || n == "" {
			return errors.New("absoluter Archivpfad")
		}
		clean := path.Clean(n)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			return errors.New("unsicherer Archivpfad")
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA && hdr.Typeflag != tar.TypeDir {
			return errors.New("unsicherer Archiveintrag")
		}
		if clean == "VERSION" {
			b, e := io.ReadAll(io.LimitReader(t, 100))
			if e != nil {
				return e
			}
			if strings.TrimSpace(string(b)) != v {
				return errors.New("VERSION im Archiv falsch")
			}
			found = true
		}
	}
	if !found {
		return errors.New("VERSION fehlt im Archiv")
	}
	return nil
}
func save(s state) {
	s.OK = s.State != "error"
	s.Updated = time.Now().Unix()
	b, _ := json.Marshal(s)
	_ = os.MkdirAll(path.Dir(statePath), 0755)
	tmp := statePath + ".tmp"
	if os.WriteFile(tmp, b, 0600) == nil {
		_ = os.Rename(tmp, statePath)
	}
}
func getState() state {
	b, e := os.ReadFile(statePath)
	if e != nil {
		return state{OK: true, State: "idle", Message: "Kein Update gestartet"}
	}
	var s state
	if json.Unmarshal(b, &s) != nil {
		return state{OK: false, State: "error", Message: "Statusdatei unlesbar"}
	}
	return s
}
func jsonOut(v interface{}) { _ = json.NewEncoder(os.Stdout).Encode(v) }
func runWorker() (err error) {
	defer os.Remove(lockPath)
	defer func() {
		if err != nil {
			save(state{State: "error", Message: "Update fehlgeschlagen: " + err.Error()})
		}
	}()
	save(state{State: "checking", Message: "Pruefe GitHub-Release"})
	v, asset, shaURL, e := latest()
	if e != nil {
		return e
	}
	old, e := installed()
	if e != nil {
		return e
	}
	yes, e := newer(v, old)
	if e != nil {
		return e
	}
	if !yes {
		save(state{State: "idle", Message: "Keine neuere Version vorhanden"})
		return nil
	}
	save(state{State: "downloading", Latest: v, Message: "Lade Version " + v})
	b, e := download(asset, 48<<20)
	if e != nil {
		return e
	}
	sum, e := download(shaURL, 1024)
	if e != nil {
		return e
	}
	fields := strings.Fields(string(sum))
	if len(fields) < 1 || !checksumRE.MatchString(fields[0]) {
		return errors.New("Ungueltige SHA-256-Datei")
	}
	hash := sha256.Sum256(b)
	if !strings.EqualFold(fields[0], hex.EncodeToString(hash[:])) {
		return errors.New("SHA-256 stimmt nicht")
	}
	if e = validateArchive(b, v); e != nil {
		return e
	}
	now, e := installed()
	if e != nil || now != old {
		return errors.New("Version waehrend Update geaendert")
	}
	if e = os.WriteFile(stagePath, b, 0600); e != nil {
		return e
	}
	save(state{State: "installing", Latest: v, Message: "Archiv geprueft; installiere Version " + v})
	cmd := exec.Command("/bin/install_addon")
	output, e := cmd.CombinedOutput()
	restart := false
	if e != nil {
		if x, ok := e.(*exec.ExitError); ok && x.ExitCode() == 10 {
			restart = true
		} else {
			return fmt.Errorf("Installer fehlgeschlagen: %v (%s)", e, string(output[:min(len(output), 200)]))
		}
	}
	now, e = installed()
	if e != nil || now != v {
		return errors.New("Versionspruefung nach Installation fehlgeschlagen")
	}
	save(state{State: "done", Installed: v, Message: "Version " + v + " installiert", RestartRequired: restart})
	return nil
}
func main() {
	if len(os.Args) != 2 {
		jsonOut(state{OK: false, Error: "Aktion fehlt"})
		os.Exit(1)
	}
	switch os.Args[1] {
	case "status":
		jsonOut(getState())
	case "check":
		v, _, _, e := latest()
		if e != nil {
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		old, e := installed()
		if e != nil {
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		yes, e := newer(v, old)
		if e != nil {
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		jsonOut(state{OK: true, Installed: old, Latest: v, Available: yes})
	case "start":
		if e := os.Mkdir(lockPath, 0700); e != nil {
			jsonOut(state{OK: false, Error: "Update laeuft bereits"})
			return
		}
		save(state{State: "queued", Message: "Update wird vorbereitet"})
		// A worker under the addon directory would be replaced by the installer.
		workerPath := "/usr/local/tmp/ccu-modbus-update-worker"
		source, e := os.Open(os.Args[0])
		if e != nil {
			_ = os.Remove(lockPath)
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		defer source.Close()
		temp, e := os.OpenFile(workerPath+".tmp", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
		if e != nil {
			_ = os.Remove(lockPath)
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		_, e = io.Copy(temp, source)
		if closeErr := temp.Close(); e == nil {
			e = closeErr
		}
		if e == nil {
			e = os.Rename(workerPath+".tmp", workerPath)
		}
		if e != nil {
			_ = os.Remove(lockPath)
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		cmd := exec.Command(workerPath, "worker")
		cmd.Stdin = nil
		cmd.Stdout = nil
		cmd.Stderr = nil
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		if e := cmd.Start(); e != nil {
			_ = os.Remove(lockPath)
			jsonOut(state{OK: false, Error: e.Error()})
			return
		}
		jsonOut(state{OK: true, State: "queued"})
	case "worker":
		if e := runWorker(); e != nil {
			os.Exit(1)
		}
	default:
		jsonOut(state{OK: false, Error: "Unbekannte Aktion"})
	}
}
