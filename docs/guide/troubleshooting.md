# Troubleshooting

This guide covers common issues you may encounter when using the `bb` CLI and how to resolve them.

## Authentication Problems

### "not authenticated" Error

**Problem:** You see an error like `error: not authenticated. Run 'bb auth login' to authenticate.`

**Solutions:**

1. Run the login command:
   ```bash
   bb auth login
   ```

2. Verify your authentication status:
   ```bash
   bb auth status
   ```

3. If you have multiple accounts, ensure you're using the correct one:
   ```bash
   bb auth status --show-token
   ```

### Token Expired

**Problem:** Commands fail with authentication errors even though you previously logged in.

**Solutions:**

1. Re-authenticate to refresh your token:
   ```bash
   bb auth login
   ```

2. If using an app password, generate a new one in Bitbucket:
   - Go to **Personal Settings** > **App passwords**
   - Create a new app password with the required permissions
   - Run `bb auth login` and enter the new password

### Wrong Permissions

**Problem:** You see `403 Forbidden` errors or "insufficient permissions" messages.

**Solutions:**

1. Check your current token's scopes:
   ```bash
   bb auth status
   ```

2. Re-authenticate with the required permissions:
   ```bash
   bb auth login --scopes repository,pullrequest,pipeline
   ```

3. For app passwords, ensure these permissions are enabled:
   - **Repository:** Read, Write
   - **Pull requests:** Read, Write
   - **Pipelines:** Read, Write
   - **Account:** Read

---

## Repository Detection Issues

### "could not detect repository" Error

**Problem:** You see `error: could not detect repository. Run this command from a git repository or use --repo flag.`

**Solutions:**

1. Ensure you're in a git repository:
   ```bash
   git status
   ```

2. Check if the remote is configured:
   ```bash
   git remote -v
   ```

3. Explicitly specify the repository:
   ```bash
   bb pr list --repo workspace/repo-name
   ```

4. If the remote URL is non-standard, set it manually:
   ```bash
   git remote set-url origin git@bitbucket.org:workspace/repo.git
   ```

### Non-Bitbucket Remotes

**Problem:** The CLI doesn't recognize your repository because the remote points to a different host.

**Solutions:**

1. Add a Bitbucket remote:
   ```bash
   git remote add bitbucket git@bitbucket.org:workspace/repo.git
   ```

2. Use the `--repo` flag to specify the Bitbucket repository:
   ```bash
   bb pr list --repo workspace/repo-name
   ```

3. Configure `bb` to use a specific remote:
   ```bash
   bb config set remote bitbucket
   ```

---

## API Errors

### Rate Limiting

**Problem:** You see `error: API rate limit exceeded` or HTTP 429 errors.

**Solutions:**

1. Wait for the rate limit to reset (usually 1 hour)

2. Check your current rate limit status:
   ```bash
   bb api /user --include-headers | grep -i rate
   ```

3. Reduce the frequency of API calls in scripts:
   ```bash
   # Add delays between calls
   for repo in repo1 repo2 repo3; do
     bb pr list --repo "workspace/$repo"
     sleep 2
   done
   ```

### 404 Not Found

**Problem:** You see `error: HTTP 404: Not Found` when accessing a resource.

**Solutions:**

1. Verify the repository exists and you have access:
   ```bash
   bb repo view workspace/repo-name
   ```

2. Check for typos in workspace or repository names (they are case-sensitive)

3. Ensure the resource (PR, issue, pipeline) exists:
   ```bash
   bb pr view 123  # Check if PR #123 exists
   ```

4. Confirm you have access to private repositories

### 403 Forbidden

**Problem:** You see `error: HTTP 403: Forbidden` when performing an action.

**Solutions:**

1. Verify you have the required permissions on the repository

2. Re-authenticate with broader scopes:
   ```bash
   bb auth login --scopes repository:admin,pullrequest:write
   ```

3. Check if the repository has branch restrictions preventing your action

4. For workspace-level operations, ensure you have workspace admin access

---

## Network Issues

### Proxy Configuration

**Problem:** Connections fail when behind a corporate proxy.

**Solutions:**

1. Set the proxy environment variables:
   ```bash
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   export NO_PROXY=localhost,127.0.0.1
   ```

2. Configure git to use the proxy:
   ```bash
   git config --global http.proxy http://proxy.example.com:8080
   git config --global https.proxy http://proxy.example.com:8080
   ```

3. For authenticated proxies:
   ```bash
   export HTTPS_PROXY=http://username:password@proxy.example.com:8080
   ```

### SSL/TLS Errors

**Problem:** You see certificate verification errors like `SSL certificate problem: unable to get local issuer certificate`.

**Solutions:**

1. Update your CA certificates:
   ```bash
   # macOS
   brew install ca-certificates

   # Ubuntu/Debian
   sudo apt-get update && sudo apt-get install ca-certificates

   # Fedora/RHEL
   sudo dnf install ca-certificates
   ```

2. If using a corporate CA, add it to your trust store:
   ```bash
   # macOS
   sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain corp-ca.crt

   # Linux
   sudo cp corp-ca.crt /usr/local/share/ca-certificates/
   sudo update-ca-certificates
   ```

3. **Not recommended for production:** Skip certificate verification:
   ```bash
   bb config set insecure true
   ```

---

## Git-Related Issues

### SSH vs HTTPS Problems

**Problem:** Git operations fail with authentication errors.

**Solutions:**

1. Check which protocol your remote uses:
   ```bash
   git remote -v
   ```

2. Switch from HTTPS to SSH:
   ```bash
   git remote set-url origin git@bitbucket.org:workspace/repo.git
   ```

