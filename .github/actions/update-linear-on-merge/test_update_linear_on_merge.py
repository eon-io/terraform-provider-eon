#!/usr/bin/env python3
"""Self-check for ticket-ID extraction: python3 test_update_linear_on_merge.py"""

from update_linear_on_merge import extract_ticket_id


def main():
    assert extract_ticket_id("feat(provider): add a thing (EON-15403)", "some-branch") == "EON-15403"
    assert extract_ticket_id("no ticket here", "assafrabinowitz-eon-15403-add-a-thing") == "EON-15403"
    # Title wins over branch, so a PR retargeted to another ticket moves the one it names.
    assert extract_ticket_id("fix: thing (EON-1)", "user-eon-2-thing") == "EON-1"
    assert extract_ticket_id("chore: no ticket", "throwaway-branch") is None

    # The PR body is deliberately not a source: title and branch both carry the ID by convention,
    # and reading the body was what let a backtick in a description run as a shell command.
    try:
        extract_ticket_id("chore: no ticket", "throwaway-branch", "mentions EON-15403")
    except TypeError:
        pass
    else:
        raise AssertionError("extract_ticket_id must not accept a PR body")

    print("ok")


if __name__ == "__main__":
    main()
