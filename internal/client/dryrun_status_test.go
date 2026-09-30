package client

import (
	"os"
	"strconv"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestDryRunSuccessStatusMatchesSpec(t *testing.T) {
	data, err := os.ReadFile("../../.speakeasy/out.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			OperationID string         `yaml:"operationId"`
			Ignore      bool           `yaml:"x-speakeasy-ignore"`
			Responses   map[string]any `yaml:"responses"`
		} `yaml:"paths"`
	}
	if err := yaml.Unmarshal(data, &spec); err != nil {
		t.Fatal(err)
	}
	want := map[string]int{}
	for _, operations := range spec.Paths {
		for _, op := range operations {
			if op.OperationID == "" || op.Ignore {
				continue
			}
			for code := range op.Responses {
				status, err := strconv.Atoi(code)
				if err == nil && status >= 200 && status < 300 && status != 200 {
					want[op.OperationID] = status
				}
			}
		}
	}
	for op, status := range want {
		if got := dryRunSuccessStatus[op]; got != status {
			t.Errorf("dryRunSuccessStatus[%q] = %d, spec says %d", op, got, status)
		}
	}
	for op := range dryRunSuccessStatus {
		if _, ok := want[op]; !ok {
			t.Errorf("dryRunSuccessStatus has %q, which the spec no longer answers with a non-200 success", op)
		}
	}
}
