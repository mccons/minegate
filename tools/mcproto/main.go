// mcproto is a code generator that automatically produces Minecraft protocol packet IDs.
//
// Usage:
//
//	cd tools/mcproto
//	go run main.go -version 1.21.4 -output ../../protocol/packetid/v1_21_4.go
//
// This tool generates packet ID constants as Go code from a Minecraft server JAR
// or from manual definitions.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

type PacketDef struct {
	Name    string
	ID      int32
	State   string
	Bound   string // serverbound / clientbound
}

func main() {
	version := flag.String("version", "1.21.4", "Minecraft version")
	output := flag.String("output", "", "Output file path")
	flag.Parse()

	packets := getDefinitions(*version)

	if *output == "" {
		pkgDir := fmt.Sprintf("../../protocol/packetid/v%s", strings.ReplaceAll(*version, ".", "_"))
		os.MkdirAll(pkgDir, 0755)
		*output = fmt.Sprintf("%s/types.go", pkgDir)
	}

	generate(packets, *version, *output)
	fmt.Printf("Generated %d packet IDs for Minecraft %s -> %s\n", len(packets), *version, *output)
}

func getDefinitions(version string) []PacketDef {
	// TODO: Extract packet IDs from the Minecraft server JAR
	// For now, using manually defined basic packets
	return []PacketDef{
		// Handshake (serverbound)
		{Name: "Handshake", ID: 0x00, State: "handshake", Bound: "serverbound"},

		// Status (serverbound)
		{Name: "StatusRequest", ID: 0x00, State: "status", Bound: "serverbound"},
		{Name: "StatusPing", ID: 0x01, State: "status", Bound: "serverbound"},

		// Status (clientbound)
		{Name: "StatusResponse", ID: 0x00, State: "status", Bound: "clientbound"},
		{Name: "StatusPong", ID: 0x01, State: "status", Bound: "clientbound"},

		// Login (serverbound)
		{Name: "LoginStart", ID: 0x00, State: "login", Bound: "serverbound"},
		{Name: "EncryptionResponse", ID: 0x01, State: "login", Bound: "serverbound"},
		{Name: "LoginPluginResponse", ID: 0x02, State: "login", Bound: "serverbound"},

		// Login (clientbound)
		{Name: "LoginDisconnect", ID: 0x00, State: "login", Bound: "clientbound"},
		{Name: "LoginSuccess", ID: 0x02, State: "login", Bound: "clientbound"},
		{Name: "SetCompression", ID: 0x03, State: "login", Bound: "clientbound"},
		{Name: "LoginPluginRequest", ID: 0x04, State: "login", Bound: "clientbound"},

		// Configuration (serverbound) - 1.20.5+
		{Name: "ConfigAck", ID: 0x00, State: "configuration", Bound: "serverbound"},
		{Name: "ConfigPluginResponse", ID: 0x01, State: "configuration", Bound: "serverbound"},

		// Configuration (clientbound) - 1.20.5+
		{Name: "ConfigPluginRequest", ID: 0x00, State: "configuration", Bound: "clientbound"},
		{Name: "ConfigDisconnect", ID: 0x01, State: "configuration", Bound: "clientbound"},
		{Name: "FinishConfiguration", ID: 0x02, State: "configuration", Bound: "clientbound"},
		{Name: "ConfigKeepAlive", ID: 0x03, State: "configuration", Bound: "clientbound"},
		{Name: "ConfigSyncData", ID: 0x04, State: "configuration", Bound: "clientbound"},

		// Play (serverbound) - KeepAlive
		{Name: "KeepAlive", ID: 0x18, State: "play", Bound: "serverbound"},
		{Name: "ChatMessage", ID: 0x05, State: "play", Bound: "serverbound"},
		{Name: "PlayerPosition", ID: 0x1A, State: "play", Bound: "serverbound"},
		{Name: "PlayerPositionRotation", ID: 0x1B, State: "play", Bound: "serverbound"},
		{Name: "PlayerRotation", ID: 0x1C, State: "play", Bound: "serverbound"},
		{Name: "PlayerMovement", ID: 0x1D, State: "play", Bound: "serverbound"},
		{Name: "ClientCommand", ID: 0x07, State: "play", Bound: "serverbound"},
		{Name: "PluginMessage", ID: 0x11, State: "play", Bound: "serverbound"},

		// Play (clientbound) - KeepAlive
		{Name: "KeepAlive", ID: 0x26, State: "play", Bound: "clientbound"},
		{Name: "JoinGame", ID: 0x2E, State: "play", Bound: "clientbound"},
		{Name: "ChatMessage", ID: 0x37, State: "play", Bound: "clientbound"},
		{Name: "SystemChat", ID: 0x66, State: "play", Bound: "clientbound"},
		{Name: "Disconnect", ID: 0x1D, State: "play", Bound: "clientbound"},
		{Name: "PluginMessage", ID: 0x25, State: "play", Bound: "clientbound"},
		{Name: "ServerData", ID: 0x47, State: "play", Bound: "clientbound"},
		{Name: "Respawn", ID: 0x44, State: "play", Bound: "clientbound"},
		{Name: "ChunkData", ID: 0x24, State: "play", Bound: "clientbound"},
		{Name: "ChunkDataUpdate", ID: 0x5B, State: "play", Bound: "clientbound"},
		{Name: "BlockUpdate", ID: 0x0A, State: "play", Bound: "clientbound"},
		{Name: "PlayerInfo", ID: 0x3D, State: "play", Bound: "clientbound"},
		{Name: "PlayerInfoRemove", ID: 0x42, State: "play", Bound: "clientbound"},
		{Name: "SynchronizePosition", ID: 0x40, State: "play", Bound: "clientbound"},
		{Name: "UpdateRecipes", ID: 0x7E, State: "play", Bound: "clientbound"},
	}
}

func generate(packets []PacketDef, version, output string) {
	var sb strings.Builder

	pkgName := fmt.Sprintf("v%s", strings.ReplaceAll(version, ".", "_"))
	pkgName = strings.ReplaceAll(pkgName, "-", "_")

	sb.WriteString(fmt.Sprintf("// Package %s - Auto-generated packet IDs for Minecraft %s\n", pkgName, version))
	sb.WriteString("// Code generated by tools/mcproto; DO NOT EDIT.\n\n")
	sb.WriteString(fmt.Sprintf("package %s\n\n", pkgName))

	sb.WriteString("const (\n")

	groups := make(map[string][]PacketDef)
	for _, p := range packets {
		key := p.State + "_" + p.Bound
		groups[key] = append(groups[key], p)
	}

	for key, defs := range groups {
		sb.WriteString(fmt.Sprintf("\t// %s\n", key))
		for _, d := range defs {
			constName := fmt.Sprintf("%s%s", d.Bound[:1], d.Name) // sHandshake, cKeepAlive
			sb.WriteString(fmt.Sprintf("\t%s PacketID = 0x%02X\n", constName, d.ID))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(")\n\n")
	sb.WriteString("type PacketID int32\n")

	if err := os.WriteFile(output, []byte(sb.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", output, err)
		os.Exit(1)
	}
}
