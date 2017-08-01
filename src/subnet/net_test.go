package subnet

import (
  "testing"
)

func TestNet(t *testing.T) {
  gw, device, err := GetNetGateway()
  if err != nil {
    t.Fatal(err)
  }
  if len(device) < 2 || len(gw) < 3 {
    t.Error(gw, device)
    t.Fatal("Expected longer results for device and interface")
  }
}
