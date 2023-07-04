# Repository settings

All these settings are applied to this repository.

| Setting | Value | Info |
|-|-|-|
| Allow merge commits | `enabled` | Default to pull request title and description |
| Allow squash merging | `disabled` | |
| Allow rebase merging | `disabled` | |
| Always suggest updating pull request branches | `enabled` | |
| Allow auto-merge | `enabled` | |
| Automatically delete head branches | `enabled` | |
| Projects | `disabled` | |
| Issues | `disabled` | |
| Allow | `enabled` | |

## Add collaborators

Default GitHub teams will be added as collaborator in the repositories granting access to all project members to contribute to the onboarded repositories.

| Team | Role |
|-|-|
| [oauth-proxy-contributors](https://github.com/orgs/philips-forks/teams/oauth-proxy-contributors) | Maintain |

## Create protection rules

These are the protection rules for the `main` and `edi-foundation-integration` branches in order to secury and standarize the way of working.

| Rules | Value | Info |
|-|-|-|
| Require a pull request before merging | `Enabled` | N/A |
| Require approvals | `1` | N/A |
| Dismiss stale pull request approvals when new commits are pushed | `Enabled` | N/A |
| Require review from Code Owners | `Enabled` | N/A |
| Allow specified actors to bypass required pull requests | `Disabled` | N/A |
| Require status checks to pass before merging | `Enabled` | N/A |
| Require conversation resolution before merging | `Enabled` | N/A |
| Require signed commits | `Enabled` | N/A |
| Do not allow bypassing the above settings | `Enabled` | Enforce rules for admin users. |
| Restrict who can push to matching branches | `Disabled` | N/A |
| Allow force pushes | `Disabled` | N/A |
| Allow deletions | `Disabled` | N/A |
| Require linear history | `Disabled` | N/A |
| Require merge queue | `Disabled` | N/A |
| Require deployments to succeed before merging | `Disabled` | N/A |
| Lock branch | `Disabled` | N/A |

## Signed commits

It was added protection for all branches (pattern: `*`, `*/*` and `edi-foundation-integration`) in order to enforce signed commits. This configuration will help avoiding pull request completion issues because it'll enforce everyone to sign the commits in every branch not only in the main branch.
