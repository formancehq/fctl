"""Open one Homebrew PR after GoReleaser has uploaded both formulas."""

import hashlib
import json
from pathlib import Path
import re
import subprocess
from urllib.parse import urlencode


REPOSITORY = "repos/formancehq/homebrew-tap"


def api(path, payload=None):
    command = ["gh", "api", "--method", "POST" if payload else "GET", path]
    if payload:
        command += ["--input", "-"]
    result = subprocess.run(
        command,
        input=json.dumps(payload) if payload else None,
        text=True,
        capture_output=True,
        check=True,
    )
    return json.loads(result.stdout)


def tree(ref):
    result = api(f"{REPOSITORY}/git/trees/{ref}?recursive=1")
    if result.get("truncated"):
        raise RuntimeError("Cannot verify formulas: GitHub returned a truncated tree")
    return {entry["path"]: entry["sha"] for entry in result["tree"]}


def ensure_pull_request(version, formula_directory):
    if not re.fullmatch(r"[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.+-]+)?", version):
        raise ValueError(f"Invalid release version: {version!r}")
    expected = {}
    for name in ("fctl.rb", f"fctl@{version.split('.')[0]}.rb"):
        content = (formula_directory / name).read_bytes()
        blob = f"blob {len(content)}\0".encode() + content
        expected[f"Formula/{name}"] = hashlib.sha1(blob, usedforsecurity=False).hexdigest()

    # Compare contents, not ancestry: a squash merge leaves a different history.
    published = tree("main")
    if all(published.get(path) == sha for path, sha in expected.items()):
        print(f"Homebrew formulas for {version} are already published")
        return

    branch = f"fctl-{version}"
    head = api(f"{REPOSITORY}/git/ref/heads/{branch}")["object"]["sha"]
    uploaded = tree(head)
    if any(uploaded.get(path) != sha for path, sha in expected.items()):
        raise RuntimeError(f"Both generated formulas must be uploaded to {branch} before opening a PR")

    query = urlencode({"state": "open", "head": f"formancehq:{branch}", "base": "main"})

    def existing_pull_requests():
        return api(f"{REPOSITORY}/pulls?{query}")

    existing = existing_pull_requests()
    if existing:
        print(existing[0]["html_url"])
        return

    try:
        created = api(f"{REPOSITORY}/pulls", {
            "base": "main",
            "head": branch,
            "title": f"fix(homebrew): update fctl to v{version}",
            "body": f"Update fctl and fctl@{version.split('.')[0]} to v{version}.",
        })
    except subprocess.CalledProcessError:
        # Another invocation may have created the same PR, or the response was lost.
        existing = existing_pull_requests()
        if not existing:
            raise
        created = existing[0]
    print(created["html_url"])


if __name__ == "__main__":
    try:
        metadata = json.loads(Path("dist/metadata.json").read_text())
        ensure_pull_request(metadata["version"], Path("dist/homebrew/Formula"))
    except subprocess.CalledProcessError as error:
        raise SystemExit(error.stderr or str(error)) from error
