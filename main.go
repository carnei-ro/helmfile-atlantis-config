package main

import (
	"bufio"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type AutoPlan struct {
	Enabled      bool     `yaml:"enabled"`
	WhenModified []string `yaml:"when_modified"`
}

type Project struct {
	Dir                       string   `yaml:"dir"`
	DeleteSourceBranchOnMerge bool     `yaml:"delete_source_branch_on_merge"`
	ApplyRequirements         []string `yaml:"apply_requirements"`
	Workflow                  string   `yaml:"workflow"`
	AutoPlan                  AutoPlan `yaml:"autoplan"`
}

type AtlantisConfig struct {
	Version                   int       `yaml:"version"`
	AutoMerge                 bool      `yaml:"automerge"`
	ParallelApply             bool      `yaml:"parallel_apply"`
	ParallelPlan              bool      `yaml:"parallel_plan"`
	DeleteSourceBranchOnMerge bool      `yaml:"delete_source_branch_on_merge"`
	Projects                  []Project `yaml:"projects"`
}

func getenvBool(name string, defaultValue string) bool {
	val := os.Getenv(name)
	if val == "" {
		val = defaultValue
	}
	return strings.ToLower(val) == "true" || strings.ToLower(val) == "y" || strings.ToLower(val) == "yes"
}

func walkDir(root string, depthToHelmfiles int) ([]string, error) {
	var dirs []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.Count(path, string(os.PathSeparator)) == depthToHelmfiles && strings.HasSuffix(path, "helmfile.yaml") {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	return dirs, err
}

func main() {
	automerge := getenvBool("AUTOMERGE", "true")
	parallelApply := getenvBool("PARALLEL_APPLY", "false")
	parallelPlan := getenvBool("PARALLEL_PLAN", "false")
	deleteSourceBranch := getenvBool("DELETE_SOURCE_BRANCH", "true")
	baseDir := os.Getenv("BASE_DIR")
	if baseDir == "" {
		baseDir = "clusters"
	}
	workflowName := os.Getenv("WORKFLOW_NAME")
	if workflowName == "" {
		workflowName = "k8s_live"
	}
	depthToHelmfilesStr := os.Getenv("DEPTH_TO_HELMFILES") // clusters/CLUSTER_NAME/PACKAGE_NAME -> 3
	if depthToHelmfilesStr == "" {
		depthToHelmfilesStr = "3"
	}
	depthToHelmfiles, err := strconv.Atoi(depthToHelmfilesStr)
	if err != nil {
		panic(err)
	}

	atlantisConfig := AtlantisConfig{
		Version:                   3,
		AutoMerge:                 automerge,
		ParallelApply:             parallelApply,
		ParallelPlan:              parallelPlan,
		DeleteSourceBranchOnMerge: deleteSourceBranch,
		Projects: []Project{
			{
				Dir:      ".",
				Workflow: "fail_empty_dir",
				AutoPlan: AutoPlan{
					Enabled: false,
				},
			},
		},
	}

	helmfileDirs, _ := walkDir(baseDir, depthToHelmfiles)

	for _, dir := range helmfileDirs {
		atlantisConfig.Projects = append(atlantisConfig.Projects, Project{
			Dir:                       dir,
			DeleteSourceBranchOnMerge: true,
			ApplyRequirements:         []string{"mergeable", "approved", "undiverged"},
			Workflow:                  workflowName,
			AutoPlan: AutoPlan{
				Enabled:      true,
				WhenModified: []string{"**"},
			},
		})
	}

	for _, dir := range helmfileDirs {
		f, _ := os.Open(dir + "/helmfile.yaml")
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "_atlantis_needs: ") {
				slc := strings.Split(line, " ")
				dependency := strings.TrimSpace(slc[len(slc)-1])
				for i, project := range atlantisConfig.Projects {
					if project.Dir == dependency {
						pathRelatedToRoot := strings.Repeat("../", len(strings.Split(dependency, string(os.PathSeparator))))
						atlantisConfig.Projects[i].AutoPlan.WhenModified = append(atlantisConfig.Projects[i].AutoPlan.WhenModified, pathRelatedToRoot+dir+"/**")
					}
				}
			}
		}
		f.Close()
	}

	data, _ := yaml.Marshal(&atlantisConfig)
	ioutil.WriteFile("atlantis.yaml", data, 0644)
}
