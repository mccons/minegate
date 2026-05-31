package main

import (
	"fmt"
	"sort"
	"strings"
)

// extractPacketIDs downloads the server JAR + Mojang mappings and extracts packet IDs.
func extractPacketIDs(version string) ([]PacketDef, error) {
	manifest, err := fetchManifest()
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}

	meta, err := getVersionMeta(manifest, version)
	if err != nil {
		return nil, fmt.Errorf("get version meta: %w", err)
	}

	if meta.Downloads.ServerMappings == nil || meta.Downloads.ServerMappings.URL == "" {
		return nil, fmt.Errorf("version %q has no server_mappings", version)
	}

	mappings, err := loadMappings(meta.Downloads.ServerMappings.URL)
	if err != nil {
		return nil, fmt.Errorf("load mappings: %w", err)
	}

	packets := mappings.protocolPackets()
	if len(packets) == 0 {
		return nil, fmt.Errorf("no protocol packets found in mappings")
	}

	jarPath, err := downloadJar(meta.Downloads.Server.URL, version)
	if err != nil {
		return nil, fmt.Errorf("download JAR: %w", err)
	}

	entries, err := readJar(jarPath)
	if err != nil {
		return nil, fmt.Errorf("read JAR: %w", err)
	}

	// Build lookup from obfuscated class name → class data
	dataByObf := make(map[string][]byte)
	for _, e := range entries {
		name := e.name
		if strings.HasSuffix(name, ".class") {
			simple := strings.TrimSuffix(name, ".class")
			dataByObf[simple] = e.data
		}
	}

	var found []PacketDef
	seen := make(map[string]bool)

	for obf, pi := range packets {
		data, ok := dataByObf[obf]
		if !ok {
			continue
		}
		jc, err := parseClass(data)
		if err != nil {
			continue
		}

		id := pickPacketID(jc.integerConstants())
		if id == -1 {
			continue
		}

		key := fmt.Sprintf("%s/%s/%s", pi.state, pi.bound, pi.name)
		if seen[key] {
			continue
		}
		seen[key] = true

		found = append(found, PacketDef{
			Name:  pi.name,
			ID:    id,
			State: pi.state,
			Bound: pi.bound,
		})
	}

	if len(found) == 0 {
		return nil, fmt.Errorf("no packet IDs extracted from JAR classes")
	}

	sort.Slice(found, func(i, j int) bool {
		if found[i].State != found[j].State {
			return found[i].State < found[j].State
		}
		if found[i].Bound != found[j].Bound {
			return found[i].Bound < found[j].Bound
		}
		return found[i].ID < found[j].ID
	})

	return found, nil
}

// pickPacketID picks the most likely packet ID from a class's integer constants.
// Typically packet IDs are in the 0-255 range; we pick the smallest such value.
func pickPacketID(ints []int32) int32 {
	best := int32(-1)
	for _, v := range ints {
		if v >= 0 && v <= 255 {
			if best == -1 || v < best {
				best = v
			}
		}
	}
	return best
}


