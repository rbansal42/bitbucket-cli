# Authentication

This guide covers how to authenticate the `bb` CLI with Bitbucket Cloud.

## Overview

The `bb` CLI supports two authentication methods:

| Method | Best For | Setup Complexity |
|--------|----------|------------------|
| **OAuth** | Interactive use, full API access | Simple (browser-based) |
| **App Passwords** | CI/CD, scripts, automation | Moderate (manual setup) |

Both methods store credentials locally and support multiple Bitbucket accounts.

## Quick Start

Run the interactive login command:

```bash
bb auth login
```

This will guide you through authentication setup.

## Authentication Methods

### OAuth (Recommended)

OAuth provides the simplest authentication experience for interactive use.

#### How It Works

1. Run `bb auth login`
2. Select "OAuth" when prompted
3. A browser window opens to Bitbucket's authorization page
4. Authorize the `bb` CLI application
5. The CLI receives and stores your access token

```bash
$ bb auth login
? Select authentication method: OAuth
Opening browser for authentication...
Waiting for authorization...
✓ Logged in as yourname
```

#### OAuth Token Refresh

OAuth tokens expire after a period of time. The CLI automatically refreshes tokens using the stored refresh token, so you typically won't need to re-authenticate.

### App Passwords

App Passwords are ideal for:
- CI/CD pipelines
- Automated scripts
- Environments without a browser
- Fine-grained permission control

#### Creating an App Password

