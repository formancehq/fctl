import hashlib
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import homebrew_pr


class HomebrewPublicationTests(unittest.TestCase):
    def setUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.directory.cleanup)
        self.formulas = Path(self.directory.name)
        self.uploaded = {}
        for name in ("fctl.rb", "fctl@3.rb"):
            content = f'formula {name} version "3.11.0"\n'.encode()
            (self.formulas / name).write_bytes(content)
            self.uploaded[f"Formula/{name}"] = hashlib.sha1(
                f"blob {len(content)}\0".encode() + content, usedforsecurity=False
            ).hexdigest()
        self.published = {}
        self.pull_requests = []
        self.creations = []
        self.create_error = None
        self.concurrent_creation = False
        self.truncated = False
        self.failure = subprocess.CalledProcessError(1, ["gh", "api"], stderr="GitHub failure")
        self.api = patch.object(homebrew_pr, "api", side_effect=self.github).start()
        self.addCleanup(patch.stopall)
        self.output = patch("builtins.print").start()

    def github(self, path, payload=None):
        prefix = homebrew_pr.REPOSITORY
        if path == f"{prefix}/git/ref/heads/fctl-3.11.0":
            return {"object": {"sha": "release-head"}}
        if path.startswith(f"{prefix}/git/trees/"):
            if path.endswith("/release-head?recursive=1"):
                entries = self.uploaded
            elif path.endswith("/main?recursive=1"):
                entries = self.published
            else:
                self.fail(f"Unexpected tree request: {path}")
            return {
                "truncated": self.truncated,
                "tree": [{"path": path, "sha": sha} for path, sha in entries.items()],
            }
        if path.startswith(f"{prefix}/pulls?"):
            self.assertIn("head=formancehq%3Afctl-3.11.0", path)
            self.assertIn("state=open", path)
            self.assertIn("base=main", path)
            return self.pull_requests.copy()
        if path == f"{prefix}/pulls" and payload:
            self.creations.append(payload)
            created = {"html_url": "https://github.com/formancehq/homebrew-tap/pull/7"}
            if self.concurrent_creation:
                self.pull_requests.append(created)
                raise self.failure
            if self.create_error:
                raise self.create_error
            self.pull_requests.append(created)
            return created
        self.fail(f"Unexpected GitHub request: {path}")

    def publish(self):
        homebrew_pr.ensure_pull_request("3.11.0", self.formulas)

    def test_both_formulas_create_one_pr_and_retry_reuses_it(self):
        self.publish()
        self.publish()
        self.assertEqual(len(self.creations), 1)
        self.assertEqual(self.creations[0]["base"], "main")
        self.assertEqual(self.creations[0]["head"], "fctl-3.11.0")
        self.assertEqual(self.output.call_count, 2)

    def test_partial_or_stale_upload_never_creates_a_pr(self):
        complete = self.uploaded.copy()
        for path in complete:
            for missing in (True, False):
                with self.subTest(path=path, missing=missing):
                    self.uploaded = complete.copy()
                    if missing:
                        del self.uploaded[path]
                    else:
                        self.uploaded[path] = "old-version"
                    with self.assertRaisesRegex(RuntimeError, "Both generated formulas"):
                        self.publish()
        self.assertEqual(self.creations, [])

    def test_already_squash_merged_formulas_are_a_noop(self):
        self.published = {**self.uploaded, "Formula/ledger.rb": "unrelated-change"}
        self.publish()
        self.assertEqual(self.creations, [])
        self.assertFalse(any("/pulls" in call.args[0] for call in self.api.call_args_list))

    def test_merged_formulas_do_not_require_the_release_branch(self):
        self.published = self.uploaded.copy()
        self.uploaded = {}
        self.publish()
        self.assertEqual(len(self.api.call_args_list), 1)
        self.assertIn("/git/trees/main?recursive=1", self.api.call_args.args[0])

    def test_crash_after_uploads_before_pr_is_recoverable(self):
        self.create_error = self.failure
        with self.assertRaises(subprocess.CalledProcessError):
            self.publish()
        self.create_error = None
        self.publish()
        self.assertEqual(len(self.pull_requests), 1)

    def test_concurrent_create_or_lost_response_reuses_existing_pr(self):
        self.concurrent_creation = True
        self.publish()
        self.assertEqual(len(self.pull_requests), 1)
        self.output.assert_called_once_with(self.pull_requests[0]["html_url"])

    def test_genuine_creation_failure_is_not_ignored(self):
        self.create_error = self.failure
        with self.assertRaises(subprocess.CalledProcessError):
            self.publish()

    def test_truncated_tree_never_creates_a_pr(self):
        self.truncated = True
        with self.assertRaisesRegex(RuntimeError, "truncated tree"):
            self.publish()
        self.assertEqual(self.creations, [])

    def test_missing_local_formula_never_contacts_github(self):
        (self.formulas / "fctl@3.rb").unlink()
        with self.assertRaises(FileNotFoundError):
            self.publish()
        self.api.assert_not_called()

    def test_invalid_version_never_contacts_github(self):
        with self.assertRaises(ValueError):
            homebrew_pr.ensure_pull_request("../main", self.formulas)
        self.api.assert_not_called()


class GitHubCommandTests(unittest.TestCase):
    @patch("homebrew_pr.subprocess.run")
    def test_payload_uses_json_stdin(self, run):
        run.return_value.stdout = '{"html_url":"pr-url"}'
        self.assertEqual(homebrew_pr.api("repos/example/pulls", {"head": "release"}), {"html_url": "pr-url"})
        self.assertEqual(run.call_args.args[0], ["gh", "api", "--method", "POST", "repos/example/pulls", "--input", "-"])
        self.assertEqual(run.call_args.kwargs["input"], '{"head": "release"}')
        self.assertTrue(run.call_args.kwargs["check"])


if __name__ == "__main__":
    unittest.main()
