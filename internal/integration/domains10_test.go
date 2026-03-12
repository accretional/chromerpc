package integration

import (
	"context"
	"testing"

	bluetoothemulationpb "github.com/accretional/chromerpc/proto/cdp/bluetoothemulation"
	deviceaccesspb "github.com/accretional/chromerpc/proto/cdp/deviceaccess"
)

// =============================================================
// DeviceAccess Domain Tests
// =============================================================

func TestDeviceAccessEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.deviceAccessClient.Enable(ctx, &deviceaccesspb.EnableRequest{})
	if err != nil {
		t.Fatalf("DeviceAccess.Enable: %v", err)
	}

	_, err = env.deviceAccessClient.Disable(ctx, &deviceaccesspb.DisableRequest{})
	if err != nil {
		t.Fatalf("DeviceAccess.Disable: %v", err)
	}
}

// =============================================================
// BluetoothEmulation Domain Tests
// =============================================================

func TestBluetoothEmulationEnableDisable(t *testing.T) {
	env := setupTestEnv(t)
	ctx := context.Background()

	_, err := env.bluetoothEmulationClient.Enable(ctx, &bluetoothemulationpb.EnableRequest{
		State: "powered-on",
	})
	if err != nil {
		// BluetoothEmulation may not be available in all Chrome versions.
		t.Logf("BluetoothEmulation.Enable: %v", err)
		return
	}

	_, err = env.bluetoothEmulationClient.Disable(ctx, &bluetoothemulationpb.DisableRequest{})
	if err != nil {
		t.Logf("BluetoothEmulation.Disable: %v", err)
	}
}
