package web

import (
	"m365-native/internal/outbound"
	"net/http"
	"os"
	"runtime"
	"time"
)

var startedAt = time.Now()

const Version = "dev"

func (s *Server) version(w http.ResponseWriter, _ *http.Request) {
	jsonOut(w, map[string]any{"version": Version, "go": runtime.Version(), "uptimeSeconds": int(time.Since(startedAt).Seconds()), "accounts": len(s.tokens.List()), "proxyPool": len(outbound.ProxyPoolStatus()), "dataDir": os.Getenv("M365_DATA_DIR")})
}
