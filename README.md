# Helmfile Atlantis Config

Heavily inspired by [terragrunt-atlantis-config](https://github.com/transcend-io/terragrunt-atlantis-config), this tool creates Atlantis YAML configurations for Helmfile projects by:

- Finding all `helmfile.yaml` (customizable via env var `HELMFILE_FILE_NAME`) inside some folder (default to `clusters`) with a fixed depth (default to `3`)
- Check if the file has any line with `_atlantis_needs: $REPO_REL_DIR_TO_THE_DEPENDENCY`
- Construct the YAML in Atlantis' config spec version 3

## Integrate into your Atlantis Server

The recommended way to use this tool is to install it onto your Atlantis server, and then use a [Pre-Workflow hook](https://www.runatlantis.io/docs/pre-workflow-hooks.html#pre-workflow-hooks) to run it after every clone. This way, Atlantis can automatically determine what modules should be planned/applied for any change to your repository.

To get started, add a `pre_workflow_hooks` field to your `repos` section of your [server-side repo config](https://www.runatlantis.io/docs/server-side-repo-config.html#do-i-need-a-server-side-repo-config-file):

```json
{
  "repos": [
    {
      "id": "<your_github_repo>",
      "workflow": "default",
      "pre_workflow_hooks": [
        {
          "run": "AUTOMERGE=true PARALLEL_APPLY=false PARALLEL_PLAN=false DELETE_SOURCE_BRANCH=true BASE_DIR=clusters WORKFLOW_NAME=default DEPTH_TO_HELMFILES=3 helmfile-atlantis-config"
        }
      ]
    }
  ]
}
```
