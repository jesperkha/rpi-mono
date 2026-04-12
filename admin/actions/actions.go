package actions

import "os/exec"

func PullLatest() error {
	_, err := exec.Command("git", "pull").Output()
	return err
}

func Rebuild(name string) error {
	composeFile := "../" + name + "/docker-compose.yaml"
	_, err := exec.Command("docker", "compose", "-f", composeFile, "up", "--build", "-d").Output()
	return err
}
