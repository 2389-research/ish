#!/usr/bin/env python3
# ABOUTME: Example script demonstrating GitHub API integration with ISH.
# ABOUTME: Shows how to manage repos, issues, PRs, and comments using the ISH fake GitHub API.

import requests
from typing import Optional, List


class ISHGitHubClient:
    """Client for interacting with ISH's fake GitHub API."""

    def __init__(self, base_url: str = "http://localhost:9000", token: str = "gh_test_token"):
        self.base_url = base_url.rstrip("/")
        self.token = token
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Content-Type": "application/json"
        }

    def list_repos(self, affiliation: str = "owner,collaborator,organization_member"):
        """List user repositories."""
        url = f"{self.base_url}/user/repos"
        params = {"affiliation": affiliation}
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

    def get_repo(self, owner: str, repo: str):
        """Get a specific repository."""
        url = f"{self.base_url}/repos/{owner}/{repo}"
        response = requests.get(url, headers=self.headers)
        response.raise_for_status()
        return response.json()

    def list_issues(self, owner: str, repo: str, state: str = "open"):
        """List repository issues."""
        url = f"{self.base_url}/repos/{owner}/{repo}/issues"
        params = {"state": state}
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

    def create_issue(self, owner: str, repo: str, title: str, body: str = "", labels: Optional[List[str]] = None):
        """Create a new issue."""
        url = f"{self.base_url}/repos/{owner}/{repo}/issues"
        payload = {
            "title": title,
            "body": body
        }
        if labels:
            payload["labels"] = labels

        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def list_pull_requests(self, owner: str, repo: str, state: str = "open"):
        """List pull requests."""
        url = f"{self.base_url}/repos/{owner}/{repo}/pulls"
        params = {"state": state}
        response = requests.get(url, headers=self.headers, params=params)
        response.raise_for_status()
        return response.json()

    def create_pull_request(self, owner: str, repo: str, title: str, head: str, base: str, body: str = ""):
        """Create a new pull request."""
        url = f"{self.base_url}/repos/{owner}/{repo}/pulls"
        payload = {
            "title": title,
            "head": head,
            "base": base,
            "body": body
        }
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()

    def add_comment(self, owner: str, repo: str, issue_number: int, body: str):
        """Add a comment to an issue or PR."""
        url = f"{self.base_url}/repos/{owner}/{repo}/issues/{issue_number}/comments"
        payload = {"body": body}
        response = requests.post(url, headers=self.headers, json=payload)
        response.raise_for_status()
        return response.json()


