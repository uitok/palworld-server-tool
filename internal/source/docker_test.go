package source

import "testing"

func TestParseDockerAddress(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		containerID, filePath, err := ParseDockerAddress("docker://palworld-server:/palworld/Pal/Saved")
		if err != nil {
			t.Fatalf("expected valid docker address, got %v", err)
		}
		if containerID != "palworld-server" || filePath != "/palworld/Pal/Saved" {
			t.Fatalf("unexpected parse result: %q %q", containerID, filePath)
		}
	})

	invalidCases := []string{
		"docker://palworld-server",
		"docker://:/palworld/Pal/Saved",
		"docker://palworld-server:",
	}
	for _, input := range invalidCases {
		t.Run(input, func(t *testing.T) {
			if _, _, err := ParseDockerAddress(input); err == nil {
				t.Fatalf("expected invalid docker address error for %s", input)
			}
		})
	}
}
