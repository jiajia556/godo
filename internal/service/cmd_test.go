package service

import "testing"

func TestValidateCmdName(t *testing.T) {
	for _, name := range []string{"default-api", "worker_v2", "服务2"} {
		if err := ValidateCmdName(name); err != nil {
			t.Errorf("ValidateCmdName(%q) error = %v", name, err)
		}
	}
	for _, name := range []string{"", ".", "..", "../api", `admin\api`, " api", "api.name", "-api"} {
		if err := ValidateCmdName(name); err == nil {
			t.Errorf("ValidateCmdName(%q) succeeded", name)
		}
	}
}
