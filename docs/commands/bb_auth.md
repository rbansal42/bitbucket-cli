# bb auth

Authenticate bb with Bitbucket Cloud.

## Synopsis

```
bb auth <subcommand> [flags]
```

## Description

Manage authentication state for bb CLI. This includes logging in, logging out, and checking your current authentication status.

## Subcommands

- [bb auth login](#bb-auth-login) - Authenticate with Bitbucket
- [bb auth logout](#bb-auth-logout) - Log out of Bitbucket
- [bb auth status](#bb-auth-status) - View authentication status

---

# bb auth login

Authenticate with Bitbucket Cloud.

## Synopsis

```
bb auth login [flags]
```

## Description

Authenticate with Bitbucket Cloud using either OAuth 2.0 or a Repository Access Token.

By default, `bb auth login` starts an OAuth flow that opens your browser to complete authentication. This requires setting up an OAuth consumer first (see Authentication Guide).

Alternatively, you can authenticate using a Repository Access Token by passing the `--with-token` flag. Repository Access Tokens can be created in your repository settings under **Repository settings > Access tokens**.

The authentication token is stored securely in your system's credential store when available, or in a local configuration file.

## Flags

| Flag | Description |
|------|-------------|
| `--with-token` | Read token from standard input instead of using OAuth flow |
| `-w, --workspace <name>` | Set default workspace after login |
| `--scopes <scopes>` | Comma-separated list of OAuth scopes to request (OAuth flow only) |
| `-h, --help` | Show help for command |

## Examples

Authenticate interactively via OAuth:

```
$ bb auth login
```

Authenticate using a Repository Access Token:

```
$ bb auth login --with-token < token.txt
```

```
$ echo "$BITBUCKET_TOKEN" | bb auth login --with-token
```

Authenticate and set a default workspace:

```
$ bb auth login -w myworkspace
```

Request specific OAuth scopes:

```
$ bb auth login --scopes repository,pullrequest:write
```

## See also

- [bb auth logout](#bb-auth-logout) - Log out of Bitbucket
- [bb auth status](#bb-auth-status) - View authentication status

---

# bb auth logout

Log out of Bitbucket Cloud.

## Synopsis

```
bb auth logout [flags]
```

## Description

Remove authentication credentials for Bitbucket Cloud from the local system.

This command removes the stored access token and any associated credentials from your system's credential store or configuration file. After logging out, you will need to run `bb auth login` again to use authenticated commands.

## Flags

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help for command |

## Examples

Log out of Bitbucket:

```
$ bb auth logout
Logged out of Bitbucket Cloud
```

## See also

- [bb auth login](#bb-auth-login) - Authenticate with Bitbucket
- [bb auth status](#bb-auth-status) - View authentication status

---

# bb auth status

View authentication status.

## Synopsis

```
bb auth status [flags]
```

## Description

Display the current authentication state for the bb CLI.

Shows whether you are logged in, the username of the authenticated account, the associated workspace(s), and the validity of the stored token.

If the token has expired or is invalid, you will be prompted to re-authenticate using `bb auth login`.

## Flags

| Flag | Description |
|------|-------------|
| `-t, --show-token` | Display the authentication token |
| `-h, --help` | Show help for command |

## Examples

Check authentication status:

```
$ bb auth status
Bitbucket Cloud
  Logged in as: johndoe
  Username: johndoe
  Workspaces: myteam, personal
  Token valid: true
  Token expires: 2026-02-06 10:30:00 UTC
```

Show the authentication token:

```
$ bb auth status --show-token
Bitbucket Cloud
  Logged in as: johndoe
  Token: ****************************************abc123
```

## See also

- [bb auth login](#bb-auth-login) - Authenticate with Bitbucket
- [bb auth logout](#bb-auth-logout) - Log out of Bitbucket
