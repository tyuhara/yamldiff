# yamldiff Project Structure

```
yamldiff/
├── cmd/
│   └── yamldiff/
│       └── main.go              # CLI entry point (using kong)
│
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration loader
│   │                            # - LoadConfig: Load yamldiff.yaml
│   │                            # - GetLabels: Determine labels based on changes
│   │
│   ├── diff/
│   │   └── diff.go              # Diff calculation engine
│   │                            # - Engine: Core of diff calculation
│   │                            # - Result: Representation of diff results
│   │                            # - Print/PrintSummary: Output functionality
│   │
│   ├── github/
│   │   └── github.go            # GitHub integration
│   │                            # - PostComment: Post comment to PR
│   │                            # - AddLabels: Add labels to PR
│   │                            # - RenderTemplate: Render comment template
│   │                            # - PrepareTemplateData: Prepare template data
│   │
│   └── parser/
│       └── parser.go            # YAML parser
│                                # - ParseMultiDocYAML: Parse multiple documents
│                                # - ExtractKey: Extract identifier
│                                # - CompareValues: Compare values
│
├── scripts/
│   ├── ci-integration-example.sh          # CI integration example
│   └── ci-integration-with-config.sh      # CI integration with config
│
├── .gitignore                   # Git ignore file
├── go.mod                       # Go module definition
├── go.sum                       # Dependency hashes
├── Makefile                     # Build task definitions
├── README.md                    # Project description
├── INSTALL.md                   # Installation guide
├── CONFIG_GUIDE.md              # Configuration guide
├── GITHUB_LABEL.md              # GitHub label integration guide
├── STRUCTURE.md                 # This file
├── test-old.yaml                # Test file (old)
├── test-new.yaml                # Test file (new)
├── yamldiff.yaml.example        # Example configuration file
└── yamldiff-microservices.yaml.example  # Example for microservices
```

## Package Dependencies

```
cmd/yamldiff (main)
    ↓
    ├─→ internal/config
    │       ↓
    │       └─→ gopkg.in/yaml.v3
    │
    ├─→ internal/diff
    │       ↓
    │       └─→ internal/parser
    │               ↓
    │               └─→ gopkg.in/yaml.v3
    │
    ├─→ internal/github
    │       ↓
    │       ├─→ internal/diff
    │       └─→ text/template
    │
    ├─→ internal/parser
    │       ↓
    │       └─→ gopkg.in/yaml.v3
    │
    └─→ github.com/alecthomas/kong
        github.com/fatih/color
```

## Main Functionality Flow

```
1. User Input
   └─→ kong parses CLI arguments

2. File Reading
   └─→ parser.ParseMultiDocYAML()
       └─→ Convert each document to Document struct

3. Diff Calculation
   └─→ diff.Engine.Compare()
       ├─→ Map documents by identifier
       ├─→ Detect added documents
       ├─→ Detect deleted documents
       └─→ Detect modified documents
           └─→ parser.CompareValues() for detailed diff

4. GitHub Integration (if configured)
   ├─→ Load config file (config.LoadConfig)
   │
   ├─→ Prepare template data (github.PrepareTemplateData)
   │   ├─→ Extract added/deleted/modified lists
   │   ├─→ Format summary
   │   └─→ Include custom variables
   │
   ├─→ Render template (github.RenderTemplate)
   │   └─→ Apply Go template with data
   │
   ├─→ Post comment (github.PostComment)
   │   └─→ Execute gh CLI command
   │
   └─→ Add labels (github.AddLabels)
       └─→ Execute gh CLI command

5. Result Output
   └─→ diff.Result.Print() or PrintSummary()
       └─→ Display with color output for readability
```

## Configuration System

```
yamldiff.yaml
    ↓
config.LoadConfig()
    ↓
    ├─→ Parse YAML structure
    ├─→ Load template string
    ├─→ Load label configurations
    │   ├─→ when_has_additions
    │   ├─→ when_has_deletions
    │   ├─→ when_has_modifications
    │   └─→ when_no_changes
    └─→ Load flags (disable_comment, disable_label)

During execution:
    ↓
config.GetLabels(added, deleted, modified)
    ↓
    ├─→ If no changes: return when_no_changes label
    └─→ If changes exist: return cumulative labels
        ├─→ added > 0 → when_has_additions
        ├─→ deleted > 0 → when_has_deletions
        └─→ modified > 0 → when_has_modifications
```

