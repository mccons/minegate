package main

func fallbackDefinitions() []PacketDef {
	return []PacketDef{
		{Name: "Handshake", ID: 0x00, State: "handshake", Bound: "serverbound"},

		{Name: "StatusRequest", ID: 0x00, State: "status", Bound: "serverbound"},
		{Name: "StatusPing", ID: 0x01, State: "status", Bound: "serverbound"},
		{Name: "StatusResponse", ID: 0x00, State: "status", Bound: "clientbound"},
		{Name: "StatusPong", ID: 0x01, State: "status", Bound: "clientbound"},

		{Name: "LoginStart", ID: 0x00, State: "login", Bound: "serverbound"},
		{Name: "EncryptionResponse", ID: 0x01, State: "login", Bound: "serverbound"},
		{Name: "LoginPluginResponse", ID: 0x02, State: "login", Bound: "serverbound"},
		{Name: "LoginDisconnect", ID: 0x00, State: "login", Bound: "clientbound"},
		{Name: "LoginSuccess", ID: 0x02, State: "login", Bound: "clientbound"},
		{Name: "SetCompression", ID: 0x03, State: "login", Bound: "clientbound"},
		{Name: "LoginPluginRequest", ID: 0x04, State: "login", Bound: "clientbound"},

		{Name: "ConfigAck", ID: 0x00, State: "configuration", Bound: "serverbound"},
		{Name: "ConfigPluginResponse", ID: 0x01, State: "configuration", Bound: "serverbound"},
		{Name: "ConfigPluginRequest", ID: 0x00, State: "configuration", Bound: "clientbound"},
		{Name: "ConfigDisconnect", ID: 0x01, State: "configuration", Bound: "clientbound"},
		{Name: "FinishConfiguration", ID: 0x02, State: "configuration", Bound: "clientbound"},
		{Name: "ConfigKeepAlive", ID: 0x03, State: "configuration", Bound: "clientbound"},
		{Name: "ConfigSyncData", ID: 0x04, State: "configuration", Bound: "clientbound"},

		{Name: "KeepAlive", ID: 0x18, State: "play", Bound: "serverbound"},
		{Name: "ChatMessage", ID: 0x05, State: "play", Bound: "serverbound"},
		{Name: "PlayerPosition", ID: 0x1A, State: "play", Bound: "serverbound"},
		{Name: "PlayerPositionRotation", ID: 0x1B, State: "play", Bound: "serverbound"},
		{Name: "PlayerRotation", ID: 0x1C, State: "play", Bound: "serverbound"},
		{Name: "PlayerMovement", ID: 0x1D, State: "play", Bound: "serverbound"},
		{Name: "ClientCommand", ID: 0x07, State: "play", Bound: "serverbound"},
		{Name: "PluginMessage", ID: 0x11, State: "play", Bound: "serverbound"},

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
