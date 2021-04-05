package pytorch

import (
	"github.com/skyhookml/skyhookml/skyhook"

	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func GetTrainArgs(url string, archID string) (*skyhook.PytorchArch, map[string]*skyhook.PytorchComponent, error) {
	// get the PytorchComponents
	var arch skyhook.PytorchArch
	err := skyhook.JsonGet(url, fmt.Sprintf("/pytorch/archs/%s", archID), &arch)
	if err != nil {
		return nil, nil, err
	}
	components := make(map[string]*skyhook.PytorchComponent)
	for _, compSpec := range arch.Params.Components {
		if components[compSpec.ID] != nil {
			continue
		}
		var comp skyhook.PytorchComponent
		err := skyhook.JsonGet(url, fmt.Sprintf("/pytorch/components/%s", compSpec.ID), &comp)
		if err != nil {
			return nil, nil, err
		}
		components[comp.ID] = &comp
	}

	return &arch, components, nil
}

// Download this repository to the models/ folder if it doesn't already exist
func EnsureRepository(repo skyhook.PytorchRepository) error {
	hash := repo.Hash()

	// does it already exist?
	path := filepath.Join("data/models", hash)
	if _, err := os.Stat(path); err == nil {
		return nil
	}

	// clone the repository
	log.Printf("[pytorch] cloning repository %s@%s", repo.URL, repo.Commit)
	cmd := exec.Command(
		"git", "clone", repo.URL, path,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	if repo.Commit != "" {
		cmd = exec.Command(
			"git", "checkout", repo.Commit,
		)
		cmd.Dir = path
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func EnsureRepositories(comps map[string]*skyhook.PytorchComponent) error {
	for _, comp := range comps {
		for _, repo := range comp.Params.Repositories {
			if err := EnsureRepository(repo); err != nil {
				return fmt.Errorf("error fetching repository %v: %v", repo, err)
			}
		}
	}
	return nil
}
