package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/redpine-ai/connect-cli/internal/config"
	"github.com/redpine-ai/connect-cli/internal/factory"
	"github.com/redpine-ai/connect-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func NewLoginCmd(f *factory.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authenticate via browser (OAuth)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return &output.CLIError{
					Code:     "non_interactive",
					Message:  "Cannot use browser login in non-interactive mode",
					Hint:     "Set CONNECT_API_KEY or use 'redpine auth set-key'",
					ExitCode: output.ExitAuth,
				}
			}

			cfg, err := f.Config()
			if err != nil {
				return &output.CLIError{Code: "config_error", Message: err.Error(), ExitCode: output.ExitError}
			}

			serverURL := f.ServerFlag
			if serverURL == "" {
				serverURL = os.Getenv("CONNECT_SERVER_URL")
			}
			if serverURL == "" {
				serverURL = cfg.ServerURLForEnv()
			}
			serverURL = strings.TrimSuffix(serverURL, "/")

			ios := f.IOStreams()

			// 1. Start local callback server
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return &output.CLIError{Code: "listen_error", Message: fmt.Sprintf("Failed to start callback server: %s", err), ExitCode: output.ExitError}
			}
			port := listener.Addr().(*net.TCPAddr).Port
			callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

			codeCh := make(chan string, 1)
			errCh := make(chan string, 1)

			mux := http.NewServeMux()
			mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
				code := r.URL.Query().Get("code")
				errParam := r.URL.Query().Get("error")
				if errParam != "" {
					errDesc := r.URL.Query().Get("error_description")
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprintf(w, "<html><body><h2>Authentication Failed</h2><p>%s: %s</p><p>You can close this tab.</p></body></html>", errParam, errDesc)
					errCh <- fmt.Sprintf("%s: %s", errParam, errDesc)
					return
				}
				if code == "" {
					w.Header().Set("Content-Type", "text/html")
					fmt.Fprint(w, "<html><body><h2>Error</h2><p>No authorization code received.</p></body></html>")
					errCh <- "no authorization code received"
					return
				}
				w.Header().Set("Content-Type", "text/html")
				fmt.Fprint(w, "<html><body><h2>Authentication Successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
				codeCh <- code
			})

			srv := &http.Server{Handler: mux}
			go srv.Serve(listener) //nolint:errcheck
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()
				srv.Shutdown(ctx) //nolint:errcheck
			}()

			// 2. Register client
			fmt.Fprintln(ios.ErrOut, "Registering client...")
			regBody, _ := json.Marshal(map[string]interface{}{
				"client_name":    "Redpine CLI",
				"redirect_uris":  []string{callbackURL},
				"grant_types":    []string{"authorization_code", "refresh_token"},
				"response_types": []string{"code"},
			})

			regResp, err := http.Post(serverURL+"/register", "application/json", strings.NewReader(string(regBody)))
			if err != nil {
				return &output.CLIError{Code: "register_error", Message: fmt.Sprintf("Failed to register client: %s", err), ExitCode: output.ExitServer}
			}
			defer regResp.Body.Close()

			if regResp.StatusCode != 201 {
				return &output.CLIError{Code: "register_error", Message: fmt.Sprintf("Client registration failed (HTTP %d)", regResp.StatusCode), ExitCode: output.ExitServer}
			}

			var regResult struct {
				ClientID     string `json:"client_id"`
				ClientSecret string `json:"client_secret"`
			}
			if err := json.NewDecoder(regResp.Body).Decode(&regResult); err != nil {
				return &output.CLIError{Code: "register_error", Message: "Failed to parse registration response", ExitCode: output.ExitServer}
			}

			// 3. Generate PKCE
			verifierBytes := make([]byte, 32)
			if _, err := rand.Read(verifierBytes); err != nil {
				return &output.CLIError{Code: "pkce_error", Message: fmt.Sprintf("Failed to generate PKCE verifier: %s", err), ExitCode: output.ExitError}
			}
			codeVerifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

			challengeHash := sha256.Sum256([]byte(codeVerifier))
			codeChallenge := base64.RawURLEncoding.EncodeToString(challengeHash[:])

			stateBytes := make([]byte, 16)
			if _, err := rand.Read(stateBytes); err != nil {
				return &output.CLIError{Code: "state_error", Message: fmt.Sprintf("Failed to generate state: %s", err), ExitCode: output.ExitError}
			}
			state := base64.RawURLEncoding.EncodeToString(stateBytes)

			// 4. Build authorize URL and open browser
			params := url.Values{
				"response_type":         {"code"},
				"client_id":             {regResult.ClientID},
				"redirect_uri":          {callbackURL},
				"code_challenge":        {codeChallenge},
				"code_challenge_method": {"S256"},
				"state":                 {state},
				"scope":                 {"mcp"},
			}
			authorizeURL := serverURL + "/authorize?" + params.Encode()

			fmt.Fprintln(ios.ErrOut, "Opening browser for authentication...")
			fmt.Fprintf(ios.ErrOut, "If the browser doesn't open, visit:\n%s\n\n", authorizeURL)
			openBrowser(authorizeURL)

			// 5. Wait for callback
			fmt.Fprintln(ios.ErrOut, "Waiting for authentication...")
			var authCode string
			select {
			case authCode = <-codeCh:
				// Got the code
			case errMsg := <-errCh:
				return &output.CLIError{Code: "auth_error", Message: errMsg, ExitCode: output.ExitAuth}
			case <-time.After(5 * time.Minute):
				return &output.CLIError{Code: "timeout", Message: "Authentication timed out (5 minutes)", Hint: "Try again with 'redpine auth login'", ExitCode: output.ExitAuth}
			}

			// 6. Exchange code for token
			tokenData := url.Values{
				"grant_type":    {"authorization_code"},
				"code":          {authCode},
				"redirect_uri":  {callbackURL},
				"client_id":     {regResult.ClientID},
				"client_secret": {regResult.ClientSecret},
				"code_verifier": {codeVerifier},
			}

			tokenResp, err := http.PostForm(serverURL+"/token", tokenData)
			if err != nil {
				return &output.CLIError{Code: "token_error", Message: fmt.Sprintf("Token exchange failed: %s", err), ExitCode: output.ExitServer}
			}
			defer tokenResp.Body.Close()

			if tokenResp.StatusCode != 200 {
				return &output.CLIError{Code: "token_error", Message: fmt.Sprintf("Token exchange failed (HTTP %d)", tokenResp.StatusCode), ExitCode: output.ExitServer}
			}

			var tokenResult struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				TokenType    string `json:"token_type"`
				ExpiresIn    int    `json:"expires_in"`
			}
			if err := json.NewDecoder(tokenResp.Body).Decode(&tokenResult); err != nil {
				return &output.CLIError{Code: "token_error", Message: "Failed to parse token response", ExitCode: output.ExitServer}
			}

			// 7. Store token + refresh token
			kr := f.Keyring()
			kr.Set(tokenResult.AccessToken) // best-effort keyring

			// Always save to file — refresh token needed for auto-refresh
			creds := &config.Credentials{
				Token:        tokenResult.AccessToken,
				Type:         "oauth",
				RefreshToken: tokenResult.RefreshToken,
				ClientID:     regResult.ClientID,
				ClientSecret: regResult.ClientSecret,
				ServerURL:    serverURL,
			}
			if err := creds.SaveTo(config.ConfigDir()); err != nil {
				return &output.CLIError{Code: "store_error", Message: fmt.Sprintf("Failed to store credentials: %s", err), ExitCode: output.ExitError}
			}

			fmt.Fprintln(ios.ErrOut, "Authentication successful!")
			if f.JSONFlag != "" {
				return ios.WriteJSON(output.NewSuccessEnvelope(map[string]string{
					"message": "Authenticated successfully",
				}))
			}
			return nil
		},
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		cmd.Start() //nolint:errcheck
	}
}
