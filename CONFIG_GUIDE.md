## Configuration File Guide

yamldiff supports configuration files in YAML format, similar to [tfcmt](https://github.com/suzuki-shunsuke/tfcmt).

## Basic Structure

```yaml
repo_owner: <GitHub organization or user>
repo_name: <Repository name>

yamldiff:
  compare:
    template: |
      <Go template for PR comment>
    when_has_additions:
      label: "<label when additions exist>"
    when_has_deletions:
      label: "<label when deletions exist>"
    when_has_modifications:
      label: "<label when modifications exist>"
    when_no_changes:
      label: "<label when no changes>"
    disable_comment: false
    disable_label: false
```

## Label Selection Logic

Labels are **cumulative** - multiple labels can be added to a single PR based on what types of changes exist:

1. **No changes** (Added = 0, Deleted = 0, Modified = 0): Only `when_no_changes` label
2. **Has additions** (Added > 0): `when_has_additions` label is added
3. **Has deletions** (Deleted > 0): `when_has_deletions` label is added
4. **Has modifications** (Modified > 0): `when_has_modifications` label is added

**Example**: If a PR has 1 addition, 1 deletion, and 1 modification, **all three labels** will be added:
- `config-sync/add`
- `config-sync/destroy`
- `config-sync/changes`

This allows you to immediately see what types of changes are in a PR at a glance.

## Usage Examples

### Basic Usage

```bash
yamldiff -v old.yaml new.yaml \
  --config=yamldiff.yaml \
  --post-comment \
  --github-pr=123
```

### With CI Link

```bash
yamldiff -v old.yaml new.yaml \
  --config=yamldiff.yaml \
  --post-comment \
  --github-pr=123 \
  --link="https://console.cloud.google.com/cloud-build/builds/..."
```

### With Custom Variables

```bash
yamldiff -v old.yaml new.yaml \
  --config=yamldiff.yaml \
  --post-comment \
  --github-pr=123 \
  --var message="Deployment to production" \
  --var environment="prod" \
  --var service="api-server"
```

Then in your template:
```
{{.Vars.message}}
Environment: {{.Vars.environment}}
Service: {{.Vars.service}}
```

## Example Configurations

### Minimal Configuration

```yaml
repo_owner: myorg
repo_name: myrepo

yamldiff:
  compare:
    template: |
      ## YAML Changes
      {{.Summary}}
    when_has_additions:
      label: "has-additions"
    when_has_deletions:
      label: "has-deletions"
    when_has_modifications:
      label: "has-changes"
    when_no_changes:
      label: "no-changes"
```

### Full-featured Configuration

```yaml
repo_owner: ${OWNER}
repo_name: ${REPO_NAME}

yamldiff:
  compare:
    template: |
      ## [{{.Vars.service}}] Configuration Changes
      
      {{if .Link}}**[View CI Build]({{.Link}})**{{end}}
      
      {{.Vars.message}}
      
      ### Change Summary
      - Added: {{.Added}} resources
      - Deleted: {{.Deleted}} resources
      - Modified: {{.Modified}} resources
      
      {{if .HasChanges}}
      {{if gt .Deleted 0}}
      ⚠️ **WARNING**: This PR contains deletions. Please review carefully.
      {{end}}
      
      <details>
      <summary>📋 Full Diff</summary>
      
      ```yaml
      {{.Details}}
      ```
      
      </details>
      
      ### ⚠️ Review Guidelines
      - Verify all resource names are correct
      - Check for unintended deletions
      - Confirm modifications are expected
      {{else}}
      ✅ No changes detected in YAML files.
      {{end}}
      
      ---
      
      **Environment**: {{.Vars.environment}}  
      **Target Branch**: {{.Vars.target_branch}}
      
      To retry this job, comment `/retry` on this PR.
      
      [Documentation](https://example.com/docs)
      
    when_has_additions:
      label: "yaml/added"
    when_has_deletions:
      label: "yaml/deleted"
    when_has_modifications:
      label: "yaml/modified"
    when_no_changes:
      label: "yaml/no-changes"
```

### Kubernetes-specific Configuration

```yaml
repo_owner: myorg
repo_name: k8s-manifests

yamldiff:
  compare:
    template: |
      ## Kubernetes Manifest Changes
      
      {{if .Link}}[CI Pipeline]({{.Link}}){{end}}
      
      Cluster: **{{.Vars.cluster}}**  
      Namespace: **{{.Vars.namespace}}**
      
      {{if .HasChanges}}
      ### Changes
      {{if gt .Added 0}}✨ **{{.Added}}** new resource(s){{end}}
      {{if gt .Deleted 0}}🗑️  **{{.Deleted}}** deleted resource(s){{end}}
      {{if gt .Modified 0}}✏️  **{{.Modified}}** modified resource(s){{end}}
      
      <details>
      <summary>View Diff</summary>
      
      ```diff
      {{.Details}}
      ```
      
      </details>
      {{else}}
      ✅ No changes to Kubernetes manifests.
      {{end}}
      
    when_has_deletions:
      label: "k8s/deletion-warning"
    when_has_additions:
      label: "k8s/additions"
    when_has_modifications:
      label: "k8s/modifications"
    when_no_changes:
      label: "k8s/no-changes"
```

## Configuration File Location

By convention, place your config file in one of these locations:
- `.yamldiff/yamldiff.yaml` (recommended for project-wide config)
- `.github/yamldiff.yaml` (for GitHub Actions)
- `yamldiff.yaml` (root of repository)

## Disabling Features

### Disable Comments Only

```yaml
yamldiff:
  compare:
    disable_comment: true
    # Labels will still be added
```

### Disable Labels Only

```yaml
yamldiff:
  compare:
    disable_label: true
    # Comments will still be posted
```

### Disable Both

Use the legacy `--github-label` flag instead, or don't use any GitHub integration flags.

## CI Integration

### Cloud Build

```yaml
steps:
  - name: 'gcr.io/cloud-builders/git'
    entrypoint: 'bash'
    args:
      - '-c'
      - |
        yamldiff -v old.yaml new.yaml \
          --config=.yamldiff/yamldiff.yaml \
          --post-comment \
          --github-pr=${_PR_NUMBER} \
          --link=${BUILD_URL} \
          --var environment=${_ENVIRONMENT}
```

### GitHub Actions

```yaml
- name: Check YAML changes
  run: |
    yamldiff -v old.yaml new.yaml \
      --config=.github/yamldiff.yaml \
      --post-comment \
      --github-pr=${{ github.event.pull_request.number }} \
      --link=${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }} \
      --var environment=${{ inputs.environment }}
```

## Comparison with tfcmt

| Feature | tfcmt | yamldiff |
|---------|-------|----------|
| Config file format | YAML | YAML |
| Template engine | Go templates | Go templates |
| Label selection | Plan/Apply states | Diff types (add/delete/modify) |
| Custom variables | ✅ | ✅ |
| GitHub API | Direct | via `gh` CLI |
| Primary use case | Terraform | YAML files |

## Troubleshooting

### Template Rendering Errors

If you see template errors:
1. Check syntax with Go template validator
2. Ensure all variables used in template are passed via `--var`
3. Use `{{if}}` checks before accessing optional variables

### Labels Not Applied

1. Verify `disable_label: false` in config
2. Check that all label conditions have a `label:` value
3. Ensure `gh` CLI is authenticated

### Comments Not Posted

1. Verify `disable_comment: false` in config
2. Check that `template:` is not empty
3. Use `-v` flag to populate `.Details` variable
4. Ensure `--post-comment` flag is present

## Best Practices

1. **Version control your config**: Commit `yamldiff.yaml` to your repository
2. **Use meaningful labels**: Choose labels that integrate with your workflow
3. **Keep templates concise**: Long comments can be hard to read
4. **Use collapsible sections**: Wrap detailed output in `<details>` tags
5. **Test templates locally**: Use `--config` without `--post-comment` first
6. **Document variables**: Add comments in config explaining custom variables
