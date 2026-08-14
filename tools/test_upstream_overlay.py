import unittest

from tools.audit_upstream_overlay import validate_policy


class UpstreamOverlayPolicyTest(unittest.TestCase):
    def test_accepts_classified_fork_and_local_only_file(self):
        policy = {
            "version": 2,
            "forks": [
                {
                    "custom_path": "frontend/src/custom/WegooView.vue",
                    "upstream_path": "frontend/src/views/View.vue",
                    "reviewed_blob": "a" * 40,
                    "reviewed_local_blob": "b" * 40,
                }
            ],
            "local_only_paths": ["frontend/src/custom/sitePolicy.ts"],
        }

        failures = validate_policy(
            policy,
            actual_custom_paths={
                "frontend/src/custom/WegooView.vue",
                "frontend/src/custom/sitePolicy.ts",
            },
            resolve_upstream_blob=lambda _path: "a" * 40,
            resolve_local_blob=lambda _path: "b" * 40,
        )

        self.assertEqual(failures, [])

    def test_rejects_unclassified_custom_file(self):
        failures = validate_policy(
            {"version": 2, "forks": [], "local_only_paths": []},
            actual_custom_paths={"frontend/src/custom/untrackedPolicy.ts"},
            resolve_upstream_blob=lambda _path: "a" * 40,
            resolve_local_blob=lambda _path: "b" * 40,
        )

        self.assertIn(
            "unclassified custom production file: frontend/src/custom/untrackedPolicy.ts",
            failures,
        )

    def test_rejects_changed_upstream_blob(self):
        failures = validate_policy(
            {
                "version": 2,
                "forks": [
                    {
                        "custom_path": "frontend/src/custom/WegooView.vue",
                        "upstream_path": "frontend/src/views/View.vue",
                        "reviewed_blob": "a" * 40,
                        "reviewed_local_blob": "b" * 40,
                    }
                ],
                "local_only_paths": [],
            },
            actual_custom_paths={"frontend/src/custom/WegooView.vue"},
            resolve_upstream_blob=lambda _path: "b" * 40,
            resolve_local_blob=lambda _path: "b" * 40,
        )

        self.assertTrue(any("upstream source changed" in failure for failure in failures))

    def test_rejects_duplicate_classification(self):
        path = "frontend/src/custom/WegooView.vue"
        failures = validate_policy(
            {
                "version": 2,
                "forks": [
                    {
                        "custom_path": path,
                        "upstream_path": "frontend/src/views/View.vue",
                        "reviewed_blob": "a" * 40,
                        "reviewed_local_blob": "b" * 40,
                    }
                ],
                "local_only_paths": [path],
            },
            actual_custom_paths={path},
            resolve_upstream_blob=lambda _path: "a" * 40,
            resolve_local_blob=lambda _path: "b" * 40,
        )

        self.assertIn(f"custom path is classified more than once: {path}", failures)

    def test_rejects_changed_custom_fork(self):
        path = "frontend/src/custom/WegooView.vue"
        failures = validate_policy(
            {
                "version": 2,
                "forks": [
                    {
                        "custom_path": path,
                        "upstream_path": "frontend/src/views/View.vue",
                        "reviewed_blob": "a" * 40,
                        "reviewed_local_blob": "b" * 40,
                    }
                ],
                "local_only_paths": [],
            },
            actual_custom_paths={path},
            resolve_upstream_blob=lambda _path: "a" * 40,
            resolve_local_blob=lambda _path: "c" * 40,
        )

        self.assertTrue(any("custom fork changed" in failure for failure in failures))


if __name__ == "__main__":
    unittest.main()
