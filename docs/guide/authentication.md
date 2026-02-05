# Authentication

This guide covers how to authenticate the `bb` CLI with Bitbucket Cloud.

## Overview

The `bb` CLI supports two authentication methods:

| Method | Best For | Setup Complexity |
|--------|----------|------------------|
| **OAuth 2.0** | Interactive use, full API access | Medium (one-time setup) |
| **Repository Access Token** | CI/CD, scripts, single repo access | Easy |

> **Important:** Bitbucket has deprecated App Passwords. Use OAuth or Repository Access Tokens instead.

## Quick Start

### For Interactive Use (OAuth)

```bash
# 1. Set up OAuth consumer (one-time, see detailed instructions below)
export BB_OAUTH_CLIENT_ID="your_client_id"
export BB_OAUTH_CLIENT_SECRET="your_client_secret"

# 2. Login
bb auth login
```

### For CI/CD (Repository Access Token)

```bash
echo "$BITBUCKET_TOKEN" | bb auth login --with-token
```

---

## OAuth 2.0 Authentication

OAuth is the recommended method for interactive use. It requires a one-time setup of an "OAuth consumer" in Bitbucket.

### Step 1: Create an OAuth Consumer

1. Go to your **Workspace Settings**:
   ```
   https://bitbucket.org/YOUR_WORKSPACE/workspace/settings/oauth-consumers
   ```

2. Click **"Add consumer"**

3. Configure the consumer:
   - **Name:** `bb CLI` (or any descriptive name)
   - **Callback URL:** `http://localhost:8372/callback`
   - **This is a private consumer:** ✓ Check this box

4. Select **Permissions** based on what you need:

   | Permission | Commands |
   |------------|----------|
   | Account: Read | `bb auth status`, user info |
   | Repositories: Read | `bb repo list`, `bb repo view`, `bb repo clone` |
   | Repositories: Write | `bb repo create`, `bb repo fork` |
   | Repositories: Admin | `bb repo delete` |
   | Pull requests: Read | `bb pr list`, `bb pr view`, `bb pr diff` |
   | Pull requests: Write | `bb pr create`, `bb pr merge`, `bb pr review` |
   | Issues: Read | `bb issue list`, `bb issue view` |
   | Issues: Write | `bb issue create`, `bb issue close` |
   | Pipelines: Read | `bb pipeline list`, `bb pipeline view`, `bb pipeline logs` |
   | Pipelines: Write | `bb pipeline run`, `bb pipeline stop` |
   | Snippets: Read | `bb snippet list`, `bb snippet view` |
   | Snippets: Write | `bb snippet create`, `bb snippet delete` |

5. Click **"Save"**

6. Copy the **Key** (Client ID) and **Secret** (Client Secret) shown

### Step 2: Configure Environment Variables

Add to your shell profile (`~/.bashrc`, `~/.zshrc`, `~/.config/fish/config.fish`):

```bash
export BB_OAUTH_CLIENT_ID="your_key_here"
export BB_OAUTH_CLIENT_SECRET="your_secret_here"
```

Reload your shell:

```bash
source ~/.zshrc  # or ~/.bashrc
```

### Step 3: Authenticate

```bash
bb auth login
```

This will:
1. Open your browser to Bitbucket's authorization page
2. Ask you to grant permissions
3. Redirect back to complete authentication
4. Store tokens securely

### Verify

```bash
bb auth status
```

### Token Refresh

OAuth tokens expire (typically after 2 hours). The CLI automatically refreshes them using the stored refresh token. If refresh fails, re-run `bb auth login`.

---

## Repository Access Tokens

Repository Access Tokens are scoped to a single repository, making them ideal for CI/CD pipelines.

### Create a Repository Access Token

1. Go to your repository:
   ```
   https://bitbucket.org/WORKSPACE/REPO/admin/access-tokens
   ```

2. Click **"Create Repository Access Token"**

3. Configure:
   - **Name:** Descriptive name (e.g., `ci-pipeline`)
   - **Scopes:** Select required permissions

