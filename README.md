# BuildBloatBuster: The Developer's Cleanup Tool

BuildBloatBuster is a fast, safe, and configurable CLI tool for cleaning up common development-related clutter from your system. It's designed to help you reclaim disk space by removing large, auto-generated folders like `node_modules`, `target`, `build`, and various cache directories.

## Key Features

- **Fast & Efficient:** Concurrently scans your filesystem and calculates directory sizes to quickly identify space hogs.
- **Safe by Default:**
    - **Dry-run mode** is enabled by default to show you what will be deleted before any action is taken.
    - **Quarantine system** moves deleted items to a temporary trash location (`~/.cache/BuildBloatBuster/trash`) instead of permanently deleting them, allowing for easy restoration.
    - **System path protection** prevents the tool from scanning or deleting from critical system directories.
- **Highly Configurable:** Use the default settings for a zero-config experience, or customize the tool's behavior with a `.BuildBloatBuster.yaml` file.
- **Interactive & User-Friendly:**
    - Clear, readable reports of deletable directories, sorted by size.
    - Interactive prompts to confirm deletions.
    - Progress bars for long-running operations.
- **Automation-Friendly:** Supports JSON output for integration with scripts and other tools.


## Disclaimer (Use at Your Own Risk)

While BuildBloatBuster is designed with multiple safety mechanisms (dry-run by default, quarantine instead of hard delete, system path protection), you use it entirely at your own risk. The author/maintainers assume no responsibility or liability for any data loss, corruption, or unintended side‑effects. Always:

- Start with a scan (`BuildBloatBuster scan` or `BuildBloatBuster clean` in dry-run mode) and review the report carefully.
- Confirm that no listed directory contains irreplaceable or untracked work before running with `--dry-run=false` / `-D`.
- Consider version control status (e.g. uncommitted changes, generated-but-modified assets).
- Keep backups or rely on your git repository before permanently purging the quarantine.

If you are uncertain about an entry, do not delete it—add it to `excludeNames` / `excludePaths` in your config first and re-run the scan.


## Usage

### Scanning for Deletable Directories

To see a report of what can be cleaned without deleting anything, run the `scan` command. You can scan the current directory or provide specific paths.

```bash
# Scan the current directory
BuildBloatBuster scan

# Scan a specific project folder
BuildBloatBuster scan ~/projects/my-app
```

### Cleaning Directories

The `clean` command will scan for deletable directories and then prompt you for confirmation before moving them to the quarantine.

```bash
# Run an interactive clean in the current directory
BuildBloatBuster clean
```

By default, `clean` runs in dry-run mode. To perform the actual deletion, use the `--dry-run=false` flag or its shorthand `-D`:

```bash
# Perform the clean operation (will prompt for confirmation)
BuildBloatBuster clean --dry-run=false

# A shorter way to do the same
BuildBloatBuster clean -D
```

To skip the confirmation prompt entirely, use the `--yes` or `-y` flag:

```bash
# Clean without interactive confirmation
BuildBloatBuster clean -D -y
```

### Restoring from Quarantine

If you accidentally delete something, you can easily restore it from the quarantine. Running the `restore` command will show you a list of quarantined items to choose from.

```bash
BuildBloatBuster restore
```

### Purging the Quarantine

To permanently delete items from the quarantine and free up the disk space, use the `purge` command.

```bash
# Purge all items from the quarantine
BuildBloatBuster purge

# Purge only items older than 30 days
BuildBloatBuster purge --days 30
```
**Warning:** This action is irreversible.

## Configuration

BuildBloatBuster can be configured using a `.BuildBloatBuster.yaml` file. The tool looks for this file in the current directory, and you can also have a global configuration at `~/.config/BuildBloatBuster/config.yaml`.

Here is an example configuration file:

```yaml
# .BuildBloatBuster.yaml

# Paths to scan. Defaults to the current directory.
scanPaths:
  - .

# Directory names to include in the scan.
includeNames:
  - "node_modules"
  - ".venv"
  - "venv"
  - ".pytest_cache"
  - "__pycache__"

# Directory names to always exclude.
excludeNames:
  - "src"
  - "lib"

# Full paths to always exclude from scanning.
excludePaths:
  - "/Applications"
  - "/Library"
  - "/System"
  - "~/Applications"

# Only report on directories larger than this size (in MB).
minSizeMB: 10

# Maximum depth to scan into directories.
maxDepth: 8

# Whether to follow symbolic links (not recommended).
followSymlinks: false

# Number of concurrent workers for size calculation.
concurrency: 16 # Defaults to NumCPU * 2

# Deletion settings.
delete:
  # "quarantine" (move to trash) or "rm" (permanent delete).
  mode: "quarantine"
  # Directory to move quarantined items to.
  quarantineDir: "~/.cache/BuildBloatBuster/trash"
  # How long to keep items in quarantine before they can be purged (in days).
  retentionDays: 14

# Output settings.
output:
  # "table" or "json".
  format: "table"
  # "size", "path", or "age".
  sortBy: "size"
```
