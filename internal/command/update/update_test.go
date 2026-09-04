package update

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestVerifyChecksum(t *testing.T) {
	data := []byte("release archive bytes")
	sum := sha256.Sum256(data)
	good := hex.EncodeToString(sum[:])
	sums := []byte(strings.Join([]string{
		"deadbeef  connect-cli_0.1.0_linux_amd64.tar.gz",
		good + "  connect-cli_0.1.0_darwin_arm64.tar.gz",
		"",
	}, "\n"))

	if err := verifyChecksum("connect-cli_0.1.0_darwin_arm64.tar.gz", data, sums); err != nil {
		t.Fatalf("matching checksum rejected: %v", err)
	}
	if err := verifyChecksum("connect-cli_0.1.0_linux_amd64.tar.gz", data, sums); err == nil {
		t.Fatal("mismatching checksum accepted")
	}
	if err := verifyChecksum("connect-cli_0.1.0_windows_amd64.zip", data, sums); err == nil {
		t.Fatal("asset missing from checksums.txt accepted")
	}
	upper := []byte(strings.ToUpper(good) + "  x.tar.gz\n")
	if err := verifyChecksum("x.tar.gz", data, upper); err != nil {
		t.Fatalf("upper-case digest rejected: %v", err)
	}
}
