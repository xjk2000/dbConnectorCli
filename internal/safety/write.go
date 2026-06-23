package safety

import (
	"dbconnector/internal/config"
	"dbconnector/internal/protocol"
)

func CheckWriteAllowed(cfg *config.Config, profile *config.Profile, explicitWrite bool) *protocol.Error {
	if !explicitWrite {
		return protocol.NewError("WRITE_NOT_ALLOWED", "write operation requires --write", false)
	}
	if !cfg.Defaults.AllowWrite {
		return protocol.NewError("WRITE_NOT_ALLOWED", "write operation requires defaults.allowWrite=true", false)
	}
	if profile.Readonly {
		return protocol.NewError("WRITE_NOT_ALLOWED", "write operation is blocked by readonly profile", false)
	}
	return nil
}
