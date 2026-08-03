#!/usr/bin/env python3
"""
update_linear_on_merge.py
Update Linear ticket to Deployed state when a PR is merged.

Extracts the EON-XXXX ticket ID from the PR title or branch name, looks it up
in Linear, and moves it to the Deployed state.
"""

import os
import re
import sys
from typing import Optional

import requests

# Linear API configuration
LINEAR_API_BASE = "https://api.linear.app/graphql"
LINEAR_OAUTH_TOKEN_URL = "https://api.linear.app/oauth/token"
DEPLOYED_STATE_ID = "0e6659ab-646c-43e4-81b1-d858126bc390"
LINEAR_TICKET_PATTERN = re.compile(r"EON-(\d+)", re.IGNORECASE)


def extract_ticket_id(pr_title: str, pr_branch: str) -> Optional[str]:
    """Extract EON-XXXX ticket ID from PR title or branch."""
    for source in [pr_title, pr_branch]:
        match = LINEAR_TICKET_PATTERN.search(source)
        if match:
            return match.group(0).upper()
    return None


def get_oauth_access_token(client_id: str, client_secret: str) -> str:
    """Exchange OAuth client credentials for a Linear access token."""
    response = requests.post(
        LINEAR_OAUTH_TOKEN_URL,
        headers={"Content-Type": "application/x-www-form-urlencoded"},
        data={
            "grant_type": "client_credentials",
            "client_id": client_id,
            "client_secret": client_secret,
            "scope": "read,write",
        },
        timeout=30,
    )
    response.raise_for_status()

    access_token = response.json().get("access_token")
    if not access_token:
        print("No access_token in the Linear OAuth response.", file=sys.stderr)
        sys.exit(1)

    return access_token


def lookup_linear_issue(access_token: str, ticket_id: str) -> Optional[dict]:
    """Look up a Linear issue by identifier."""
    query = f'{{ issue(id: "{ticket_id}") {{ id identifier title state {{ name }} }} }}'

    response = requests.post(
        LINEAR_API_BASE,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {access_token}",
        },
        json={"query": query},
        timeout=30,
    )
    response.raise_for_status()

    data = response.json()

    if "errors" in data:
        for err in data["errors"]:
            msg = err.get("extensions", {}).get("userPresentableMessage", err.get("message", "Unknown error"))
            print(f"Warning: Could not find Linear issue {ticket_id}: {msg}", file=sys.stderr)
        return None

    issue = data.get("data", {}).get("issue")
    if not issue:
        print(f"Warning: No data returned for {ticket_id}", file=sys.stderr)
        return None

    return issue


def update_linear_issue(access_token: str, issue_id: str, ticket_id: str) -> bool:
    """Update a Linear issue to Deployed state."""
    mutation = f"""
    mutation {{
      issueUpdate(
        id: "{issue_id}",
        input: {{ stateId: "{DEPLOYED_STATE_ID}" }}
      ) {{
        success
        issue {{
          identifier
          title
          state {{ name }}
        }}
      }}
    }}
    """

    response = requests.post(
        LINEAR_API_BASE,
        headers={
            "Content-Type": "application/json",
            "Authorization": f"Bearer {access_token}",
        },
        json={"query": mutation},
        timeout=30,
    )
    response.raise_for_status()

    data = response.json()

    if "errors" in data:
        for err in data["errors"]:
            msg = err.get("extensions", {}).get("userPresentableMessage", err.get("message", "Unknown error"))
            print(f"Error updating {ticket_id}: {msg}", file=sys.stderr)
        return False

    result = data.get("data", {}).get("issueUpdate", {})
    if result.get("success"):
        issue = result.get("issue", {})
        print(f"Moved {issue.get('identifier')} - {issue.get('title')} -> {issue.get('state', {}).get('name')}")
        return True

    print(f"Failed to update {ticket_id}: {result}", file=sys.stderr)
    return False


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Update Linear ticket to Deployed on PR merge")
    parser.add_argument("--pr-title", required=True, help="Pull request title")
    parser.add_argument("--pr-branch", required=True, help="Pull request head branch name")
    args = parser.parse_args()

    # Credentials come from the environment, not argv: anything else on the runner can read
    # another process's command line out of /proc.
    client_id = os.environ.get("OAUTH_CLIENT_ID", "")
    client_secret = os.environ.get("OAUTH_CLIENT_SECRET", "")
    if not client_id or not client_secret:
        print("OAUTH_CLIENT_ID and OAUTH_CLIENT_SECRET are required.", file=sys.stderr)
        sys.exit(1)

    # Extract ticket ID
    ticket_id = extract_ticket_id(args.pr_title, args.pr_branch)
    if not ticket_id:
        print("No Linear ticket ID found in PR title or branch. Skipping.")
        return

    print(f"Found Linear ticket: {ticket_id}")

    access_token = get_oauth_access_token(client_id, client_secret)

    # Look up the issue
    issue = lookup_linear_issue(access_token, ticket_id)
    if not issue:
        return

    # Skip if already in a terminal state
    state_name = issue.get("state", {}).get("name", "")
    if state_name in ("Done", "Deployed"):
        print(f"Issue {ticket_id} is already {state_name}, skipping.")
        return

    # Update to Deployed
    print(f"Updating {ticket_id} from {state_name} to Deployed...")
    update_linear_issue(access_token, issue["id"], ticket_id)


if __name__ == "__main__":
    main()
