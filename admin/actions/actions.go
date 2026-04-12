package actions

import "os/exec"

func PullLatest() error {
	_, err := exec.Command("git", "pull").Output()
	return err
}

func CreateRecipeBackup() error {
	_, err := exec.Command("bash", "./scripts/backup.sh").Output()
	return err
}

func Rebuild(name string) error {
	_, err := exec.Command("docker", "compose", "up", "--build", "-d", name).Output()
	return err
}
