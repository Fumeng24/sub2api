import os
import sys
import unittest

sys.path.insert(0, os.path.dirname(__file__))

import upstream_sync


class UpstreamSyncTest(unittest.TestCase):
    def test_topology_accepts_exact_upstream_and_local_parents(self):
        self.assertEqual(
            upstream_sync.topology_failures(
                ["upstream", "local"],
                upstream_sha="upstream",
                local_sha="local",
                upstream_ref="upstream/main",
            ),
            [],
        )

    def test_topology_rejects_local_ref_that_is_not_second_parent(self):
        failures = upstream_sync.topology_failures(
            ["upstream", "intermediate"],
            upstream_sha="upstream",
            local_sha="local",
            upstream_ref="upstream/main",
        )

        self.assertEqual(len(failures), 1)
        self.assertIn("second parent", failures[0])

    def test_topology_requires_exactly_two_parents(self):
        failures = upstream_sync.topology_failures(
            ["upstream"],
            upstream_sha="upstream",
            local_sha="local",
            upstream_ref="upstream/main",
        )

        self.assertEqual(failures, ["candidate must have exactly two parents, found 1"])


if __name__ == "__main__":
    unittest.main()