def main():
    """Demonstrate GitHub API integration."""
    print("=" * 60)
    print("ISH GitHub API Integration Example")
    print("=" * 60)

    client = ISHGitHubClient()

    # 1. List repositories
    print("\n1. Listing repositories:")
    print("-" * 60)
    try:
        repos = client.list_repos()
        print(f"  Found {len(repos)} repositories")
        for repo in repos[:5]:
            print(f"\n  📦 {repo.get('full_name', 'Unknown')}")
            print(f"     Description: {repo.get('description', 'No description')}")
            print(f"     Language: {repo.get('language', 'Unknown')}")
            print(f"     Stars: ⭐ {repo.get('stargazers_count', 0)}")
            print(f"     Forks: 🍴 {repo.get('forks_count', 0)}")
            print(f"     Private: {'🔒 Yes' if repo.get('private') else '🌐 No'}")
    except Exception as e:
        print(f"  Error: {e}")

    # 2. Get specific repository details
    print("\n2. Getting repository details:")
    print("-" * 60)
    if repos:
        try:
            first_repo = repos[0]
            owner, repo_name = first_repo["full_name"].split("/")
            detailed = client.get_repo(owner, repo_name)
            print(f"  Repository: {detailed.get('full_name')}")
            print(f"  Created: {detailed.get('created_at')}")
            print(f"  Updated: {detailed.get('updated_at')}")
            print(f"  Default Branch: {detailed.get('default_branch', 'main')}")
            print(f"  Open Issues: {detailed.get('open_issues_count', 0)}")
            print(f"  Watchers: {detailed.get('watchers_count', 0)}")
        except Exception as e:
            print(f"  Error: {e}")

    # 3. List issues
    print("\n3. Listing open issues:")
    print("-" * 60)
    if repos:
        try:
            owner, repo_name = repos[0]["full_name"].split("/")
            issues = client.list_issues(owner, repo_name, state="open")
            print(f"  Found {len(issues)} open issues in {repos[0]['full_name']}")
            for issue in issues[:5]:
                print(f"\n  #{issue.get('number')} {issue.get('title', 'Untitled')}")
                print(f"     State: {issue.get('state', 'unknown')}")
                print(f"     Author: {issue.get('user', {}).get('login', 'Unknown')}")
                if issue.get('labels'):
                    labels = [l.get('name', '') for l in issue['labels']]
                    print(f"     Labels: {', '.join(labels)}")
                if issue.get('body'):
                    print(f"     Body: {issue['body'][:80]}...")
        except Exception as e:
            print(f"  Error: {e}")

    # 4. Create a new issue
    print("\n4. Creating a new issue:")
    print("-" * 60)
    if repos:
        try:
            owner, repo_name = repos[0]["full_name"].split("/")
            new_issue = client.create_issue(
                owner=owner,
                repo=repo_name,
                title="Add dark mode support",
                body="We should add a dark mode theme option to improve user experience in low-light environments.",
                labels=["enhancement", "ui"]
            )
            print(f"  Issue created successfully!")
            print(f"  Number: #{new_issue.get('number')}")
            print(f"  Title: {new_issue.get('title')}")
            print(f"  URL: {new_issue.get('html_url', 'N/A')}")
        except Exception as e:
            print(f"  Error: {e}")

    # 5. List pull requests
    print("\n5. Listing pull requests:")
    print("-" * 60)
    if repos:
        try:
            owner, repo_name = repos[0]["full_name"].split("/")
            prs = client.list_pull_requests(owner, repo_name, state="open")
            print(f"  Found {len(prs)} open pull requests")
            for pr in prs[:5]:
                print(f"\n  #{pr.get('number')} {pr.get('title', 'Untitled')}")
                print(f"     {pr.get('head', {}).get('ref', '?')} → {pr.get('base', {}).get('ref', '?')}")
                print(f"     State: {pr.get('state', 'unknown')}")
                print(f"     Author: {pr.get('user', {}).get('login', 'Unknown')}")
                if pr.get('draft'):
                    print(f"     Status: 📝 Draft")
        except Exception as e:
            print(f"  Error: {e}")

    # 6. Create a pull request
    print("\n6. Creating a pull request:")
    print("-" * 60)
    if repos:
        try:
            owner, repo_name = repos[0]["full_name"].split("/")
            new_pr = client.create_pull_request(
                owner=owner,
                repo=repo_name,
                title="feat: implement user authentication",
                head="feature/auth",
                base="main",
                body="""## Changes
- Added login/logout endpoints
- Implemented JWT token generation
- Added user session management
- Updated API documentation

## Testing
- Added unit tests for auth endpoints
- Tested with Postman collection
- All tests passing ✅"""
            )
            print(f"  Pull request created!")
            print(f"  Number: #{new_pr.get('number')}")
            print(f"  Title: {new_pr.get('title')}")
            print(f"  Branch: {new_pr.get('head', {}).get('ref')} → {new_pr.get('base', {}).get('ref')}")
        except Exception as e:
            print(f"  Error: {e}")

    # 7. Add comment to issue
    print("\n7. Adding comment to issue:")
    print("-" * 60)
    if repos and issues:
        try:
            owner, repo_name = repos[0]["full_name"].split("/")
            issue_num = issues[0]["number"]
            comment = client.add_comment(
                owner=owner,
                repo=repo_name,
                issue_number=issue_num,
                body="I can help with this! I'll start working on it this week."
            )
            print(f"  Comment added to issue #{issue_num}")
            print(f"  Author: {comment.get('user', {}).get('login', 'Unknown')}")
            print(f"  Body: {comment.get('body', '')[:60]}...")
        except Exception as e:
            print(f"  Error: {e}")

    # 8. List closed issues
    print("\n8. Listing recently closed issues:")
    print("-" * 60)
    if repos:
        try:
            owner, repo_name = repos[0]["full_name"].split("/")
            closed_issues = client.list_issues(owner, repo_name, state="closed")
            print(f"  Found {len(closed_issues)} closed issues")
            for issue in closed_issues[:3]:
                print(f"  ✓ #{issue.get('number')} {issue.get('title', 'Untitled')}")
        except Exception as e:
            print(f"  Error: {e}")

    print("\n" + "=" * 60)
    print("Example complete!")
    print("=" * 60)


if __name__ == "__main__":
    main()