4. Click **"Create"** and **copy the token immediately**

### Use the Token

```bash
# Interactive
bb auth login --with-token
# Paste your token when prompted

# Non-interactive (CI/CD)
echo "$BITBUCKET_TOKEN" | bb auth login --with-token

# Or use environment variable directly (no login needed)
export BB_TOKEN="your_repository_access_token"
bb pr list
```

---

## Token Storage

### Storage Locations

| OS | Primary Storage | Fallback |
|----|-----------------|----------|
| macOS | Keychain | `~/.config/bb/hosts.yml` |
| Linux | Secret Service (GNOME Keyring, KWallet) | `~/.config/bb/hosts.yml` |
| Windows | Credential Manager | `%APPDATA%\bb\hosts.yml` |

### hosts.yml Format

```yaml
bitbucket.org:
  user: yourname
  oauth_token: <stored securely>
```

The CLI sets restrictive permissions (0600) on the credentials file.

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `BB_TOKEN` | Access token (highest priority) |
| `BITBUCKET_TOKEN` | Alternative token variable |
| `BB_OAUTH_CLIENT_ID` | OAuth consumer key |
| `BB_OAUTH_CLIENT_SECRET` | OAuth consumer secret |

### Precedence Order

1. `BB_TOKEN` environment variable
2. `BITBUCKET_TOKEN` environment variable  
3. Stored OAuth token (from `bb auth login`)

---

## CI/CD Examples

### GitHub Actions

```yaml
jobs:
  bitbucket-sync:
    runs-on: ubuntu-latest
    steps:
      - name: Install bb CLI
        run: go install github.com/rbansal42/bitbucket-cli/cmd/bb@latest

      - name: Authenticate
        run: echo "${{ secrets.BITBUCKET_TOKEN }}" | bb auth login --with-token

      - name: List PRs
        run: bb pr list --repo myworkspace/myrepo
```

### Bitbucket Pipelines

```yaml
pipelines:
  default:
    - step:
        script:
          - go install github.com/rbansal42/bitbucket-cli/cmd/bb@latest
          - export BB_TOKEN=$REPOSITORY_ACCESS_TOKEN
          - bb pipeline list
```

### GitLab CI

```yaml
bitbucket-integration:
  script:
    - go install github.com/rbansal42/bitbucket-cli/cmd/bb@latest
    - echo "$BITBUCKET_TOKEN" | bb auth login --with-token
    - bb pr create --title "Sync from GitLab"
```

---

## Troubleshooting

### "OAuth client credentials not configured"

You haven't set up the OAuth consumer environment variables:

```bash
export BB_OAUTH_CLIENT_ID="your_key"
export BB_OAUTH_CLIENT_SECRET="your_secret"
```

See [Step 1: Create an OAuth Consumer](#step-1-create-an-oauth-consumer).

### "invalid token" error

- Token may have expired
- Token doesn't have required permissions
- Try re-authenticating: `bb auth login`

### "authorization failed: access_denied"

You denied the permission request in the browser. Run `bb auth login` again and click "Grant access".

### 401 Unauthorized

- Token is invalid or expired
- Re-authenticate: `bb auth logout && bb auth login`

### 403 Forbidden

- Token doesn't have required permissions
- Check OAuth consumer permissions or create a new token with correct scopes

---

## Logging Out

Remove stored credentials:

```bash
bb auth logout
```

---

## Security Best Practices

### Do

- Use **OAuth** for interactive sessions
- Use **Repository Access Tokens** (scoped to one repo) for CI/CD
- Store tokens in CI/CD **secret management** (not in code)
- **Rotate tokens** periodically
- Use **minimal permissions** for your use case

### Don't

- Don't commit tokens to version control
- Don't share tokens between users or systems
- Don't use overly broad permissions
- Don't store tokens in shell history

---

## Related

- [Configuration Guide](configuration.md)
- [Troubleshooting Guide](troubleshooting.md)
- [bb auth command reference](../commands/bb_auth.md)
