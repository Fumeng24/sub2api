import unittest

from tools.audit_upstream_owned import ModifiedPath, RenamedPath, Snapshot, validate_policy


def snapshot(*, local_blob="b" * 40, upstream_blob="a" * 40):
    return Snapshot(
        modified=(ModifiedPath("official.go", upstream_blob, local_blob),),
        renamed=(RenamedPath("old.sql", "new.sql", "c" * 40, "d" * 40),),
        deleted=(),
    )


def policy():
    return {
        "version": 1,
        "modified": [
            {
                "path": "official.go",
                "upstream_blob": "a" * 40,
                "local_blob": "b" * 40,
            }
        ],
        "renamed": [
            {
                "upstream_path": "old.sql",
                "local_path": "new.sql",
                "upstream_blob": "c" * 40,
                "local_blob": "d" * 40,
            }
        ],
    }


class UpstreamOwnedPolicyTest(unittest.TestCase):
    def test_accepts_exact_reviewed_blobs(self):
        self.assertEqual(validate_policy(policy(), snapshot()), [])

    def test_rejects_new_official_file_override(self):
        actual = Snapshot(
            modified=snapshot().modified + (ModifiedPath("new.go", "e" * 40, "f" * 40),),
            renamed=snapshot().renamed,
            deleted=(),
        )
        self.assertIn(
            "unreviewed upstream-owned modification: new.go",
            validate_policy(policy(), actual),
        )

    def test_rejects_changed_upstream_source(self):
        failures = validate_policy(policy(), snapshot(upstream_blob="e" * 40))
        self.assertIn("upstream source changed for reviewed override: official.go", failures)

    def test_rejects_changed_local_override(self):
        failures = validate_policy(policy(), snapshot(local_blob="f" * 40))
        self.assertIn("local reviewed override changed: official.go", failures)

    def test_rejects_stale_and_deleted_entries(self):
        actual = Snapshot(modified=(), renamed=(), deleted=("removed.go",))
        failures = validate_policy(policy(), actual)
        self.assertIn("stale upstream-owned modification policy entry: official.go", failures)
        self.assertIn("stale upstream-owned rename policy entry: old.sql -> new.sql", failures)
        self.assertIn("upstream-owned path deleted locally: removed.go", failures)


if __name__ == "__main__":
    unittest.main()
