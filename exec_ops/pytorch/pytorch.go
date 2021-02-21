package pytorch

import (
	"../../skyhook"

	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func GetArgs(url string, node skyhook.ExecNode) (*skyhook.PytorchArch, map[int]*skyhook.PytorchComponent, map[int]*skyhook.Dataset, error) {
	var params skyhook.PytorchNodeParams
	skyhook.JsonUnmarshal([]byte(node.Params), &params)

	// get the PytorchComponents
	var arch skyhook.PytorchArch
	err := skyhook.JsonGet(url, fmt.Sprintf("/pytorch/archs/%d", params.ArchID), &arch)
	if err != nil {
		return nil, nil, nil, err
	}
	components := make(map[int]*skyhook.PytorchComponent)
	for _, compSpec := range arch.Params.Components {
		if components[compSpec.ID] != nil {
			continue
		}
		var comp skyhook.PytorchComponent
		err := skyhook.JsonGet(url, fmt.Sprintf("/pytorch/components/%d", compSpec.ID), &comp)
		if err != nil {
			return nil, nil, nil, err
		}
		components[comp.ID] = &comp
	}

	// get the Datasets
	datasets := make(map[int]*skyhook.Dataset)
	for _, dsSpec := range params.InputDatasets {
		if datasets[dsSpec.ID] != nil {
			continue
		}
		var ds skyhook.Dataset
		err := skyhook.JsonGet(url, fmt.Sprintf("/datasets/%d", dsSpec.ID), &ds)
		if err != nil {
			return nil, nil, nil, err
		}
		datasets[dsSpec.ID] = &ds
	}

	return &arch, components, datasets, nil
}

// Download this repository to the models/ folder if it doesn't already exist
func EnsureRepository(repo skyhook.PytorchRepository) error {
	// first compute hash as sha256(url[@commit])
	h := sha256.New()
	h.Write([]byte(repo.URL))
	if repo.Commit != "" {
		h.Write([]byte("@"+repo.Commit))
	}
	bytes := h.Sum(nil)
	hash := hex.EncodeToString(bytes)

	// does it already exist?
	path := filepath.Join("models", hash)
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

func EnsureRepositories(comps map[int]*skyhook.PytorchComponent) error {
	for _, comp := range comps {
		for _, repo := range comp.Params.Repositories {
			if err := EnsureRepository(repo); err != nil {
				return fmt.Errorf("error fetching repository %v: %v", repo, err)
			}
		}
	}
	return nil
}
