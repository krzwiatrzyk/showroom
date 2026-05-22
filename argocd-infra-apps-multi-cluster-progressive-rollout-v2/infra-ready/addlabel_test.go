package main

import (
	"strings"
	"testing"
)

const sample = `apiVersion: v1
kind: Secret
metadata:
  name: beta1
  namespace: argocd
  labels:
    argocd.argoproj.io/secret-type: cluster
    type: beta
type: Opaque
stringData:
  name: beta1
  server: https://kubernetes.default.svc
  config: |
    {
      "tlsClientConfig": {
        "insecure": false
      }
    }
`

func TestAddInfraReadyLabel_Adds(t *testing.T) {
	out, changed, err := addInfraReadyLabel([]byte(sample))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	s := string(out)
	t.Logf("output:\n%s", s)
	if !strings.Contains(s, "windkube.com/infra-ready: \"true\"") {
		t.Fatalf("missing new label in output:\n%s", s)
	}
	if !strings.Contains(s, "argocd.argoproj.io/secret-type: cluster") {
		t.Fatalf("existing label lost:\n%s", s)
	}
}

func TestAddInfraReadyLabel_Idempotent(t *testing.T) {
	once, changed, err := addInfraReadyLabel([]byte(sample))
	if err != nil || !changed {
		t.Fatalf("first pass: err=%v changed=%v", err, changed)
	}
	_, changed2, err := addInfraReadyLabel(once)
	if err != nil {
		t.Fatal(err)
	}
	if changed2 {
		t.Fatalf("second pass should be no-op, got changed=true")
	}
}