3. Switch from SSH to HTTPS:
   ```bash
   git remote set-url origin https://bitbucket.org/workspace/repo.git
   ```

4. For SSH, ensure your key is added to the agent:
   ```bash
   ssh-add -l  # List loaded keys
   ssh-add ~/.ssh/id_ed25519  # Add your key
   ```

5. Test SSH connectivity:
   ```bash
   ssh -T git@bitbucket.org
   ```

### Clone Failures

**Problem:** `bb repo clone` fails to clone a repository.

**Solutions:**

1. Verify you have access to the repository:
   ```bash
   bb repo view workspace/repo-name
   ```

2. Check your SSH configuration:
   ```bash
   ssh -vT git@bitbucket.org
   ```

3. Try cloning with HTTPS instead:
   ```bash
   bb repo clone workspace/repo-name --protocol https
   ```

4. Ensure you have enough disk space

5. For large repositories, try a shallow clone:
   ```bash
   bb repo clone workspace/repo-name -- --depth 1
   ```

---

## Pipeline Issues

### Pipeline Not Triggering

**Problem:** Push events don't trigger pipelines as expected.

**Solutions:**

1. Verify pipelines are enabled for the repository:
   ```bash
   bb repo view --json | jq '.pipelines_enabled'
   ```

2. Check if `bitbucket-pipelines.yml` exists and is valid:
   ```bash
   bb pipeline validate
   ```

3. View recent pipeline runs:
   ```bash
   bb pipeline list
   ```

4. Manually trigger a pipeline:
   ```bash
   bb pipeline run --branch main
   ```

5. Check branch patterns in your pipeline configuration match your branch name

### Log Viewing Problems

**Problem:** Unable to view pipeline logs or logs appear incomplete.

**Solutions:**

1. Ensure the pipeline has completed or is running:
   ```bash
   bb pipeline view 123
   ```

2. View logs for a specific step:
   ```bash
   bb pipeline logs 123 --step "Build"
   ```

3. For long logs, use pagination:
   ```bash
   bb pipeline logs 123 --step "Build" | less
   ```

4. Download full logs:
   ```bash
   bb pipeline logs 123 --step "Build" > build.log
   ```

---

## Getting Debug Output

Enable verbose output to diagnose issues:

```bash
# Basic verbose output
bb --verbose pr list

# Full debug output including HTTP requests
bb --debug pr list

# Log to a file for sharing
bb --debug pr list 2>&1 | tee debug.log
```

Debug output includes:
- HTTP request/response details
- API endpoints being called
- Authentication method used
- Configuration values

Environment variable alternative:
```bash
export BB_DEBUG=1
bb pr list
```

---

## Reporting Bugs

If you encounter a bug, please report it with the following information:

### 1. Gather System Information

```bash
bb --version
git --version
uname -a  # or systeminfo on Windows
```

### 2. Reproduce with Debug Output

```bash
bb --debug <command-that-fails> 2>&1 | tee bug-report.log
```

### 3. Redact Sensitive Information

Before sharing logs, remove:
- Authentication tokens
- Private repository names (if sensitive)
- Personal information

```bash
# Redact tokens from debug output
sed -i 's/Bearer [A-Za-z0-9_-]*/Bearer [REDACTED]/g' bug-report.log
```

### 4. Submit the Report

Open an issue on the project repository with:
- Description of the problem
- Steps to reproduce
- Expected behavior
- Actual behavior
- Debug output (redacted)
- System information

---

## FAQ

### How do I update bb to the latest version?

```bash
# If installed via Homebrew
brew upgrade bb

# If installed via go install
go install github.com/your-org/bb@latest

# Check current version
bb --version
```

### How do I authenticate with multiple Bitbucket accounts?

```bash
# Login to different accounts
bb auth login --hostname bitbucket.org --user work-account
bb auth login --hostname bitbucket.org --user personal-account

# Switch between accounts
bb auth switch work-account

# List all authenticated accounts
bb auth status --all
```

### How do I use bb in CI/CD environments?

Set the `BB_TOKEN` environment variable:

```bash
export BB_TOKEN=your-app-password
bb pr list --repo workspace/repo
```

Or use `BB_USERNAME` and `BB_APP_PASSWORD`:

```bash
export BB_USERNAME=your-username
export BB_APP_PASSWORD=your-app-password
```

### How do I configure bb for my organization?

Create a configuration file at `~/.config/bb/config.yml`:

```yaml
default_workspace: my-workspace
default_protocol: ssh
editor: vim
pager: less
```

Or use `bb config`:

```bash
bb config set default_workspace my-workspace
bb config set default_protocol ssh
```

### Why is bb slow?

Common causes and solutions:

1. **Network latency:** Use `--cache` for repeated queries
   ```bash
   bb pr list --cache
   ```

2. **Large responses:** Use filters to reduce data
   ```bash
   bb pr list --state open --limit 10
   ```

3. **Debug to identify bottlenecks:**
   ```bash
   bb --debug pr list 2>&1 | grep "took"
   ```

### How do I reset bb to default settings?

```bash
# Remove configuration
rm -rf ~/.config/bb

# Remove cached credentials (be careful!)
bb auth logout --all

# Start fresh
bb auth login
```

### Can I use bb with Bitbucket Server (self-hosted)?

Yes, specify your server's hostname:

```bash
bb auth login --hostname bitbucket.mycompany.com
```

Then use the `--hostname` flag or set it in config:

```bash
bb config set default_hostname bitbucket.mycompany.com
```

### How do I contribute to bb?

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

See the CONTRIBUTING.md file in the repository for detailed guidelines.
