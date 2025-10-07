# Release Automation

This project uses GoReleaser and GitHub Actions to automate releases and distribution through multiple channels.

## What Gets Published

When you push a version tag (e.g., `v1.0.0`), the following happens automatically:

1. **Binaries** - Built for all platforms (Linux, macOS, Windows) and published to GitHub Releases
2. **Docker Images** - Published to GitHub Container Registry (`ghcr.io/carlisia/mcp-factcheck`)
3. **MCP Registry** - Published to the [Model Context Protocol registry](https://github.com/modelcontextprotocol/servers) for easy installation in MCP clients

## Prerequisites

### Repository Permissions

The GitHub Actions workflow requires these permissions (already configured in `.github/workflows/goreleaser.yml`):

- `contents: write` - For creating releases
- `packages: write` - For pushing Docker images
- `id-token: write` - For MCP registry authentication via OIDC

## Creating a Release

### 1. Ensure everything is ready

```bash
# All changes committed
git status

# Tests passing
go test ./...

# Build succeeds
go build ./cmd/server
```

### 2. Create and push a version tag

```bash
# Create a tag
git tag -a v1.0.0 -m "Release version 1.0.0"

# Push the tag (this triggers the release)
git push origin v1.0.0
```

### 3. Monitor the release

- Go to Actions tab in GitHub
- Watch the "Release with GoReleaser" workflow
- Check for successful completion of all steps

## Testing Locally

Test the release process without publishing:

```bash
# Install GoReleaser (if not already installed)
# See https://goreleaser.com/install/ for installation options

# Test release process
goreleaser release --snapshot --clean --skip=publish

# Check the dist/ directory for generated artifacts
ls -la dist/
```

## Release Channels

### GitHub Releases

- **URL**: `https://github.com/carlisia/mcp-factcheck/releases`
- **Artifacts**: Platform-specific binaries (tar.gz/zip), checksums, changelog
- **Naming**: `mcp-factcheck_Darwin_arm64.tar.gz`, `mcp-factcheck_Linux_x86_64.tar.gz`, etc.

### Docker

- **Registry**: GitHub Container Registry
- **Images**:
  - `ghcr.io/carlisia/mcp-factcheck:latest`
  - `ghcr.io/carlisia/mcp-factcheck:v1.0.0` (full version)
  - `ghcr.io/carlisia/mcp-factcheck:v1` (major version)
  - `ghcr.io/carlisia/mcp-factcheck:v1.0` (minor version)

### MCP Registry

- **Name**: `mcp-factcheck`
- **Namespace**: `io.github.carlisia.mcp-factcheck`
- **Installation**: Available in MCP client marketplaces (Claude Desktop, etc.)
- **Authentication**: Automatic via GitHub OIDC (no manual login required)

## Configuration Files

### `.goreleaser.yml`

Controls the build and release process:

- Build targets (platforms, architectures)
- Archive formats and naming
- Changelog generation
- Docker image tags

### `server.json`

MCP registry metadata:

- Server description
- Deployment configuration
- Environment variables
- Package information

### `.github/workflows/goreleaser.yml`

GitHub Actions workflow that:

- Triggers on version tags
- Builds all artifacts
- Publishes to all channels
- Authenticates with MCP registry via OIDC

## Troubleshooting

### Release fails on GoReleaser step

Check:

- Tag format is `v*` (e.g., `v1.0.0`)
- All tests pass
- Build succeeds locally

### Docker push fails

Check:

- `GITHUB_TOKEN` has packages write permission (should be automatic)
- Dockerfile is valid
- Build succeeds locally with `docker build .`

### MCP registry publish fails

Check:

- `server.json` is valid
- Namespace matches GitHub repository (`io.github.carlisia.mcp-factcheck`)
- Workflow has `id-token: write` permission

## Version Numbering

Follow [Semantic Versioning](https://semver.org/):

- **MAJOR** (`v2.0.0`): Breaking changes
- **MINOR** (`v1.1.0`): New features, backward compatible
- **PATCH** (`v1.0.1`): Bug fixes, backward compatible

## Post-Release

After a successful release:

1. **Verify installations**:

   ```bash
   # Docker
   docker pull ghcr.io/carlisia/mcp-factcheck:latest

   # GitHub releases
   # Check https://github.com/carlisia/mcp-factcheck/releases
   ```

2. **Check MCP registry**: Search for "mcp-factcheck" in Claude Desktop

3. **Update documentation**: If needed, update README or CHANGELOG

4. **Announce**: Share release notes with users