1. Log in to [Bitbucket Cloud](https://bitbucket.org)
2. Click your avatar (bottom-left) > **Personal settings**
3. Under **Access management**, click **App passwords**
4. Click **Create app password**
5. Enter a descriptive label (e.g., "bb CLI - work laptop")
6. Select the required permissions (see below)
7. Click **Create**
8. **Copy the password immediately** - it won't be shown again

#### Required Permissions

For full `bb` CLI functionality, enable these permissions:

| Permission | Required For |
|------------|--------------|
| **Account: Read** | `bb auth status`, user info |
| **Repositories: Read** | `bb repo list`, `bb repo view` |
| **Repositories: Write** | `bb repo clone`, `bb repo create` |
| **Pull requests: Read** | `bb pr list`, `bb pr view` |
| **Pull requests: Write** | `bb pr create`, `bb pr merge` |
| **Issues: Read** | `bb issue list`, `bb issue view` |
| **Issues: Write** | `bb issue create` |
| **Pipelines: Read** | `bb pipeline list`, `bb pipeline view` |
| **Pipelines: Write** | `bb pipeline run` |
| **Webhooks: Read & Write** | Webhook management |

**Minimal permissions** for read-only access:
- Account: Read
- Repositories: Read
- Pull requests: Read

#### Using an App Password

```bash
$ bb auth login
? Select authentication method: App Password
? Bitbucket username: yourname
? App password: ********
✓ Logged in as yourname
```

Or provide credentials directly:

```bash
bb auth login --username yourname --app-password your-app-password
```

## Token Storage

### Storage Location

Credentials are stored in:

```
~/.config/bb/hosts.yml
```

On Windows:

```
%APPDATA%\bb\hosts.yml
```

### File Format

```yaml
bitbucket.org:
  user: yourname
  oauth_token: your-oauth-token
  oauth_refresh_token: your-refresh-token
  # OR for app passwords:
  # app_password: your-app-password
```

### File Permissions

The CLI automatically sets restrictive permissions (0600) on the credentials file. Only you can read or write to it.

To verify:

```bash
ls -la ~/.config/bb/hosts.yml
# -rw------- 1 yourname yourname 256 Jan 15 10:30 hosts.yml
```

## Environment Variables

Override stored credentials using environment variables:

| Variable | Description |
|----------|-------------|
| `BB_TOKEN` | OAuth token or App Password |
| `BITBUCKET_TOKEN` | Alternative to `BB_TOKEN` |
| `BB_USERNAME` | Bitbucket username (required with App Passwords) |
| `BITBUCKET_USERNAME` | Alternative to `BB_USERNAME` |

### Examples

```bash
# Using an App Password
export BB_USERNAME=yourname
export BB_TOKEN=your-app-password
bb pr list

# Single command
BB_USERNAME=yourname BB_TOKEN=your-app-password bb pr list
```

### Precedence

The CLI checks for credentials in this order:

1. Environment variables (`BB_TOKEN` / `BITBUCKET_TOKEN`)
2. Stored credentials (`~/.config/bb/hosts.yml`)

Environment variables always take priority over stored credentials.

## Multiple Accounts

### Adding Additional Accounts

The `bb` CLI supports multiple Bitbucket accounts. Add accounts by logging in with different usernames:

```bash
# First account
bb auth login
# Login as: personal-account

# Second account
bb auth login
# Login as: work-account
```

### Switching Accounts

Use the `--account` flag to specify which account to use:

```bash
# List PRs using work account
bb pr list --account work-account

# Create repo using personal account
bb repo create my-project --account personal-account
```

### Setting Default Account

Set a default account for a repository:

```bash
bb config set account work-account
```

Or globally:

```bash
bb config set --global account personal-account
```

### Viewing Configured Accounts

```bash
$ bb auth status
bitbucket.org
  ✓ personal-account (default)
  ✓ work-account
```

## Checking Authentication Status

View your current authentication state:

```bash
$ bb auth status
bitbucket.org
  ✓ Logged in as yourname
  ✓ Token valid until 2024-03-15 14:30:00
  ✓ Scopes: repository, pullrequest, issue, pipeline
```

Check with verbose output:

```bash
bb auth status --verbose
```

## Logging Out

Remove stored credentials:

```bash
# Log out of default account
bb auth logout

# Log out of specific account
bb auth logout --account work-account

# Log out of all accounts
bb auth logout --all
```

## Troubleshooting

### "Authentication required" errors

```
Error: authentication required. Run 'bb auth login' to authenticate.
```

**Solutions:**
1. Run `bb auth login` to authenticate
2. Check if `~/.config/bb/hosts.yml` exists and is readable
3. Verify environment variables are set correctly

### "Bad credentials" or 401 errors

```
Error: 401 Unauthorized
```

**Solutions:**
1. Regenerate your App Password in Bitbucket settings
2. Re-authenticate: `bb auth login`
3. Check that your account has access to the repository

### "Forbidden" or 403 errors

```
Error: 403 Forbidden
```

**Solutions:**
1. Verify your App Password has the required permissions
2. Check repository access permissions in Bitbucket
3. Ensure you're using the correct account for the repository

### OAuth token refresh failures

```
Error: failed to refresh OAuth token
```

**Solutions:**
1. Log out and log back in: `bb auth logout && bb auth login`
2. Check your internet connection
3. Verify the OAuth application hasn't been revoked in Bitbucket settings

### Permission denied on credentials file

```
Error: permission denied reading ~/.config/bb/hosts.yml
```

**Solutions:**
```bash
# Fix file permissions
chmod 600 ~/.config/bb/hosts.yml

# Fix directory permissions
chmod 700 ~/.config/bb
```

### Environment variable not recognized

Ensure you're exporting variables correctly:

```bash
# Wrong - variable only set in subshell
BB_TOKEN=xxx
bb pr list  # Won't see the token

# Correct - export the variable
export BB_TOKEN=xxx
bb pr list

# Correct - inline for single command
BB_TOKEN=xxx bb pr list
```

## Security Best Practices

### Do

- **Use OAuth** for interactive sessions - tokens auto-expire and refresh
- **Use App Passwords** with minimal required permissions for automation
- **Rotate App Passwords** periodically and after team member departures
- **Use environment variables** in CI/CD instead of storing credentials in code
- **Review authorized applications** in Bitbucket settings regularly

### Don't

- **Don't commit credentials** to version control
- **Don't share App Passwords** between users or systems
- **Don't grant unnecessary permissions** to App Passwords
- **Don't store credentials** in shell history (use `--app-password` flag carefully)

### CI/CD Security

For CI/CD pipelines:

```yaml
# GitHub Actions example
- name: List open PRs
  env:
    BB_USERNAME: ${{ secrets.BITBUCKET_USERNAME }}
    BB_TOKEN: ${{ secrets.BITBUCKET_APP_PASSWORD }}
  run: bb pr list
```

```yaml
# Bitbucket Pipelines example
script:
  - export BB_USERNAME=$BITBUCKET_USERNAME
  - export BB_TOKEN=$BITBUCKET_APP_PASSWORD
  - bb pr list
```

### Auditing Access

Periodically review your App Passwords:

1. Go to Bitbucket > Personal settings > App passwords
2. Remove any unused or unrecognized passwords
3. Check the "Last used" date for each password
4. Revoke access for former team members or decommissioned systems

## Related Commands

- [`bb auth login`](/docs/commands/auth-login.md) - Authenticate with Bitbucket
- [`bb auth logout`](/docs/commands/auth-logout.md) - Remove stored credentials
- [`bb auth status`](/docs/commands/auth-status.md) - View authentication status
- [`bb config`](/docs/commands/config.md) - Manage CLI configuration