## Template System

```
Template Definition (in yamldiff.yaml)
    ↓
Available Variables:
    ├─→ .Summary        (e.g., "1 added, 2 deleted, 0 modified")
    ├─→ .Details        (verbose diff output)
    ├─→ .HasChanges     (true/false)
    ├─→ .Added          (number of added documents)
    ├─→ .Deleted        (number of deleted documents)
    ├─→ .Modified       (number of modified documents)
    ├─→ .AddedList      ([]string of added document names)
    ├─→ .DeletedList    ([]string of deleted document names)
    ├─→ .ModifiedList   ([]string of modified document names)
    ├─→ .Link           (CI build link, optional)
    └─→ .Vars           (custom variables, map[string]interface{})

Rendering Process:
    ↓
github.RenderTemplate(templateStr, data)
    ↓
text/template execution
    ↓
Formatted Markdown output
    ↓
Posted to GitHub PR as comment
```

## Similarities with expiry-monitor

This project adopts the same design pattern as `expiry-monitor`:

### Common Structure
- `cmd/<tool>/main.go`: CLI entry point
- `internal/`: Internal packages (not importable from outside)
- `github.com/alecthomas/kong`: CLI framework
- Subcommand-based design

### expiry-monitor structure (reference)
```
expiry-monitor/
├── cmd/
│   └── expiry-monitor/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   └── datadog/
│       ├── client.go
│       ├── metrics.go
│       └── monitor.go
└── ...
```

### yamldiff structure (this project)
```
yamldiff/
├── cmd/
│   └── yamldiff/
│       └── main.go
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── diff/
│   │   └── diff.go
│   ├── github/
│   │   └── github.go
│   └── parser/
│       └── parser.go
└── ...
```

## Extensibility

Examples of subcommands that can be added in the future:

```go
type CLI struct {
    Compare  CompareCmd  `cmd:"" help:"Compare two YAML files."`
    Validate ValidateCmd `cmd:"" help:"Validate YAML syntax."`
    Format   FormatCmd   `cmd:"" help:"Format YAML files."`
    Merge    MergeCmd    `cmd:"" help:"Merge multiple YAML files."`
}
```

This design allows easy addition of new features.

## Color Output System

```
internal/diff/diff.go
    ↓
Uses github.com/fatih/color
    ↓
    ├─→ Red: Deleted documents/lines
    ├─→ Green: Added documents/lines
    ├─→ Yellow: Modified documents
    └─→ Cyan: Document identifiers

Can be disabled with:
    ├─→ --no-color flag
    └─→ color.NoColor = true
```

## Error Handling

```
Main execution flow:
    ↓
    ├─→ File parsing errors
    │   └─→ Return fmt.Errorf with context
    │
    ├─→ Config loading errors
    │   └─→ Return fmt.Errorf with context
    │
    ├─→ GitHub API errors
    │   ├─→ PostComment failure
    │   │   └─→ Return error with gh output
    │   └─→ AddLabels failure
    │       └─→ Return error with gh output
    │
    └─→ Exit codes
        ├─→ 0: No differences or successful execution
        └─→ 1: Differences found (expected behavior)
```

## Testing Files

The project includes test files for validation:

```
test-old.yaml:
- Contains sample Kubernetes resources
- Used as baseline for comparison

test-new.yaml:
- Modified version of test-old.yaml
- Demonstrates different types of changes:
  ├─→ Added resources
  ├─→ Deleted resources
  └─→ Modified resources

Usage:
./yamldiff test-old.yaml test-new.yaml
```

## Development Workflow

```
1. Make changes to code
   └─→ Edit files in cmd/ or internal/

2. Install dependencies (if needed)
   └─→ make dev-deps

3. Build
   └─→ make build

4. Test
   └─→ ./yamldiff test-old.yaml test-new.yaml
   └─→ Test with actual YAML files

5. Install (optional)
   └─→ make install
   └─→ Installs to $GOPATH/bin
```
