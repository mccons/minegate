package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 60 * time.Second}

type versionManifest struct {
	Latest struct {
		Release  string `json:"release"`
		Snapshot string `json:"snapshot"`
	} `json:"latest"`
	Versions []struct {
		ID   string `json:"id"`
		URL  string `json:"url"`
		Type string `json:"type"`
	} `json:"versions"`
}

type versionMeta struct {
	Downloads struct {
		Server struct {
			URL string `json:"url"`
		} `json:"server"`
		ServerMappings *struct {
			URL string `json:"url"`
		} `json:"server_mappings"`
	} `json:"downloads"`
}

const manifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

func fetchManifest() (*versionManifest, error) {
	resp, err := httpClient.Get(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("manifest HTTP %d", resp.StatusCode)
	}
	var m versionManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	return &m, nil
}

func getVersionMeta(manifest *versionManifest, version string) (*versionMeta, error) {
	for _, v := range manifest.Versions {
		if v.ID == version {
			resp, err := httpClient.Get(v.URL)
			if err != nil {
				return nil, fmt.Errorf("fetch version %s: %w", version, err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != 200 {
				return nil, fmt.Errorf("version HTTP %d", resp.StatusCode)
			}
			var meta versionMeta
			if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
				return nil, fmt.Errorf("decode version meta: %w", err)
			}
			return &meta, nil
		}
	}
	return nil, fmt.Errorf("version %q not found in manifest", version)
}

func downloadFile(url, cachePath string) error {
	if _, err := os.Stat(cachePath); err == nil {
		return nil
	}
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", cachePath, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(cachePath)
	if err != nil {
		return fmt.Errorf("create %s: %w", cachePath, err)
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func cachePath(name string) string {
	dir := filepath.Join(os.TempDir(), "mcproto-cache")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, name)
}

// jarMapping holds clean→obfuscated name mapping from Mojang's server.txt.
type jarMapping map[string]string // clean_name -> obfuscated

// protocolSubpackages lists the known Minecraft protocol sub-packages.
var protocolSubpackages = []string{
	"handshake",
	"status",
	"login",
	"configuration",
	"common",
	"game",
	"ping",
}

func loadMappings(url string) (jarMapping, error) {
	cp := cachePath("server.txt")
	if err := downloadFile(url, cp); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(cp)
	if err != nil {
		return nil, err
	}

	m := make(jarMapping)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		// Class-level mapping: clean.name -> obf:
		if line[0] != ' ' && line[0] != '\t' {
			if idx := strings.Index(line, " -> "); idx > 0 {
				clean := line[:idx]
				rest := line[idx+4:]
				if strings.HasSuffix(rest, ":") {
					obf := strings.TrimSuffix(rest, ":")
					m[clean] = obf
				}
			}
		}
	}
	return m, nil
}

func (m jarMapping) protocolPackets() map[string]packetInfo {
	result := make(map[string]packetInfo)
	for clean, obf := range m {
		// Only classes in protocol subpackages that end with "Packet" and aren't inner classes
		if !strings.HasPrefix(clean, "net.minecraft.network.protocol.") {
			continue
		}
		if !strings.HasSuffix(clean, "Packet") {
			continue
		}
		if strings.Contains(clean, "$") {
			continue
		}

		state, bound := classifyClean(clean)
		name := extractCleanName(clean)
		if state == "" || name == "" {
			continue
		}

		result[obf] = packetInfo{
			state: state,
			bound: bound,
			name:  name,
			obf:   obf,
		}
	}
	return result
}

type packetInfo struct {
	state string
	bound string
	name  string
	obf   string
}

func classifyClean(clean string) (state, bound string) {
	// e.g. net.minecraft.network.protocol.login.ClientboundLoginSuccessPacket
	parts := strings.Split(clean, ".")
	state = "play"
	for _, p := range parts {
		for _, sp := range protocolSubpackages {
			if p == sp {
				state = sp
			}
		}
	}
	if state == "common" {
		state = "play"
	}
	if state == "ping" {
		state = "status"
	}

	simple := parts[len(parts)-1]
	if strings.Contains(simple, "Clientbound") || strings.Contains(simple, "Client") {
		bound = "clientbound"
	} else if strings.Contains(simple, "Serverbound") || strings.Contains(simple, "Server") {
		bound = "serverbound"
	} else {
		bound = "serverbound"
	}
	return state, bound
}

func extractCleanName(clean string) string {
	parts := strings.Split(clean, ".")
	simple := parts[len(parts)-1]
	cleanName := simple
	for _, prefix := range []string{"Clientbound", "Serverbound", "Client", "Server"} {
		if strings.HasPrefix(cleanName, prefix) {
			cleanName = strings.TrimPrefix(cleanName, prefix)
			break
		}
	}
	cleanName = strings.TrimSuffix(cleanName, "Packet")
	if cleanName == "" {
		cleanName = simple
	}
	return cleanName
}

func downloadJar(url, version string) (string, error) {
	cp := cachePath(fmt.Sprintf("server-%s.jar", version))
	if err := downloadFile(url, cp); err != nil {
		return "", err
	}
	return cp, nil
}

type jarEntry struct {
	name string
	data []byte
}

func readJar(path string) ([]jarEntry, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, fmt.Errorf("open JAR: %w", err)
	}
	defer zr.Close()

	// For bundler JARs, the real classes are inside META-INF/versions/<ver>/*.jar
	var nestedJAR string
	for _, f := range zr.File {
		if strings.HasPrefix(f.Name, "META-INF/versions/") && strings.HasSuffix(f.Name, ".jar") && !f.FileInfo().IsDir() {
			nestedJAR = f.Name
			break
		}
	}

	if nestedJAR != "" {
		rc, err := zr.Open(nestedJAR)
		if err != nil {
			return nil, fmt.Errorf("open nested JAR %s: %w", nestedJAR, err)
		}
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, fmt.Errorf("read nested JAR: %w", err)
		}

		nzr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return nil, fmt.Errorf("parse nested JAR: %w", err)
		}
		return readZip(nzr), nil
	}

	return readZip(&zr.Reader), nil
}

func readZip(zr *zip.Reader) []jarEntry {
	var entries []jarEntry
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		entries = append(entries, jarEntry{name: f.Name, data: data})
	}
	return entries
}


