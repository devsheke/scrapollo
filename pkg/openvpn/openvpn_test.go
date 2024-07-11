package openvpn

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

func testCheckOpenVPNInstalled(t *testing.T) bool {
	err := exec.Command("which", "openvpn").Run()
	if err == nil {
		t.Log("openvpn executable not found. skipping test.")
		return true
	}

	switch err.(type) {
	case *exec.ExitError:
		return false
	default:
		t.Fatal(err)
		return false
	}
}

func testEnvLookup(t *testing.T, env string) string {
	envvar, ok := os.LookupEnv(env)
	if !ok {
		t.Fatalf("missing %q from env", env)
	}

	return envvar
}

func TestOpenVPN(t *testing.T) {
	if !testCheckOpenVPNInstalled(t) {
		return
	}

	config := testEnvLookup(t, "VPN_CONFIG")
	auth := testEnvLookup(t, "VPN_AUTH")

	cmd, _, err := Start(config, auth)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	if err := Stop(cmd); err != nil {
		t.Fatal(err)
	}
}

func TestOpenVPNRestart(t *testing.T) {
	if !testCheckOpenVPNInstalled(t) {
		return
	}

	config := testEnvLookup(t, "VPN_CONFIG")
	altConfig := testEnvLookup(t, "VPN_ALT_CONFIG")
	auth := testEnvLookup(t, "VPN_AUTH")

	cmd, _, err := Start(config, auth)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	cmd, _, err = Restart(cmd, altConfig, auth)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	if err := Stop(cmd); err != nil {
		t.Fatal(err)
	}
}
