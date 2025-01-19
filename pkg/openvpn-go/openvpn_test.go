// Copyright 2025 Abhisheke Acharya
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openvpn

import (
	"os"
	"os/exec"
	"testing"
	"time"
)

const testTimeout time.Duration = 30 * time.Second

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

	cmd, _, err := Start(config, auth, os.Getenv("VPN_ARGS"), testTimeout)
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

	cmd, _, err := Start(config, auth, os.Getenv("VPN_ARGS"), testTimeout)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	cmd, _, err = Restart(cmd, altConfig, auth, os.Getenv("VPN_ARGS"), testTimeout)
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(2 * time.Second)

	if err := Stop(cmd); err != nil {
		t.Fatal(err)
	}
}
