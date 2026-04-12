#!/usr/bin/env python3
"""
update_linear_on_merge.py
Update Linear ticket to Deployed state when a PR is merged.

Extracts the EON-XXXX ticket ID from the PR title, branch name, or body,
looks it up in Linear, and moves it to the Deployed state.
"""

import re
import sys
from typing import Optional

import requests

# Linear API configuration
LINEAR_API_BASE = "https://api.linear.app/graphql"
DEPLOYED_STATE_ID = "0e6659ab-646c-43e4-81b1-d858126bc390"
LINEAR_TICKET_PATTERN = re.compile(r"EON-(\d+)", re.IGNORECASE)


def extract_ticket_id(pr_title: str, pr_branch: str, pr_body: str) -> Optional[str]:
    """Extract EON-XXXX ticket ID from PR title, branch, or body."""
    for source in [pr_title, pr_branch, pr_body]:
        match = LINEAR_TICKET_PATTERN.search(source)
        if match:
            return match.group(0).upper()
    return None


def lookup_linear_issue(linear_api_key: str, ticket_id: str) -> Optional[dict]:
    """Look up a Linear issue by identifier."""
    query = f'{{ issue(id: "{ticket_id}") {{ id identifier title state {{ name }} }} }}'

    response = requests.post(
        LINEAR_API_BASE,
        headers={
            "Content-Type": "application/json",
            "Authorization": linear_api_key,
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


def update_linear_issue(linear_api_key: str, issue_id: str, ticket_id: str) -> bool:
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
            "Authorization": linear_api_key,
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
    parser.add_argument("--pr-body", default="", help="Pull request body")
    parser.add_argument("--linear-api-key", required=True, help="Linear API key")
    args = parser.parse_args()

    # Extract ticket ID
    ticket_id = extract_ticket_id(args.pr_title, args.pr_branch, args.pr_body)
    if not ticket_id:
        print("No Linear ticket ID found in PR title, branch, or body. Skipping.")
        return

    print(f"Found Linear ticket: {ticket_id}")

    # Look up the issue
    issue = lookup_linear_issue(args.linear_api_key, ticket_id)
    if not issue:
        return

    # Skip if already in a terminal state
    state_name = issue.get("state", {}).get("name", "")
    if state_name in ("Done", "Deployed"):
        print(f"Issue {ticket_id} is already {state_name}, skipping.")
        return

    # Update to Deployed
    print(f"Updating {ticket_id} from {state_name} to Deployed...")
    update_linear_issue(args.linear_api_key, issue["id"], ticket_id)


if __name__ == "__main__":
    main()
