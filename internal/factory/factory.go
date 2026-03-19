package factory

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/redpine-ai/connect-cli/internal/cache"
	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/mcp"
	"github.com/redpine-ai/connect-cli/internal/output"
)

type Factory struct {
	IOStreams func() *output.IOStreams
	Config   func() (*config.Config, error)
	Keyring  func() config.Keyring
	Token    func(flagValue string) (token, source string)

	MCPClient func(token string) *mcp.Client
	ToolCache func() *cache.ToolCache

	// Global flag values — set by root command
	APIKeyFlag   string
	ServerFlag   string
	JSONFlag     string
	PrettyFlag   bool
	QuietFlag    bool
	InsecureFlag bool
}

func New() *Factory {
	f := &Factory{}

	var cachedIOS *output.IOStreams
	f.IOStreams = func() *output.IOStreams {
		if cachedIOS == nil {
			cachedIOS = output.New()
		}
		return cachedIOS
	}

	var cachedConfig *config.Config
	f.Config = func() (*config.Config, error) {
		if cachedConfig != nil {
			return cachedConfig, nil
		}
		cfg, err := config.Load()
		if err != nil {
			return nil, err
		}
		cachedConfig = cfg
		return cfg, nil
	}

	var cachedKeyring config.Keyring
	f.Keyring = func() config.Keyring {
		if cachedKeyring == nil {
			cachedKeyring = config.NewSystemKeyring()
		}
		return cachedKeyring
	}

	f.Token = func(flagValue string) (string, string) {
		return config.ResolveToken(flagValue, f.Keyring())
	}

	f.MCPClient = func(token string) *mcp.Client {
		cfg, err := f.Config()
		if err != nil {
			return mcp.NewClient(config.DefaultServerURL, token, f.InsecureFlag)
		}
		serverURL := f.ServerFlag
		if serverURL == "" {
			serverURL = os.Getenv("CONNECT_SERVER_URL")
		}
		if serverURL == "" {
			serverURL = cfg.ServerURL
		}
		return mcp.NewClient(serverURL, token, f.InsecureFlag)
	}

	f.ToolCache = func() *cache.ToolCache {
		return cache.NewToolCache(filepath.Join(config.ConfigDir(), "cache"))
	}

	return f
}

// MCPClientWithSession creates an MCP client and attempts to reuse a cached
// session ID. On a cold start it calls Initialize and caches the new session.
func (f *Factory) MCPClientWithSession(token string) (*mcp.Client, *mcp.SessionCache, error) {
	cfg, err := f.Config()
	if err != nil {
		cfg = &config.Config{ServerURL: config.DefaultServerURL}
	}
	serverURL := f.ServerFlag
	if serverURL == "" {
		serverURL = os.Getenv("CONNECT_SERVER_URL")
	}
	if serverURL == "" {
		serverURL = cfg.ServerURL
	}

	client := mcp.NewClient(serverURL, token, f.InsecureFlag)
	sc := mcp.DefaultSessionCache(serverURL)

	// Try cached session
	if sid := sc.Load(); sid != "" {
		client.SetSessionID(sid)
		return client, sc, nil
	}

	// Cold start — initialize
	if err := client.Initialize(); err != nil {
		// If 401, try refreshing the OAuth token
		if strings.Contains(err.Error(), "401") || strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
			newToken, refreshErr := config.RefreshOAuthToken(f.Keyring())
			if refreshErr == nil && newToken != "" {
				client = mcp.NewClient(serverURL, newToken, f.InsecureFlag)
				if err := client.Initialize(); err != nil {
					return nil, nil, err
				}
				sc.Save(client.SessionID())
				return client, sc, nil
			}
		}
		return nil, nil, err
	}
	sc.Save(client.SessionID())
	return client, sc, nil
}

func NewTest(ios *output.IOStreams, cfg *config.Config, kr config.Keyring) *Factory {
	f := &Factory{}
	f.IOStreams = func() *output.IOStreams { return ios }
	f.Config = func() (*config.Config, error) { return cfg, nil }
	f.Keyring = func() config.Keyring { return kr }
	f.Token = func(flagValue string) (string, string) {
		return config.ResolveToken(flagValue, kr)
	}
	f.MCPClient = func(token string) *mcp.Client {
		return mcp.NewClient(cfg.ServerURL, token, true)
	}
	f.ToolCache = func() *cache.ToolCache {
		return cache.NewToolCache("")
	}
	return f
}
