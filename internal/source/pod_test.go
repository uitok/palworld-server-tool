package source

import "testing"

func TestParseK8sAddress(t *testing.T) {
	t.Run("with namespace", func(t *testing.T) {
		namespace, pod, container, filePath, err := ParseK8sAddress("k8s://default/palworld-0/palworld:/palworld/Pal/Saved")
		if err != nil {
			t.Fatalf("expected valid k8s address, got %v", err)
		}
		if namespace != "default" || pod != "palworld-0" || container != "palworld" || filePath != "/palworld/Pal/Saved" {
			t.Fatalf("unexpected parse result: %q %q %q %q", namespace, pod, container, filePath)
		}
	})

	t.Run("without namespace", func(t *testing.T) {
		namespace, pod, container, filePath, err := ParseK8sAddress("k8s://palworld-0/palworld:/palworld/Pal/Saved")
		if err != nil {
			t.Fatalf("expected valid k8s address, got %v", err)
		}
		if namespace != "" || pod != "palworld-0" || container != "palworld" || filePath != "/palworld/Pal/Saved" {
			t.Fatalf("unexpected parse result: %q %q %q %q", namespace, pod, container, filePath)
		}
	})

	invalidCases := []string{
		"k8s://default/palworld-0/palworld",
		"k8s://palworld-0/:/palworld/Pal/Saved",
		"k8s://default//palworld:/palworld/Pal/Saved",
		"k8s://default/palworld-0/palworld:",
	}
	for _, input := range invalidCases {
		t.Run(input, func(t *testing.T) {
			if _, _, _, _, err := ParseK8sAddress(input); err == nil {
				t.Fatalf("expected invalid k8s address error for %s", input)
			}
		})
	}
}
