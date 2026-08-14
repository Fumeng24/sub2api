import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))

import upstream_delta


class UpstreamDeltaTest(unittest.TestCase):
    def test_generated_paths_exclude_handwritten_ent_schema(self):
        self.assertTrue(upstream_delta.is_generated_path("backend/ent/mutation.go"))
        self.assertTrue(upstream_delta.is_generated_path("backend/cmd/server/wire_gen.go"))
        self.assertFalse(upstream_delta.is_generated_path("backend/ent/schema/group.go"))
        self.assertFalse(upstream_delta.is_generated_path("backend/internal/service/group.go"))

    def test_check_requires_real_ancestry_and_preserves_upstream_files(self):
        report = {
            "upstream_tip_is_ancestor": False,
            "upstream_only_commits": 3,
            "unaccounted_missing_upstream_commits": 1,
            "deleted_upstream_files": 2,
            "handwritten_upstream_overlap_files": 458,
        }

        failures = upstream_delta.check_failures(report)

        self.assertEqual(len(failures), 4)

    def test_check_can_allow_reviewed_upstream_deletion(self):
        report = {
            "upstream_tip_is_ancestor": True,
            "upstream_only_commits": 0,
            "unaccounted_missing_upstream_commits": 0,
            "deleted_upstream_files": 1,
            "handwritten_upstream_overlap_files": 458,
        }

        self.assertEqual(
            upstream_delta.check_failures(report, allow_upstream_deletions=True),
            [],
        )

    def test_check_rejects_handwritten_overlap_regression(self):
        report = {
            "upstream_tip_is_ancestor": True,
            "upstream_only_commits": 0,
            "unaccounted_missing_upstream_commits": 0,
            "deleted_upstream_files": 0,
            "handwritten_upstream_overlap_files": 459,
        }

        self.assertEqual(
            upstream_delta.check_failures(report, max_handwritten_overlap=458),
            ["handwritten upstream overlap increased to 459; maximum allowed is 458"],
        )

    def test_check_accepts_current_handwritten_overlap_budget(self):
        report = {
            "upstream_tip_is_ancestor": True,
            "upstream_only_commits": 0,
            "unaccounted_missing_upstream_commits": 0,
            "deleted_upstream_files": 0,
            "handwritten_upstream_overlap_files": 458,
        }

        self.assertEqual(
            upstream_delta.check_failures(report, max_handwritten_overlap=458),
            [],
        )


if __name__ == "__main__":
    unittest.main()
